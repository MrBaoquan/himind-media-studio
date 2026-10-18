package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// storyboard_test.go —— 守住 Storyboard 编辑器的"声明驱动"边界。
//
// 编辑器不能硬编码 big-number / route-growth 之类的题材 id。
// 哪些模板能当镜头、默认顺序是什么，必须来自 catalog/templates.json；
// 渲染器里的 SHOT_REGISTRY 必须与它一一对应。

func loadShotRegistryIDs(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), "renderer", "remotion", "shots", "registry.js"))
	if err != nil {
		t.Fatalf("read shot registry: %v", err)
	}
	block := regexp.MustCompile(`(?s)SHOT_REGISTRY\s*=\s*\{(.*?)\n\};`).FindStringSubmatch(string(raw))
	if block == nil {
		t.Fatal("cannot locate SHOT_REGISTRY object")
	}
	keys := regexp.MustCompile(`(?m)^\s*(?:'([a-z0-9-]+)'|([a-z0-9-]+))\s*:`).FindAllStringSubmatch(block[1], -1)
	out := map[string]bool{}
	for _, match := range keys {
		key := match[1]
		if key == "" {
			key = match[2]
		}
		out[key] = true
	}
	return out
}

func TestStoryboardShotCatalogMatchesRendererRegistry(t *testing.T) {
	registry := loadShotRegistryIDs(t)
	catalog := map[string]bool{}
	orders := map[int]string{}
	for _, item := range loadTemplates() {
		if !item.StoryboardShot {
			continue
		}
		if item.Status != "ready" {
			t.Errorf("%s is a storyboard shot but status is %q", item.ID, item.Status)
		}
		if item.StoryboardOrder <= 0 {
			t.Errorf("%s has invalid storyboard_order %d", item.ID, item.StoryboardOrder)
		}
		if previous, exists := orders[item.StoryboardOrder]; exists {
			t.Errorf("storyboard_order %d is used by both %s and %s", item.StoryboardOrder, previous, item.ID)
		}
		orders[item.StoryboardOrder] = item.ID
		catalog[item.ID] = true
	}
	for id := range registry {
		if !catalog[id] {
			t.Errorf("renderer registers shot %q but templates.json does not mark it storyboard_shot", id)
		}
	}
	for id := range catalog {
		if !registry[id] {
			t.Errorf("templates.json marks %q as storyboard_shot but renderer does not register it", id)
		}
	}
	if len(catalog) == 0 {
		t.Fatal("no storyboard shots declared")
	}
}

func TestTemplateSchemaProvidesDefaultsForStoryboardShots(t *testing.T) {
	checked := 0
	for _, item := range loadTemplates() {
		if !item.StoryboardShot {
			continue
		}
		result, rpcErr := templateSchema(requestInput{TemplateID: item.ID, StyleID: "tech-blue"})
		if rpcErr != nil {
			t.Errorf("%s: templateSchema: %s", item.ID, rpcErr.Message)
			continue
		}
		payload, _ := result.(map[string]any)
		defaultData, _ := payload["default_data"].(map[string]any)
		if defaultData == nil || len(defaultData) == 0 {
			t.Errorf("%s: templateSchema returned no default_data", item.ID)
			continue
		}
		if item.DataContract != "" {
			if _, ok := payload["schema"].(map[string]any); !ok {
				t.Errorf("%s: templateSchema returned no schema", item.ID)
			}
			// 默认数据可以带 $ref（例如 choropleth 的 GeoJSON）；契约校验前先物化。
			materialized, err := materializeRecipe(t.TempDir(), map[string]any{"data": defaultData})
			if err != nil {
				t.Errorf("%s: materialize default_data: %v", item.ID, err)
				continue
			}
			resolvedData, _ := materialized["data"].(map[string]any)
			if issues := validateTemplateData(item, map[string]any{"data": resolvedData}); len(issues) > 0 {
				t.Errorf("%s: default_data violates contract: %#v", item.ID, issues)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no storyboard shot defaults checked")
	}
}

func TestProjectGetReturnsWorkspaceRoot(t *testing.T) {
	workspace := t.TempDir()
	if _, rpcErr := handle(himindjsonrpc.Request{
		Method: "media.workbench.project.create",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","name":"workspace-root-check","template_id":"storyboard","style_id":"tech-blue"}`),
	}); rpcErr != nil {
		t.Fatalf("create: %s", rpcErr.Message)
	}
	got, rpcErr := handle(himindjsonrpc.Request{
		Method: "media.workbench.project.get",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"workspace-root-check"}`),
	})
	if rpcErr != nil {
		t.Fatalf("get: %s", rpcErr.Message)
	}
	payload := got.(map[string]any)
	item := payload["project"].(project)
	if item.WorkspaceRoot == "" {
		t.Fatal("project.get did not return project.workspace_root")
	}
	if payload["workspace_root"] != item.WorkspaceRoot {
		t.Fatalf("top-level workspace_root %q != project.workspace_root %q", payload["workspace_root"], item.WorkspaceRoot)
	}
}

func TestStoryboardEditorDoesNotHardcodeShotIDs(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(pluginRoot(), "..", "..", "..", "ui", "src", "features", "storyboard-editor.tsx"))
	if err != nil {
		t.Fatalf("read editor source: %v", err)
	}
	text := string(source)
	for id := range loadShotRegistryIDs(t) {
		if strings.Contains(text, "'"+id+"'") || strings.Contains(text, "\""+id+"\"") {
			t.Errorf("storyboard editor hardcodes shot id %q; read storyboard_shot from catalog instead", id)
		}
	}
	if strings.Contains(text, "route-growth") || strings.Contains(text, "choropleth") {
		t.Error("storyboard editor contains genre-specific identifiers")
	}
}
