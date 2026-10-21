package main

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type editTemplateSlot struct {
	Role          string  `json:"role"`
	DurationRatio float64 `json:"duration_ratio"`
	MinSeconds    float64 `json:"min_seconds,omitempty"`
}

type editTemplate struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Slots       []editTemplateSlot `json:"slots"`
	Tags        []string           `json:"tags,omitempty"`
}

var editTemplates = []editTemplate{
	{ID: "hook-proof-payoff", Name: "钩子-论证-收束", Description: "3 段式科普/观点短片，先抛结论，再给证据，最后收束。", Tags: []string{"explainer", "three-act"}, Slots: []editTemplateSlot{{Role: "hook", DurationRatio: 0.2, MinSeconds: 2}, {Role: "proof", DurationRatio: 0.5, MinSeconds: 3}, {Role: "payoff", DurationRatio: 0.3, MinSeconds: 2}}},
	{ID: "six-beat-story", Name: "六拍故事", Description: "起势、激励、推进、转折、高潮、余韵。", Tags: []string{"story", "narrative"}, Slots: []editTemplateSlot{{Role: "setup"}, {Role: "inciting"}, {Role: "progress"}, {Role: "turn"}, {Role: "climax"}, {Role: "resolution"}}},
	{ID: "explainer-five", Name: "五段式讲解", Description: "问题、背景、机制、证据、结论。", Tags: []string{"science", "explainer"}, Slots: []editTemplateSlot{{Role: "question"}, {Role: "context"}, {Role: "mechanism"}, {Role: "evidence"}, {Role: "conclusion"}}},
}

func editTemplateByID(id string) editTemplate {
	id = strings.TrimSpace(id)
	for _, template := range editTemplates {
		if template.ID == id {
			return template
		}
	}
	return editTemplates[0]
}

func editTemplateListCapability() any {
	return map[string]any{"schema": "himind.edit-template.v1", "templates": editTemplates, "total": len(editTemplates)}
}

func timelineTemplateApplyCapability(in requestInput) (any, *himindjsonrpc.Error) {
	template := editTemplateByID(in.TemplateID)
	if len(in.MediaIDs) < len(template.Slots) {
		return map[string]any{"state": "blocked", "blockers": []string{fmt.Sprintf("TEMPLATE_REQUIRES_%d_MEDIA", len(template.Slots))}}, nil
	}
	ids := in.MediaIDs[:len(template.Slots)]
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
	duration := in.DurationSeconds
	if duration <= 0 {
		duration = 15
	}
	targetFrames := int(math.Round(duration * float64(fps) / float64(den)))
	if targetFrames < len(ids) {
		targetFrames = len(ids)
	}
	slotFrames := allocateTemplateFrames(template, targetFrames, float64(fps)/float64(den))
	clips := make([]timelineClip, 0, len(ids))
	markers := make([]map[string]any, 0, len(ids))
	cursor := 0
	for index, id := range ids {
		frames := slotFrames[index]
		clip := timelineClip{ID: fmt.Sprintf("tpl-%02d-%s", index+1, sanitizeTimelineID(id)), MediaID: id, TrackID: "V1", StartFrame: cursor, DurationFrames: frames, SourceInFrame: 0, SourceOutFrame: frames, Speed: 1}
		if index > 0 {
			clip.TransitionIn = &timelineTransition{Type: firstNonEmpty(in.Transition, "fade"), DurationFrames: minTransitionFrames(6, frames/4)}
		}
		if index < len(ids)-1 {
			clip.TransitionOut = &timelineTransition{Type: firstNonEmpty(in.Transition, "fade"), DurationFrames: minTransitionFrames(6, frames/4)}
		}
		clips = append(clips, clip)
		markers = append(markers, map[string]any{"frame": cursor, "label": template.Slots[index].Role, "kind": "template-slot", "template_id": template.ID})
		cursor += frames
	}
	doc := timelineDoc{Schema: "himind.timeline.v1", Revision: 0, Timebase: timelineTimebase{Num: fps, Den: den}, DurationFrames: targetFrames, Canvas: canvas, Tracks: []timelineTrack{{ID: "V1", Kind: "video", Index: 1, Role: "program", Clips: clips}}, Media: append([]timelineMediaRef(nil), in.Timeline.Media...), Markers: markers}
	if blockers := validateTimeline(doc); len(blockers) > 0 {
		return map[string]any{"state": "blocked", "blockers": blockers, "timeline": doc}, nil
	}
	return map[string]any{"state": "ready", "template_id": template.ID, "timeline": doc, "slots": template.Slots, "clip_count": len(clips), "duration_seconds": float64(targetFrames) * float64(den) / float64(fps)}, nil
}

func allocateTemplateFrames(template editTemplate, total int, fps float64) []int {
	ratios := make([]float64, len(template.Slots))
	sum := 0.0
	for index, slot := range template.Slots {
		if slot.DurationRatio <= 0 {
			ratios[index] = 1
		} else {
			ratios[index] = slot.DurationRatio
		}
		sum += ratios[index]
	}
	frames := make([]int, len(ratios))
	used := 0
	for index, ratio := range ratios {
		value := int(math.Floor(float64(total) * ratio / sum))
		minFrames := 1
		if template.Slots[index].MinSeconds > 0 {
			minFrames = int(math.Ceil(template.Slots[index].MinSeconds * fps))
		}
		if value < minFrames {
			value = minFrames
		}
		frames[index] = value
		used += value
	}
	frames[len(frames)-1] += total - used
	if frames[len(frames)-1] < 1 {
		frames[len(frames)-1] = 1
	}
	return frames
}

func minTransitionFrames(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func transcriptQueryTerms(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return r == ' ' || r == ',' || r == rune(0xFF0C) || r == rune(0x3001) || r == rune(10) || r == rune(9)
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func matchSubtitleCue(text, query, mode string) bool {
	text = strings.ToLower(text)
	terms := transcriptQueryTerms(query)
	if len(terms) == 0 {
		return true
	}
	if strings.EqualFold(mode, "all") {
		for _, term := range terms {
			if !strings.Contains(text, term) {
				return false
			}
		}
		return true
	}
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func timelineTranscriptSearchCapability(in requestInput) any {
	cues := append([]subtitleCue(nil), in.Subtitles...)
	if len(cues) == 0 {
		cues = subtitleCuesFromTimeline(in.Timeline)
	}
	hits := make([]map[string]any, 0)
	for index, cue := range cues {
		if matchSubtitleCue(cue.Text, in.Query, in.MatchMode) {
			hits = append(hits, map[string]any{"index": index, "start": cue.Start, "end": cue.End, "text": cue.Text})
		}
	}
	return map[string]any{"state": "ready", "query": in.Query, "hits": hits, "total": len(hits), "cues": len(cues)}
}

func timelineRoughCutCapability(in requestInput) (any, *himindjsonrpc.Error) {
	doc := in.Timeline
	if len(doc.Tracks) == 0 {
		return nil, himindjsonrpc.InvalidParams("timeline is required for rough cut")
	}
	cues := subtitleCuesFromTimeline(doc)
	var videoTrack *timelineTrack
	for index := range doc.Tracks {
		if doc.Tracks[index].Kind == "video" && len(doc.Tracks[index].Clips) > 0 {
			videoTrack = &doc.Tracks[index]
			break
		}
	}
	if videoTrack == nil || len(cues) == 0 {
		return map[string]any{"state": "blocked", "blockers": []string{"TIMELINE_VIDEO_OR_SUBTITLES_MISSING"}}, nil
	}
	selected := make([]int, 0)
	if len(in.CueIndexes) > 0 {
		selected = append(selected, in.CueIndexes...)
	} else {
		for index, cue := range cues {
			if matchSubtitleCue(cue.Text, in.Query, in.MatchMode) {
				selected = append(selected, index)
			}
		}
	}
	if len(selected) == 0 {
		return map[string]any{"state": "empty", "hits": 0, "blockers": []string{"NO_TRANSCRIPT_HITS"}}, nil
	}
	sort.Ints(selected)
	fps := doc.Timebase.Num
	if fps <= 0 {
		fps = 30
	}
	den := doc.Timebase.Den
	if den <= 0 {
		den = 1
	}
	baseMedia := videoTrack.Clips[0].MediaID
	padding := in.PaddingSeconds
	if padding < 0 {
		padding = 0
	}
	if padding == 0 {
		padding = 0.15
	}
	maxFrames := int(math.Round(in.DurationSeconds * float64(fps) / float64(den)))
	if maxFrames <= 0 {
		maxFrames = doc.DurationFrames
	}
	clips := make([]timelineClip, 0)
	subtitleClips := make([]timelineClip, 0)
	cursor := 0
	for _, cueIndex := range selected {
		if cueIndex < 0 || cueIndex >= len(cues) {
			continue
		}
		cue := cues[cueIndex]
		start := math.Max(0, cue.Start-padding)
		end := cue.End + padding
		frames := int(math.Round((end - start) * float64(fps) / float64(den)))
		if frames <= 0 {
			continue
		}
		if maxFrames > 0 && cursor+frames > maxFrames {
			frames = maxFrames - cursor
			if frames <= 0 {
				break
			}
		}
		clips = append(clips, timelineClip{ID: fmt.Sprintf("rough-%03d", len(clips)+1), MediaID: baseMedia, TrackID: "V1", StartFrame: cursor, DurationFrames: frames, SourceInFrame: int(math.Round(start * float64(fps) / float64(den))), SourceOutFrame: int(math.Round(start*float64(fps)/float64(den))) + frames, Speed: 1})
		subtitleClips = append(subtitleClips, timelineClip{ID: fmt.Sprintf("rough-sub-%03d", len(subtitleClips)+1), TrackID: "S1", StartFrame: cursor, DurationFrames: frames, Text: cue.Text})
		cursor += frames
	}
	if len(clips) == 0 {
		return map[string]any{"state": "empty", "blockers": []string{"ROUGH_CUT_EMPTY"}}, nil
	}
	out := timelineDoc{Schema: "himind.timeline.v1", Revision: 0, Timebase: doc.Timebase, DurationFrames: cursor, Canvas: doc.Canvas, Tracks: []timelineTrack{{ID: "V1", Kind: "video", Index: 1, Role: "program", Clips: clips}, {ID: "S1", Kind: "subtitle", Index: 1, Role: "caption", Clips: subtitleClips}}, Media: append([]timelineMediaRef(nil), doc.Media...), Markers: []map[string]any{{"frame": 0, "label": "AI 粗剪", "kind": "rough-cut", "query": in.Query}}}
	if blockers := validateTimeline(out); len(blockers) > 0 {
		return map[string]any{"state": "blocked", "blockers": blockers, "timeline": out}, nil
	}
	return map[string]any{"state": "ready", "timeline": out, "selected_cues": len(selected), "clip_count": len(clips), "duration_seconds": float64(cursor) * float64(den) / float64(fps), "base_media_id": baseMedia}, nil
}
