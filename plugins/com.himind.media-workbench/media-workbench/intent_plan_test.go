package main

import (
	"strings"
	"testing"
)

func TestIntentPlanRoutesFastAndStory(t *testing.T) {
	hasData := true
	cases := []struct {
		name    string
		input   requestInput
		route   string
		approve bool
	}{
		{
			name:  "gdp ranking is fast",
			input: requestInput{Request: "请制作2025年全国省会GDP排名动画", Platform: "抖音", StyleHint: "信息图", HasDataSource: &hasData},
			route: "fast",
		},
		{
			name:  "anhui high speed rail is fast",
			input: requestInput{Request: "请制作安徽省高铁路线动画示意图", TargetDurationSeconds: 30},
			route: "fast",
		},
		{
			name:    "hefei artificial sun science story is story",
			input:   requestInput{Request: "做一期合肥人造太阳科普短片，手绘风格，有旁白和字幕", Platform: "视频号", WantsNarration: boolPtr(true)},
			route:   "story",
			approve: true,
		},
		{
			name:    "himind product promo is promo",
			input:   requestInput{Request: "请为 HiMind Agent 做一支 30 秒产品宣传片，官网 https://example.com，电影感，有旁白和字幕", Platform: "B站", TargetDurationSeconds: 30, StyleHint: "电影感", WantsNarration: boolPtr(true)},
			route:   "promo",
			approve: true,
		},
		{
			name:    "celestial motion science story is story",
			input:   requestInput{Request: "制作一期天体运动科普视频，带分镜和旁白", TargetDurationSeconds: 60, StyleHint: "电影感"},
			route:   "story",
			approve: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, rpcErr := intentPlan(tc.input)
			if rpcErr != nil {
				t.Fatalf("unexpected error: %v", rpcErr)
			}
			plan, ok := result.(intentPlanResult)
			if !ok {
				t.Fatalf("unexpected result type %T", result)
			}
			if plan.Route != tc.route {
				t.Fatalf("route = %s, want %s (reason=%s signals=%v)", plan.Route, tc.route, plan.Reason, plan.Signals)
			}
			if plan.RequiresCreativeApproval != tc.approve {
				t.Fatalf("requires approval = %v, want %v", plan.RequiresCreativeApproval, tc.approve)
			}
			if len(plan.RecommendedCapabilities) == 0 {
				t.Fatal("recommended capabilities must not be empty")
			}
			switch plan.Route {
			case "story":
				assertHasStep(t, plan, "media.workbench.creative.approve")
				assertHasStep(t, plan, "media.workbench.audio.speech")
				assertHasStep(t, plan, "media.workbench.render.status")
			case "promo":
				assertHasStep(t, plan, "media.workbench.creative.approve")
				assertHasStep(t, plan, "media.workbench.recipe.compile")
				assertHasStep(t, plan, "media.workbench.render.status")
			default:
				assertHasStep(t, plan, "media.workbench.plan.from-recipe")
				assertHasStep(t, plan, "media.workbench.render.status")
				for _, step := range plan.RecommendedCapabilities {
					if step.Capability == "media.workbench.creative.approve" {
						t.Fatal("fast path must not require creative approval")
					}
				}
			}
		})
	}
}

func TestIntentPlanReportsMissingInputs(t *testing.T) {
	result, rpcErr := intentPlan(requestInput{Request: "请制作2025年全国省会GDP排名动画"})
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}
	plan := result.(intentPlanResult)
	ids := map[string]bool{}
	for _, item := range plan.Missing {
		ids[item.ID] = true
	}
	for _, want := range []string{"duration", "aspect_ratio", "style", "data_source"} {
		if !ids[want] {
			t.Fatalf("missing %q in %v", want, ids)
		}
	}
	if len(plan.Questions) != len(plan.Missing) {
		t.Fatalf("questions %d != missing %d", len(plan.Questions), len(plan.Missing))
	}
}

func TestIntentPlanExtractsDuration(t *testing.T) {
	result, rpcErr := intentPlan(requestInput{Request: "做一条 45 秒的科普短片，手绘风格，有旁白和字幕", Platform: "视频号"})
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}
	plan := result.(intentPlanResult)
	if got := plan.Signals["duration"].(float64); got != 45 {
		t.Fatalf("duration = %v, want 45", got)
	}
	for _, item := range plan.Missing {
		if item.ID == "duration" {
			t.Fatal("duration should be inferred from the request")
		}
	}
}

func TestIntentPlanRequiresRequest(t *testing.T) {
	_, rpcErr := intentPlan(requestInput{})
	if rpcErr == nil || !strings.Contains(rpcErr.Message, "request is required") {
		t.Fatalf("expected request error, got %#v", rpcErr)
	}
}

func assertHasStep(t *testing.T, plan intentPlanResult, capability string) {
	t.Helper()
	for _, step := range plan.RecommendedCapabilities {
		if step.Capability == capability {
			return
		}
	}
	t.Fatalf("capability %s missing from %d-step plan", capability, len(plan.RecommendedCapabilities))
}

func boolPtr(value bool) *bool { return &value }

func TestIntentPlanPromoReportsProductReference(t *testing.T) {
	result, rpcErr := intentPlan(requestInput{Request: "给我做一支产品宣传片，有旁白和字幕"})
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}
	plan := result.(intentPlanResult)
	if plan.Route != "promo" {
		t.Fatalf("route = %s, want promo", plan.Route)
	}
	ids := map[string]bool{}
	for _, item := range plan.Missing {
		ids[item.ID] = true
	}
	for _, want := range []string{"product_reference", "style", "duration", "aspect_ratio"} {
		if !ids[want] {
			t.Fatalf("missing %q in %v", want, ids)
		}
	}
}
