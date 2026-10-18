package main

import "testing"

func checkByID(checks []visualCheck, id string) visualCheck {
	for _, check := range checks {
		if check.ID == id {
			return check
		}
	}
	return visualCheck{}
}

func TestEvaluateTextAuditPassesValidScene(t *testing.T) {
	payload := textAuditPayload{
		Width: 1080, Height: 1920,
		Scenes: []textAuditScene{{
			Time: 3, Title: "地球在公转",
			Texts: []textAuditText{
				{Text: "地球在公转", Left: 80, Right: 400, Top: 180, Bottom: 250, FontSize: 60},
				{Text: "轨道接近圆形", Left: 80, Right: 500, Top: 270, Bottom: 300, FontSize: 24},
			},
		}},
	}
	checks, findings, _ := evaluateTextAudit(payload)
	if len(findings) != 0 {
		t.Fatalf("unexpected findings: %#v", findings)
	}
	if checkByID(checks, "text_bounds").Status != "passed" || checkByID(checks, "safe_area").Status != "passed" || checkByID(checks, "title_hit").Status != "passed" {
		t.Fatalf("unexpected checks: %#v", checks)
	}
}

func TestEvaluateTextAuditFailsOutOfBoundsAndOverlap(t *testing.T) {
	payload := textAuditPayload{
		Width: 1080, Height: 1920,
		Scenes: []textAuditScene{{
			Time: 3, Title: "标题",
			Texts: []textAuditText{
				{Text: "标题", Left: 1000, Right: 1200, Top: 180, Bottom: 260, FontSize: 60},
				{Text: "副标题", Left: 1020, Right: 1180, Top: 200, Bottom: 240, FontSize: 30},
			},
		}},
	}
	checks, findings, _ := evaluateTextAudit(payload)
	if checkByID(checks, "text_bounds").Status != "failed" {
		t.Fatal("expected text_bounds failed")
	}
	if checkByID(checks, "text_overlap").Status != "failed" {
		t.Fatal("expected text_overlap failed")
	}
	hasOverlap := false
	for _, finding := range findings {
		if finding["code"] == "TEXT_OVERLAP" {
			hasOverlap = true
		}
	}
	if !hasOverlap {
		t.Fatal("expected TEXT_OVERLAP finding")
	}
}
