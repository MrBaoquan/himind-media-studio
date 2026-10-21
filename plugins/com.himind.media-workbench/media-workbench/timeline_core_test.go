package main

import (
	"testing"
)

func timelineFixture() timelineDoc {
	return timelineDoc{
		Schema: "himind.timeline.v1", Revision: 0,
		Timebase: timelineTimebase{Num: 30, Den: 1},
		Canvas:   timelineCanvas{Width: 1080, Height: 1920},
		Tracks: []timelineTrack{{
			ID: "V1", Kind: "video", Index: 1,
			Clips: []timelineClip{{
				ID: "c1", MediaID: "m1", TrackID: "V1",
				StartFrame: 0, DurationFrames: 300, SourceInFrame: 0, SourceOutFrame: 300, Speed: 1,
			}},
		}},
	}
}

func TestTimelineBlade(t *testing.T) {
	next, blockers := applyTimelineCommands(timelineFixture(), []timelineCommand{{Op: "blade", TrackID: "V1", ClipID: "c1", AtFrame: 120}})
	if len(blockers) != 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	clips := next.Tracks[0].Clips
	if len(clips) != 2 || clips[0].DurationFrames != 120 || clips[1].StartFrame != 120 || clips[1].SourceInFrame != 120 {
		t.Fatalf("unexpected blade result: %+v", clips)
	}
	if next.Revision != 1 {
		t.Fatalf("revision = %d", next.Revision)
	}
}

func TestTimelineRippleDelete(t *testing.T) {
	doc := timelineFixture()
	doc.Tracks[0].Clips = append(doc.Tracks[0].Clips, timelineClip{ID: "c2", MediaID: "m2", TrackID: "V1", StartFrame: 300, DurationFrames: 60, SourceInFrame: 0, SourceOutFrame: 60, Speed: 1})
	next, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "ripple_delete", TrackID: "V1", ClipID: "c1"}})
	if len(blockers) != 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	if len(next.Tracks[0].Clips) != 1 || next.Tracks[0].Clips[0].StartFrame != 0 {
		t.Fatalf("unexpected ripple result: %+v", next.Tracks[0].Clips)
	}
}

func TestTimelineInsertSplitsAndShifts(t *testing.T) {
	insertee := timelineClip{ID: "c9", MediaID: "m9", TrackID: "V1", DurationFrames: 60, SourceInFrame: 0, SourceOutFrame: 60, Speed: 1}
	next, blockers := applyTimelineCommands(timelineFixture(), []timelineCommand{{Op: "insert", TrackID: "V1", AtFrame: 120, Clip: &insertee}})
	if len(blockers) != 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	clips := next.Tracks[0].Clips
	if len(clips) != 3 {
		t.Fatalf("expected 3 clips, got %+v", clips)
	}
	if clips[1].ID != "c9" || clips[1].StartFrame != 120 || clips[2].StartFrame != 180 {
		t.Fatalf("unexpected insert result: %+v", clips)
	}
}

func TestTimelineOverwriteTrims(t *testing.T) {
	insertee := timelineClip{ID: "c9", MediaID: "m9", TrackID: "V1", DurationFrames: 100, SourceInFrame: 0, SourceOutFrame: 100, Speed: 1}
	next, blockers := applyTimelineCommands(timelineFixture(), []timelineCommand{{Op: "overwrite", TrackID: "V1", AtFrame: 100, Clip: &insertee}})
	if len(blockers) != 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	clips := next.Tracks[0].Clips
	if len(clips) != 3 {
		t.Fatalf("expected 3 clips, got %+v", clips)
	}
	if clips[0].DurationFrames != 100 || clips[2].StartFrame != 200 || clips[2].SourceInFrame != 200 {
		t.Fatalf("unexpected overwrite result: %+v", clips)
	}
}

func TestTimelineValidateOverlap(t *testing.T) {
	doc := timelineFixture()
	doc.Tracks[0].Clips = append(doc.Tracks[0].Clips, timelineClip{ID: "c2", MediaID: "m2", TrackID: "V1", StartFrame: 200, DurationFrames: 100, SourceInFrame: 0, SourceOutFrame: 100, Speed: 1})
	blockers := validateTimeline(doc)
	found := false
	for _, blocker := range blockers {
		if len(blocker) >= 12 && blocker[:12] == "CLIP_OVERLAP" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected overlap blocker, got %v", blockers)
	}
}

func TestTimelineRevisionConflict(t *testing.T) {
	_, blockers := applyTimelineCommands(timelineFixture(), []timelineCommand{{Op: "blade", TrackID: "V1", ClipID: "c1", AtFrame: 120, ExpectedRevision: 99}})
	if len(blockers) == 0 || blockers[0][:18] != "REVISION_CONFLICT:" {
		t.Fatalf("expected revision conflict, got %v", blockers)
	}
}
