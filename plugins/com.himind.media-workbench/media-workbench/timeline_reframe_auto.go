package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

var cropDetectPattern = regexp.MustCompile(`crop=([0-9]+):([0-9]+):([0-9]+):([0-9]+)`)

func detectContentCrop(path string, timeSeconds float64) (int, int, int, int, error) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("ffmpeg not found in PATH")
	}
	args := []string{"-hide_banner", "-nostats"}
	if timeSeconds > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.6f", timeSeconds))
	}
	args = append(args, "-i", path, "-vf", "cropdetect=limit=24:round=2:reset=0", "-frames:v", "4", "-f", "null", "-")
	output, runErr := exec.Command(ffmpeg, args...).CombinedOutput()
	text := string(output)
	if runErr != nil && !strings.Contains(text, "crop=") {
		return 0, 0, 0, 0, runErr
	}
	matches := cropDetectPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return 0, 0, 0, 0, fmt.Errorf("cropdetect produced no crop")
	}
	last := matches[len(matches)-1]
	w, _ := strconv.Atoi(last[1])
	h, _ := strconv.Atoi(last[2])
	x, _ := strconv.Atoi(last[3])
	y, _ := strconv.Atoi(last[4])
	return w, h, x, y, nil
}

func timelineAutoReframeCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	doc := in.Timeline
	if len(doc.Tracks) == 0 {
		return nil, himindjsonrpc.InvalidParams("timeline is required")
	}
	width, height := in.Width, in.Height
	if width <= 0 || height <= 0 {
		return nil, himindjsonrpc.InvalidParams("width and height are required")
	}
	fit := firstNonEmpty(in.FitMode, "cover")
	num, den := doc.Timebase.Num, doc.Timebase.Den
	if num <= 0 {
		num = 30
	}
	if den <= 0 {
		den = 1
	}
	updated := 0
	warnings := make([]string, 0)
	for trackIndex := range doc.Tracks {
		if doc.Tracks[trackIndex].Kind != "video" {
			continue
		}
		for clipIndex := range doc.Tracks[trackIndex].Clips {
			clip := &doc.Tracks[trackIndex].Clips[clipIndex]
			mediaPath := strings.TrimSpace(in.MediaPaths[clip.MediaID])
			if mediaPath == "" {
				warnings = append(warnings, "MEDIA_PATH_MISSING: "+clip.MediaID)
				continue
			}
			resolved, pathErr := workspaceFile(root, mediaPath)
			if pathErr != nil {
				warnings = append(warnings, "MEDIA_PATH_INVALID: "+clip.MediaID)
				continue
			}
			rel, _ := filepath.Rel(root, resolved)
			probe, probeErr := probeMediaRef(root, clip.MediaID, filepath.ToSlash(rel))
			if probeErr != nil || probe.Width <= 0 || probe.Height <= 0 {
				warnings = append(warnings, "MEDIA_PROBE_FAILED: "+clip.MediaID)
				continue
			}
			midFrame := clip.SourceInFrame + clip.DurationFrames/2
			timeSeconds := float64(midFrame) * float64(den) / float64(num)
			cw, ch, cx, cy, cropErr := detectContentCrop(resolved, timeSeconds)
			if cropErr != nil {
				warnings = append(warnings, "CROPDETECT_FAILED: "+clip.MediaID)
				continue
			}
			offsetX := (float64(cx) + float64(cw)/2 - float64(probe.Width)/2) / float64(probe.Width)
			offsetY := (float64(cy) + float64(ch)/2 - float64(probe.Height)/2) / float64(probe.Height)
			transform := timelineTransform{}
			if clip.Transform != nil {
				transform = *clip.Transform
			}
			transform.Fit = fit
			transform.Scale = 1
			transform.PositionX = offsetX
			transform.PositionY = offsetY
			clip.Transform = &transform
			updated++
		}
	}
	doc.Canvas = timelineCanvas{Width: width, Height: height}
	return map[string]any{"state": "ready", "timeline": doc, "canvas": doc.Canvas, "updated_clips": updated, "warnings": warnings, "subject_tracking": "content-aware-center"}, nil
}
