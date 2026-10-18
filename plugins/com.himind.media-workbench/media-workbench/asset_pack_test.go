package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeAssetPackFixture(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "asset-packs", "demo")
	template := filepath.Join(dir, "template")
	if err := os.MkdirAll(filepath.Join(template, "reel"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	write(filepath.Join(template, "style-pack.json"), `{"schema":"himind.video.style-pack.v1","id":"demo","version":"1.0.0","name":"Demo","renderer":"browser-code","entry":"render.mjs","video_out":"out/video.mp4","duration_seconds":10,"preview":{"sheet_out":"out/check/sheet.jpg","strip_out":"out/check/strip.jpg","stills_out":"out/stills"}}`)
	write(filepath.Join(template, "render.mjs"), "// --sheet --strip --stills --frames --encode\n")
	write(filepath.Join(template, "package.json"), `{"name":"demo","private":true}`)
	write(filepath.Join(template, "STORYBOARD.md"), "# Demo\n")
	write(filepath.Join(template, "reel", "reel.js"), "export const version = 1;\n")
	write(filepath.Join(dir, "preview", "sheet.jpg"), "fake-jpeg")
	write(filepath.Join(dir, "asset-pack.json"), `{
  "schema":"himind.asset-pack.v1","id":"demo","version":"1.0.0","name":"Demo Asset Pack","license":"MIT",
  "resources":[
    {"id":"template","type":"template","path":"template","role":"render-template","runtime":"browser-code","entry":"render.mjs"},
    {"id":"preview","type":"image","path":"preview/sheet.jpg","role":"preview","mime":"image/jpeg"}
  ],
  "entrypoints":[{"id":"render","resource":"template","runtime":"browser-code","entry":"render.mjs"}],
  "tags":["video","style"]
}`)
}

func TestAssetPackListReadAndScaffold(t *testing.T) {
	root := t.TempDir()
	writeAssetPackFixture(t, root)

	listResult, rpcErr := assetList(requestInput{WorkspaceRoot: root})
	if rpcErr != nil {
		t.Fatalf("asset.list: %s", rpcErr.Message)
	}
	items := listResult.(map[string]any)["items"].([]assetPack)
	found := false
	for _, pack := range items {
		if pack.ID == "demo" && pack.Version == "1.0.0" {
			found = true
		}
	}
	if !found {
		t.Fatalf("demo pack not listed: %+v", items)
	}

	readResult, rpcErr := assetRead(requestInput{WorkspaceRoot: root, PackID: "demo", ResourceID: "preview"})
	if rpcErr != nil {
		t.Fatalf("asset.read: %s", rpcErr.Message)
	}
	if readResult.(map[string]any)["base64"] == "" {
		t.Fatal("asset.read returned empty base64")
	}

	scaffoldResult, rpcErr := stylePackScaffold(requestInput{WorkspaceRoot: root, PackID: "demo", Name: "Demo Project"})
	if rpcErr != nil {
		t.Fatalf("stylepack.scaffold: %s", rpcErr.Message)
	}
	project := scaffoldResult.(map[string]any)["project"].(project)
	if project.Mode != "code" || project.StylePack != "demo" || project.SourceDir != "demo-project" {
		t.Fatalf("unexpected scaffolded project: %+v", project)
	}
	if !fileExists(filepath.Join(root, "demo-project", "render.mjs")) {
		t.Fatal("template was not copied to project")
	}
}

func TestAssetPackTypeFilter(t *testing.T) {
	root := t.TempDir()
	writeAssetPackFixture(t, root)
	result, rpcErr := assetList(requestInput{WorkspaceRoot: root, TypeFilter: "image"})
	if rpcErr != nil {
		t.Fatalf("asset.list: %s", rpcErr.Message)
	}
	if len(result.(map[string]any)["items"].([]assetPack)) == 0 {
		t.Fatal("expected image resource pack")
	}
}
