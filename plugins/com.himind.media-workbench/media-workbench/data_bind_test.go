package main

import (
	"encoding/json"
	"testing"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func bindForTest(t *testing.T, templateID string, data map[string]any) map[string]any {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{"template_id": templateID, "data": data})
	result, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.data.bind", Params: payload})
	if rpcErr != nil {
		t.Fatalf("data.bind: %s", rpcErr.Message)
	}
	return result.(map[string]any)
}

func TestDataBindComposition(t *testing.T) {
	result := bindForTest(t, "composition", map[string]any{
		"unit": "亿元",
		"items": []any{
			map[string]any{"name": "交通", "value": 60},
			map[string]any{"name": "住宿", "value": 40},
		},
	})
	if result["valid"] != true {
		t.Fatalf("invalid: %#v", result["issues"])
	}
	data := result["data"].(map[string]any)
	categories := data["categories"].([]any)
	if len(categories) != 2 || categories[0].(map[string]any)["label"] != "交通" {
		t.Fatalf("categories = %#v", categories)
	}
}

func TestDataBindLeaderboard(t *testing.T) {
	result := bindForTest(t, "leaderboard-tech", map[string]any{
		"items": []any{
			map[string]any{"name": "合肥", "value": 120, "change": "+5%"},
			map[string]any{"name": "芜湖", "value": 98},
		},
	})
	if result["valid"] != true {
		t.Fatalf("invalid: %#v", result["issues"])
	}
	data := result["data"].(map[string]any)
	items := data["items"].([]any)
	if len(items) != 2 || items[0].(map[string]any)["sub_value"] != "+5%" {
		t.Fatalf("items = %#v", items)
	}
}

func TestDataBindFlow(t *testing.T) {
	result := bindForTest(t, "flow", map[string]any{
		"items": []any{
			map[string]any{"name": "洞察", "description": "找问题"},
			map[string]any{"name": "脚本", "description": "写叙事"},
		},
	})
	if result["valid"] != true {
		t.Fatalf("invalid: %#v", result["issues"])
	}
	data := result["data"].(map[string]any)
	steps := data["steps"].([]any)
	if len(steps) != 2 || steps[0].(map[string]any)["label"] != "洞察" {
		t.Fatalf("steps = %#v", steps)
	}
}

func TestDataBindRejectsUnstructuredRoute(t *testing.T) {
	result := bindForTest(t, "route-growth", map[string]any{
		"items": []any{map[string]any{"name": "合肥", "value": 1}},
	})
	if result["valid"] != false {
		t.Fatalf("expected invalid route data: %#v", result)
	}
	issues, _ := result["issues"].([]map[string]any)
	if !hasIssueCode(issues, "data_bind_lines_missing") {
		t.Fatalf("unexpected issues: %#v", issues)
	}
}

func TestDataBindSankeyItemsToLinks(t *testing.T) {
	result := bindForTest(t, "sankey-flow", map[string]any{
		"unit":        "亿元",
		"total_label": "文旅消费",
		"items": []any{
			map[string]any{"name": "交通", "value": 2860},
			map[string]any{"name": "住宿", "value": 2450},
		},
	})
	if result["valid"] != true {
		t.Fatalf("invalid: %#v", result["issues"])
	}
	data := result["data"].(map[string]any)
	if data["total_label"] != "文旅消费" {
		t.Fatalf("total_label = %#v", data["total_label"])
	}
	links := data["links"].([]any)
	if len(links) != 2 {
		t.Fatalf("links = %#v", links)
	}
	first := links[0].(map[string]any)
	if first["source"] != "文旅消费" || first["target"] != "交通" || first["value"] != float64(2860) {
		t.Fatalf("first link = %#v", first)
	}
}

func TestDataBindSankeyKeepsExplicitLinks(t *testing.T) {
	result := bindForTest(t, "sankey-flow", map[string]any{
		"links": []any{
			map[string]any{"source": "总量", "target": "A", "value": 3},
			map[string]any{"source": "总量", "target": "B", "value": 2},
		},
	})
	if result["valid"] != true {
		t.Fatalf("invalid: %#v", result["issues"])
	}
	links := result["data"].(map[string]any)["links"].([]any)
	if len(links) != 2 {
		t.Fatalf("links = %#v", links)
	}
	if links[0].(map[string]any)["target"] != "A" {
		t.Fatalf("links = %#v", links)
	}
}
