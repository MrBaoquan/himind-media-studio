package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeUpgradeStylePack(t *testing.T, root string) {
	t.Helper()
	packRoot := filepath.Join(root, "style-packs", "demo-style")
	templateRoot := filepath.Join(packRoot, "template")
	if err := os.MkdirAll(templateRoot, 0755); err != nil {
		t.Fatalf("mkdir template: %v", err)
	}
	manifest := `{"schema":"himind.asset-pack.v1","id":"demo-style","version":"0.1.0","name":"Demo","resources":[{"id":"template","type":"template","path":"template","role":"render-template","runtime":"browser-code","entry":"render.mjs"}]}`
	if err := os.WriteFile(filepath.Join(packRoot, "asset-pack.json"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write asset pack: %v", err)
	}
	renderer := "// upgraded renderer\n// --sheet --strip --stills --frames --encode\n"
	if err := os.WriteFile(filepath.Join(templateRoot, "render.mjs"), []byte(renderer), 0644); err != nil {
		t.Fatalf("write renderer: %v", err)
	}
}

func TestProjectUpgradeStylePreservesProjectCode(t *testing.T) {
	root := t.TempDir()
	writeCodeDemo(t, root)
	writeUpgradeStylePack(t, root)
	importResult, rpcErr := projectImport(requestInput{WorkspaceRoot: root, Name: "code-demo", SourceDir: "code-demo"})
	if rpcErr != nil {
		t.Fatalf("import: %s", rpcErr.Message)
	}
	item := importResult.(map[string]any)["project"].(project)
	reelPath := filepath.Join(root, "code-demo", "reel", "reel.js")
	beforeReel, _ := os.ReadFile(reelPath)

	planResult, rpcErr := projectUpgradeStyle(requestInput{WorkspaceRoot: root, ProjectID: item.ID, DryRun: true})
	if rpcErr != nil {
		t.Fatalf("dry run: %s", rpcErr.Message)
	}
	plan := planResult.(map[string]any)
	if plan["state"] != "planned" {
		t.Fatalf("dry run state = %v", plan["state"])
	}
	if raw, _ := os.ReadFile(filepath.Join(root, "code-demo", "render.mjs")); !strings.Contains(string(raw), "demo renderer") {
		t.Fatal("dry run modified render.mjs")
	}

	upgradeResult, rpcErr := projectUpgradeStyle(requestInput{WorkspaceRoot: root, ProjectID: item.ID})
	if rpcErr != nil {
		t.Fatalf("upgrade: %s", rpcErr.Message)
	}
	upgraded := upgradeResult.(map[string]any)
	if upgraded["state"] != "upgraded" {
		t.Fatalf("upgrade state = %v", upgraded["state"])
	}
	raw, _ := os.ReadFile(filepath.Join(root, "code-demo", "render.mjs"))
	if !strings.Contains(string(raw), "upgraded renderer") {
		t.Fatal("render.mjs was not upgraded")
	}
	afterReel, _ := os.ReadFile(reelPath)
	if string(beforeReel) != string(afterReel) {
		t.Fatal("project reel code was modified")
	}
	updated := upgraded["project"].(project)
	if updated.PreviewStatus != "stale" || updated.DeterminismStatus != "stale" || updated.DeterminismHash != "" {
		t.Fatalf("project was not marked stale: %+v", updated)
	}
	if updated.Revision != item.Revision+1 {
		t.Fatalf("revision = %d, want %d", updated.Revision, item.Revision+1)
	}
	if _, err := os.Stat(upgraded["backup"].(string)); err != nil {
		t.Fatalf("backup missing: %v", err)
	}

	again, rpcErr := projectUpgradeStyle(requestInput{WorkspaceRoot: root, ProjectID: item.ID})
	if rpcErr != nil {
		t.Fatalf("second upgrade: %s", rpcErr.Message)
	}
	if again.(map[string]any)["state"] != "current" {
		t.Fatalf("second upgrade state = %v", again.(map[string]any)["state"])
	}
}
