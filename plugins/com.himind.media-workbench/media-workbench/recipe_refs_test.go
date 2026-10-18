package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// recipe_refs_test.go —— 数据资产引用的契约。
//
// 三条必须守住：
//   1. plugin: 引用能读到随包资产；
//   2. workspace: 引用能读到工作区资产；
//   3. 任何越界引用必须被拒绝（不能借 $ref 读工作区外的文件）。

func TestRecipeRefsMaterializeFromPluginAndWorkspace(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "values.json"), []byte(`{"合肥市":14210}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	recipe := map[string]any{
		"data": map[string]any{
			"geo":    map[string]any{"$ref": "plugin:catalog/data/geo/anhui-cities.geojson"},
			"values": map[string]any{"$ref": "workspace:assets/values.json"},
		},
	}

	resolved, err := materializeRecipe(root, recipe)
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	data, _ := resolved["data"].(map[string]any)
	geo, _ := data["geo"].(map[string]any)
	features, _ := geo["features"].([]any)
	if len(features) != 16 {
		t.Fatalf("expected 16 features from the plugin asset, got %d", len(features))
	}
	values, _ := data["values"].(map[string]any)
	if values["合肥市"] != float64(14210) {
		t.Fatalf("workspace asset not materialized: %v", values)
	}
}

func TestRecipeRefsRejectEscapingPaths(t *testing.T) {
	root := t.TempDir()
	cases := []string{
		"plugin:../../secret.txt",
		"workspace:../outside.json",
		"https://example.com/data.json",
		"relative/path.json",
	}
	for _, ref := range cases {
		recipe := map[string]any{"x": map[string]any{"$ref": ref}}
		if _, err := materializeRecipe(root, recipe); err == nil {
			t.Errorf("expected ref %q to be rejected", ref)
		}
	}
}

func TestDefaultChoroplethRecipeUsesReference(t *testing.T) {
	recipe := defaultRecipe("choropleth", "tech-blue", nil)
	data, _ := recipe["data"].(map[string]any)
	geo, _ := data["geo"].(map[string]any)
	ref, _ := geo["$ref"].(string)
	if ref == "" {
		t.Fatal("choropleth default recipe must use a $ref instead of inlining the GeoJSON")
	}
	if _, ok := geo["features"]; ok {
		t.Fatal("choropleth default recipe should not inline features")
	}
	encoded, err := jsonMarshalSize(recipe)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if encoded > 8*1024 {
		t.Fatalf("choropleth default recipe is %d bytes; expected the reference form to stay small", encoded)
	}

	// 物化之后必须拿到真实的 16 个区域。
	materialized, err := materializeRecipe(t.TempDir(), recipe)
	if err != nil {
		t.Fatalf("materialize default recipe: %v", err)
	}
	materializedData, _ := materialized["data"].(map[string]any)
	materializedGeo, _ := materializedData["geo"].(map[string]any)
	features, _ := materializedGeo["features"].([]any)
	if len(features) != 16 {
		t.Fatalf("expected 16 features after materialization, got %d", len(features))
	}
}

func jsonMarshal(value any) ([]byte, error) { return json.Marshal(value) }

func jsonMarshalSize(value any) (int, error) {
	raw, err := jsonMarshal(value)
	if err != nil {
		return 0, err
	}
	return len(raw), nil
}
