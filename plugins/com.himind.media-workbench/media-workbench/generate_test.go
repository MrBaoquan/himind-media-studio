package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func TestGenerateRunCreatesProjectPlanAndRecipe(t *testing.T) {
	workspace := t.TempDir()
	planRaw, err := os.ReadFile(filepath.Join(pluginRoot(), "catalog", "examples", "plans", "composition.plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	var plan map[string]any
	if err := json.Unmarshal(planRaw, &plan); err != nil {
		t.Fatal(err)
	}
	recipe := readRecipeFixture(t, "composition/recipe.json")
	payload, _ := json.Marshal(map[string]any{
		"workspace_root": workspace,
		"name":           "plan-generate-check",
		"plan":           plan,
		"data":           recipe["data"],
		"preview":        false,
	})
	result, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.generate.run", Params: payload})
	if rpcErr != nil {
		t.Fatalf("generate.run: %s", rpcErr.Message)
	}
	payloadMap := result.(map[string]any)
	if asString(payloadMap["state"]) != "created" {
		t.Fatalf("state = %v, blockers=%#v", payloadMap["state"], payloadMap["blockers"])
	}
	projectRoot := filepath.Join(workspace, metadataDir, projectDir, "plan-generate-check")
	for _, name := range []string{"project.json", "recipe.json", "plan.json", "storyboard.json", "brief.md"} {
		if _, err := os.Stat(filepath.Join(projectRoot, name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
	item, err := loadProject(workspace, "plan-generate-check")
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	if item.Template != "composition" {
		t.Errorf("template = %s, want composition", item.Template)
	}
	if item.Revision != 1 {
		t.Errorf("revision = %d, want 1", item.Revision)
	}
	if asString(item.Recipe["template_id"]) != "composition" {
		t.Errorf("recipe template = %s", asString(item.Recipe["template_id"]))
	}
}

func TestGenerateRunBlocksInvalidPlan(t *testing.T) {
	workspace := t.TempDir()
	payload, _ := json.Marshal(map[string]any{
		"workspace_root": workspace,
		"name":           "invalid-plan",
		"plan": map[string]any{
			"storyboard":   map[string]any{"shots": []any{map[string]any{"shot": "missing-shot"}}},
			"presentation": map[string]any{"title": "坏计划"},
		},
		"preview": false,
	})
	result, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.generate.run", Params: payload})
	if rpcErr != nil {
		t.Fatalf("generate.run: %s", rpcErr.Message)
	}
	payloadMap := result.(map[string]any)
	if asString(payloadMap["state"]) != "blocked_plan" {
		t.Fatalf("state = %v", payloadMap["state"])
	}
	blockers, _ := payloadMap["blockers"].([]map[string]any)
	if !hasIssueCode(blockers, "plan_shot_not_found") {
		t.Fatalf("unexpected blockers: %#v", payloadMap["blockers"])
	}
}

func TestProjectGetReturnsStoredPlan(t *testing.T) {
	workspace := t.TempDir()
	planRaw, _ := os.ReadFile(filepath.Join(pluginRoot(), "catalog", "examples", "plans", "composition.plan.json"))
	var plan map[string]any
	if err := json.Unmarshal(planRaw, &plan); err != nil {
		t.Fatal(err)
	}
	recipe := readRecipeFixture(t, "composition/recipe.json")
	payload, _ := json.Marshal(map[string]any{
		"workspace_root": workspace,
		"name":           "stored-plan",
		"plan":           plan,
		"data":           recipe["data"],
		"preview":        false,
	})
	if _, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.generate.run", Params: payload}); rpcErr != nil {
		t.Fatalf("generate.run: %s", rpcErr.Message)
	}
	got, rpcErr := handle(himindjsonrpc.Request{
		Method: "media.workbench.project.get",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"stored-plan"}`),
	})
	if rpcErr != nil {
		t.Fatalf("project.get: %s", rpcErr.Message)
	}
	projectPlan, ok := got.(map[string]any)["plan"].(map[string]any)
	if !ok {
		t.Fatal("project.get did not return plan")
	}
	storyboard := projectPlan["storyboard"].(map[string]any)
	shots := storyboard["shots"].([]any)
	if len(shots) != 1 {
		t.Fatalf("plan shots = %d, want 1", len(shots))
	}
}

func TestProjectGetSynthesizesPlanForLegacyProject(t *testing.T) {
	workspace := t.TempDir()
	payload, _ := json.Marshal(map[string]any{
		"workspace_root": workspace,
		"name":           "legacy-project",
		"template_id":    "leaderboard-tech",
		"style_id":       "tech-blue",
	})
	if _, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.project.create", Params: payload}); rpcErr != nil {
		t.Fatalf("project.create: %s", rpcErr.Message)
	}
	got, rpcErr := handle(himindjsonrpc.Request{
		Method: "media.workbench.project.get",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"legacy-project"}`),
	})
	if rpcErr != nil {
		t.Fatalf("project.get: %s", rpcErr.Message)
	}
	projectPlan, ok := got.(map[string]any)["plan"].(map[string]any)
	if !ok {
		t.Fatal("legacy project.get did not synthesize a plan")
	}
	storyboard := projectPlan["storyboard"].(map[string]any)
	shots := storyboard["shots"].([]any)
	if len(shots) != 1 {
		t.Fatalf("synthesized plan shots = %d, want 1", len(shots))
	}
}

func TestProjectIDForNameStableAndUnique(t *testing.T) {
	first := projectIDForName("系列A · 2024")
	second := projectIDForName("系列B · 2024")
	if first == second {
		t.Fatalf("different Chinese series names produced the same project id: %s", first)
	}
	if first != projectIDForName("系列A · 2024") {
		t.Fatal("project id is not deterministic")
	}
	for _, id := range []string{first, second, projectIDForName("普通项目")} {
		if !idPattern.MatchString(id) {
			t.Errorf("invalid generated project id: %s", id)
		}
	}
}
