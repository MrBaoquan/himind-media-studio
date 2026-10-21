package main

import (
	"strings"
	"testing"
)

func TestComplementSilenceRanges(t *testing.T) {
	spans := complementRanges([]silenceRange{{Start: 1, End: 2, Duration: 1}, {Start: 4, End: 5, Duration: 1}}, 6, 0.1, 0.35)
	if len(spans) != 3 {
		t.Fatalf("spans=%+v", spans)
	}
	if spans[0].Start != 0 || spans[0].End != 0.9 || spans[1].Start != 2.1 || spans[1].End != 3.9 || spans[2].Start != 5.1 {
		t.Fatalf("unexpected spans=%+v", spans)
	}
}

func TestFormatTimecode(t *testing.T) {
	if got := formatTimecode(65*30+12, 30); got != "00:01:05:12" {
		t.Fatalf("timecode=%s", got)
	}
}

func TestTimelineCommentsRoundTrip(t *testing.T) {
	root := t.TempDir()
	items := []timelineComment{{ID: "c1", ProjectID: "p1", Frame: 30, Timecode: "00:00:01:00", Text: "check", Status: "open", CreatedAt: "now"}}
	if err := saveTimelineComments(root, "p1", items); err != nil {
		t.Fatal(err)
	}
	loaded := loadTimelineComments(root, "p1")
	if len(loaded) != 1 || loaded[0].Text != "check" {
		t.Fatalf("loaded=%+v", loaded)
	}
}

func TestColorKeyframeExpr(t *testing.T) {
	keyframes := []timelineColorKeyframe{{Frame: 0, Brightness: -0.04}, {Frame: 30, Brightness: 0.08}, {Frame: 60, Brightness: -0.04}}
	expr := colorKeyframeExpr(keyframes, 30, 1, "brightness")
	if expr == "" || expr == "0" {
		t.Fatalf("empty expr")
	}
	if !strings.Contains(expr, "if(lt(t") {
		t.Fatalf("expr=%s", expr)
	}
}
