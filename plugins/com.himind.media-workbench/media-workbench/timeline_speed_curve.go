package main

import (
	"fmt"
	"math"
	"sort"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type timelineSpeedKeyframe struct {
	Frame int     `json:"frame"`
	Speed float64 `json:"speed"`
}

func timelineSpeedCurveCapability(in requestInput) (any, *himindjsonrpc.Error) {
	if len(in.SpeedKeyframes) == 0 {
		return nil, himindjsonrpc.InvalidParams("speed_keyframes is required")
	}
	doc := in.Timeline
	trackIndex, clipIndex := -1, -1
	for ti := range doc.Tracks {
		if doc.Tracks[ti].Kind != "video" {
			continue
		}
		for ci := range doc.Tracks[ti].Clips {
			if doc.Tracks[ti].Clips[ci].ID == in.ClipID {
				trackIndex, clipIndex = ti, ci
				break
			}
		}
		if trackIndex >= 0 {
			break
		}
	}
	if trackIndex < 0 || clipIndex < 0 {
		return nil, himindjsonrpc.InvalidParams("clip_id not found")
	}
	track := &doc.Tracks[trackIndex]
	original := track.Clips[clipIndex]
	if original.Locked {
		return map[string]any{"state": "blocked", "blockers": []string{"CLIP_LOCKED: " + original.ID}}, nil
	}
	keyframes := append([]timelineSpeedKeyframe(nil), in.SpeedKeyframes...)
	sort.SliceStable(keyframes, func(i, j int) bool { return keyframes[i].Frame < keyframes[j].Frame })
	if keyframes[0].Frame > 0 {
		keyframes = append([]timelineSpeedKeyframe{{Frame: 0, Speed: keyframes[0].Speed}}, keyframes...)
	}
	if keyframes[len(keyframes)-1].Frame < original.DurationFrames {
		keyframes = append(keyframes, timelineSpeedKeyframe{Frame: original.DurationFrames, Speed: keyframes[len(keyframes)-1].Speed})
	}
	for _, keyframe := range keyframes {
		if keyframe.Frame < 0 || keyframe.Frame > original.DurationFrames || keyframe.Speed < 0.25 || keyframe.Speed > 4 {
			return map[string]any{"state": "blocked", "blockers": []string{"SPEED_KEYFRAME_INVALID"}}, nil
		}
	}
	sourceFrames := original.SourceOutFrame - original.SourceInFrame
	if sourceFrames <= 0 {
		return map[string]any{"state": "blocked", "blockers": []string{"CLIP_SOURCE_RANGE_INVALID"}}, nil
	}
	type segment struct{ start, duration, speed float64 }
	segments := make([]segment, 0, len(keyframes)-1)
	totalWeight := 0.0
	for i := 0; i < len(keyframes)-1; i++ {
		duration := float64(keyframes[i+1].Frame - keyframes[i].Frame)
		if duration <= 0 {
			continue
		}
		speed := (keyframes[i].Speed + keyframes[i+1].Speed) / 2
		weight := speed * duration
		segments = append(segments, segment{start: float64(keyframes[i].Frame), duration: duration, speed: speed})
		totalWeight += weight
	}
	if len(segments) == 0 || totalWeight <= 0 {
		return map[string]any{"state": "blocked", "blockers": []string{"SPEED_CURVE_EMPTY"}}, nil
	}
	newClips := make([]timelineClip, 0, len(segments))
	sourceCursor := original.SourceInFrame
	usedSource := 0
	for index, seg := range segments {
		sourceSpan := int(math.Round(float64(sourceFrames) * (seg.speed * seg.duration) / totalWeight))
		if index == len(segments)-1 {
			sourceSpan = sourceFrames - usedSource
		}
		if sourceSpan <= 0 {
			continue
		}
		clip := original
		clip.ID = fmt.Sprintf("%s-sc%02d", original.ID, index+1)
		clip.StartFrame = original.StartFrame + int(math.Round(seg.start))
		clip.DurationFrames = int(math.Round(seg.duration))
		clip.SourceInFrame = sourceCursor
		clip.SourceOutFrame = sourceCursor + sourceSpan
		clip.Speed = float64(sourceSpan) / seg.duration
		clip.TransitionIn = nil
		clip.TransitionOut = nil
		newClips = append(newClips, clip)
		sourceCursor += sourceSpan
		usedSource += sourceSpan
	}
	track.Clips = append(track.Clips[:clipIndex], append(newClips, track.Clips[clipIndex+1:]...)...)
	sortTrackClips(track)
	doc.Revision++
	if blockers := validateTimeline(doc); len(blockers) > 0 {
		return map[string]any{"state": "blocked", "blockers": blockers, "timeline": doc}, nil
	}
	return map[string]any{"state": "ready", "timeline": doc, "segments": len(newClips), "source_frames": sourceFrames}, nil
}
