package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"

	"himind-plugin/media-visual/internal/himindjsonrpc"
)

const sceneTimePattern = `pts_time:\s*([0-9]+(?:\.[0-9]+)?)`

var sceneTimeRegexp = regexp.MustCompile(sceneTimePattern)

type sceneCut struct {
	Index int     `json:"index"`
	Time  float64 `json:"time_seconds"`
}

func videoScenesCapability(in requestInput) (any, *himindjsonrpc.Error) {
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
	threshold := in.SceneThreshold
	if threshold <= 0 {
		threshold = 0.30
	}
	if threshold > 1 {
		return nil, himindjsonrpc.InvalidParams("scene_threshold must be within 0..1")
	}
	maxCuts := in.MaxCuts
	if maxCuts <= 0 {
		maxCuts = 80
	}
	if maxCuts > 500 {
		maxCuts = 500
	}
	filter := fmt.Sprintf("select='gt(scene,%.3f)',showinfo", threshold)
	command := exec.Command(ffmpeg, "-hide_banner", "-loglevel", "info", "-i", videoPath,
		"-an", "-filter:v", filter, "-f", "null", "-")
	output, runErr := command.CombinedOutput()
	if runErr != nil {
		return nil, himindjsonrpc.InternalError("ffmpeg scene detection failed: " + string(output))
	}
	times := parseSceneTimes(string(output))
	cuts := make([]sceneCut, 0, len(times))
	for index, at := range times {
		if index+1 > maxCuts {
			break
		}
		cuts = append(cuts, sceneCut{Index: index + 1, Time: at})
	}
	return map[string]any{
		"state": "ready", "probe": probe, "threshold": threshold,
		"cut_count": len(cuts), "cuts": cuts, "truncated": len(times) > len(cuts),
		"note": "基于 ffmpeg 场景分数的确定性切换检测；阈值越高切分越少，用于分镜切分与素材检索",
	}, nil
}

func parseSceneTimes(output string) []float64 {
	matches := sceneTimeRegexp.FindAllStringSubmatch(output, -1)
	seen := map[float64]bool{}
	times := make([]float64, 0, len(matches))
	for _, match := range matches {
		at, err := strconv.ParseFloat(match[1], 64)
		if err != nil || at <= 0 {
			continue
		}
		key := round4(at)
		if seen[key] {
			continue
		}
		seen[key] = true
		times = append(times, key)
	}
	sort.Float64s(times)
	return times
}
