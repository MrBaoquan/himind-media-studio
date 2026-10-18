package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// TestStylePacksUseLosslessFinalFrames guards the rendering lesson from the
// celestial-seasons case: JPEG intermediate frames flatten dark gradients into
// large color bands. Final frames must be lossless PNG unless the caller
// explicitly opts into JPEG for a quick test.
func TestStylePacksUseLosslessFinalFrames(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	patterns := []string{
		filepath.Join(root, "style-packs", "*", "template", "render.mjs"),
		filepath.Join(root, "examples", "code-project-demo", "render.mjs"),
	}
	var files []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob %s: %v", pattern, err)
		}
		files = append(files, matches...)
	}
	if len(files) == 0 {
		t.Fatal("no render.mjs files found")
	}
	for _, file := range files {
		file := file
		t.Run(filepath.Base(filepath.Dir(filepath.Dir(file))), func(t *testing.T) {
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read %s: %v", file, err)
			}
			text := string(raw)
			if !strings.Contains(text, "const frameFormat") {
				t.Fatalf("%s must default final frames to PNG (missing frameFormat)", file)
			}
			if !strings.Contains(text, "frame-format") {
				t.Fatalf("%s must accept --frame-format for quick JPEG tests", file)
			}
			if strings.Contains(text, "f%05d.jpg") {
				t.Fatalf("%s still hardcodes JPEG frame input", file)
			}
			if strings.Contains(text, "shot(page, index / FPS, 'jpeg'") {
				t.Fatalf("%s still captures final frames as JPEG", file)
			}
			if strings.Contains(text, "frameOf(page, i / fps, 'image/jpeg'") {
				t.Fatalf("%s still captures final frames as JPEG", file)
			}
		})
	}
}

func TestArtifactExportBlocksFailedQuality(t *testing.T) {
	workspace := t.TempDir()
	artifactRoot := filepath.Join(workspace, metadataDir, artifactDir)
	if err := os.MkdirAll(artifactRoot, 0755); err != nil {
		t.Fatal(err)
	}
	videoPath := filepath.Join(artifactRoot, "video-artifact-quality-check.mp4")
	if err := os.WriteFile(videoPath, []byte("fake mp4"), 0644); err != nil {
		t.Fatal(err)
	}
	item, err := makeArtifact("video-artifact-quality-check", "video/mp4", videoPath, "project-quality", "browser-code", true)
	if err != nil {
		t.Fatal(err)
	}
	item.QualityStatus = "error"
	item.QualityReport = map[string]any{"ok": false, "error": "ffprobe unavailable"}
	if err := writeJSON(videoPath+".json", item); err != nil {
		t.Fatal(err)
	}

	blocked, rpcError := handle(himindjsonrpc.Request{
		Method: "media.workbench.artifact.export",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","artifact_id":"video-artifact-quality-check"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	result := blocked.(map[string]any)
	if result["state"] != "blocked" || result["exported"] != nil {
		t.Fatalf("failed quality must block export: %#v", result)
	}
	blockers, _ := result["blockers"].([]map[string]any)
	if len(blockers) != 1 || blockers[0]["code"] != "ARTIFACT_QUALITY_FAILED" {
		t.Fatalf("unexpected blockers: %#v", result["blockers"])
	}

	forced, rpcError := handle(himindjsonrpc.Request{
		Method: "media.workbench.artifact.export",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","artifact_id":"video-artifact-quality-check","allow_failed":true}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	if forced.(map[string]any)["exported"] != true {
		t.Fatalf("allow_failed should permit debug export: %#v", forced)
	}
}
