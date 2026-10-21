package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func timelineAutoEditCapability(in requestInput) (any, *himindjsonrpc.Error) {
	ids := make([]string, 0, len(in.MediaIDs))
	seen := map[string]bool{}
	for _, id := range in.MediaIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, himindjsonrpc.InvalidParams("media_ids is required")
	}
	fps := in.Timeline.Timebase.Num
	if fps <= 0 {
		fps = 30
	}
	den := in.Timeline.Timebase.Den
	if den <= 0 {
		den = 1
	}
	duration := in.DurationSeconds
	if duration <= 0 {
		duration = float64(len(ids)) * 3
	}
	targetFrames := int(math.Round(duration * float64(fps) / float64(den)))
	if targetFrames < len(ids) {
		targetFrames = len(ids)
	}
	canvas := in.Timeline.Canvas
	if canvas.Width <= 0 || canvas.Height <= 0 {
		canvas = timelineCanvas{Width: 1080, Height: 1920}
	}
	transitionFrames := in.TransitionFrames
	if transitionFrames < 0 {
		transitionFrames = 0
	}
	transitionType := strings.TrimSpace(in.Transition)
	clips := make([]timelineClip, 0, len(ids))
	cursor := 0
	remaining := targetFrames
	for index, mediaID := range ids {
		remainingSlots := len(ids) - index
		frames := remaining / remainingSlots
		if frames <= 0 {
			frames = 1
		}
		if index == len(ids)-1 {
			frames = remaining
		}
		clip := timelineClip{
			ID:             fmt.Sprintf("auto-%02d-%s", index+1, sanitizeTimelineID(mediaID)),
			MediaID:        mediaID,
			TrackID:        "V1",
			StartFrame:     cursor,
			DurationFrames: frames,
			SourceInFrame:  0,
			SourceOutFrame: frames,
			Speed:          1,
		}
		if transitionType != "" && transitionFrames > 0 {
			used := transitionFrames
			if used > frames/4 {
				used = frames / 4
			}
			if used > 0 {
				t := &timelineTransition{Type: transitionType, DurationFrames: used}
				if index > 0 {
					clip.TransitionIn = t
				}
				if index < len(ids)-1 {
					clip.TransitionOut = &timelineTransition{Type: transitionType, DurationFrames: used}
				}
			}
		}
		clips = append(clips, clip)
		cursor += frames
		remaining -= frames
	}
	tracks := []timelineTrack{{ID: "V1", Kind: "video", Index: 1, Role: "program", Clips: clips}}
	media := append([]timelineMediaRef(nil), in.Timeline.Media...)
	if strings.TrimSpace(in.MusicPath) != "" {
		tracks = append(tracks, timelineTrack{
			ID: "A1", Kind: "audio", Index: 1, Role: "music",
			Clips: []timelineClip{{ID: "auto-music", MediaID: "auto-music", TrackID: "A1", StartFrame: 0, DurationFrames: targetFrames, SourceInFrame: 0, SourceOutFrame: targetFrames, Speed: 1, VolumeDB: -18}},
		})
		media = append(media, timelineMediaRef{ID: "auto-music", Kind: "audio", URI: in.MusicPath})
	}
	if strings.TrimSpace(in.NarrationPath) != "" {
		tracks = append(tracks, timelineTrack{
			ID: "A2", Kind: "audio", Index: 2, Role: "narration",
			Clips: []timelineClip{{ID: "auto-narration", MediaID: "auto-narration", TrackID: "A2", StartFrame: 0, DurationFrames: targetFrames, SourceInFrame: 0, SourceOutFrame: targetFrames, Speed: 1, VolumeDB: 0}},
		})
		media = append(media, timelineMediaRef{ID: "auto-narration", Kind: "audio", URI: in.NarrationPath})
	}
	doc := timelineDoc{
		Schema: "himind.timeline.v1", Revision: 0,
		Timebase: timelineTimebase{Num: fps, Den: den}, DurationFrames: targetFrames,
		Canvas: canvas, Tracks: tracks, Media: media,
		Markers: []map[string]any{{"frame": 0, "label": "自动成片起点", "kind": "autoedit"}},
	}
	if len(in.Subtitles) > 0 {
		doc.Tracks = append(doc.Tracks, subtitleTrackFromCues(doc, in.Subtitles))
	}
	if blockers := validateTimeline(doc); len(blockers) > 0 {
		return map[string]any{"state": "blocked", "blockers": blockers, "timeline": doc}, nil
	}
	return map[string]any{
		"state": "ready", "timeline": doc, "clip_count": len(clips),
		"duration_seconds": float64(targetFrames) * float64(den) / float64(fps),
		"next":             "保存 Timeline，预览后按需精修或直接渲染",
	}, nil
}

func sanitizeTimelineID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 32 {
		out = out[len(out)-32:]
	}
	if out == "" {
		out = "media"
	}
	return out
}

func subtitleTrackFromCues(doc timelineDoc, cues []subtitleCue) timelineTrack {
	fps := doc.Timebase.Num
	if fps <= 0 {
		fps = 30
	}
	den := doc.Timebase.Den
	if den <= 0 {
		den = 1
	}
	clips := make([]timelineClip, 0, len(cues))
	for index, cue := range cues {
		start := int(math.Round(cue.Start * float64(fps) / float64(den)))
		end := int(math.Round(cue.End * float64(fps) / float64(den)))
		if end <= start {
			end = start + 1
		}
		clips = append(clips, timelineClip{
			ID: fmt.Sprintf("subtitle-%03d", index+1), TrackID: "S1", StartFrame: start,
			DurationFrames: end - start, Text: strings.TrimSpace(cue.Text),
		})
	}
	sort.SliceStable(clips, func(i, j int) bool { return clips[i].StartFrame < clips[j].StartFrame })
	return timelineTrack{ID: "S1", Kind: "subtitle", Index: 1, Role: "caption", Clips: clips}
}

func timelineSubtitleAttachCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	doc := in.Timeline
	if len(doc.Tracks) == 0 && strings.TrimSpace(in.ProjectID) != "" {
		item, loadErr := loadProject(root, in.ProjectID)
		if loadErr != nil {
			return nil, himindjsonrpc.InvalidParams(loadErr.Error())
		}
		path := timelineStorePath(root, item.ID)
		if readErr := readJSON(path, &doc); readErr != nil {
			return nil, himindjsonrpc.InvalidParams("timeline not found: " + readErr.Error())
		}
	}
	cues := append([]subtitleCue(nil), in.Subtitles...)
	if len(cues) == 0 {
		if strings.TrimSpace(in.SubtitlePath) != "" {
			path, pathErr := workspaceFile(root, in.SubtitlePath)
			if pathErr != nil {
				return nil, himindjsonrpc.InvalidParams("subtitle_path: " + pathErr.Error())
			}
			_, loaded, readErr := readSubtitlePayload(path)
			if readErr != nil {
				return nil, himindjsonrpc.InvalidParams("subtitle_path: " + readErr.Error())
			}
			cues = loaded
		} else if strings.TrimSpace(in.ProjectID) != "" {
			item, loadErr := loadProject(root, in.ProjectID)
			if loadErr == nil {
				if path, pathErr := subtitleDataPath(root, item); pathErr == nil {
					_, loaded, readErr := readSubtitlePayload(path)
					if readErr == nil {
						cues = loaded
					}
				}
			}
		}
	}
	if len(cues) == 0 {
		return map[string]any{"state": "missing", "message": "没有可挂载的字幕", "blockers": []string{"SUBTITLES_EMPTY"}}, nil
	}
	track := subtitleTrackFromCues(doc, cues)
	replaced := false
	for index := range doc.Tracks {
		if doc.Tracks[index].Kind == "subtitle" {
			doc.Tracks[index] = track
			replaced = true
			break
		}
	}
	if !replaced {
		doc.Tracks = append(doc.Tracks, track)
	}
	if blockers := validateTimeline(doc); len(blockers) > 0 {
		return map[string]any{"state": "blocked", "blockers": blockers, "timeline": doc}, nil
	}
	return map[string]any{"state": "ready", "timeline": doc, "cue_count": len(cues), "track_id": "S1"}, nil
}

func subtitleCuesFromTimeline(doc timelineDoc) []subtitleCue {
	fps := doc.Timebase.Num
	if fps <= 0 {
		fps = 30
	}
	den := doc.Timebase.Den
	if den <= 0 {
		den = 1
	}
	cues := make([]subtitleCue, 0)
	for _, track := range doc.Tracks {
		if track.Kind != "subtitle" || track.Muted {
			continue
		}
		for _, clip := range track.Clips {
			text := strings.TrimSpace(clip.Text)
			if text == "" {
				continue
			}
			cues = append(cues, subtitleCue{
				Start: float64(clip.StartFrame) * float64(den) / float64(fps),
				End:   float64(clip.StartFrame+clip.DurationFrames) * float64(den) / float64(fps),
				Text:  text,
			})
		}
	}
	sort.SliceStable(cues, func(i, j int) bool { return cues[i].Start < cues[j].Start })
	return cues
}

func writeSRT(path string, cues []subtitleCue) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	var b strings.Builder
	for index, cue := range cues {
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", index+1, srtTimestamp(cue.Start), srtTimestamp(cue.End), cue.Text)
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}

func srtTimestamp(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	totalMillis := int(math.Round(seconds * 1000))
	hours := totalMillis / 3600000
	totalMillis %= 3600000
	minutes := totalMillis / 60000
	totalMillis %= 60000
	secs := totalMillis / 1000
	millis := totalMillis % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}
