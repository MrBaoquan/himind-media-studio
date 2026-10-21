package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type deliveryPreset struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	MaxLongSide  int     `json:"max_long_side,omitempty"`
	Width        int     `json:"width,omitempty"`
	Height       int     `json:"height,omitempty"`
	CRF          int     `json:"crf"`
	AudioBitrate string  `json:"audio_bitrate"`
	MaxRate      string  `json:"max_rate,omitempty"`
	Container    string  `json:"container"`
	LoudnessLUFS float64 `json:"loudness_lufs"`
}

var deliveryPresets = []deliveryPreset{
	{ID: "master", Name: "母版", Description: "最高质量存档，不缩放画布", CRF: 16, AudioBitrate: "256k", Container: "mp4", LoudnessLUFS: -16},
	{ID: "web", Name: "Web 预览", Description: "长边 1280，快速分享", MaxLongSide: 1280, CRF: 21, AudioBitrate: "160k", Container: "mp4", LoudnessLUFS: -16},
	{ID: "douyin", Name: "抖音竖屏", Description: "1080×1920，H.264，高峰值码率", Width: 1080, Height: 1920, CRF: 20, AudioBitrate: "192k", MaxRate: "8M", Container: "mp4", LoudnessLUFS: -14},
	{ID: "bilibili", Name: "B 站", Description: "按画布比例输出 1080 长边", MaxLongSide: 1920, CRF: 19, AudioBitrate: "192k", Container: "mp4", LoudnessLUFS: -16},
	{ID: "youtube", Name: "YouTube", Description: "按画布比例输出 1080 长边，响度 -14 LUFS", MaxLongSide: 1920, CRF: 18, AudioBitrate: "192k", Container: "mp4", LoudnessLUFS: -14},
	{ID: "xigua", Name: "西瓜视频", Description: "1080 长边，稳健码率", MaxLongSide: 1920, CRF: 20, AudioBitrate: "192k", Container: "mp4", LoudnessLUFS: -15},
}

func deliveryPresetByID(id string) deliveryPreset {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, preset := range deliveryPresets {
		if preset.ID == id {
			return preset
		}
	}
	return deliveryPresets[0]
}

func deliveryPresetCatalogCapability() any {
	return map[string]any{"schema": "himind.delivery-presets.v1", "presets": deliveryPresets, "total": len(deliveryPresets)}
}

func deliveryDimensions(preset deliveryPreset, canvas timelineCanvas) (int, int) {
	width, height := canvas.Width, canvas.Height
	if width <= 0 || height <= 0 {
		width, height = 1080, 1920
	}
	if preset.Width > 0 && preset.Height > 0 {
		return preset.Width, preset.Height
	}
	if preset.MaxLongSide <= 0 || (width <= preset.MaxLongSide && height <= preset.MaxLongSide) {
		return evenDimension(width), evenDimension(height)
	}
	if width >= height {
		newWidth := preset.MaxLongSide
		newHeight := int(float64(height) * float64(newWidth) / float64(width))
		return evenDimension(newWidth), evenDimension(newHeight)
	}
	newHeight := preset.MaxLongSide
	newWidth := int(float64(width) * float64(newHeight) / float64(height))
	return evenDimension(newWidth), evenDimension(newHeight)
}

func evenDimension(value int) int {
	if value < 2 {
		return 2
	}
	if value%2 != 0 {
		return value - 1
	}
	return value
}

func timelineQAEvaluation(root string, in requestInput) map[string]any {
	doc := in.Timeline
	blockers := append([]string(nil), validateTimeline(doc)...)
	warnings := make([]string, 0)
	checks := make([]map[string]any, 0)
	checks = append(checks, map[string]any{"id": "structure", "status": qaStatusFor(len(blockers)), "blockers": len(blockers)})
	videoTracks := 0
	videoClips := 0
	audioTracks := 0
	subtitleCues := 0
	missingMedia := make([]string, 0)
	for _, track := range doc.Tracks {
		switch track.Kind {
		case "video":
			videoTracks++
			videoClips += len(track.Clips)
		case "audio":
			audioTracks++
		case "subtitle":
			subtitleCues += len(track.Clips)
		}
		for _, clip := range track.Clips {
			if track.Kind == "subtitle" || clip.MediaID == "" {
				continue
			}
			path := strings.TrimSpace(in.MediaPaths[clip.MediaID])
			if path == "" {
				missingMedia = append(missingMedia, clip.MediaID)
				continue
			}
			if _, err := workspaceFile(root, path); err != nil {
				missingMedia = append(missingMedia, clip.MediaID)
			}
		}
	}
	if videoTracks == 0 || videoClips == 0 {
		blockers = append(blockers, "VIDEO_TRACK_EMPTY")
	}
	if len(missingMedia) > 0 {
		sort.Strings(missingMedia)
		missingMedia = uniqueStrings(missingMedia)
		blockers = append(blockers, "MEDIA_PATH_MISSING: "+strings.Join(missingMedia, ","))
	}
	checks = append(checks, map[string]any{"id": "media", "status": qaStatusFor(len(missingMedia)), "missing": missingMedia})
	for _, track := range doc.Tracks {
		if track.Kind != "video" || len(track.Clips) < 2 {
			continue
		}
		clips := append([]timelineClip(nil), track.Clips...)
		sort.SliceStable(clips, func(i, j int) bool { return clips[i].StartFrame < clips[j].StartFrame })
		for index := 1; index < len(clips); index++ {
			if clips[index].StartFrame > clipEnd(clips[index-1]) {
				warnings = append(warnings, fmt.Sprintf("TRACK_GAP: %s %df", track.ID, clips[index].StartFrame-clipEnd(clips[index-1])))
			}
		}
	}
	checks = append(checks, map[string]any{"id": "sync", "status": qaStatusFor(len(warnings))})
	checks = append(checks, map[string]any{"id": "subtitles", "status": "passed", "cues": subtitleCues})
	passed := len(blockers) == 0
	return map[string]any{
		"passed": passed, "blockers": uniqueStrings(blockers), "warnings": uniqueStrings(warnings),
		"checks": checks, "video_tracks": videoTracks, "video_clips": videoClips,
		"audio_tracks": audioTracks, "subtitle_cues": subtitleCues,
	}
}

func qaStatusFor(count int) string {
	if count > 0 {
		return "failed"
	}
	return "passed"
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func timelineQACapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	return timelineQAEvaluation(root, in), nil
}

func deliveryQACapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	pathText := strings.TrimSpace(in.Path)
	if pathText == "" && strings.TrimSpace(in.OutputPath) != "" {
		pathText = in.OutputPath
	}
	if pathText == "" {
		return nil, himindjsonrpc.InvalidParams("path is required")
	}
	path, err := workspaceFile(root, pathText)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("path: " + err.Error())
	}
	info, err := os.Stat(path)
	if err != nil {
		return map[string]any{"state": "missing", "passed": false, "blockers": []string{"OUTPUT_NOT_FOUND"}}, nil
	}
	preset := deliveryPresetByID(in.DeliveryPreset)
	probe, probeErr := probeMediaRef(root, "delivery-output", pathText)
	checks := make([]map[string]any, 0)
	blockers := make([]string, 0)
	warnings := make([]string, 0)
	if probeErr != nil {
		blockers = append(blockers, "PROBE_FAILED: "+probeErr.Error())
	} else {
		checks = append(checks, map[string]any{"id": "container", "status": "passed", "kind": probe.Kind, "width": probe.Width, "height": probe.Height, "duration_frames": probe.DurationFrames})
		if info.Size() == 0 {
			blockers = append(blockers, "OUTPUT_EMPTY")
		}
		if probe.Kind != "video" {
			blockers = append(blockers, "OUTPUT_NOT_VIDEO")
		}
	}
	audio, audioErr := analyzeAudio(path, 0)
	if audioErr == nil {
		lufs := audio.IntegratedLUFS
		peak := audio.TruePeakDBTP
		status := "passed"
		if lufs != 0 && (lufs < preset.LoudnessLUFS-3 || lufs > preset.LoudnessLUFS+2) {
			status = "warning"
			warnings = append(warnings, fmt.Sprintf("LOUDNESS_OUT_OF_TARGET: %.1f LUFS target %.1f", lufs, preset.LoudnessLUFS))
		}
		if peak > -0.8 {
			status = "warning"
			warnings = append(warnings, fmt.Sprintf("TRUE_PEAK_HIGH: %.1f dBFS", peak))
		}
		checks = append(checks, map[string]any{"id": "audio_loudness", "status": status, "lufs": lufs, "peak_dbfs": peak})
	}
	return map[string]any{
		"state": "ready", "passed": len(blockers) == 0, "blockers": blockers, "warnings": warnings,
		"checks": checks, "preset": preset.ID, "path": filepath.ToSlash(pathText), "size": info.Size(),
	}, nil
}
