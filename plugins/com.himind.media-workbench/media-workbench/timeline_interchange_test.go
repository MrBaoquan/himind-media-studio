package main

import (
	"reflect"
	"testing"
)

func interchangeFixture() timelineDoc {
	doc := timelineDoc{
		Schema: "himind.timeline.v1", Revision: 3,
		Timebase: timelineTimebase{Num: 30, Den: 1},
		Canvas:   timelineCanvas{Width: 1920, Height: 1080},
		Media: []timelineMediaRef{
			{ID: "m1", Kind: "video", URI: "media/a.mp4", DurationFrames: 600, Width: 1920, Height: 1080, FPS: "30/1"},
			{ID: "m2", Kind: "audio", URI: "media/b.wav", DurationFrames: 900},
		},
		Tracks: []timelineTrack{
			{
				ID: "V1", Kind: "video", Index: 1, Role: "main",
				Clips: []timelineClip{
					{
						ID: "c1", MediaID: "m1", TrackID: "V1",
						StartFrame: 0, DurationFrames: 180,
						SourceInFrame: 60, SourceOutFrame: 240, Speed: 1,
						Transform:    &timelineTransform{Scale: 1.25, PositionX: 12, PositionY: -8, Rotation: 2.5, Fit: "cover"},
						Color:        &timelineColor{Brightness: 1.08, Contrast: 1.12, Saturation: 1.2, Gamma: 0.95, LUT: "film.cube"},
						TransitionIn: &timelineTransition{Type: "fade", DurationFrames: 12},
					},
					{
						ID: "c2", MediaID: "m1", TrackID: "V1",
						StartFrame: 180, DurationFrames: 120,
						SourceInFrame: 300, SourceOutFrame: 420, Speed: 0.5,
						LinkedGroup: "g1",
					},
				},
			},
			{
				ID: "A1", Kind: "audio", Index: 2, Role: "voiceover", Muted: false,
				Clips: []timelineClip{
					{ID: "a1", MediaID: "m2", TrackID: "A1", StartFrame: 0, DurationFrames: 300, SourceInFrame: 0, SourceOutFrame: 300, VolumeDB: -6, LinkedGroup: "g1"},
				},
			},
		},
	}
	return doc
}

func TestOTIORoundTripPreservesTimeline(t *testing.T) {
	source := interchangeFixture()
	payload := timelineToOTIO(source)
	decoded, err := otioToTimeline(payload)
	if err != nil {
		t.Fatalf("otioToTimeline: %v", err)
	}
	if !reflect.DeepEqual(source, decoded) {
		t.Fatalf("OTIO round-trip mismatch\nsource: %#v\ndecoded: %#v", source, decoded)
	}
}

func TestEDLRoundTripPreservesClips(t *testing.T) {
	source := interchangeFixture()
	content := timelineToEDL(source)
	decoded, err := edlToTimeline(content)
	if err != nil {
		t.Fatalf("edlToTimeline: %v", err)
	}
	if decoded.Schema != source.Schema || decoded.Revision != source.Revision {
		t.Fatalf("EDL header metadata lost: %#v", decoded)
	}
	if decoded.Timebase != source.Timebase || decoded.Canvas != source.Canvas {
		t.Fatalf("EDL timebase/canvas lost: %#v", decoded)
	}
	if len(decoded.Tracks) != len(source.Tracks) {
		t.Fatalf("track count = %d, want %d", len(decoded.Tracks), len(source.Tracks))
	}
	for i, track := range source.Tracks {
		got := decoded.Tracks[i]
		if got.ID != track.ID || got.Kind != track.Kind || len(got.Clips) != len(track.Clips) {
			t.Fatalf("track %d mismatch: %#v vs %#v", i, got, track)
		}
		for j, clip := range track.Clips {
			if !reflect.DeepEqual(clip, got.Clips[j]) {
				t.Fatalf("clip %s/%d mismatch\nwant: %#v\ngot:  %#v", track.ID, j, clip, got.Clips[j])
			}
		}
	}
}

func TestInterchangeRejectsStandardOnlyOTIO(t *testing.T) {
	_, err := otioToTimeline(map[string]any{"OTIO_SCHEMA": "Timeline.1"})
	if err == nil {
		t.Fatal("expected error for OTIO payload without himind metadata")
	}
}

func TestTimecodeRoundTrip(t *testing.T) {
	for _, frame := range []int{0, 1, 29, 30, 1799, 108000} {
		got, ok := timecodeToFrames(framesToTimecode(frame, 30), 30)
		if !ok || got != frame {
			t.Fatalf("timecode round-trip for %d = %d (ok=%v)", frame, got, ok)
		}
	}
}
