package main

import "testing"

func TestTemplateAllocation(t *testing.T) {
	template := editTemplateByID("hook-proof-payoff")
	frames := allocateTemplateFrames(template, 300, 30)
	total := 0
	for _, value := range frames {
		total += value
	}
	if total != 300 {
		t.Fatalf("total=%d", total)
	}
	if frames[0] != 60 || frames[2] != 90 {
		t.Fatalf("frames=%v", frames)
	}
}

func TestTimelineTemplateApply(t *testing.T) {
	in := requestInput{TemplateID: "six-beat-story", MediaIDs: []string{"a", "b", "c", "d", "e", "f"}, DurationSeconds: 12}
	in.Timeline.Timebase = timelineTimebase{Num: 30, Den: 1}
	value, rpcErr := timelineTemplateApplyCapability(in)
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	payload := value.(map[string]any)
	if payload["state"] != "ready" {
		t.Fatalf("payload=%+v", payload)
	}
	doc := payload["timeline"].(timelineDoc)
	if len(doc.Tracks[0].Clips) != 6 || doc.DurationFrames != 360 {
		t.Fatalf("doc=%+v", doc)
	}
	if len(doc.Markers) != 6 {
		t.Fatalf("markers=%d", len(doc.Markers))
	}
}

func TestTranscriptSearchAndRoughCut(t *testing.T) {
	doc := timelineDoc{Schema: "himind.timeline.v1", Timebase: timelineTimebase{Num: 30, Den: 1}, DurationFrames: 900, Canvas: timelineCanvas{Width: 1080, Height: 1920}, Tracks: []timelineTrack{{ID: "V1", Kind: "video", Index: 1, Clips: []timelineClip{{ID: "v1", MediaID: "m1", TrackID: "V1", StartFrame: 0, DurationFrames: 900, SourceInFrame: 0, SourceOutFrame: 900, Speed: 1}}}, {ID: "S1", Kind: "subtitle", Index: 1, Clips: []timelineClip{{ID: "s1", TrackID: "S1", StartFrame: 30, DurationFrames: 60, Text: "合肥人造太阳"}, {ID: "s2", TrackID: "S1", StartFrame: 120, DurationFrames: 60, Text: "磁场约束等离子体"}, {ID: "s3", TrackID: "S1", StartFrame: 240, DurationFrames: 60, Text: "结论与能源未来"}}}}}
	search := timelineTranscriptSearchCapability(requestInput{Timeline: doc, Query: "等离子体"})
	if search.(map[string]any)["total"].(int) != 1 {
		t.Fatalf("search=%+v", search)
	}
	value, rpcErr := timelineRoughCutCapability(requestInput{Timeline: doc, Query: "等离子体 未来"})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	payload := value.(map[string]any)
	if payload["state"] != "ready" || payload["selected_cues"].(int) != 2 {
		t.Fatalf("roughcut=%+v", payload)
	}
	rough := payload["timeline"].(timelineDoc)
	if len(rough.Tracks[0].Clips) != 2 || len(rough.Tracks[1].Clips) != 2 {
		t.Fatalf("rough=%+v", rough)
	}
	if blockers := validateTimeline(rough); len(blockers) > 0 {
		t.Fatalf("blockers=%v", blockers)
	}
}

func TestTimelinePIPAndReframe(t *testing.T) {
	doc := timelineEditFixture()
	value, rpcErr := timelinePIPCapability(requestInput{Timeline: doc, ClipID: "a"})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	pipDoc := value.(map[string]any)["timeline"].(timelineDoc)
	if len(pipDoc.Tracks) != 2 || pipDoc.Tracks[1].ID != "V2" || len(pipDoc.Tracks[1].Clips) != 1 {
		t.Fatalf("pip=%+v", pipDoc.Tracks)
	}
	reframe, rpcErr := timelineReframeCapability(requestInput{Timeline: pipDoc, Width: 1080, Height: 1080, FitMode: "cover"})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	reframed := reframe.(map[string]any)["timeline"].(timelineDoc)
	if reframed.Canvas.Width != 1080 || reframed.Canvas.Height != 1080 || reframed.Tracks[0].Clips[0].Transform == nil {
		t.Fatalf("reframe=%+v", reframed)
	}
}

func TestTimelineSpeedCurveSegments(t *testing.T) {
	doc := timelineEditFixture()
	value, rpcErr := timelineSpeedCurveCapability(requestInput{Timeline: doc, ClipID: "a", SpeedKeyframes: []timelineSpeedKeyframe{{Frame: 0, Speed: 1}, {Frame: 30, Speed: 4}, {Frame: 60, Speed: 4}}})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	result := value.(map[string]any)
	if result["state"] != "ready" {
		t.Fatalf("result=%+v", result)
	}
	out := result["timeline"].(timelineDoc)
	clips := out.Tracks[0].Clips
	if len(clips) != 3 {
		t.Fatalf("clips=%+v", clips)
	}
	duration := 0
	source := 0
	for _, clip := range clips {
		duration += clip.DurationFrames
		source += clip.SourceOutFrame - clip.SourceInFrame
	}
	if duration != 120 {
		t.Fatalf("duration=%d", duration)
	}
	if source != 240 {
		t.Fatalf("source=%d", source)
	}
}

func TestTimelineCommandPlan(t *testing.T) {
	doc := timelineEditFixture()
	result := timelineCommandPlanCapability(requestInput{Timeline: doc, CommandText: "把第二段剪到 0.5 秒并加 0.1 秒淡入"}).(map[string]any)
	if result["state"] != "ready" {
		t.Fatalf("result=%+v", result)
	}
	commands := result["commands"].([]timelineCommand)
	if len(commands) != 2 || commands[0].Op != "update_clip" || commands[1].Op != "ripple_trim" {
		t.Fatalf("commands=%+v", commands)
	}
	if result["target_clip"] != "b" {
		t.Fatalf("target=%v", result["target_clip"])
	}
}
