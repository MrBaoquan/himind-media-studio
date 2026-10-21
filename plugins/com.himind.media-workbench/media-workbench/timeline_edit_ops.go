package main

import (
	"fmt"
	"math"
	"strings"
)

func timelineRoll(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	trackIndex := timelineTrackIndex(&doc, command.TrackID)
	if trackIndex < 0 {
		return doc, []string{"TRACK_NOT_FOUND: " + command.TrackID}
	}
	track := &doc.Tracks[trackIndex]
	clipIndex := timelineClipIndex(track, command.ClipID)
	if clipIndex < 0 {
		return doc, []string{"CLIP_NOT_FOUND: " + command.ClipID}
	}
	if clipIndex+1 >= len(track.Clips) {
		return doc, []string{"ROLL_REQUIRES_NEXT_CLIP: " + command.ClipID}
	}
	left := &track.Clips[clipIndex]
	right := &track.Clips[clipIndex+1]
	if left.Locked || right.Locked {
		return doc, []string{"CLIP_LOCKED: " + command.ClipID}
	}
	if left.StartFrame+left.DurationFrames != right.StartFrame {
		return doc, []string{"ROLL_CLIPS_NOT_ADJACENT"}
	}
	delta := command.DeltaFrames
	if delta == 0 && command.AtFrame > 0 {
		delta = command.AtFrame - clipEnd(*left)
	}
	newLeftDuration := left.DurationFrames + delta
	newRightDuration := right.DurationFrames - delta
	if newLeftDuration <= 0 || newRightDuration <= 0 {
		return doc, []string{"ROLL_DURATION_INVALID"}
	}
	if track.Kind != "subtitle" {
		if left.SourceInFrame+newLeftDuration > left.SourceOutFrame {
			left.SourceOutFrame = left.SourceInFrame + newLeftDuration
		}
		if right.SourceInFrame+delta < 0 || right.SourceInFrame+delta+newRightDuration > right.SourceOutFrame {
			return doc, []string{"ROLL_SOURCE_EXHAUSTED"}
		}
		right.SourceInFrame += delta
	}
	left.DurationFrames = newLeftDuration
	right.StartFrame = left.StartFrame + newLeftDuration
	right.DurationFrames = newRightDuration
	sortTrackClips(track)
	return doc, nil
}

func timelineSlip(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	trackIndex := timelineTrackIndex(&doc, command.TrackID)
	if trackIndex < 0 {
		return doc, []string{"TRACK_NOT_FOUND: " + command.TrackID}
	}
	track := &doc.Tracks[trackIndex]
	clipIndex := timelineClipIndex(track, command.ClipID)
	if clipIndex < 0 {
		return doc, []string{"CLIP_NOT_FOUND: " + command.ClipID}
	}
	clip := &track.Clips[clipIndex]
	if clip.Locked {
		return doc, []string{"CLIP_LOCKED: " + clip.ID}
	}
	delta := command.DeltaFrames
	if delta == 0 {
		return doc, []string{"SLIP_DELTA_REQUIRED"}
	}
	newIn := clip.SourceInFrame + delta
	newOut := clip.SourceOutFrame + delta
	if newIn < 0 || newOut <= newIn {
		return doc, []string{"SLIP_SOURCE_OUT_OF_BOUNDS"}
	}
	clip.SourceInFrame = newIn
	clip.SourceOutFrame = newOut
	return doc, nil
}

func timelineSlide(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	trackIndex := timelineTrackIndex(&doc, command.TrackID)
	if trackIndex < 0 {
		return doc, []string{"TRACK_NOT_FOUND: " + command.TrackID}
	}
	track := &doc.Tracks[trackIndex]
	clipIndex := timelineClipIndex(track, command.ClipID)
	if clipIndex < 0 {
		return doc, []string{"CLIP_NOT_FOUND: " + command.ClipID}
	}
	clip := &track.Clips[clipIndex]
	if clip.Locked {
		return doc, []string{"CLIP_LOCKED: " + clip.ID}
	}
	delta := command.DeltaFrames
	if delta == 0 {
		return doc, []string{"SLIDE_DELTA_REQUIRED"}
	}
	oldStart := clip.StartFrame
	oldEnd := clipEnd(*clip)
	newStart := oldStart + delta
	if newStart < 0 {
		return doc, []string{"SLIDE_BEFORE_ZERO"}
	}
	// Classic slide keeps total duration stable: the previous clip gives or
	// takes time on the left, the next clip compensates on the right.
	if clipIndex > 0 && clipEnd(track.Clips[clipIndex-1]) == oldStart {
		prev := &track.Clips[clipIndex-1]
		if prev.Locked {
			return doc, []string{"CLIP_LOCKED: " + prev.ID}
		}
		if prev.DurationFrames-delta <= 0 {
			return doc, []string{"SLIDE_PREVIOUS_TOO_SHORT"}
		}
		prev.DurationFrames += delta
		if track.Kind != "subtitle" && prev.SourceInFrame+prev.DurationFrames > prev.SourceOutFrame {
			prev.SourceOutFrame = prev.SourceInFrame + prev.DurationFrames
		}
	} else if newStart < oldStart {
		return doc, []string{"SLIDE_PREVIOUS_GAP"}
	}
	if clipIndex+1 < len(track.Clips) && track.Clips[clipIndex+1].StartFrame == oldEnd {
		next := &track.Clips[clipIndex+1]
		if next.Locked {
			return doc, []string{"CLIP_LOCKED: " + next.ID}
		}
		nextStart := next.StartFrame + delta
		nextDuration := next.DurationFrames - delta
		nextSourceIn := next.SourceInFrame + delta
		if nextDuration <= 0 || (track.Kind != "subtitle" && nextSourceIn < 0) {
			return doc, []string{"SLIDE_NEXT_TOO_SHORT"}
		}
		next.StartFrame = nextStart
		next.DurationFrames = nextDuration
		if track.Kind != "subtitle" {
			next.SourceInFrame = nextSourceIn
		}
	} else if newStart+clip.DurationFrames > oldEnd {
		return doc, []string{"SLIDE_NEXT_GAP"}
	}
	clip.StartFrame = newStart
	sortTrackClips(track)
	return doc, nil
}

func timelineRetime(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	trackIndex := timelineTrackIndex(&doc, command.TrackID)
	if trackIndex < 0 {
		return doc, []string{"TRACK_NOT_FOUND: " + command.TrackID}
	}
	track := &doc.Tracks[trackIndex]
	clipIndex := timelineClipIndex(track, command.ClipID)
	if clipIndex < 0 {
		return doc, []string{"CLIP_NOT_FOUND: " + command.ClipID}
	}
	clip := &track.Clips[clipIndex]
	if clip.Locked {
		return doc, []string{"CLIP_LOCKED: " + clip.ID}
	}
	if command.Speed == nil {
		return doc, []string{"RETIME_SPEED_REQUIRED"}
	}
	speed := *command.Speed
	if speed < 0.25 || speed > 4 {
		return doc, []string{"RETIME_SPEED_OUT_OF_RANGE"}
	}
	sourceFrames := clip.SourceOutFrame - clip.SourceInFrame
	if sourceFrames <= 0 {
		return doc, []string{"CLIP_SOURCE_RANGE_INVALID"}
	}
	newDuration := int(math.Round(float64(sourceFrames) / speed))
	if newDuration <= 0 {
		newDuration = 1
	}
	oldEnd := clipEnd(*clip)
	delta := newDuration - clip.DurationFrames
	clip.DurationFrames = newDuration
	clip.Speed = speed
	if command.Ripple {
		for index := range track.Clips {
			if track.Clips[index].ID != clip.ID && track.Clips[index].StartFrame >= oldEnd {
				track.Clips[index].StartFrame += delta
				if track.Clips[index].StartFrame < 0 {
					track.Clips[index].StartFrame = 0
				}
			}
		}
	}
	sortTrackClips(track)
	return doc, nil
}

func timelineUpdateClip(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	if command.Clip == nil {
		return doc, []string{"CLIP_REQUIRED"}
	}
	trackID := firstNonEmpty(command.TrackID, command.Clip.TrackID)
	trackIndex := timelineTrackIndex(&doc, trackID)
	if trackIndex < 0 {
		return doc, []string{"TRACK_NOT_FOUND: " + trackID}
	}
	track := &doc.Tracks[trackIndex]
	clipID := firstNonEmpty(command.ClipID, command.Clip.ID)
	clipIndex := timelineClipIndex(track, clipID)
	if clipIndex < 0 {
		return doc, []string{"CLIP_NOT_FOUND: " + clipID}
	}
	current := track.Clips[clipIndex]
	if current.Locked {
		return doc, []string{"CLIP_LOCKED: " + current.ID}
	}
	replacement := *command.Clip
	replacement.ID = current.ID
	replacement.TrackID = track.ID
	if replacement.StartFrame != current.StartFrame || replacement.DurationFrames != current.DurationFrames {
		return doc, []string{"UPDATE_CLIP_CANNOT_CHANGE_TIMING"}
	}
	if replacement.SourceInFrame == 0 && replacement.SourceOutFrame == 0 && track.Kind != "subtitle" {
		replacement.SourceInFrame = current.SourceInFrame
		replacement.SourceOutFrame = current.SourceOutFrame
	}
	if replacement.MediaID == "" && track.Kind != "subtitle" {
		replacement.MediaID = current.MediaID
	}
	if replacement.Speed <= 0 {
		replacement.Speed = 1
	}
	if track.Kind == "subtitle" && len([]rune(replacement.Text)) == 0 {
		return doc, []string{"SUBTITLE_TEXT_REQUIRED: " + current.ID}
	}
	if track.Kind != "subtitle" && replacement.MediaID == "" {
		return doc, []string{"CLIP_MEDIA_REQUIRED: " + current.ID}
	}
	track.Clips[clipIndex] = replacement
	return doc, nil
}

func timelineMediaRefIndex(doc timelineDoc, mediaID string) int {
	for index := range doc.Media {
		if doc.Media[index].ID == mediaID {
			return index
		}
	}
	return -1
}

func timelineClipIDs(doc timelineDoc, clipID string) []string {
	if clipID == "" {
		return nil
	}
	for _, track := range doc.Tracks {
		for _, clip := range track.Clips {
			if clip.ID == clipID {
				return []string{clipID}
			}
		}
	}
	return nil
}

func clipDebug(clip timelineClip) string {
	return fmt.Sprintf("%s@%d+%d", clip.ID, clip.StartFrame, clip.DurationFrames)
}

// timelineMove moves a clip (and its linked group) without changing source range.
// It is the deterministic primitive behind direct timeline dragging.
func timelineMove(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	trackIndex := timelineTrackIndex(&doc, command.TrackID)
	if trackIndex < 0 {
		return doc, []string{"TRACK_NOT_FOUND: " + command.TrackID}
	}
	track := &doc.Tracks[trackIndex]
	clipIndex := timelineClipIndex(track, command.ClipID)
	if clipIndex < 0 {
		return doc, []string{"CLIP_NOT_FOUND: " + command.ClipID}
	}
	clip := track.Clips[clipIndex]
	if clip.Locked || track.Locked {
		return doc, []string{"CLIP_LOCKED: " + clip.ID}
	}
	target := command.AtFrame
	if target < 0 {
		target = 0
	}
	delta := target - clip.StartFrame
	if delta == 0 {
		return doc, nil
	}
	type moveTarget struct {
		trackIndex int
		clipIndex  int
		start      int
	}
	targets := []moveTarget{{trackIndex: trackIndex, clipIndex: clipIndex, start: target}}
	if clip.LinkedGroup != "" {
		for ti := range doc.Tracks {
			for ci := range doc.Tracks[ti].Clips {
				other := doc.Tracks[ti].Clips[ci]
				if other.ID == clip.ID || other.LinkedGroup != clip.LinkedGroup {
					continue
				}
				if other.Locked || doc.Tracks[ti].Locked {
					return doc, []string{"CLIP_LOCKED: " + other.ID}
				}
				start := other.StartFrame + delta
				if start < 0 {
					start = 0
				}
				targets = append(targets, moveTarget{trackIndex: ti, clipIndex: ci, start: start})
			}
		}
	}
	for _, item := range targets {
		candidate := doc.Tracks[item.trackIndex].Clips[item.clipIndex]
		candidateStart := item.start
		for ci := range doc.Tracks[item.trackIndex].Clips {
			other := doc.Tracks[item.trackIndex].Clips[ci]
			if other.ID == candidate.ID {
				continue
			}
			moving := false
			for _, t := range targets {
				if t.trackIndex == item.trackIndex && doc.Tracks[t.trackIndex].Clips[t.clipIndex].ID == other.ID {
					moving = true
					break
				}
			}
			if moving {
				continue
			}
			if candidateStart < clipEnd(other) && other.StartFrame < candidateStart+candidate.DurationFrames {
				return doc, []string{"CLIP_OVERLAP: " + other.ID + " / " + candidate.ID}
			}
		}
	}
	touched := map[int]bool{}
	for _, item := range targets {
		doc.Tracks[item.trackIndex].Clips[item.clipIndex].StartFrame = item.start
		touched[item.trackIndex] = true
	}
	for trackIndex := range touched {
		sortTrackClips(&doc.Tracks[trackIndex])
	}
	return doc, nil
}

// timelineTrimEdge trims the left or right edge directly. delta_frames > 0
// extends (right) or moves the left edge left (left); < 0 shortens.
func timelineTrimEdge(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	trackIndex := timelineTrackIndex(&doc, command.TrackID)
	if trackIndex < 0 {
		return doc, []string{"TRACK_NOT_FOUND: " + command.TrackID}
	}
	track := &doc.Tracks[trackIndex]
	clipIndex := timelineClipIndex(track, command.ClipID)
	if clipIndex < 0 {
		return doc, []string{"CLIP_NOT_FOUND: " + command.ClipID}
	}
	clip := &track.Clips[clipIndex]
	if clip.Locked || track.Locked {
		return doc, []string{"CLIP_LOCKED: " + clip.ID}
	}
	edge := strings.ToLower(strings.TrimSpace(command.Edge))
	if edge == "" {
		edge = "right"
	}
	if edge != "left" && edge != "right" {
		return doc, []string{"TRIM_EDGE_INVALID: " + edge}
	}
	delta := command.DeltaFrames
	if delta == 0 {
		return doc, nil
	}
	newStart, newDuration := clip.StartFrame, clip.DurationFrames
	if edge == "left" {
		newStart = clip.StartFrame + delta
		newDuration = clip.DurationFrames - delta
	} else {
		newDuration = clip.DurationFrames + delta
	}
	if newStart < 0 {
		return doc, []string{"CLIP_START_NEGATIVE: " + clip.ID}
	}
	if newDuration <= 0 {
		return doc, []string{"TRIM_DURATION_INVALID"}
	}
	if track.Kind != "subtitle" {
		sourceIn := clip.SourceInFrame
		if edge == "left" {
			sourceIn += delta
		}
		if sourceIn < 0 || sourceIn+newDuration > clip.SourceOutFrame && edge == "left" {
			return doc, []string{"CLIP_SOURCE_RANGE_INVALID: " + clip.ID}
		}
	}
	for ci := range track.Clips {
		other := track.Clips[ci]
		if other.ID == clip.ID {
			continue
		}
		if newStart < clipEnd(other) && other.StartFrame < newStart+newDuration {
			return doc, []string{"CLIP_OVERLAP: " + other.ID + " / " + clip.ID}
		}
	}
	clip.StartFrame = newStart
	clip.DurationFrames = newDuration
	if edge == "left" && track.Kind != "subtitle" {
		clip.SourceInFrame += delta
	}
	if edge == "right" && track.Kind != "subtitle" && clip.SourceInFrame+clip.DurationFrames > clip.SourceOutFrame {
		clip.SourceOutFrame = clip.SourceInFrame + clip.DurationFrames
	}
	sortTrackClips(track)
	return doc, nil
}
