package main

import (
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type silenceRange struct {
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Duration float64 `json:"duration"`
}

var silenceStartPattern = regexp.MustCompile(`silence_start: ([0-9.]+)`)
var silenceEndPattern = regexp.MustCompile(`silence_end: ([0-9.]+)`)

func detectSilence(path string, noiseDB, minSilence float64) ([]silenceRange, string, error) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, "", fmt.Errorf("ffmpeg not found in PATH")
	}
	if noiseDB == 0 {
		noiseDB = -35
	}
	if minSilence <= 0 {
		minSilence = 0.35
	}
	af := fmt.Sprintf("silencedetect=noise=%.1fdB:d=%.3f", noiseDB, minSilence)
	command := exec.Command(ffmpeg, "-hide_banner", "-nostats", "-i", path, "-af", af, "-f", "null", "-")
	output, runErr := command.CombinedOutput()
	text := string(output)
	if runErr != nil && !strings.Contains(text, "silence_start") {
		return nil, text, runErr
	}
	starts := silenceStartPattern.FindAllStringSubmatch(text, -1)
	ends := silenceEndPattern.FindAllStringSubmatch(text, -1)
	ranges := make([]silenceRange, 0, len(starts))
	for index, match := range starts {
		start, _ := strconv.ParseFloat(match[1], 64)
		end := start
		if index < len(ends) {
			end, _ = strconv.ParseFloat(ends[index][1], 64)
		}
		if end <= start {
			continue
		}
		ranges = append(ranges, silenceRange{Start: start, End: end, Duration: end - start})
	}
	sort.SliceStable(ranges, func(i, j int) bool { return ranges[i].Start < ranges[j].Start })
	return ranges, text, nil
}

func resolveSilenceMedia(root, pathText string) (string, string, error) {
	pathText = strings.TrimSpace(pathText)
	if pathText == "" {
		return "", "", fmt.Errorf("media path is required")
	}
	path, err := workspaceFile(root, pathText)
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", "", err
	}
	return path, filepath.ToSlash(rel), nil
}

func timelineSilenceDetectCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	path, rel, err := resolveSilenceMedia(root, in.Path)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	ranges, raw, runErr := detectSilence(path, in.NoiseDB, in.MinSilenceSeconds)
	if runErr != nil {
		return nil, himindjsonrpc.InternalError("silence detect failed: " + raw)
	}
	total := 0.0
	for _, item := range ranges {
		total += item.Duration
	}
	probe, _ := probeMediaRef(root, in.MediaID, rel)
	duration := 0.0
	if probe.FPS != "" {
		if fps := parseRate(probe.FPS); fps > 0 {
			duration = float64(probe.DurationFrames) / fps
		}
	}
	return map[string]any{"state": "ready", "path": rel, "media_id": in.MediaID, "ranges": ranges, "total": len(ranges), "silence_seconds": total, "duration_seconds": duration}, nil
}

func timelineSilenceTrimCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	path, rel, err := resolveSilenceMedia(root, in.Path)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	ranges, raw, runErr := detectSilence(path, in.NoiseDB, in.MinSilenceSeconds)
	if runErr != nil {
		return nil, himindjsonrpc.InternalError("silence detect failed: " + raw)
	}
	probe, probeErr := probeMediaRef(root, in.MediaID, rel)
	if probeErr != nil {
		return nil, himindjsonrpc.InvalidParams(probeErr.Error())
	}
	fps := in.Timeline.Timebase.Num
	if fps <= 0 {
		fps = 30
	}
	den := in.Timeline.Timebase.Den
	if den <= 0 {
		den = 1
	}
	canvas := in.Timeline.Canvas
	if canvas.Width <= 0 || canvas.Height <= 0 {
		canvas = timelineCanvas{Width: 1080, Height: 1920}
	}
	duration := float64(probe.DurationFrames) * float64(den) / float64(fps)
	if duration <= 0 {
		return map[string]any{"state": "empty", "blockers": []string{"MEDIA_DURATION_UNKNOWN"}}, nil
	}
	pad := in.PaddingSeconds
	if pad <= 0 {
		pad = 0.08
	}
	minSilence := in.MinSilenceSeconds
	if minSilence <= 0 {
		minSilence = 0.35
	}
	kept := complementRanges(ranges, duration, pad, minSilence)
	if len(kept) == 0 {
		return map[string]any{"state": "empty", "blockers": []string{"NO_SPEECH_RANGES"}}, nil
	}
	clips := make([]timelineClip, 0, len(kept))
	cursor := 0
	for index, span := range kept {
		frames := int(math.Round((span.End - span.Start) * float64(fps) / float64(den)))
		if frames <= 0 {
			continue
		}
		sourceIn := int(math.Round(span.Start * float64(fps) / float64(den)))
		clips = append(clips, timelineClip{ID: fmt.Sprintf("silence-cut-%03d", index+1), MediaID: in.MediaID, TrackID: "V1", StartFrame: cursor, DurationFrames: frames, SourceInFrame: sourceIn, SourceOutFrame: sourceIn + frames, Speed: 1})
		cursor += frames
	}
	doc := timelineDoc{Schema: "himind.timeline.v1", Revision: 0, Timebase: timelineTimebase{Num: fps, Den: den}, DurationFrames: cursor, Canvas: canvas, Tracks: []timelineTrack{{ID: "V1", Kind: "video", Index: 1, Role: "program", Clips: clips}}, Media: []timelineMediaRef{{ID: in.MediaID, Kind: "video", URI: rel}}, Markers: []map[string]any{{"frame": 0, "label": "去静音", "kind": "silence-trim", "removed_seconds": totalRangeDuration(ranges)}}}
	if blockers := validateTimeline(doc); len(blockers) > 0 {
		return map[string]any{"state": "blocked", "blockers": blockers, "timeline": doc}, nil
	}
	return map[string]any{"state": "ready", "timeline": doc, "removed_seconds": totalRangeDuration(ranges), "kept_segments": len(clips), "duration_seconds": float64(cursor) * float64(den) / float64(fps)}, nil
}

type timeSpan struct {
	Start float64
	End   float64
}

func complementRanges(ranges []silenceRange, duration, padding, minSilence float64) []timeSpan {
	kept := make([]timeSpan, 0)
	cursor := 0.0
	for _, item := range ranges {
		if item.Duration < minSilence {
			continue
		}
		cutStart := item.Start - padding
		cutEnd := item.End + padding
		if cutStart < cursor {
			cutStart = cursor
		}
		if cutStart > cursor {
			kept = append(kept, timeSpan{Start: cursor, End: cutStart})
		}
		if cutEnd > cursor {
			cursor = cutEnd
		}
	}
	if cursor < duration {
		kept = append(kept, timeSpan{Start: cursor, End: duration})
	}
	out := kept[:0]
	for _, span := range kept {
		if span.End-span.Start >= 0.12 {
			out = append(out, span)
		}
	}
	return out
}

func totalRangeDuration(ranges []silenceRange) float64 {
	total := 0.0
	for _, item := range ranges {
		total += item.Duration
	}
	return total
}
