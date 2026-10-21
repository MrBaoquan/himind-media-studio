package main

import (
	"testing"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func seedShotCreative(t *testing.T, root string) creativePackage {
	t.Helper()
	writeCreativeDemoPack(t, root)
	created, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.creative.create", Params: mustJSON(t, map[string]any{
		"workspace_root": root,
		"title":          "镜头能力测试",
		"brief":          map[string]any{"goal": "测试多 provider 分镜", "audience": "创作者", "platform": "douyin", "duration_seconds": 5},
	})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	item := created.(map[string]any)["creative"].(creativePackage)
	updates := []struct {
		section string
		value   map[string]any
	}{
		{"research", map[string]any{"mode": "required", "sources": []any{map[string]any{"id": "s1", "title": "Demo"}}, "claims": []any{map[string]any{"id": "c1", "text": "Demo", "evidence_refs": []any{"s1"}}}}},
		{"script", map[string]any{"logline": "Demo", "beats": []any{map[string]any{"id": "b1", "purpose": "hook", "duration_seconds": 5, "narration": "Demo"}}}},
		{"storyboard", map[string]any{"shots": []any{map[string]any{"id": "shot1", "shot": "title", "beat_id": "b1", "duration_seconds": 5, "visual": "Demo"}}}},
		{"production", map[string]any{"mode": "code", "style_pack": "demo-science", "duration_seconds": 5}},
	}
	for _, update := range updates {
		result, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.creative.update", Params: mustJSON(t, map[string]any{
			"workspace_root": root, "creative_id": item.ID, "section": update.section, "value": update.value,
		})})
		if rpcErr != nil {
			t.Fatal(rpcErr)
		}
		item = result.(map[string]any)["creative"].(creativePackage)
	}
	return item
}

func TestShotProvidersAssignAndTimelinePlan(t *testing.T) {
	root := t.TempDir()
	item := seedShotCreative(t, root)

	providersResult, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.providers", Params: mustJSON(t, map[string]any{})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	providers := providersResult.(map[string]any)["items"].([]shotProvider)
	if len(providers) < 5 {
		t.Fatalf("expected built-in and external providers, got %#v", providers)
	}

	defaultPlan, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.timeline.plan", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID,
	})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	defaultPayload := defaultPlan.(map[string]any)
	if defaultPayload["state"] != "ready" {
		t.Fatalf("default plan should be ready: %#v", defaultPayload)
	}
	warnings := defaultPayload["warnings"].([]map[string]any)
	if len(warnings) == 0 || warnings[0]["code"] != "SHOT_PROVIDER_DEFAULT" {
		t.Fatalf("expected default provider warning: %#v", warnings)
	}
	planShots := defaultPayload["shots"].([]map[string]any)
	if len(planShots) != 1 || planShots[0]["provider"] != "programmatic.browser-code" {
		t.Fatalf("default provider not resolved: %#v", planShots)
	}

	if _, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.assign", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID, "shot_id": "shot1", "provider": "unknown.provider",
	})}); rpcErr == nil {
		t.Fatal("unknown provider should be rejected")
	}

	assigned, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.assign", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID, "shot_id": "shot1", "provider": "ai.video", "technique": "cinematic", "prompt": "a cinematic shot", "fallback_provider": "programmatic.browser-code",
	})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	assignedItem := assigned.(map[string]any)["creative"].(creativePackage)
	if assignedItem.Status != "draft" || assignedItem.Revision <= item.Revision {
		t.Fatalf("shot.assign should create a draft revision: %+v", assignedItem)
	}

	aiPlan, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.timeline.plan", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID,
	})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	aiPayload := aiPlan.(map[string]any)
	if aiPayload["state"] != "ready" {
		t.Fatalf("ai.video is a built-in agent capability and should be ready with prompt: %#v", aiPayload)
	}
	aiShots := aiPayload["shots"].([]map[string]any)
	if len(aiShots) != 1 || aiShots[0]["provider"] != "ai.video" || aiShots[0]["capability"] != "media.video.generate" {
		t.Fatalf("ai.video should resolve to media.video.generate: %#v", aiShots)
	}

	if _, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.assign", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID, "shot_id": "shot1", "provider": "stock.video",
	})}); rpcErr != nil {
		t.Fatal(rpcErr)
	}
	blockedPlan, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.timeline.plan", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID,
	})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	blockedPayload := blockedPlan.(map[string]any)
	if blockedPayload["state"] != "blocked" {
		t.Fatalf("external provider without pack should block: %#v", blockedPayload)
	}
	blockers := blockedPayload["blockers"].([]map[string]any)
	foundUnavailable := false
	for _, blocker := range blockers {
		if blocker["code"] == "SHOT_PROVIDER_UNAVAILABLE" {
			foundUnavailable = true
		}
	}
	if !foundUnavailable {
		t.Fatalf("expected SHOT_PROVIDER_UNAVAILABLE: %#v", blockers)
	}

	if _, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.assign", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID, "shot_id": "shot1", "provider": "programmatic.browser-code", "technique": "hand-drawn",
	})}); rpcErr != nil {
		t.Fatal(rpcErr)
	}
	readyPlan, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.timeline.plan", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID,
	})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	if readyPlan.(map[string]any)["state"] != "ready" {
		t.Fatalf("programmatic provider should be ready: %#v", readyPlan)
	}
}
