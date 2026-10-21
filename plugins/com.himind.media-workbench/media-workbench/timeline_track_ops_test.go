package main

import "testing"

func TestTimelineAddTrackOrdersByKind(t *testing.T) {
	doc := timelineFixture()
	next, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "add_track", TrackKind: "video", TrackRole: "overlay"}})
	if len(blockers) != 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	if len(next.Tracks) != 2 || next.Tracks[1].ID != "V2" || next.Tracks[1].Index != 2 || next.Tracks[1].Role != "overlay" {
		t.Fatalf("unexpected video track: %+v", next.Tracks)
	}
	next, blockers = applyTimelineCommands(next, []timelineCommand{{Op: "add_track", TrackKind: "audio"}})
	if len(blockers) != 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	if next.Tracks[len(next.Tracks)-1].ID != "A1" || next.Tracks[len(next.Tracks)-1].Kind != "audio" {
		t.Fatalf("unexpected audio track: %+v", next.Tracks)
	}
}

func TestTimelineSetTrackStateAndRemoveEmpty(t *testing.T) {
	muted, locked := true, true
	doc := timelineFixture()
	next, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "add_track", TrackKind: "video"}})
	if len(blockers) != 0 {
		t.Fatalf("add blockers: %v", blockers)
	}
	next, blockers = applyTimelineCommands(next, []timelineCommand{{Op: "set_track_state", TrackID: "V2", Muted: &muted, Locked: &locked}})
	if len(blockers) != 0 {
		t.Fatalf("state blockers: %v", blockers)
	}
	if !next.Tracks[1].Muted || !next.Tracks[1].Locked {
		t.Fatalf("track state not applied: %+v", next.Tracks[1])
	}
	// Locked empty track must not be removable until unlocked.
	if _, blockers = applyTimelineCommands(next, []timelineCommand{{Op: "remove_track", TrackID: "V2"}}); len(blockers) == 0 {
		t.Fatal("expected TRACK_LOCKED")
	}
	unlocked := false
	next, blockers = applyTimelineCommands(next, []timelineCommand{{Op: "set_track_state", TrackID: "V2", Locked: &unlocked}})
	if len(blockers) != 0 {
		t.Fatalf("unlock blockers: %v", blockers)
	}
	next, blockers = applyTimelineCommands(next, []timelineCommand{{Op: "remove_track", TrackID: "V2"}})
	if len(blockers) != 0 {
		t.Fatalf("remove blockers: %v", blockers)
	}
	if len(next.Tracks) != 1 {
		t.Fatalf("expected 1 track, got %+v", next.Tracks)
	}
}

func TestTimelineRemoveTrackRejectsNonEmptyAndLastTrack(t *testing.T) {
	doc := timelineFixture()
	if _, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "remove_track", TrackID: "V1"}}); len(blockers) == 0 {
		t.Fatal("expected TRACK_NOT_EMPTY or LAST_TRACK_CANNOT_BE_REMOVED")
	}
}

func TestPrimaryVideoTrackSkipsMuted(t *testing.T) {
	doc := timelineFixture()
	next, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "add_track", TrackKind: "video"}})
	if len(blockers) != 0 {
		t.Fatalf("add blockers: %v", blockers)
	}
	muted := true
	next, blockers = applyTimelineCommands(next, []timelineCommand{{Op: "set_track_state", TrackID: "V1", Muted: &muted}})
	if len(blockers) != 0 {
		t.Fatalf("mute blockers: %v", blockers)
	}
	next.Tracks[1].Clips = []timelineClip{{ID: "v2c1", TrackID: "V2", StartFrame: 0, DurationFrames: 60}}
	primary := primaryVideoTrack(next)
	if primary == nil || primary.ID != "V2" {
		t.Fatalf("primary = %+v, want V2", primary)
	}
	unmuted := false
	next, _ = applyTimelineCommands(next, []timelineCommand{{Op: "set_track_state", TrackID: "V1", Muted: &unmuted}})
	if primary = primaryVideoTrack(next); primary == nil || primary.ID != "V1" {
		t.Fatalf("primary after unmute = %+v, want V1", primary)
	}
}
func TestTimelineSetTrackHeight(t *testing.T) {
	height := 84
	doc := timelineFixture()
	next, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "set_track_height", TrackID: "V1", Height: &height}})
	if len(blockers) != 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	if next.Tracks[0].Height != 84 {
		t.Fatalf("height = %d, want 84", next.Tracks[0].Height)
	}
	height = 200
	if _, blockers = applyTimelineCommands(next, []timelineCommand{{Op: "set_track_height", TrackID: "V1", Height: &height}}); len(blockers) == 0 {
		t.Fatal("expected TRACK_HEIGHT_OUT_OF_RANGE")
	}
	height = 0
	next, blockers = applyTimelineCommands(next, []timelineCommand{{Op: "set_track_height", TrackID: "V1", Height: &height}})
	if len(blockers) != 0 || next.Tracks[0].Height != 0 {
		t.Fatalf("clear height failed: %+v %v", next.Tracks[0], blockers)
	}
}
func TestAudioSoloOverridesMute(t *testing.T) {
	doc := timelineFixture()
	doc.Tracks[0].Kind = "video"
	doc.Tracks = append(doc.Tracks,
		timelineTrack{ID: "A1", Kind: "audio", Index: 1, Muted: true, Clips: []timelineClip{}},
		timelineTrack{ID: "A2", Kind: "audio", Index: 2, Clips: []timelineClip{}},
	)
	a1, a2 := doc.Tracks[1], doc.Tracks[2]
	if audioTrackAudible(doc, a1) || !audioTrackAudible(doc, a2) {
		t.Fatalf("base audible mismatch: a1=%v a2=%v", audioTrackAudible(doc, a1), audioTrackAudible(doc, a2))
	}
	solo := true
	doc, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "set_track_state", TrackID: "A2", Solo: &solo}})
	if len(blockers) != 0 {
		t.Fatalf("solo blockers: %v", blockers)
	}
	a1, a2 = doc.Tracks[1], doc.Tracks[2]
	if audioTrackAudible(doc, a1) || !audioTrackAudible(doc, a2) {
		t.Fatalf("solo audible mismatch: a1=%v a2=%v", audioTrackAudible(doc, a1), audioTrackAudible(doc, a2))
	}
	doc, blockers = applyTimelineCommands(doc, []timelineCommand{{Op: "set_track_state", TrackID: "A1", Solo: &solo}})
	if len(blockers) != 0 {
		t.Fatalf("second solo blockers: %v", blockers)
	}
	if !audioTrackAudible(doc, doc.Tracks[1]) || !audioTrackAudible(doc, doc.Tracks[2]) {
		t.Fatal("both solo tracks should be audible")
	}
}
func TestTimelineReorderTracksWithinKind(t *testing.T) {
	doc := timelineFixture()
	next, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "add_track", TrackKind: "video"}})
	if len(blockers) != 0 {
		t.Fatalf("add blockers: %v", blockers)
	}
	next, blockers = applyTimelineCommands(next, []timelineCommand{{Op: "reorder_track", TrackID: "V1", TargetIndex: 1}})
	if len(blockers) != 0 {
		t.Fatalf("reorder blockers: %v", blockers)
	}
	if next.Tracks[0].ID != "V2" || next.Tracks[1].ID != "V1" {
		t.Fatalf("order = %s,%s", next.Tracks[0].ID, next.Tracks[1].ID)
	}
	if next.Tracks[0].Index != 1 || next.Tracks[1].Index != 2 {
		t.Fatalf("indexes = %d,%d", next.Tracks[0].Index, next.Tracks[1].Index)
	}
	if _, blockers = applyTimelineCommands(next, []timelineCommand{{Op: "reorder_track", TrackID: "V1", TargetIndex: 2}}); len(blockers) == 0 {
		t.Fatal("expected TRACK_TARGET_OUT_OF_RANGE")
	}
}

func TestTimelineTrackNameAndColor(t *testing.T) {
	doc := timelineFixture()
	next, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "set_track_state", TrackID: "V1", TrackName: "主画面", TrackColor: "#5aa9ff"}})
	if len(blockers) != 0 {
		t.Fatalf("state blockers: %v", blockers)
	}
	if next.Tracks[0].Name != "主画面" || next.Tracks[0].Color != "#5aa9ff" {
		t.Fatalf("track identity = %+v", next.Tracks[0])
	}
	if _, blockers = applyTimelineCommands(next, []timelineCommand{{Op: "set_track_state", TrackID: "V1", TrackColor: "red"}}); len(blockers) == 0 {
		t.Fatal("expected TRACK_COLOR_INVALID")
	}
}
func TestReorderChangesPrimaryVideoTrack(t *testing.T) {
	doc := timelineFixture()
	next, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "add_track", TrackKind: "video"}})
	if len(blockers) != 0 {
		t.Fatalf("add blockers: %v", blockers)
	}
	next.Tracks[1].Clips = []timelineClip{{ID: "v2c1", TrackID: "V2", StartFrame: 0, DurationFrames: 60}}
	if primary := primaryVideoTrack(next); primary == nil || primary.ID != "V1" {
		t.Fatalf("initial primary = %+v, want V1", primary)
	}
	next, blockers = applyTimelineCommands(next, []timelineCommand{{Op: "reorder_track", TrackID: "V2", TargetIndex: 0}})
	if len(blockers) != 0 {
		t.Fatalf("reorder blockers: %v", blockers)
	}
	if primary := primaryVideoTrack(next); primary == nil || primary.ID != "V2" {
		t.Fatalf("primary after reorder = %+v, want V2", primary)
	}
}
func TestTimelineTrackVolume(t *testing.T) {
	volume := -12.5
	doc := timelineFixture()
	next, blockers := applyTimelineCommands(doc, []timelineCommand{{Op: "set_track_state", TrackID: "V1", TrackVolumeDB: &volume}})
	if len(blockers) != 0 {
		t.Fatalf("blockers: %v", blockers)
	}
	if next.Tracks[0].VolumeDB != -12.5 {
		t.Fatalf("volume = %v", next.Tracks[0].VolumeDB)
	}
	volume = 20
	if _, blockers = applyTimelineCommands(next, []timelineCommand{{Op: "set_track_state", TrackID: "V1", TrackVolumeDB: &volume}}); len(blockers) == 0 {
		t.Fatal("expected TRACK_VOLUME_OUT_OF_RANGE")
	}
}
func TestAudioClipGainIncludesTrackVolume(t *testing.T) {
	gain := audioClipGain(timelineTrack{VolumeDB: -6}, timelineClip{VolumeDB: -6})
	want := 0.251188643150958
	if gain < want-0.0001 || gain > want+0.0001 {
		t.Fatalf("gain = %v, want %v", gain, want)
	}
}
