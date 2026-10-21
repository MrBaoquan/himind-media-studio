package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func TestShotAttachAssetCopiesExternalFile(t *testing.T) {
	root := t.TempDir()
	item := seedShotCreative(t, root)
	if _, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.assign", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID, "shot_id": "shot1", "provider": "ai.image", "prompt": "real image",
	})}); rpcErr != nil {
		t.Fatal(rpcErr)
	}
	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "real.svg")
	if err := os.WriteFile(source, []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10" fill="#123456"/></svg>`), 0644); err != nil {
		t.Fatal(err)
	}
	result, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.attach-asset", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID, "shot_id": "shot1", "path": source,
		"provider": "ai.image", "model": "doubao-seedream-5.0-pro", "width": 1024, "height": 1024,
	})})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	payload := result.(map[string]any)
	if payload["state"] != "attached" {
		t.Fatalf("attach should succeed: %#v", payload)
	}
	asset := payload["asset"].(map[string]any)
	if asset["exists"] != true || !strings.HasPrefix(asString(asset["preview_data_url"]), "data:image/svg+xml;base64,") {
		t.Fatalf("asset preview missing: %#v", asset)
	}
	rel := asString(asset["path"])
	if !strings.HasPrefix(rel, ".himind-media/creatives/") {
		t.Fatalf("asset should be copied into creative assets: %s", rel)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		t.Fatalf("copied asset missing: %v", err)
	}
	updated := payload["creative"].(creativePackage)
	shots := asMapSlice(asMap(updated.Storyboard)["shots"])
	if len(shots) != 1 || asString(shots[0]["status"]) != "generated" || len(asStringSlice(shots[0]["asset_refs"])) == 0 {
		t.Fatalf("shot not updated: %#v", shots)
	}
	if shots[0]["mock"] == true {
		t.Fatalf("real asset should not be marked mock: %#v", shots[0])
	}
}

func TestShotAttachAssetRejectsUnsupportedExtension(t *testing.T) {
	root := t.TempDir()
	item := seedShotCreative(t, root)
	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "notes.txt")
	if err := os.WriteFile(source, []byte("not an asset"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.shot.attach-asset", Params: mustJSON(t, map[string]any{
		"workspace_root": root, "creative_id": item.ID, "shot_id": "shot1", "path": source,
	})}); rpcErr == nil {
		t.Fatal("unsupported asset extension should be rejected")
	}
}
