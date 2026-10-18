package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// TestWorkflowContextBridge 守住 Workflow 调用 Capability 的三段桥：
// plan_step / project_step / workflow_context。
func TestWorkflowContextBridge(t *testing.T) {
	workspace := t.TempDir()
	rawPlan, err := os.ReadFile(filepath.Join(pluginRoot(), "catalog", "examples", "plans", "composition.plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	var plan map[string]any
	if err := json.Unmarshal(rawPlan, &plan); err != nil {
		t.Fatal(err)
	}

	normalizePayload, _ := json.Marshal(map[string]any{
		"plan_step":        "GV-PLAN",
		"workflow_context": map[string]any{"steps": map[string]any{"GV-PLAN": plan}},
	})
	normalized, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.plan.normalize", Params: normalizePayload})
	if rpcErr != nil {
		t.Fatalf("plan.normalize via workflow context: %s", rpcErr.Message)
	}
	if valid := normalized.(map[string]any)["valid"].(bool); !valid {
		t.Fatalf("normalized plan invalid: %#v", normalized)
	}

	generatePayload, _ := json.Marshal(map[string]any{
		"workspace_root": workspace,
		"name":           "workflow-bridge-check",
		"plan_step":      "GV-NORMALIZE",
		"workflow_context": map[string]any{"steps": map[string]any{
			"GV-NORMALIZE": normalized,
		}},
		"preview": false,
	})
	generated, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.generate.run", Params: generatePayload})
	if rpcErr != nil {
		t.Fatalf("generate.run via workflow context: %s", rpcErr.Message)
	}
	generatedMap := generated.(map[string]any)
	if asString(generatedMap["state"]) != "created" {
		t.Fatalf("generate state = %v, blockers=%#v", generatedMap["state"], generatedMap["blockers"])
	}

	qualityPayload, _ := json.Marshal(map[string]any{
		"workspace_root": workspace,
		"project_step":   "GV-GENERATE",
		"workflow_context": map[string]any{"steps": map[string]any{
			"GV-GENERATE": generatedMap,
		}},
	})
	quality, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.quality.check", Params: qualityPayload})
	if rpcErr != nil {
		t.Fatalf("quality.check via workflow context: %s", rpcErr.Message)
	}
	if passed := quality.(map[string]any)["passed"].(bool); !passed {
		t.Fatalf("quality failed: %#v", quality)
	}
}

func TestWorkflowDataStepBridge(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "items.csv"), []byte("name,value\n省会A,120\n省会B,98\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ingestPayload, _ := json.Marshal(map[string]any{"workspace_root": workspace, "path": "items.csv"})
	ingested, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.data.ingest", Params: ingestPayload})
	if rpcErr != nil {
		t.Fatalf("data.ingest: %s", rpcErr.Message)
	}
	plan := map[string]any{
		"storyboard": map[string]any{"shots": []any{
			map[string]any{"shot": "leaderboard-tech", "duration_seconds": 5, "data_binding": "data"},
		}},
		"presentation": map[string]any{"title": "数据入口桥接", "style_id": "tech-blue"},
	}
	generatePayload, _ := json.Marshal(map[string]any{
		"workspace_root": workspace,
		"name":           "workflow-data-bridge",
		"plan":           plan,
		"data_step":      "GV-DATA",
		"workflow_context": map[string]any{"steps": map[string]any{
			"GV-DATA": ingested,
		}},
		"preview": false,
	})
	generated, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.generate.run", Params: generatePayload})
	if rpcErr != nil {
		t.Fatalf("generate.run with data_step: %s", rpcErr.Message)
	}
	result := generated.(map[string]any)
	if asString(result["state"]) != "created" {
		t.Fatalf("state = %v, blockers=%#v", result["state"], result["blockers"])
	}
	recipe := result["recipe"].(map[string]any)
	data := recipe["data"].(map[string]any)
	items, ok := data["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("recipe items = %#v", data["items"])
	}
}
