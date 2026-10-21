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
	"time"

	"himind-plugin/media-visual/internal/himindjsonrpc"
)

type videoProbe struct {
	Path     string  `json:"path"`
	Duration float64 `json:"duration_seconds"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	FPS      string  `json:"fps"`
	Codec    string  `json:"codec"`
}

type frameQA struct {
	Time     float64          `json:"time_seconds"`
	Frame    string           `json:"frame"`
	OK       bool             `json:"ok"`
	Metrics  imageMetrics     `json:"metrics"`
	Blockers []map[string]any `json:"blockers"`
	Warnings []map[string]any `json:"warnings"`
}

func videoQACapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	videoPath, err := workspaceFile(root, in.VideoPath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("video_path: " + err.Error())
	}
	info, statErr := os.Stat(videoPath)
	if statErr != nil || info.IsDir() || info.Size() == 0 {
		return nil, himindjsonrpc.InvalidParams("video_path must be a non-empty file")
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, himindjsonrpc.InternalError("ffmpeg not found in PATH")
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, himindjsonrpc.InternalError("ffprobe not found in PATH")
	}
	probe, probeErr := probeVideo(ffprobe, videoPath)
	if probeErr != nil {
		return nil, himindjsonrpc.InvalidParams("probe video: " + probeErr.Error())
	}
	times := parseFrameTimes(in.Times)
	if len(times) == 0 {
		times = defaultFrameTimes(probe.Duration, in.SampleCount)
	}
	if len(times) == 0 {
		return nil, himindjsonrpc.InvalidParams("no valid sample times; provide times or a video with duration > 0")
	}
	if len(times) > 12 {
		times = times[:12]
	}
	tempDir := filepath.Join(root, ".himind-media", "qa", fmt.Sprintf("video-%d", time.Now().UnixNano()))
	if mkErr := os.MkdirAll(tempDir, 0755); mkErr != nil {
		return nil, himindjsonrpc.InternalError(mkErr.Error())
	}
	if !in.KeepFrames {
		defer os.RemoveAll(tempDir)
	}
	frames := make([]frameQA, 0, len(times))
	var blockerList []map[string]any
	var warningList []map[string]any
	worstBanding := -1.0
	worstBandingTime := 0.0
	worstBlockiness := 0.0
	maxClipHigh := 0.0
	minSharpness := 1.0
	for index, at := range times {
		framePath := filepath.Join(tempDir, fmt.Sprintf("frame-%02d.png", index+1))
		if extractErr := extractVideoFrame(ffmpeg, videoPath, at, framePath); extractErr != nil {
			blocker := issue("VIDEO_FRAME_EXTRACT_FAILED", "frame", fmt.Sprintf("%.2fs 抽帧失败: %v", at, extractErr), "检查视频编码与时间点，或改用更小的采样时间")
			blockerList = append(blockerList, blocker)
			frames = append(frames, frameQA{Time: at, Frame: "", OK: false, Blockers: []map[string]any{blocker}})
			continue
		}
		img, decodeErr := decodeImage(framePath)
		if decodeErr != nil {
			blocker := issue("VIDEO_FRAME_DECODE_FAILED", "frame", fmt.Sprintf("%.2fs 抽帧无法解码: %s", at, decodeErr.Error()), "改用 PNG 中间帧或检查 ffmpeg 输出")
			blockerList = append(blockerList, blocker)
			frames = append(frames, frameQA{Time: at, Frame: workspaceRelative(root, framePath), OK: false, Blockers: []map[string]any{blocker}})
			continue
		}
		metrics := analyzeImage(img)
		frameBlockers, frameWarnings := evaluateImageMetrics(metrics, fmt.Sprintf("%.2f", at))
		blockerList = append(blockerList, frameBlockers...)
		warningList = append(warningList, frameWarnings...)
		if metrics.BandingScore > worstBanding {
			worstBanding = metrics.BandingScore
			worstBandingTime = at
		}
		if metrics.Blockiness > worstBlockiness {
			worstBlockiness = metrics.Blockiness
		}
		if metrics.ClipHigh > maxClipHigh {
			maxClipHigh = metrics.ClipHigh
		}
		if metrics.Sharpness < minSharpness {
			minSharpness = metrics.Sharpness
		}
		ok := len(frameBlockers) == 0
		frames = append(frames, frameQA{
			Time: at, Frame: workspaceRelative(root, framePath), OK: ok,
			Metrics: metrics, Blockers: frameBlockers, Warnings: frameWarnings,
		})
	}
	state := "ready"
	if len(blockerList) > 0 {
		state = "blocked"
	}
	result := map[string]any{
		"state": state, "ok": len(blockerList) == 0,
		"probe": probe, "frames": frames,
		"summary": map[string]any{
			"frames_checked":      len(frames),
			"worst_banding":       map[string]any{"time_seconds": worstBandingTime, "score": round4(worstBanding)},
			"worst_blockiness":    round4(worstBlockiness),
			"max_clip_high_ratio": round4(maxClipHigh),
			"min_sharpness":       round4(minSharpness),
		},
		"blockers": blockerList, "warnings": warningList,
		"note": "确定性抽帧统计：用于发现色带/色块、过曝与失焦风险；不做语义审美判断",
	}
	if in.KeepFrames {
		result["frame_dir"] = workspaceRelative(root, tempDir)
	}
	return result, nil
}

func probeVideo(ffprobe, path string) (videoProbe, error) {
	probe := videoProbe{Path: path}
	command := exec.Command(ffprobe, "-v", "error",
		"-show_entries", "format=duration:stream=codec_name,width,height,r_frame_rate",
		"-of", "json", path)
	output, err := command.Output()
	if err != nil {
		return probe, fmt.Errorf("ffprobe failed: %w", err)
	}
	var payload struct {
		Streams []struct {
			CodecName  string `json:"codec_name"`
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			RFrameRate string `json:"r_frame_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return probe, err
	}
	probe.Duration, _ = strconv.ParseFloat(payload.Format.Duration, 64)
	for _, stream := range payload.Streams {
		if stream.Width > 0 && stream.Height > 0 {
			probe.Width = stream.Width
			probe.Height = stream.Height
			probe.FPS = stream.RFrameRate
			probe.Codec = stream.CodecName
			break
		}
	}
	return probe, nil
}

func parseFrameTimes(value string) []float64 {
	parts := strings.Split(value, ",")
	seen := map[float64]bool{}
	times := make([]float64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		at, err := strconv.ParseFloat(part, 64)
		if err != nil || at < 0 {
			continue
		}
		if seen[at] {
			continue
		}
		seen[at] = true
		times = append(times, at)
	}
	sort.Float64s(times)
	return times
}

func defaultFrameTimes(duration float64, sampleCount int) []float64 {
	if duration <= 0 {
		return nil
	}
	if sampleCount <= 0 {
		sampleCount = 5
	}
	if sampleCount > 12 {
		sampleCount = 12
	}
	limit := duration - 0.05
	times := make([]float64, 0, sampleCount)
	for index := 1; index <= sampleCount; index++ {
		at := duration * float64(index) / float64(sampleCount+1)
		if at > limit {
			at = limit
		}
		if at < 0 {
			at = 0
		}
		times = append(times, round4(at))
	}
	return times
}
