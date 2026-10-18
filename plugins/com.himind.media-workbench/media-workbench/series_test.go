package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func baseCompositionPlan(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), "catalog", "examples", "plans", "composition.plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	var plan map[string]any
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatal(err)
	}
	return plan
}

func seriesItems() []any {
	return []any{
		map[string]any{
			"name":  "2024",
			"title": "2024 年消费构成",
			"data": map[string]any{
				"unit":        "亿元",
				"total_label": "文旅消费",
				"categories": []any{
					map[string]any{"label": "交通", "value": 60},
					map[string]any{"label": "住宿", "value": 40},
				},
			},
		},
		map[string]any{
			"name":  "2025",
			"title": "2025 年消费构成",
			"data": map[string]any{
				"unit":        "亿元",
				"total_label": "文旅消费",
				"categories": []any{
					map[string]any{"label": "交通", "value": 55},
					map[string]any{"label": "住宿", "value": 45},
				},
			},
		},
	}
}

func TestSeriesPlanBuildsSummary(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"name_prefix": "文旅构成",
		"plan":        baseCompositionPlan(t),
		"items":       seriesItems(),
	})
	result, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.series.plan", Params: payload})
	if rpcErr != nil {
		t.Fatalf("series.plan: %s", rpcErr.Message)
	}
	payloadMap := result.(map[string]any)
	if payloadMap["valid"] != true {
		t.Fatalf("series.plan invalid: %#v", payloadMap["issues"])
	}
	summary := payloadMap["summary"].(map[string]any)
	if summary["count"] != 2 {
		t.Fatalf("count = %v", summary["count"])
	}
	if summary["total_seconds"] != float64(20) {
		t.Fatalf("total_seconds = %v", summary["total_seconds"])
	}
	items := payloadMap["items"].([]seriesPlanItem)
	if items[0].ProjectName != "文旅构成 · 2024" {
		t.Fatalf("project name = %s", items[0].ProjectName)
	}
}

func TestSeriesGenerateCreatesProjects(t *testing.T) {
	workspace := t.TempDir()
	payload, _ := json.Marshal(map[string]any{
		"workspace_root": workspace,
		"name_prefix":    "文旅构成",
		"plan":           baseCompositionPlan(t),
		"items":          seriesItems(),
		"preview":        false,
	})
	result, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.series.generate", Params: payload})
	if rpcErr != nil {
		t.Fatalf("series.generate: %s", rpcErr.Message)
	}
	payloadMap := result.(map[string]any)
	if payloadMap["state"] != "completed" {
		t.Fatalf("state = %v, failures=%#v", payloadMap["state"], payloadMap["failures"])
	}
	created := payloadMap["created"].([]any)
	if len(created) != 2 {
		t.Fatalf("created = %d, want 2", len(created))
	}
	projects, err := os.ReadDir(filepath.Join(workspace, metadataDir, projectDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 2 {
		t.Fatalf("project dirs = %d, want 2", len(projects))
	}
}

func TestSeriesPlanBlocksMultiShot(t *testing.T) {
	plan := baseCompositionPlan(t)
	storyboard := plan["storyboard"].(map[string]any)
	storyboard["shots"] = []any{
		map[string]any{"shot": "big-number", "duration_seconds": 3},
		map[string]any{"shot": "compare", "duration_seconds": 4},
	}
	payload, _ := json.Marshal(map[string]any{
		"plan":  plan,
		"items": seriesItems(),
	})
	result, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.series.plan", Params: payload})
	if rpcErr != nil {
		t.Fatalf("series.plan: %s", rpcErr.Message)
	}
	issues, _ := result.(map[string]any)["issues"].([]map[string]any)
	if !hasIssueCode(issues, "series_multi_shot_not_supported") {
		t.Fatalf("expected series_multi_shot_not_supported, got %#v", issues)
	}
}
