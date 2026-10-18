package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCodeDemo(t *testing.T, root string) string {
	t.Helper()
	source := filepath.Join(root, "code-demo")
	if err := os.MkdirAll(filepath.Join(source, "reel"), 0755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	files := map[string]string{
		"style-pack.json": `{"schema":"himind.video.style-pack.v1","id":"demo-style","version":"0.1.0","name":"Demo","renderer":"browser-code","entry":"render.mjs","video_out":"out/video.mp4","preview":{"sheet_out":"out/check/sheet.jpg","strip_out":"out/check/strip.jpg","stills_out":"out/stills"},"duration_seconds":10}`,
		"package.json":    `{"name":"demo","private":true,"type":"module"}`,
		"STORYBOARD.md":   "# Demo\n\n## Reads\n\n- 标题入场\n- 数据卡\n",
		"render.mjs":      "// demo renderer\\n// --sheet --strip --stills --frames --encode\\n",
		"reel/reel.js":    "export const version = 1;\n",
	}
	for name, content := range files {
		path := filepath.Join(source, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return source
}

func TestProjectModeDefaultsToTemplate(t *testing.T) {
	if got := projectMode(project{}); got != "template" {
		t.Fatalf("projectMode(empty) = %q", got)
	}
	if got := projectMode(project{Mode: "code"}); got != "code" {
		t.Fatalf("projectMode(code) = %q", got)
	}
}

func TestHashCodeSourceIgnoresGeneratedOutputs(t *testing.T) {
	root := t.TempDir()
	source := writeCodeDemo(t, root)
	first, err := hashCodeSource(source)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(source, "out"), 0755); err != nil {
		t.Fatalf("mkdir out: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "out", "sheet.jpg"), []byte("generated"), 0644); err != nil {
		t.Fatalf("write out: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "package-lock.json"), []byte("lock"), 0644); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	second, err := hashCodeSource(source)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if first != second {
		t.Fatalf("generated outputs changed source hash: %s != %s", first, second)
	}
	if err := os.WriteFile(filepath.Join(source, "reel", "reel.js"), []byte("export const version = 2;\n"), 0644); err != nil {
		t.Fatalf("edit source: %v", err)
	}
	third, err := hashCodeSource(source)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if third == second {
		t.Fatal("source edit did not change hash")
	}
}

func TestCodeProjectImportSyncAndContext(t *testing.T) {
	root := t.TempDir()
	writeCodeDemo(t, root)
	result, rpcErr := projectImport(requestInput{WorkspaceRoot: root, Name: "code-demo", SourceDir: "code-demo"})
	if rpcErr != nil {
		t.Fatalf("import: %s", rpcErr.Message)
	}
	payload := result.(map[string]any)
	item := payload["project"].(project)
	if item.Mode != "code" || item.StylePack != "demo-style" || item.SourceHash == "" {
		t.Fatalf("unexpected imported project: %+v", item)
	}
	if item.SourceDir != "code-demo" || item.Entry != "render.mjs" {
		t.Fatalf("unexpected source fields: %+v", item)
	}

	if err := os.WriteFile(filepath.Join(root, "code-demo", "reel", "reel.js"), []byte("export const version = 3;\n"), 0644); err != nil {
		t.Fatalf("edit: %v", err)
	}
	syncResult, rpcErr := projectSync(requestInput{WorkspaceRoot: root, ProjectID: item.ID})
	if rpcErr != nil {
		t.Fatalf("sync: %s", rpcErr.Message)
	}
	syncPayload := syncResult.(map[string]any)
	if syncPayload["changed"] != true {
		t.Fatalf("sync did not detect change: %#v", syncPayload)
	}
	updated := syncPayload["project"].(project)
	if updated.PreviewStatus != "stale" || updated.Revision != item.Revision+1 {
		t.Fatalf("unexpected synced project: %+v", updated)
	}

	contextResult, rpcErr := projectContext(requestInput{WorkspaceRoot: root, ProjectID: item.ID})
	if rpcErr != nil {
		t.Fatalf("context: %s", rpcErr.Message)
	}
	contextPayload := contextResult.(map[string]any)
	if contextPayload["mode"] != "code" {
		t.Fatalf("context mode = %v", contextPayload["mode"])
	}
	if contextPayload["storyboard"] == "" {
		t.Fatal("context is missing storyboard")
	}
}

func TestCodeProjectQualityRequiresPreview(t *testing.T) {
	root := t.TempDir()
	writeCodeDemo(t, root)
	result, rpcErr := projectImport(requestInput{WorkspaceRoot: root, Name: "code-demo", SourceDir: "code-demo"})
	if rpcErr != nil {
		t.Fatalf("import: %s", rpcErr.Message)
	}
	item := result.(map[string]any)["project"].(project)
	quality, rpcErr := qualityCheck(requestInput{WorkspaceRoot: root, ProjectID: item.ID})
	if rpcErr != nil {
		t.Fatalf("quality: %s", rpcErr.Message)
	}
	payload := quality.(map[string]any)
	if payload["passed"] != false {
		t.Fatalf("quality should fail without preview: %#v", payload)
	}
	render, rpcErr := renderStart(requestInput{WorkspaceRoot: root, ProjectID: item.ID})
	if rpcErr != nil {
		t.Fatalf("render: %s", rpcErr.Message)
	}
	renderPayload := render.(map[string]any)
	if renderPayload["state"] != "blocked" {
		t.Fatalf("render should be blocked before preview: %#v", renderPayload)
	}
}
func TestStylePackValidate(t *testing.T) {
	root := t.TempDir()
	writeCodeDemo(t, root)
	result, rpcErr := stylePackValidate(requestInput{WorkspaceRoot: root, SourceDir: "code-demo"})
	if rpcErr != nil {
		t.Fatalf("stylepack.validate: %s", rpcErr.Message)
	}
	payload := result.(map[string]any)
	if payload["valid"] != true {
		t.Fatalf("valid style pack rejected: %#v", payload["issues"])
	}
	pack := payload["style_pack"].(codeStylePack)
	if pack.ID != "demo-style" || pack.DurationSeconds != 10 {
		t.Fatalf("unexpected pack: %+v", pack)
	}
}
func TestProjectDataWrite(t *testing.T) {
	root := t.TempDir()
	writeCodeDemo(t, root)
	result, rpcErr := projectImport(requestInput{WorkspaceRoot: root, Name: "code-demo", SourceDir: "code-demo"})
	if rpcErr != nil {
		t.Fatalf("import: %s", rpcErr.Message)
	}
	item := result.(map[string]any)["project"].(project)
	written, rpcErr := projectDataWrite(requestInput{
		WorkspaceRoot: root,
		ProjectID:     item.ID,
		Data: map[string]any{
			"title": "2025 省会 GDP",
			"items": []any{
				map[string]any{"name": "合肥", "value": 120},
				map[string]any{"name": "芜湖", "value": 98},
			},
		},
		Brief: map[string]any{"goal": "排名动画"},
	})
	if rpcErr != nil {
		t.Fatalf("data.write: %s", rpcErr.Message)
	}
	payload := written.(map[string]any)
	if _, err := os.Stat(filepath.Join(root, "code-demo", "input", "data.json")); err != nil {
		t.Fatalf("data.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "code-demo", "input", "brief.json")); err != nil {
		t.Fatalf("brief.json missing: %v", err)
	}
	updated := payload["project"].(project)
	if updated.PreviewStatus != "stale" || updated.Revision != item.Revision+1 {
		t.Fatalf("unexpected project after data write: %+v", updated)
	}
}

func TestWebCopyScaleFilterUsesLongEdge(t *testing.T) {
	cases := map[int]string{
		1280: "scale='if(gte(iw,ih),min(1280,iw),-2)':'if(gte(iw,ih),-2,min(1280,ih))'",
		0:    "scale='if(gte(iw,ih),min(1280,iw),-2)':'if(gte(iw,ih),-2,min(1280,ih))'",
		720:  "scale='if(gte(iw,ih),min(720,iw),-2)':'if(gte(iw,ih),-2,min(720,ih))'",
	}
	for input, want := range cases {
		if got := webCopyScaleFilter(input); got != want {
			t.Fatalf("webCopyScaleFilter(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestEffectiveWorkerCountRespectsStylePackLimit(t *testing.T) {
	cases := []struct{ requested, max, want int }{
		{0, 0, 1}, {4, 0, 4}, {4, 1, 1}, {3, 2, 2}, {-1, 2, 1},
	}
	for _, c := range cases {
		if got := effectiveWorkerCount(c.requested, c.max); got != c.want {
			t.Fatalf("effectiveWorkerCount(%d,%d)=%d want %d", c.requested, c.max, got, c.want)
		}
	}
}
