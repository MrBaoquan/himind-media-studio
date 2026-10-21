package main

import (
	"encoding/json"
	"testing"
)

func TestTimelineMoveChangesStartOnly(t *testing.T) {
	next, blockers := applyTimelineCommands(timelineEditFixture(), []timelineCommand{{Op: "move", TrackID: "V1", ClipID: "b", AtFrame: 90}})
	if len(blockers) > 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	clip := next.Tracks[0].Clips[1]
	if clip.ID != "b" || clip.StartFrame != 90 || clip.DurationFrames != 60 || clip.SourceInFrame != 0 {
		t.Fatalf("unexpected move: %+v", clip)
	}
}

func TestTimelineMoveRejectsOverlap(t *testing.T) {
	_, blockers := applyTimelineCommands(timelineEditFixture(), []timelineCommand{{Op: "move", TrackID: "V1", ClipID: "a", AtFrame: 70}})
	if len(blockers) == 0 || blockers[0] != "CLIP_OVERLAP: b / a" {
		t.Fatalf("expected overlap blocker, got %v", blockers)
	}
}

func TestTimelineMoveLinkedGroupTogether(t *testing.T) {
	doc := timelineEditFixture()
	doc.Tracks[0].Clips[0].LinkedGroup = "av-1"
	doc.Tracks = append(doc.Tracks, timelineTrack{ID: "A1", Kind: "audio", Index: 2, Clips: []timelineClip{{
		ID: "a-audio", MediaID: "m1", TrackID: "A1", StartFrame: 0, DurationFrames: 60, SourceInFrame: 0, SourceOutFrame: 120, Speed: 1, LinkedGroup: "av-1",
	}}})
	next, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "move", TrackID: "V1", ClipID: "a", AtFrame: 120}})
	if len(blockers) > 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	startByID := map[string]int{}
	for _, track := range next.Tracks {
		for _, clip := range track.Clips {
			startByID[clip.ID] = clip.StartFrame
		}
	}
	if startByID["a"] != 120 || startByID["a-audio"] != 120 {
		t.Fatalf("linked move failed: %+v", startByID)
	}
}

func TestOTIORoundTripPreservesClipTiming(t *testing.T) {
	doc := timelineEditFixture()
	doc.Tracks[0].Clips[0].TransitionIn = &timelineTransition{Type: "fade", DurationFrames: 8}
	payload := timelineToOTIO(doc)
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	restored, err := otioToTimeline(decoded)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	before, after := doc.Tracks[0].Clips, restored.Tracks[0].Clips
	for i := range before {
		if before[i].StartFrame != after[i].StartFrame || before[i].DurationFrames != after[i].DurationFrames || before[i].SourceInFrame != after[i].SourceInFrame {
			t.Fatalf("OTIO changed timing: before=%+v after=%+v", before[i], after[i])
		}
	}
	if restored.Tracks[0].Clips[0].TransitionIn == nil || restored.Tracks[0].Clips[0].TransitionIn.DurationFrames != 8 {
		t.Fatalf("OTIO lost transition metadata: %+v", restored.Tracks[0].Clips[0])
	}
}

func TestEDLRoundTripPreservesClipTiming(t *testing.T) {
	doc := timelineEditFixture()
	restored, err := edlToTimeline(timelineToEDL(doc))
	if err != nil {
		t.Fatalf("EDL import: %v", err)
	}
	before, after := doc.Tracks[0].Clips, restored.Tracks[0].Clips
	if len(before) != len(after) {
		t.Fatalf("EDL clip count changed: %d vs %d", len(before), len(after))
	}
	for i := range before {
		if before[i].StartFrame != after[i].StartFrame || before[i].DurationFrames != after[i].DurationFrames || before[i].SourceInFrame != after[i].SourceInFrame {
			t.Fatalf("EDL changed timing: before=%+v after=%+v", before[i], after[i])
		}
	}
}

func TestTimelineSemanticSearchFindsClipAndMedia(t *testing.T) {
	doc := timelineEditFixture()
	doc.Tracks[0].Clips[0].ID = "hefei-sun-video"
	doc.Tracks = append(doc.Tracks, timelineTrack{ID: "S1", Kind: "subtitle", Index: 2, Clips: []timelineClip{{
		ID: "sub-sun", TrackID: "S1", StartFrame: 10, DurationFrames: 40, Text: "合肥人造太阳托卡马克",
	}}})
	result, rpcErr := timelineSemanticSearchCapability(requestInput{Timeline: doc, Query: "合肥 人造太阳 视频", MediaPaths: map[string]string{"m1": "assets/hefei-sun-lab.mp4"}, Limit: 10})
	if rpcErr != nil {
		t.Fatalf("semantic: %v", rpcErr)
	}
	payload := result.(map[string]any)
	hits := payload["hits"].([]semanticHit)
	if len(hits) == 0 {
		t.Fatalf("expected semantic hits, got %+v", payload)
	}
	foundClip := false
	for _, hit := range hits {
		if hit.Kind == "clip" && (hit.ClipID == "sub-sun" || hit.ClipID == "hefei-sun-video") {
			foundClip = true
		}
	}
	if !foundClip {
		t.Fatalf("expected a matching clip, got %+v", hits)
	}
}

func TestTrimEdgeLeftKeepsEndAndAdvancesSourceIn(t *testing.T) {
	next, blockers := applyTimelineCommands(timelineEditFixture(), []timelineCommand{{Op: "trim_edge", TrackID: "V1", ClipID: "a", Edge: "left", DeltaFrames: 10}})
	if len(blockers) > 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	clip := next.Tracks[0].Clips[0]
	if clip.StartFrame != 10 || clip.DurationFrames != 50 || clip.SourceInFrame != 10 || clipEnd(clip) != 60 {
		t.Fatalf("unexpected left trim: %+v", clip)
	}
}

func TestTrimEdgeRightShrinksDuration(t *testing.T) {
	next, blockers := applyTimelineCommands(timelineEditFixture(), []timelineCommand{{Op: "trim_edge", TrackID: "V1", ClipID: "a", Edge: "right", DeltaFrames: -10}})
	if len(blockers) > 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	clip := next.Tracks[0].Clips[0]
	if clip.StartFrame != 0 || clip.DurationFrames != 50 {
		t.Fatalf("unexpected right trim: %+v", clip)
	}
}
