package main

import (
	"fmt"
	"sort"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type timelineTimebase struct {
	Num int `json:"num"`
	Den int `json:"den"`
}

type timelineCanvas struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type timelineCrop struct {
	Left   float64 `json:"left,omitempty"`
	Right  float64 `json:"right,omitempty"`
	Top    float64 `json:"top,omitempty"`
	Bottom float64 `json:"bottom,omitempty"`
}

type timelineTransform struct {
	Scale     float64       `json:"scale,omitempty"`
	PositionX float64       `json:"position_x,omitempty"`
	PositionY float64       `json:"position_y,omitempty"`
	Rotation  float64       `json:"rotation,omitempty"`
	Fit       string        `json:"fit,omitempty"`
	Crop      *timelineCrop `json:"crop,omitempty"`
}

type timelineColor struct {
	Brightness float64                 `json:"brightness,omitempty"`
	Contrast   float64                 `json:"contrast,omitempty"`
	Saturation float64                 `json:"saturation,omitempty"`
	Gamma      float64                 `json:"gamma,omitempty"`
	LUT        string                  `json:"lut,omitempty"`
	Keyframes  []timelineColorKeyframe `json:"keyframes,omitempty"`
}

type timelineColorKeyframe struct {
	Frame      int     `json:"frame"`
	Brightness float64 `json:"brightness,omitempty"`
	Contrast   float64 `json:"contrast,omitempty"`
	Saturation float64 `json:"saturation,omitempty"`
	Gamma      float64 `json:"gamma,omitempty"`
}

type timelineTransition struct {
	Type           string `json:"type"`
	DurationFrames int    `json:"duration_frames,omitempty"`
}

type timelineClip struct {
	ID             string              `json:"id"`
	MediaID        string              `json:"media_id,omitempty"`
	TrackID        string              `json:"track_id"`
	StartFrame     int                 `json:"start_frame"`
	DurationFrames int                 `json:"duration_frames"`
	SourceInFrame  int                 `json:"source_in_frame,omitempty"`
	SourceOutFrame int                 `json:"source_out_frame,omitempty"`
	Speed          float64             `json:"speed,omitempty"`
	LinkedGroup    string              `json:"linked_group,omitempty"`
	Locked         bool                `json:"locked,omitempty"`
	Text           string              `json:"text,omitempty"`
	Transform      *timelineTransform  `json:"transform,omitempty"`
	Color          *timelineColor      `json:"color,omitempty"`
	TransitionIn   *timelineTransition `json:"transition_in,omitempty"`
	TransitionOut  *timelineTransition `json:"transition_out,omitempty"`
	VolumeDB       float64             `json:"volume_db,omitempty"`
	Opacity        float64             `json:"opacity,omitempty"`
}

type timelineMediaRef struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	URI            string `json:"uri"`
	ProxyURI       string `json:"proxy_uri,omitempty"`
	SHA256         string `json:"sha256,omitempty"`
	DurationFrames int    `json:"duration_frames,omitempty"`
	Width          int    `json:"width,omitempty"`
	Height         int    `json:"height,omitempty"`
	FPS            string `json:"fps,omitempty"`
	ColorSpace     string `json:"color_space,omitempty"`
	License        string `json:"license,omitempty"`
}

type timelineTrack struct {
	ID       string         `json:"id"`
	Kind     string         `json:"kind"`
	Index    int            `json:"index"`
	Role     string         `json:"role,omitempty"`
	Height   int            `json:"height,omitempty"`
	Name     string         `json:"name,omitempty"`
	Color    string         `json:"color,omitempty"`
	VolumeDB float64        `json:"volume_db,omitempty"`
	Muted    bool           `json:"muted"`
	Solo     bool           `json:"solo,omitempty"`
	Locked   bool           `json:"locked"`
	Clips    []timelineClip `json:"clips"`
}

type timelineDoc struct {
	Schema         string             `json:"schema"`
	Revision       int                `json:"revision"`
	Timebase       timelineTimebase   `json:"timebase"`
	DurationFrames int                `json:"duration_frames"`
	Canvas         timelineCanvas     `json:"canvas"`
	Tracks         []timelineTrack    `json:"tracks"`
	Media          []timelineMediaRef `json:"media,omitempty"`
	Markers        []map[string]any   `json:"markers,omitempty"`
}

type timelineCommand struct {
	CommandID        string        `json:"command_id,omitempty"`
	Op               string        `json:"op"`
	ExpectedRevision int           `json:"expected_revision,omitempty"`
	TrackID          string        `json:"track_id,omitempty"`
	ClipID           string        `json:"clip_id,omitempty"`
	AtFrame          int           `json:"at_frame,omitempty"`
	Clip             *timelineClip `json:"clip,omitempty"`
	Edge             string        `json:"edge,omitempty"`
	DurationFrames   int           `json:"duration_frames,omitempty"`
	DeltaFrames      int           `json:"delta_frames,omitempty"`
	Ripple           bool          `json:"ripple,omitempty"`
	LinkedGroup      string        `json:"linked_group,omitempty"`
	ClipIDs          []string      `json:"clip_ids,omitempty"`
	Speed            *float64      `json:"speed,omitempty"`
	TrackKind        string        `json:"track_kind,omitempty"`
	TrackRole        string        `json:"track_role,omitempty"`
	Muted            *bool         `json:"muted,omitempty"`
	Solo             *bool         `json:"solo,omitempty"`
	Locked           *bool         `json:"locked,omitempty"`
	Height           *int          `json:"height,omitempty"`
	TrackName        string        `json:"track_name,omitempty"`
	TrackColor       string        `json:"track_color,omitempty"`
	TargetIndex      int           `json:"target_index,omitempty"`
	TrackVolumeDB    *float64      `json:"track_volume_db,omitempty"`
}

func clipEnd(clip timelineClip) int { return clip.StartFrame + clip.DurationFrames }

func timelineTrackIndex(doc *timelineDoc, trackID string) int {
	for index := range doc.Tracks {
		if doc.Tracks[index].ID == trackID {
			return index
		}
	}
	return -1
}

func timelineClipIndex(track *timelineTrack, clipID string) int {
	for index := range track.Clips {
		if track.Clips[index].ID == clipID {
			return index
		}
	}
	return -1
}

func sortTrackClips(track *timelineTrack) {
	sort.SliceStable(track.Clips, func(left, right int) bool {
		if track.Clips[left].StartFrame == track.Clips[right].StartFrame {
			return track.Clips[left].ID < track.Clips[right].ID
		}
		return track.Clips[left].StartFrame < track.Clips[right].StartFrame
	})
}

func applyTimelineCommands(doc timelineDoc, commands []timelineCommand) (timelineDoc, []string) {
	if doc.Schema == "" {
		doc.Schema = "himind.timeline.v1"
	}
	revision := doc.Revision
	for _, command := range commands {
		if command.ExpectedRevision > 0 && command.ExpectedRevision != revision {
			return doc, []string{fmt.Sprintf("REVISION_CONFLICT: expected %d, current %d", command.ExpectedRevision, revision)}
		}
		next, blockers := applyTimelineCommand(doc, command)
		if len(blockers) > 0 {
			return doc, blockers
		}
		doc = next
		revision++
		doc.Revision = revision
	}
	return doc, nil
}

func applyTimelineCommand(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	switch command.Op {
	case "blade":
		return timelineBlade(doc, command)
	case "lift":
		return timelineLift(doc, command, false)
	case "ripple_delete":
		return timelineLift(doc, command, true)
	case "insert":
		return timelineInsert(doc, command, false)
	case "overwrite":
		return timelineInsert(doc, command, true)
	case "trim_edge":
		return timelineTrimEdge(doc, command)
	case "trim", "ripple_trim":
		return timelineTrim(doc, command, command.Op == "ripple_trim" || command.Ripple)
	case "roll":
		return timelineRoll(doc, command)
	case "slip":
		return timelineSlip(doc, command)
	case "slide":
		return timelineSlide(doc, command)
	case "move":
		return timelineMove(doc, command)
	case "retime":
		return timelineRetime(doc, command)
	case "update_clip":
		return timelineUpdateClip(doc, command)
	case "link", "unlink":
		return timelineLink(doc, command)
	case "add_track":
		return timelineAddTrack(doc, command)
	case "remove_track":
		return timelineRemoveTrack(doc, command)
	case "set_track_state":
		return timelineSetTrackState(doc, command)
	case "set_track_height":
		return timelineSetTrackHeight(doc, command)
	case "reorder_track":
		return timelineReorderTrack(doc, command)
	default:
		return doc, []string{"UNKNOWN_COMMAND: " + command.Op}
	}
}

func timelineBlade(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
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
	if clip.Locked {
		return doc, []string{"CLIP_LOCKED: " + clip.ID}
	}
	if command.AtFrame <= clip.StartFrame || command.AtFrame >= clipEnd(clip) {
		return doc, []string{"BLADE_OUTSIDE_CLIP: " + clip.ID}
	}
	leftDuration := command.AtFrame - clip.StartFrame
	right := clip
	right.ID = fmt.Sprintf("%s-b%d", clip.ID, command.AtFrame)
	right.StartFrame = command.AtFrame
	right.DurationFrames = clip.DurationFrames - leftDuration
	right.SourceInFrame = clip.SourceInFrame + leftDuration
	track.Clips[clipIndex].DurationFrames = leftDuration
	track.Clips = append(track.Clips, right)
	sortTrackClips(track)
	return doc, nil
}

func timelineLift(doc timelineDoc, command timelineCommand, ripple bool) (timelineDoc, []string) {
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
	if clip.Locked {
		return doc, []string{"CLIP_LOCKED: " + clip.ID}
	}
	track.Clips = append(track.Clips[:clipIndex], track.Clips[clipIndex+1:]...)
	if ripple {
		for index := range track.Clips {
			if track.Clips[index].StartFrame >= clipEnd(clip) {
				track.Clips[index].StartFrame -= clip.DurationFrames
				if track.Clips[index].StartFrame < 0 {
					track.Clips[index].StartFrame = 0
				}
			}
		}
	}
	return doc, nil
}

func normalizeClip(clip timelineClip, trackID string) (timelineClip, []string) {
	if clip.MediaID == "" {
		return clip, []string{"CLIP_MEDIA_REQUIRED"}
	}
	if clip.DurationFrames <= 0 {
		return clip, []string{"CLIP_DURATION_INVALID"}
	}
	if clip.Speed <= 0 {
		clip.Speed = 1
	}
	if clip.SourceOutFrame <= clip.SourceInFrame {
		return clip, []string{"CLIP_SOURCE_RANGE_INVALID"}
	}
	if clip.ID == "" {
		return clip, []string{"CLIP_ID_REQUIRED"}
	}
	if trackID != "" {
		clip.TrackID = trackID
	}
	return clip, nil
}

func timelineInsert(doc timelineDoc, command timelineCommand, overwrite bool) (timelineDoc, []string) {
	if command.Clip == nil {
		return doc, []string{"CLIP_REQUIRED"}
	}
	trackID := firstNonEmpty(command.TrackID, command.Clip.TrackID)
	trackIndex := timelineTrackIndex(&doc, trackID)
	if trackIndex < 0 {
		return doc, []string{"TRACK_NOT_FOUND: " + trackID}
	}
	clip, blockers := normalizeClip(*command.Clip, trackID)
	if len(blockers) > 0 {
		return doc, blockers
	}
	track := &doc.Tracks[trackIndex]
	if track.Locked {
		return doc, []string{"TRACK_LOCKED: " + trackID}
	}
	at := command.AtFrame
	if at < 0 {
		at = 0
	}
	clip.StartFrame = at
	if overwrite {
		track.Clips = overwriteRange(track.Clips, at, at+clip.DurationFrames)
	} else {
		track.Clips = splitSpanningClip(track.Clips, at)
		for index := range track.Clips {
			if track.Clips[index].StartFrame >= at {
				track.Clips[index].StartFrame += clip.DurationFrames
			}
		}
	}
	track.Clips = append(track.Clips, clip)
	sortTrackClips(track)
	return doc, nil
}

func splitSpanningClip(clips []timelineClip, at int) []timelineClip {
	result := make([]timelineClip, 0, len(clips)+1)
	for _, clip := range clips {
		if at > clip.StartFrame && at < clipEnd(clip) {
			left := clip
			right := clip
			right.ID = fmt.Sprintf("%s-s%d", clip.ID, at)
			right.StartFrame = at
			right.DurationFrames = clipEnd(clip) - at
			right.SourceInFrame = clip.SourceInFrame + (at - clip.StartFrame)
			left.DurationFrames = at - clip.StartFrame
			result = append(result, left, right)
			continue
		}
		result = append(result, clip)
	}
	return result
}

func overwriteRange(clips []timelineClip, start, end int) []timelineClip {
	result := make([]timelineClip, 0, len(clips)+2)
	for _, clip := range clips {
		clipStart, clipStop := clip.StartFrame, clipEnd(clip)
		if clipStop <= start || clipStart >= end {
			result = append(result, clip)
			continue
		}
		if clipStart < start {
			left := clip
			left.DurationFrames = start - clipStart
			result = append(result, left)
		}
		if clipStop > end {
			right := clip
			right.ID = fmt.Sprintf("%s-o%d", clip.ID, end)
			right.StartFrame = end
			right.DurationFrames = clipStop - end
			right.SourceInFrame = clip.SourceInFrame + (end - clipStart)
			result = append(result, right)
		}
	}
	return result
}

func timelineTrim(doc timelineDoc, command timelineCommand, ripple bool) (timelineDoc, []string) {
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
	newDuration := command.DurationFrames
	if newDuration <= 0 {
		return doc, []string{"TRIM_DURATION_INVALID"}
	}
	delta := newDuration - clip.DurationFrames
	oldEnd := clipEnd(*clip)
	clip.DurationFrames = newDuration
	if track.Kind != "subtitle" && clip.SourceInFrame+newDuration > clip.SourceOutFrame {
		clip.SourceOutFrame = clip.SourceInFrame + newDuration
	}
	if ripple {
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

func timelineLink(doc timelineDoc, command timelineCommand) (timelineDoc, []string) {
	group := command.LinkedGroup
	if command.Op == "unlink" {
		group = ""
	}
	ids := append([]string(nil), command.ClipIDs...)
	if len(ids) == 0 && command.ClipID != "" {
		ids = append(ids, command.ClipID)
	}
	if len(ids) == 0 {
		return doc, []string{"CLIP_IDS_REQUIRED"}
	}
	if command.Op == "link" && group == "" {
		group = "link-" + ids[0]
	}
	found := map[string]bool{}
	for trackIndex := range doc.Tracks {
		for clipIndex := range doc.Tracks[trackIndex].Clips {
			for _, id := range ids {
				if doc.Tracks[trackIndex].Clips[clipIndex].ID == id {
					doc.Tracks[trackIndex].Clips[clipIndex].LinkedGroup = group
					found[id] = true
				}
			}
		}
	}
	for _, id := range ids {
		if !found[id] {
			return doc, []string{"CLIP_NOT_FOUND: " + id}
		}
	}
	return doc, nil
}

func validateTimeline(doc timelineDoc) []string {
	blockers := make([]string, 0)
	if doc.Schema != "" && doc.Schema != "himind.timeline.v1" {
		blockers = append(blockers, "SCHEMA_UNSUPPORTED: "+doc.Schema)
	}
	if doc.Timebase.Num <= 0 || doc.Timebase.Den <= 0 {
		blockers = append(blockers, "TIMEBASE_INVALID")
	}
	if doc.Canvas.Width <= 0 || doc.Canvas.Height <= 0 {
		blockers = append(blockers, "CANVAS_INVALID")
	}
	seenMedia := map[string]bool{}
	for _, media := range doc.Media {
		if media.ID == "" || seenMedia[media.ID] {
			blockers = append(blockers, "MEDIA_ID_DUPLICATE_OR_EMPTY: "+media.ID)
		}
		seenMedia[media.ID] = true
		if media.Kind != "" && media.Kind != "video" && media.Kind != "audio" && media.Kind != "image" && media.Kind != "subtitle" {
			blockers = append(blockers, "MEDIA_KIND_INVALID: "+media.ID)
		}
	}
	seenTracks := map[string]bool{}
	seenClips := map[string]bool{}
	for _, track := range doc.Tracks {
		if track.ID == "" || seenTracks[track.ID] {
			blockers = append(blockers, "TRACK_ID_DUPLICATE_OR_EMPTY: "+track.ID)
		}
		seenTracks[track.ID] = true
		if track.Kind != "video" && track.Kind != "audio" && track.Kind != "subtitle" {
			blockers = append(blockers, "TRACK_KIND_INVALID: "+track.ID)
		}
		if track.Height != 0 && (track.Height < 28 || track.Height > 160) {
			blockers = append(blockers, "TRACK_HEIGHT_INVALID: "+track.ID)
		}
		if len([]rune(track.Name)) > 40 {
			blockers = append(blockers, "TRACK_NAME_TOO_LONG: "+track.ID)
		}
		if track.Color != "" && !validTrackColor(track.Color) {
			blockers = append(blockers, "TRACK_COLOR_INVALID: "+track.ID)
		}
		if track.VolumeDB < -60 || track.VolumeDB > 12 {
			blockers = append(blockers, "TRACK_VOLUME_OUT_OF_RANGE: "+track.ID)
		}
		clips := append([]timelineClip(nil), track.Clips...)
		sort.SliceStable(clips, func(left, right int) bool { return clips[left].StartFrame < clips[right].StartFrame })
		for index, clip := range clips {
			if clip.ID == "" || seenClips[clip.ID] {
				blockers = append(blockers, "CLIP_ID_DUPLICATE_OR_EMPTY: "+clip.ID)
			}
			seenClips[clip.ID] = true
			if clip.DurationFrames <= 0 {
				blockers = append(blockers, "CLIP_DURATION_INVALID: "+clip.ID)
			}
			if clip.StartFrame < 0 {
				blockers = append(blockers, "CLIP_START_NEGATIVE: "+clip.ID)
			}
			if track.Kind == "subtitle" {
				if len([]rune(clip.Text)) == 0 {
					blockers = append(blockers, "SUBTITLE_TEXT_REQUIRED: "+clip.ID)
				}
			} else {
				if clip.MediaID == "" {
					blockers = append(blockers, "CLIP_MEDIA_REQUIRED: "+clip.ID)
				}
				if clip.SourceOutFrame <= clip.SourceInFrame {
					blockers = append(blockers, "CLIP_SOURCE_RANGE_INVALID: "+clip.ID)
				}
			}
			if clip.TransitionIn != nil && clip.TransitionIn.DurationFrames > clip.DurationFrames {
				blockers = append(blockers, "TRANSITION_IN_TOO_LONG: "+clip.ID)
			}
			if clip.TransitionOut != nil && clip.TransitionOut.DurationFrames > clip.DurationFrames {
				blockers = append(blockers, "TRANSITION_OUT_TOO_LONG: "+clip.ID)
			}
			if clip.Transform != nil && clip.Transform.Scale < 0 {
				blockers = append(blockers, "TRANSFORM_SCALE_INVALID: "+clip.ID)
			}
			if clip.Color != nil && len(clip.Color.Keyframes) > 0 {
				lastFrame := -1
				for _, keyframe := range clip.Color.Keyframes {
					if keyframe.Frame < 0 || keyframe.Frame > clip.DurationFrames || keyframe.Frame < lastFrame {
						blockers = append(blockers, "COLOR_KEYFRAME_ORDER_OR_RANGE_INVALID: "+clip.ID)
					}
					lastFrame = keyframe.Frame
				}
			}
			if index > 0 && clip.StartFrame < clipEnd(clips[index-1]) {
				blockers = append(blockers, "CLIP_OVERLAP: "+clips[index-1].ID+" / "+clip.ID)
			}
		}
	}
	return blockers
}

func timelineApply(in requestInput) (any, *himindjsonrpc.Error) {
	if len(in.Commands) == 0 {
		return nil, himindjsonrpc.InvalidParams("commands is required")
	}
	next, blockers := applyTimelineCommands(in.Timeline, in.Commands)
	if len(blockers) > 0 {
		return map[string]any{"ok": false, "applied": false, "revision": next.Revision, "timeline": next, "blockers": blockers}, nil
	}
	if validation := validateTimeline(next); len(validation) > 0 {
		return map[string]any{"ok": false, "applied": true, "revision": next.Revision, "timeline": next, "blockers": validation}, nil
	}
	if strings.TrimSpace(in.ProjectID) != "" {
		if root, workspaceErr := workspace(in.WorkspaceRoot); workspaceErr == nil {
			if item, loadErr := loadProject(root, in.ProjectID); loadErr == nil {
				startRevision := next.Revision - len(in.Commands) + 1
				if logErr := appendTimelineCommandLog(root, item.ID, in.Commands, startRevision); logErr != nil {
					return nil, himindjsonrpc.InternalError("append timeline command log: " + logErr.Error())
				}
			}
		}
	}
	return map[string]any{"ok": true, "applied": true, "revision": next.Revision, "timeline": next, "blockers": []string{}}, nil
}

func timelineValidate(in requestInput) (any, *himindjsonrpc.Error) {
	blockers := validateTimeline(in.Timeline)
	return map[string]any{"ok": len(blockers) == 0, "revision": in.Timeline.Revision, "blockers": blockers}, nil
}
