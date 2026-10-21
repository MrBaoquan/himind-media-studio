package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func writeCreativeDemoPack(t *testing.T, root string) {
	t.Helper()
	packRoot := filepath.Join(root, "style-packs", "demo-science")
	template := filepath.Join(packRoot, "template")
	if err := os.MkdirAll(filepath.Join(template, "reel"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{
		"schema": "himind.asset-pack.v1", "id": "demo-science", "version": "0.1.0", "name": "Demo Science", "license": "MIT",
		"resources": []any{map[string]any{"id": "template", "type": "template", "path": "template", "runtime": "browser-code", "entry": "render.mjs"}},
	}
	writeCreativeTestJSON(t, filepath.Join(packRoot, "asset-pack.json"), manifest)
	stylePack := `{"schema":"himind.video.style-pack.v1","id":"demo-science","version":"0.1.0","name":"Demo Science","renderer":"browser-code","entry":"render.mjs","video_out":"out/video.mp4","duration_seconds":5,"preview":{"sheet_out":"out/check/sheet.jpg","strip_out":"out/check/strip.jpg","stills_out":"out/stills"},"network":{"mode":"none","allow":[]}}`
	writeCreativeTestFile(t, filepath.Join(template, "style-pack.json"), stylePack)
	writeCreativeTestFile(t, filepath.Join(template, "package.json"), `{"name":"demo-science","private":true,"type":"module"}`)
	writeCreativeTestFile(t, filepath.Join(template, "STORYBOARD.md"), "# Demo Science\n")
	writeCreativeTestFile(t, filepath.Join(template, "render.mjs"), "// --sheet --strip --stills --frames --encode\nconst frameFormat='png';\n")
	writeCreativeTestFile(t, filepath.Join(template, "reel", "index.html"), "<canvas id=\"stage\"></canvas>")
}

func writeCreativeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeCreativeTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	writeCreativeTestFile(t, path, string(raw))
}

func TestCreativePackageLifecycleMaterializesCodeProject(t *testing.T) {
	root := t.TempDir()
	writeCreativeDemoPack(t, root)
	created, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.creative.create", Params: mustJSON(t, map[string]any{
		"workspace_root": root,
		"title":          "天空为什么是蓝色的？",
		"brief":          map[string]any{"goal": "解释瑞利散射", "audience": "泛知识用户", "platform": "douyin", "duration_seconds": 5},
	})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	item := created.(map[string]any)["creative"].(creativePackage)
	if item.ID == "" || item.Status != "draft" {
		t.Fatalf("unexpected creative: %+v", item)
	}
	updates := []struct {
		section string
		value   map[string]any
	}{
		{"research", map[string]any{"mode": "required", "sources": []any{map[string]any{"id": "s1", "title": "NOAA"}}, "claims": []any{map[string]any{"id": "c1", "text": "蓝光更容易被散射", "evidence_refs": []any{"s1"}}}}},
		{"script", map[string]any{"logline": "阳光被空气散射", "beats": []any{map[string]any{"id": "b1", "purpose": "hook", "duration_seconds": 5, "narration": "天空为什么是蓝色的？"}}}},
		{"storyboard", map[string]any{"shots": []any{map[string]any{"id": "shot1", "shot": "title", "beat_id": "b1", "duration_seconds": 5}}}},
		{"production", map[string]any{"mode": "code", "style_pack": "demo-science", "duration_seconds": 5, "data": map[string]any{"title": "天空为什么是蓝色的？"}}},
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
	validated, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.creative.validate", Params: mustJSON(t, map[string]any{"workspace_root": root, "creative_id": item.ID})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	if validated.(map[string]any)["valid"] != true {
		t.Fatalf("creative should validate: %#v", validated)
	}
	approved, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.creative.approve", Params: mustJSON(t, map[string]any{"workspace_root": root, "creative_id": item.ID})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	if approved.(map[string]any)["state"] != "approved" {
		t.Fatalf("creative should approve: %#v", approved)
	}
	materialized, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.creative.materialize", Params: mustJSON(t, map[string]any{"workspace_root": root, "creative_id": item.ID, "name": "science-demo"})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	payload := materialized.(map[string]any)
	if payload["state"] != "materialized" {
		t.Fatalf("creative should materialize: %#v", payload)
	}
	projectItem := payload["project"].(project)
	if projectItem.SourceCreativeID != item.ID || projectItem.SourceCreativeRevision != item.Revision {
		t.Fatalf("project is missing creative provenance: %+v", projectItem)
	}
	if !fileExists(filepath.Join(root, projectItem.SourceDir, "input", "data.json")) {
		t.Fatal("materialized code project did not receive input/data.json")
	}
	loaded, loadErr := loadCreative(root, item.ID)
	if loadErr != nil || loaded.Status != "materialized" {
		t.Fatalf("creative link not persisted: %+v %v", loaded, loadErr)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
