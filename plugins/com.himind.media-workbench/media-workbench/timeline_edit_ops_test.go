package main

import "testing"

func timelineEditFixture() timelineDoc {
	return timelineDoc{
		Schema: "himind.timeline.v1", Revision: 0,
		Timebase: timelineTimebase{Num: 30, Den: 1},
		Canvas:   timelineCanvas{Width: 1080, Height: 1920},
		Tracks: []timelineTrack{{
			ID: "V1", Kind: "video", Index: 1,
			Clips: []timelineClip{
				{ID: "a", MediaID: "m1", TrackID: "V1", StartFrame: 0, DurationFrames: 60, SourceInFrame: 0, SourceOutFrame: 120, Speed: 1},
				{ID: "b", MediaID: "m2", TrackID: "V1", StartFrame: 60, DurationFrames: 60, SourceInFrame: 0, SourceOutFrame: 120, Speed: 1},
			},
		}},
	}
}

func TestTimelineRippleTrimShiftsFollowingClips(t *testing.T) {
	next, blockers := applyTimelineCommands(timelineEditFixture(), []timelineCommand{{Op: "ripple_trim", TrackID: "V1", ClipID: "a", DurationFrames: 40}})
	if len(blockers) > 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	if next.Tracks[0].Clips[0].DurationFrames != 40 || next.Tracks[0].Clips[1].StartFrame != 40 {
		t.Fatalf("unexpected ripple trim: %+v", next.Tracks[0].Clips)
	}
}

func TestTimelineRollKeepsTotalDuration(t *testing.T) {
	next, blockers := applyTimelineCommands(timelineEditFixture(), []timelineCommand{{Op: "roll", TrackID: "V1", ClipID: "a", DeltaFrames: 10}})
	if len(blockers) > 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	clips := next.Tracks[0].Clips
	if clips[0].DurationFrames != 70 || clips[1].StartFrame != 70 || clips[1].DurationFrames != 50 || clips[1].SourceInFrame != 10 {
		t.Fatalf("unexpected roll: %+v", clips)
	}
	if clipEnd(clips[1]) != 120 {
		t.Fatalf("roll changed total duration: %+v", clips)
	}
}

func TestTimelineSlipChangesSourceOnly(t *testing.T) {
	next, blockers := applyTimelineCommands(timelineEditFixture(), []timelineCommand{{Op: "slip", TrackID: "V1", ClipID: "a", DeltaFrames: 15}})
	if len(blockers) > 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	clip := next.Tracks[0].Clips[0]
	if clip.StartFrame != 0 || clip.DurationFrames != 60 || clip.SourceInFrame != 15 || clip.SourceOutFrame != 135 {
		t.Fatalf("unexpected slip: %+v", clip)
	}
}

func TestTimelineSlidePreservesTrackLength(t *testing.T) {
	next, blockers := applyTimelineCommands(timelineEditFixture(), []timelineCommand{{Op: "slide", TrackID: "V1", ClipID: "b", DeltaFrames: -10}})
	if len(blockers) > 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	clips := next.Tracks[0].Clips
	if clips[0].DurationFrames != 50 || clips[1].StartFrame != 50 || clips[1].DurationFrames != 60 || clipEnd(clips[1]) != 110 {
		t.Fatalf("unexpected slide: %+v", clips)
	}
}

func TestTimelineUpdateClipTransformAndLink(t *testing.T) {
	doc := timelineEditFixture()
	updated := doc.Tracks[0].Clips[0]
	updated.Transform = &timelineTransform{Scale: 1.2, PositionX: 0.1, Fit: "cover"}
	next, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "update_clip", TrackID: "V1", ClipID: "a", Clip: &updated}})
	if len(blockers) > 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	if next.Tracks[0].Clips[0].Transform == nil || next.Tracks[0].Clips[0].Transform.Scale != 1.2 {
		t.Fatalf("transform not updated: %+v", next.Tracks[0].Clips[0])
	}
	linked, blockers := applyTimelineCommands(next, []timelineCommand{{Op: "link", ClipIDs: []string{"a", "b"}, LinkedGroup: "av-1"}})
	if len(blockers) > 0 {
		t.Fatalf("link blockers: %v", blockers)
	}
	if linked.Tracks[0].Clips[0].LinkedGroup != "av-1" || linked.Tracks[0].Clips[1].LinkedGroup != "av-1" {
		t.Fatalf("link failed: %+v", linked.Tracks[0].Clips)
	}
}

func TestTimelineSubtitleValidation(t *testing.T) {
	doc := timelineDoc{
		Schema: "himind.timeline.v1", Timebase: timelineTimebase{Num: 30, Den: 1}, Canvas: timelineCanvas{Width: 1080, Height: 1920},
		Tracks: []timelineTrack{{ID: "S1", Kind: "subtitle", Index: 1, Clips: []timelineClip{{ID: "s1", TrackID: "S1", StartFrame: 0, DurationFrames: 30, Text: "你好"}}}},
	}
	if blockers := validateTimeline(doc); len(blockers) > 0 {
		t.Fatalf("subtitle blockers: %v", blockers)
	}
	doc.Tracks[0].Clips[0].Text = ""
	if blockers := validateTimeline(doc); len(blockers) == 0 {
		t.Fatal("empty subtitle text should block")
	}
}

func TestTimelineRetime(t *testing.T) {
	doc := timelineEditFixture()
	doc.Tracks[0].Clips[0].SourceOutFrame = 60
	speed := 2.0
	next, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "retime", TrackID: "V1", ClipID: "a", Speed: &speed, Ripple: true}})
	if len(blockers) > 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	clips := next.Tracks[0].Clips
	if clips[0].DurationFrames != 30 || clips[0].Speed != 2 || clips[1].StartFrame != 30 {
		t.Fatalf("unexpected retime: %+v", clips)
	}
}
