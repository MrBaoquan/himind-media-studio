package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type ffprobeOutput struct {
	Streams []struct {
		CodecType  string `json:"codec_type"`
		CodecName  string `json:"codec_name"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"`
		ColorSpace string `json:"color_space"`
		Duration   string `json:"duration"`
		SampleRate string `json:"sample_rate"`
		Channels   int    `json:"channels"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		Size     string `json:"size"`
	} `json:"format"`
}

func probeMediaRef(root, id, relativePath string) (timelineMediaRef, error) {
	path, err := workspaceFile(root, relativePath)
	if err != nil {
		return timelineMediaRef{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return timelineMediaRef{}, err
	}
	if info.IsDir() {
		return timelineMediaRef{}, fmt.Errorf("media path is a directory")
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return timelineMediaRef{}, fmt.Errorf("ffprobe not found in PATH")
	}
	command := exec.Command(ffprobe, "-v", "error", "-print_format", "json", "-show_streams", "-show_format", path)
	output, err := command.Output()
	if err != nil {
		return timelineMediaRef{}, fmt.Errorf("ffprobe failed: %w", err)
	}
	var probe ffprobeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return timelineMediaRef{}, err
	}
	ref := timelineMediaRef{ID: id, URI: filepath.ToSlash(relativePath), ColorSpace: "sdr"}
	kind := ""
	duration, _ := strconv.ParseFloat(probe.Format.Duration, 64)
	for _, stream := range probe.Streams {
		switch stream.CodecType {
		case "video":
			kind = "video"
			ref.Width = stream.Width
			ref.Height = stream.Height
			ref.FPS = stream.RFrameRate
			if stream.ColorSpace != "" {
				ref.ColorSpace = stream.ColorSpace
			}
			if duration <= 0 {
				duration, _ = strconv.ParseFloat(stream.Duration, 64)
			}
		case "audio":
			if kind == "" {
				kind = "audio"
			}
			if duration <= 0 {
				duration = numberFloat(stream.Duration)
			}
		}
	}
	if kind == "" && strings.HasPrefix(strings.ToLower(filepath.Ext(path)), ".png") {
		kind = "image"
	}
	ref.Kind = kind
	fps := parseRate(ref.FPS)
	if fps <= 0 {
		fps = 30
	}
	ref.DurationFrames = int(duration * fps)
	if ref.DurationFrames <= 0 {
		ref.DurationFrames = 1
	}
	if hash, hashErr := fileSHA256(path); hashErr == nil {
		ref.SHA256 = hash
	}
	return ref, nil
}

func parseRate(value string) float64 {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) == 2 {
		num, _ := strconv.ParseFloat(parts[0], 64)
		den, _ := strconv.ParseFloat(parts[1], 64)
		if den > 0 {
			return num / den
		}
	}
	rate, _ := strconv.ParseFloat(value, 64)
	return rate
}

func mediaPathsForRequest(root string, in requestInput) (map[string]string, *himindjsonrpc.Error) {
	paths := map[string]string{}
	for id, path := range in.MediaPaths {
		if strings.TrimSpace(id) == "" || strings.TrimSpace(path) == "" {
			continue
		}
		if _, err := workspaceFile(root, path); err != nil {
			return nil, himindjsonrpc.InvalidParams("media_paths[" + id + "]: " + err.Error())
		}
		paths[id] = path
	}
	if len(paths) == 0 && strings.TrimSpace(in.ProjectID) != "" {
		for _, item := range projectArtifacts(root, in.ProjectID) {
			if !item.Ready {
				continue
			}
			if !strings.HasPrefix(item.Kind, "video/") && !strings.HasPrefix(item.Kind, "image/") && !strings.HasPrefix(item.Kind, "audio/") {
				continue
			}
			paths[item.ID] = item.Path
		}
	}
	return paths, nil
}

func timelineMediaCatalogCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	paths, rpcErr := mediaPathsForRequest(root, in)
	if rpcErr != nil {
		return nil, rpcErr
	}
	ids := make([]string, 0, len(paths))
	for id := range paths {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]timelineMediaRef, 0, len(ids))
	errors := make([]map[string]any, 0)
	for _, id := range ids {
		ref, probeErr := probeMediaRef(root, id, paths[id])
		if probeErr != nil {
			errors = append(errors, map[string]any{"id": id, "message": probeErr.Error()})
			continue
		}
		items = append(items, ref)
	}
	return map[string]any{"state": "ready", "items": items, "total": len(items), "errors": errors}, nil
}

func timelineProxyBuildCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	paths, rpcErr := mediaPathsForRequest(root, in)
	if rpcErr != nil {
		return nil, rpcErr
	}
	if len(paths) == 0 {
		return map[string]any{"state": "empty", "items": []any{}, "blockers": []string{"MEDIA_PATHS_EMPTY"}}, nil
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, himindjsonrpc.InternalError("ffmpeg not found in PATH")
	}
	width := in.ProxyWidth
	if width <= 0 {
		width = 540
	}
	proxyRoot := filepath.Join(root, metadataDir, "proxies")
	if err := os.MkdirAll(proxyRoot, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	ids := make([]string, 0, len(paths))
	for id := range paths {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		source, _ := workspaceFile(root, paths[id])
		output := filepath.Join(proxyRoot, sanitizeTimelineID(id)+"-proxy.mp4")
		command := exec.Command(ffmpeg, "-hide_banner", "-loglevel", "error", "-y", "-i", source,
			"-vf", fmt.Sprintf("scale=%d:-2", width), "-c:v", "libx264", "-preset", "veryfast", "-crf", "26",
			"-c:a", "aac", "-b:a", "96k", output)
		if outputBytes, runErr := command.CombinedOutput(); runErr != nil {
			items = append(items, map[string]any{"id": id, "state": "failed", "message": strings.TrimSpace(string(outputBytes))})
			continue
		}
		rel, _ := filepath.Rel(root, output)
		items = append(items, map[string]any{"id": id, "state": "ready", "proxy_uri": filepath.ToSlash(rel), "width": width})
	}
	return map[string]any{"state": "ready", "items": items, "total": len(items)}, nil
}
