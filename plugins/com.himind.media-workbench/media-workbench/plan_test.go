package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func readRecipeFixture(t *testing.T, relative string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), "catalog", "fixtures", filepath.FromSlash(relative)))
	if err != nil {
		t.Fatalf("read fixture %s: %v", relative, err)
	}
	var recipe map[string]any
	if err := json.Unmarshal(raw, &recipe); err != nil {
		t.Fatalf("parse fixture %s: %v", relative, err)
	}
	return recipe
}

func TestPlanFromRecipeCompositionRoundTrip(t *testing.T) {
	recipe := readRecipeFixture(t, "composition/recipe.json")
	plan, err := planFromRecipe(recipe)
	if err != nil {
		t.Fatalf("planFromRecipe: %v", err)
	}
	if issues := validatePlan(plan); len(issues) > 0 {
		t.Fatalf("plan from composition recipe is invalid: %#v", issues)
	}
	if len(plan.Storyboard.Shots) != 1 || plan.Storyboard.Shots[0].Shot != "composition" {
		t.Fatalf("unexpected shots: %#v", plan.Storyboard.Shots)
	}
	data, _ := recipe["data"].(map[string]any)
	compiled, issues := compilePlan(plan, data)
	if len(issues) > 0 {
		t.Fatalf("compile: %#v", issues)
	}
	for _, key := range []string{"template_id", "style_id", "title", "subtitle", "source"} {
		if compiled[key] != recipe[key] {
			t.Errorf("%s mismatch: compiled=%v fixture=%v", key, compiled[key], recipe[key])
		}
	}
	if numberValue(compiled["duration_seconds"]) != numberValue(recipe["duration_seconds"]) {
		t.Errorf("duration mismatch: compiled=%v fixture=%v", compiled["duration_seconds"], recipe["duration_seconds"])
	}
	if !reflect.DeepEqual(compiled["data"], recipe["data"]) {
		t.Errorf("data changed during Plan round-trip")
	}
}

func TestPlanFromRecipeStoryboard(t *testing.T) {
	recipe := readRecipeFixture(t, "storyboard/recipe.json")
	plan, err := planFromRecipe(recipe)
	if err != nil {
		t.Fatalf("planFromRecipe: %v", err)
	}
	if len(plan.Storyboard.Shots) != 4 {
		t.Fatalf("shot count = %d, want 4", len(plan.Storyboard.Shots))
	}
	if issues := validatePlan(plan); len(issues) > 0 {
		t.Fatalf("plan from storyboard recipe is invalid: %#v", issues)
	}
	if total := totalShotSeconds(plan.Storyboard.Shots); total != 19 {
		t.Fatalf("duration = %v, want 19", total)
	}
	compiled, issues := compilePlan(plan, nil)
	if len(issues) > 0 {
		t.Fatalf("compile storyboard: %#v", issues)
	}
	if asString(compiled["template_id"]) != "storyboard" {
		t.Fatalf("template_id = %s, want storyboard", asString(compiled["template_id"]))
	}
	data, _ := compiled["data"].(map[string]any)
	shots, _ := data["storyboard"].([]any)
	if len(shots) != 4 {
		t.Fatalf("compiled storyboard shots = %d, want 4", len(shots))
	}
}

func TestPlanNormalizeFillsDefaults(t *testing.T) {
	plan, err := normalizePlan(map[string]any{
		"storyboard": map[string]any{
			"shots": []any{map[string]any{"shot": "composition"}},
		},
		"presentation": map[string]any{"title": "测试标题"},
	})
	if err != nil {
		t.Fatalf("normalizePlan: %v", err)
	}
	if plan.Version != planVersion {
		t.Errorf("version = %s", plan.Version)
	}
	if plan.Presentation.StyleID != "tech-blue" {
		t.Errorf("style_id = %s, want tech-blue", plan.Presentation.StyleID)
	}
	if plan.Storyboard.Shots[0].DurationSeconds != 10 {
		t.Errorf("duration = %v, want template default 10", plan.Storyboard.Shots[0].DurationSeconds)
	}
	if plan.Narrative.Structure != "single-shot" {
		t.Errorf("structure = %s", plan.Narrative.Structure)
	}
	if plan.Brief.DurationSeconds != 10 {
		t.Errorf("brief duration = %v", plan.Brief.DurationSeconds)
	}
	if issues := validatePlan(plan); len(issues) > 0 {
		t.Fatalf("normalized plan should be valid: %#v", issues)
	}
}

func TestPlanValidateRejectsUnknownShot(t *testing.T) {
	plan, err := normalizePlan(map[string]any{
		"storyboard": map[string]any{
			"shots": []any{map[string]any{"shot": "does-not-exist", "duration_seconds": 5}},
		},
		"presentation": map[string]any{"title": "坏计划"},
	})
	if err != nil {
		t.Fatalf("normalizePlan: %v", err)
	}
	issues := validatePlan(plan)
	if !hasIssueCode(issues, "plan_shot_not_found") {
		t.Fatalf("expected plan_shot_not_found, got %#v", issues)
	}
}

func TestRecipeCompileComposition(t *testing.T) {
	plan, err := normalizePlan(map[string]any{
		"storyboard": map[string]any{
			"shots": []any{map[string]any{"shot": "composition", "duration_seconds": 8}},
		},
		"presentation": map[string]any{
			"title":    "消费构成",
			"style_id": "tech-blue",
		},
	})
	if err != nil {
		t.Fatalf("normalizePlan: %v", err)
	}
	data := map[string]any{
		"unit": "亿元",
		"categories": []any{
			map[string]any{"label": "交通", "value": 60},
			map[string]any{"label": "住宿", "value": 40},
		},
	}
	recipe, issues := compilePlan(plan, data)
	if len(issues) > 0 {
		t.Fatalf("compile: %#v", issues)
	}
	if asString(recipe["template_id"]) != "composition" {
		t.Fatalf("template_id = %s", asString(recipe["template_id"]))
	}
	if numberValue(recipe["duration_seconds"]) != 8 {
		t.Fatalf("duration = %v", recipe["duration_seconds"])
	}
	if asString(recipe["title"]) != "消费构成" {
		t.Fatalf("title = %s", asString(recipe["title"]))
	}
}

func TestRecipeCompileMultiShotRequiresData(t *testing.T) {
	plan, err := normalizePlan(map[string]any{
		"storyboard": map[string]any{
			"shots": []any{
				map[string]any{"shot": "big-number", "duration_seconds": 3},
				map[string]any{"shot": "compare", "duration_seconds": 4},
			},
		},
		"presentation": map[string]any{"title": "多镜头", "style_id": "tech-blue"},
	})
	if err != nil {
		t.Fatalf("normalizePlan: %v", err)
	}
	_, issues := compilePlan(plan, nil)
	if !hasIssueCode(issues, "compile_shot_data_missing") {
		t.Fatalf("expected compile_shot_data_missing, got %#v", issues)
	}
}

func TestExamplePlansValidate(t *testing.T) {
	files := []string{
		"catalog/examples/plans/composition.plan.json",
		"catalog/examples/plans/flow.plan.json",
		"catalog/examples/plans/route-growth.plan.json",
	}
	for _, relative := range files {
		raw, err := os.ReadFile(filepath.Join(pluginRoot(), filepath.FromSlash(relative)))
		if err != nil {
			t.Errorf("read %s: %v", relative, err)
			continue
		}
		var planMap map[string]any
		if err := json.Unmarshal(raw, &planMap); err != nil {
			t.Errorf("parse %s: %v", relative, err)
			continue
		}
		plan, err := normalizePlan(planMap)
		if err != nil {
			t.Errorf("normalize %s: %v", relative, err)
			continue
		}
		if issues := validatePlan(plan); len(issues) > 0 {
			t.Errorf("%s: %#v", relative, issues)
		}
	}
}

func TestPlanCapabilityPipeline(t *testing.T) {
	rawPlan, err := os.ReadFile(filepath.Join(pluginRoot(), "catalog", "examples", "plans", "composition.plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	var planMap map[string]any
	if err := json.Unmarshal(rawPlan, &planMap); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(map[string]any{"plan": planMap})
	normalized, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.plan.normalize", Params: encoded})
	if rpcErr != nil {
		t.Fatalf("plan.normalize: %s", rpcErr.Message)
	}
	if valid := normalized.(map[string]any)["valid"].(bool); !valid {
		t.Fatalf("normalize result invalid: %#v", normalized)
	}
	validated, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.plan.validate", Params: encoded})
	if rpcErr != nil {
		t.Fatalf("plan.validate: %s", rpcErr.Message)
	}
	if valid := validated.(map[string]any)["valid"].(bool); !valid {
		t.Fatalf("validate result invalid: %#v", validated)
	}
	recipe := readRecipeFixture(t, "composition/recipe.json")
	compilePayload, _ := json.Marshal(map[string]any{"plan": planMap, "data": recipe["data"]})
	compiled, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.recipe.compile", Params: compilePayload})
	if rpcErr != nil {
		t.Fatalf("recipe.compile: %s", rpcErr.Message)
	}
	result := compiled.(map[string]any)
	if asString(result["state"]) != "compiled" {
		t.Fatalf("compile state = %v, blockers=%#v", result["state"], result["blockers"])
	}
	compiledRecipe := result["recipe"].(map[string]any)
	if asString(compiledRecipe["template_id"]) != "composition" {
		t.Fatalf("template_id = %s", asString(compiledRecipe["template_id"]))
	}
}

func TestPlanNormalizeRejectsUnknownFields(t *testing.T) {
	_, err := normalizePlan(map[string]any{
		"storyboard": map[string]any{
			"shots": []any{map[string]any{"shot": "composition", "duration_seconds": 5}},
		},
		"presentation": map[string]any{"title": "未知字段测试"},
		"unexpected":   true,
	})
	if err == nil {
		t.Fatal("normalizePlan accepted an unknown top-level field")
	}
}
