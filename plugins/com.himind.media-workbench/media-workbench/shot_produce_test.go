package main

import (
	"os"
	"strings"
	"testing"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func TestShotProduceMockImageAndStatus(t *testing.T) {
	root := t.TempDir()
	item := seedShotCreative(t, root)

	if _, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.assign", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID, "shot_id": "shot1", "provider": "ai.image", "technique": "storybook", "prompt": "a blue sky over a small town",
	})}); rpcErr != nil {
		t.Fatal(rpcErr)
	}

	produced, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.produce", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID, "shot_ids": []string{"shot1"}, "mode": "mock",
	})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	payload := produced.(map[string]any)
	if payload["state"] != "completed" {
		t.Fatalf("mock image produce should complete: %#v", payload)
	}
	assets := payload["assets"].([]producedShotAsset)
	if len(assets) != 1 {
		t.Fatalf("expected one produced asset: %#v", assets)
	}
	asset := assets[0]
	if !asset.Mock || asset.Path == "" || !strings.HasPrefix(asset.PreviewDataURL, "data:image/svg+xml;base64,") {
		t.Fatalf("unexpected mock asset: %#v", asset)
	}
	absPath, err := workspaceFile(root, asset.Path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "a blue sky over a sm") || !strings.Contains(string(raw), "all town") || !strings.Contains(string(raw), "MOCK AI IMAGE") {
		t.Fatalf("mock SVG is missing prompt or label: %s", raw)
	}
	updated := payload["creative"].(creativePackage)
	if updated.Revision <= item.Revision || updated.Status != "draft" {
		t.Fatalf("shot.produce should create a draft revision: %+v", updated)
	}
	shots := asMapSlice(asMap(updated.Storyboard)["shots"])
	if len(shots) != 1 || asString(shots[0]["status"]) != "generated" || len(asStringSlice(shots[0]["asset_refs"])) == 0 {
		t.Fatalf("shot status/asset_refs not updated: %#v", shots)
	}

	statusResult, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.status", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID,
	})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	items := statusResult.(map[string]any)["items"].([]map[string]any)
	if len(items) != 1 {
		t.Fatalf("expected one shot status: %#v", items)
	}
	assetsStatus := items[0]["assets"].([]map[string]any)
	if len(assetsStatus) != 1 || assetsStatus[0]["exists"] != true {
		t.Fatalf("asset should exist: %#v", assetsStatus)
	}
	if !strings.HasPrefix(asString(assetsStatus[0]["preview_data_url"]), "data:image/svg+xml;base64,") {
		t.Fatalf("asset preview missing: %#v", assetsStatus[0])
	}

	if _, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.assign", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID, "shot_id": "shot1", "provider": "ai.video", "prompt": "a cinematic shot",
	})}); rpcErr != nil {
		t.Fatal(rpcErr)
	}
	blocked, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.produce", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID, "shot_ids": []string{"shot1"}, "mode": "mock",
	})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	blockedPayload := blocked.(map[string]any)
	if blockedPayload["state"] != "blocked" {
		t.Fatalf("ai.video should be blocked until async production is wired: %#v", blockedPayload)
	}
	foundAsync := false
	for _, blocker := range blockedPayload["blockers"].([]map[string]any) {
		if blocker["code"] == "SHOT_PROVIDER_ASYNC_NOT_IMPLEMENTED" {
			foundAsync = true
		}
	}
	if !foundAsync {
		t.Fatalf("expected SHOT_PROVIDER_ASYNC_NOT_IMPLEMENTED: %#v", blockedPayload["blockers"])
	}

	if _, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.assign", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID, "shot_id": "shot1", "provider": "programmatic.browser-code",
	})}); rpcErr != nil {
		t.Fatal(rpcErr)
	}
	programmatic, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.produce", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID, "shot_ids": []string{"shot1"}, "mode": "mock",
	})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	programmaticPayload := programmatic.(map[string]any)
	if programmaticPayload["state"] != "blocked" {
		t.Fatalf("programmatic shot should not be produced by mock image producer: %#v", programmaticPayload)
	}
	foundProgrammaticWarning := false
	for _, warning := range programmaticPayload["warnings"].([]map[string]any) {
		if warning["code"] == "SHOT_PROGRAMMATIC_PRODUCE_NOT_IMPLEMENTED" {
			foundProgrammaticWarning = true
		}
	}
	if !foundProgrammaticWarning {
		t.Fatalf("expected programmatic warning: %#v", programmaticPayload["warnings"])
	}
}
