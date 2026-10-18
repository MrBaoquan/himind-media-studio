package main

import (
	"math"
	"testing"
)

func TestVisualDetectorParsers(t *testing.T) {
	if got := parseFrameRate("30000/1001"); math.Abs(got-29.97002997) > 0.001 {
		t.Fatalf("parseFrameRate = %v", got)
	}
	if got := countDetectorEvents("black_start:1.2 black_end:2.0 black_start:3.0", "black_start"); got != 2 {
		t.Fatalf("black events = %d", got)
	}
	if got := countDetectorEvents("freeze_start: 5.0 freeze_end: 7.0", "freeze_start"); got != 1 {
		t.Fatalf("freeze events = %d", got)
	}
	output := "[Parsed_ebur128_0] I: -22.2 LUFS\n[Parsed_ebur128_0] Peak: -13.6 dBFS\n"
	if got := lastFloatMatch(output, `I:\s*(-?\d+(?:\.\d+)?)\s*LUFS`); math.Abs(got+22.2) > 0.001 {
		t.Fatalf("integrated loudness = %v", got)
	}
	if got := lastFloatMatch(output, `Peak:\s*(-?\d+(?:\.\d+)?)\s*dBFS`); math.Abs(got+13.6) > 0.001 {
		t.Fatalf("true peak = %v", got)
	}
	times := parseTimeList("1.5, 5, 9")
	if len(times) != 3 || times[0] != 1.5 || times[2] != 9 {
		t.Fatalf("parseTimeList = %#v", times)
	}
	relative := relativeSampleTimes(30)
	if len(relative) != 3 || math.Abs(relative[0]-4.5) > 0.001 || math.Abs(relative[1]-15) > 0.001 || math.Abs(relative[2]-25.5) > 0.001 {
		t.Fatalf("relativeSampleTimes = %#v", relative)
	}
}
func TestApplyVisualQuality(t *testing.T) {
	passed := &job{}
	applyVisualQuality(passed, visualReport{OK: true})
	if passed.QualityStatus != "passed" || passed.QualityReport == nil {
		t.Fatalf("passed quality = %+v", passed)
	}
	warned := &job{}
	applyVisualQuality(warned, visualReport{OK: true, Warnings: []string{"freeze"}})
	if warned.QualityStatus != "warning" {
		t.Fatalf("warning quality = %+v", warned)
	}
	failed := &job{}
	applyVisualQuality(failed, visualReport{OK: false})
	if failed.QualityStatus != "failed" {
		t.Fatalf("failed quality = %+v", failed)
	}
}
