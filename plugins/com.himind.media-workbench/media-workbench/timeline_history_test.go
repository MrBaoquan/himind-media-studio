package main

import (
	"encoding/json"
	"testing"
)

func timelineJSON(t *testing.T, doc timelineDoc) string {
	t.Helper()
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestTimelineUndoRedo10Steps(t *testing.T) {
	history := timelineHistory{Current: timelineFixture()}
	original := timelineJSON(t, history.Current)
	for step := 0; step < 10; step++ {
		duration := 299 - step
		blockers := history.apply([]timelineCommand{{Op: "trim", TrackID: "V1", ClipID: "c1", DurationFrames: duration}})
		if len(blockers) > 0 {
			t.Fatalf("step %d blockers: %v", step, blockers)
		}
	}
	final := timelineJSON(t, history.Current)
	if history.Current.Revision != 10 {
		t.Fatalf("revision = %d", history.Current.Revision)
	}
	for step := 0; step < 10; step++ {
		if !history.undo() {
			t.Fatalf("undo %d failed", step)
		}
	}
	if got := timelineJSON(t, history.Current); got != original {
		t.Fatalf("undo did not restore original:\n%s\n%s", got, original)
	}
	for step := 0; step < 10; step++ {
		if !history.redo() {
			t.Fatalf("redo %d failed", step)
		}
	}
	if got := timelineJSON(t, history.Current); got != final {
		t.Fatalf("redo did not restore final:\n%s\n%s", got, final)
	}
}

func TestTimelineHistoryClearsRedoOnNewApply(t *testing.T) {
	history := timelineHistory{Current: timelineFixture()}
	_ = history.apply([]timelineCommand{{Op: "trim", TrackID: "V1", ClipID: "c1", DurationFrames: 299}})
	_ = history.undo()
	if len(history.Redo) != 1 {
		t.Fatalf("redo stack = %d", len(history.Redo))
	}
	_ = history.apply([]timelineCommand{{Op: "trim", TrackID: "V1", ClipID: "c1", DurationFrames: 298}})
	if len(history.Redo) != 0 {
		t.Fatalf("redo stack should clear, got %d", len(history.Redo))
	}
}

func TestTimelineDiff(t *testing.T) {
	base := timelineFixture()
	target := timelineFixture()
	target.Revision = 2
	target.Tracks[0].Clips[0].DurationFrames = 200
	value, rpcErr := timelineDiffCapability(requestInput{BaseTimeline: base, TargetTimeline: target})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	result := value.(map[string]any)
	summary := result["summary"].(map[string]any)
	if summary["changed"].(int) != 1 {
		t.Fatalf("summary=%+v", summary)
	}
}
