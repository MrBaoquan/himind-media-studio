package main

import "testing"

func TestTimelineAutoEditDistributesDuration(t *testing.T) {
	in := requestInput{MediaIDs: []string{"a", "b", "c"}, DurationSeconds: 10, Transition: "fade", TransitionFrames: 6}
	in.Timeline.Timebase = timelineTimebase{Num: 30, Den: 1}
	value, rpcErr := timelineAutoEditCapability(in)
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	payload := value.(map[string]any)
	if payload["state"] != "ready" {
		t.Fatalf("state: %+v", payload)
	}
	doc := payload["timeline"].(timelineDoc)
	if doc.DurationFrames != 300 {
		t.Fatalf("duration=%d", doc.DurationFrames)
	}
	if len(doc.Tracks[0].Clips) != 3 || doc.Tracks[0].Clips[0].DurationFrames != 100 || doc.Tracks[0].Clips[2].StartFrame != 200 {
		t.Fatalf("unexpected clips: %+v", doc.Tracks[0].Clips)
	}
	if blockers := validateTimeline(doc); len(blockers) > 0 {
		t.Fatalf("blockers: %v", blockers)
	}
}

func TestSubtitleTrackFromCues(t *testing.T) {
	doc := timelineDoc{Schema: "himind.timeline.v1", Timebase: timelineTimebase{Num: 30, Den: 1}, Canvas: timelineCanvas{Width: 1080, Height: 1920}}
	track := subtitleTrackFromCues(doc, []subtitleCue{{Start: 0.5, End: 1.5, Text: "第一句"}, {Start: 2, End: 3, Text: "第二句"}})
	if track.Kind != "subtitle" || len(track.Clips) != 2 || track.Clips[0].StartFrame != 15 || track.Clips[0].DurationFrames != 30 {
		t.Fatalf("unexpected subtitle track: %+v", track)
	}
	doc.Tracks = append(doc.Tracks, track)
	if blockers := validateTimeline(doc); len(blockers) > 0 {
		t.Fatalf("blockers: %v", blockers)
	}
}

func TestDeliveryDimensionsPortrait(t *testing.T) {
	width, height := deliveryDimensions(deliveryPresetByID("web"), timelineCanvas{Width: 1080, Height: 1920})
	if width != 720 || height != 1280 {
		t.Fatalf("web dimensions=%dx%d", width, height)
	}
	width, height = deliveryDimensions(deliveryPresetByID("douyin"), timelineCanvas{Width: 1920, Height: 1080})
	if width != 1080 || height != 1920 {
		t.Fatalf("douyin dimensions=%dx%d", width, height)
	}
}

func TestSRTTimestamp(t *testing.T) {
	if got := srtTimestamp(65.25); got != "00:01:05,250" {
		t.Fatalf("timestamp=%s", got)
	}
}
