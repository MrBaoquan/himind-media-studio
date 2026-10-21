package main

// timeline_track_ops.go — track-level editing commands.
//
// Tracks are part of the same himind.timeline.v1 document. These commands
// keep track management explicit instead of letting the UI mutate arrays.

import (
	"fmt"
	"strings"
)

func timelineAddTrack(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	kind := strings.ToLower(strings.TrimSpace(command.TrackKind))
	if kind == "" {
		kind = "video"
	}
	if kind != "video" && kind != "audio" && kind != "subtitle" {
		return doc, []string{"TRACK_KIND_INVALID: " + command.TrackKind}
	}
	if len(doc.Tracks) >= 32 {
		return doc, []string{"TRACK_LIMIT_REACHED"}
	}
	index := 1
	insertAt := len(doc.Tracks)
	prefix := map[string]string{"video": "V", "audio": "A", "subtitle": "S"}[kind]
	for position, track := range doc.Tracks {
		if track.Kind != kind {
			continue
		}
		if track.Index >= index {
			index = track.Index + 1
		}
		insertAt = position + 1
	}
	if kind == "video" && insertAt == len(doc.Tracks) {
		hasVideo := false
		for _, track := range doc.Tracks {
			if track.Kind == "video" {
				hasVideo = true
				break
			}
		}
		if !hasVideo {
			insertAt = 0
		}
	}
	id := strings.TrimSpace(command.TrackID)
	if id == "" {
		id = fmt.Sprintf("%s%d", prefix, index)
	}
	if timelineTrackIndex(&doc, id) >= 0 {
		return doc, []string{"TRACK_ID_EXISTS: " + id}
	}
	role := strings.TrimSpace(command.TrackRole)
	if role == "" && kind == "audio" {
		role = "audio"
	}
	track := timelineTrack{ID: id, Kind: kind, Index: index, Role: role, Clips: []timelineClip{}}
	doc.Tracks = append(doc.Tracks, timelineTrack{})
	copy(doc.Tracks[insertAt+1:], doc.Tracks[insertAt:])
	doc.Tracks[insertAt] = track
	return doc, nil
}

func timelineRemoveTrack(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	index := timelineTrackIndex(&doc, command.TrackID)
	if index < 0 {
		return doc, []string{"TRACK_NOT_FOUND: " + command.TrackID}
	}
	if len(doc.Tracks) <= 1 {
		return doc, []string{"LAST_TRACK_CANNOT_BE_REMOVED"}
	}
	if len(doc.Tracks[index].Clips) > 0 {
		return doc, []string{"TRACK_NOT_EMPTY: " + command.TrackID}
	}
	if doc.Tracks[index].Locked {
		return doc, []string{"TRACK_LOCKED: " + command.TrackID}
	}
	doc.Tracks = append(doc.Tracks[:index], doc.Tracks[index+1:]...)
	return doc, nil
}

func timelineSetTrackState(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	index := timelineTrackIndex(&doc, command.TrackID)
	if index < 0 {
		return doc, []string{"TRACK_NOT_FOUND: " + command.TrackID}
	}
	track := &doc.Tracks[index]
	if command.Muted != nil {
		track.Muted = *command.Muted
	}
	if command.Locked != nil {
		track.Locked = *command.Locked
	}
	if command.Solo != nil {
		track.Solo = *command.Solo
	}
	if name := strings.TrimSpace(command.TrackName); name != "" {
		if len([]rune(name)) > 40 {
			return doc, []string{"TRACK_NAME_TOO_LONG"}
		}
		track.Name = name
	}
	if color := strings.TrimSpace(command.TrackColor); color != "" {
		if !validTrackColor(color) {
			return doc, []string{"TRACK_COLOR_INVALID"}
		}
		track.Color = color
	}
	if command.TrackVolumeDB != nil {
		volume := *command.TrackVolumeDB
		if volume < -60 || volume > 12 {
			return doc, []string{"TRACK_VOLUME_OUT_OF_RANGE"}
		}
		track.VolumeDB = volume
	}
	if role := strings.TrimSpace(command.TrackRole); role != "" {
		track.Role = role
	}
	return doc, nil
}
func timelineSetTrackHeight(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	index := timelineTrackIndex(&doc, command.TrackID)
	if index < 0 {
		return doc, []string{"TRACK_NOT_FOUND: " + command.TrackID}
	}
	if command.Height == nil {
		return doc, []string{"TRACK_HEIGHT_REQUIRED"}
	}
	height := *command.Height
	if height != 0 && (height < 28 || height > 160) {
		return doc, []string{"TRACK_HEIGHT_OUT_OF_RANGE"}
	}
	doc.Tracks[index].Height = height
	return doc, nil
}

func validTrackColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, char := range value[1:] {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

func timelineReorderTrack(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	source := timelineTrackIndex(&doc, command.TrackID)
	if source < 0 {
		return doc, []string{"TRACK_NOT_FOUND: " + command.TrackID}
	}
	kind := doc.Tracks[source].Kind
	positions := make([]int, 0)
	tracks := make([]timelineTrack, 0)
	for index, track := range doc.Tracks {
		if track.Kind == kind {
			positions = append(positions, index)
			tracks = append(tracks, track)
		}
	}
	sourcePosition := -1
	for index, track := range tracks {
		if track.ID == command.TrackID {
			sourcePosition = index
		}
	}
	target := command.TargetIndex
	if target < 0 || target >= len(tracks) {
		return doc, []string{"TRACK_TARGET_OUT_OF_RANGE"}
	}
	if target == sourcePosition {
		return doc, nil
	}
	moved := tracks[sourcePosition]
	tracks = append(tracks[:sourcePosition], tracks[sourcePosition+1:]...)
	tracks = append(tracks, timelineTrack{})
	copy(tracks[target+1:], tracks[target:])
	tracks[target] = moved
	for index, position := range positions {
		tracks[index].Index = index + 1
		doc.Tracks[position] = tracks[index]
	}
	return doc, nil
}
