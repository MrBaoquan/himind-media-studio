package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func hasIssueCode(issues []map[string]any, code string) bool {
	for _, item := range issues {
		if asString(item["code"]) == code {
			return true
		}
	}
	return false
}

func TestCompositionDataContractRejectsInvalidCategories(t *testing.T) {
	tpl, ok := findTemplate("composition")
	if !ok {
		t.Fatal("composition template not found")
	}
	recipe := map[string]any{"data": map[string]any{"categories": []any{
		map[string]any{"label": "交通", "value": -1, "percentage": 50},
	}}}
	issues := validateTemplateData(tpl, recipe)
	for _, code := range []string{"data_contract_min_items", "data_contract_minimum", "data_contract_additional_property"} {
		if !hasIssueCode(issues, code) {
			t.Errorf("expected issue %s, got %#v", code, issues)
		}
	}
}

func TestFlowDataContractRejectsTooManySteps(t *testing.T) {
	tpl, ok := findTemplate("flow")
	if !ok {
		t.Fatal("flow template not found")
	}
	steps := make([]any, 0, 7)
	for i := 0; i < 7; i++ {
		steps = append(steps, map[string]any{"label": "步骤"})
	}
	issues := validateTemplateData(tpl, map[string]any{"data": map[string]any{"steps": steps}})
	if !hasIssueCode(issues, "data_contract_max_items") {
		t.Fatalf("expected data_contract_max_items, got %#v", issues)
	}
}

func TestRecipeValidateRunsTemplateDataContract(t *testing.T) {
	recipe := defaultRecipe("composition", "tech-blue", nil)
	data := recipe["data"].(map[string]any)
	categories := data["categories"].([]any)
	categories[0].(map[string]any)["value"] = -5
	issues := validateRecipe(recipe, &project{})
	if !hasIssueCode(issues, "data_contract_minimum") {
		t.Fatalf("recipe.validate did not surface data contract issues: %#v", issues)
	}
}

func TestDefaultRecipesAndFixturesSatisfyDataContracts(t *testing.T) {
	templates := map[string]videoTemplate{}
	for _, tpl := range loadTemplates() {
		templates[tpl.ID] = tpl
	}

	for _, tpl := range templates {
		if tpl.DataContract == "" {
			continue
		}
		recipe := defaultRecipe(tpl.ID, "tech-blue", nil)
		materialized, err := materializeRecipe(t.TempDir(), recipe)
		if err != nil {
			t.Errorf("%s: materialize default recipe: %v", tpl.ID, err)
			continue
		}
		if issues := validateTemplateData(tpl, materialized); len(issues) > 0 {
			t.Errorf("%s: default recipe violates data contract: %#v", tpl.ID, issues)
		}
	}

	manifest := loadFixtures(t)
	root := filepath.Join(pluginRoot(), "catalog", "fixtures")
	for _, fixture := range manifest.Fixtures {
		tpl, ok := templates[fixture.Template]
		if !ok || tpl.DataContract == "" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(fixture.Recipe)))
		if err != nil {
			t.Errorf("%s: read fixture recipe: %v", fixture.ID, err)
			continue
		}
		var recipe map[string]any
		if err := json.Unmarshal(raw, &recipe); err != nil {
			t.Errorf("%s: parse fixture recipe: %v", fixture.ID, err)
			continue
		}
		materialized, err := materializeRecipe(t.TempDir(), recipe)
		if err != nil {
			t.Errorf("%s: materialize fixture: %v", fixture.ID, err)
			continue
		}
		if issues := validateTemplateData(tpl, materialized); len(issues) > 0 {
			t.Errorf("%s: fixture violates data contract: %#v", fixture.ID, issues)
		}
	}
}

func TestProjectUpdateValidatesMaterializedRefs(t *testing.T) {
	workspace := t.TempDir()
	if _, rpcErr := handle(himindjsonrpc.Request{
		Method: "media.workbench.project.create",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","name":"ref-update","template_id":"choropleth","style_id":"tech-blue"}`),
	}); rpcErr != nil {
		t.Fatalf("create: %s", rpcErr.Message)
	}
	got, rpcErr := handle(himindjsonrpc.Request{
		Method: "media.workbench.project.get",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"ref-update"}`),
	})
	if rpcErr != nil {
		t.Fatalf("get: %s", rpcErr.Message)
	}
	item := got.(map[string]any)["project"].(project)
	updated, rpcErr := handle(himindjsonrpc.Request{
		Method: "media.workbench.project.update",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"ref-update","expected_updated_at":"` + item.UpdatedAt + `","recipe":` + mustMarshal(t, item.Recipe) + `}`),
	})
	if rpcErr != nil {
		t.Fatalf("update: %s", rpcErr.Message)
	}
	result := updated.(map[string]any)
	if asString(result["state"]) == "blocked" {
		t.Fatalf("valid $ref recipe was blocked by data contract: %#v", result["blockers"])
	}
	if saved := result["project"].(project); saved.Revision != 2 {
		t.Fatalf("revision = %d, want 2", saved.Revision)
	}
}

func mustMarshal(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(raw)
}

func TestStoryboardNestedShotContractIsValidated(t *testing.T) {
	tpl, ok := findTemplate("storyboard")
	if !ok {
		t.Fatal("storyboard template not found")
	}
	recipe := map[string]any{"data": map[string]any{"storyboard": []any{
		map[string]any{
			"shot":             "composition",
			"duration_seconds": 4,
			"data": map[string]any{"categories": []any{
				map[string]any{"label": "交通", "value": -1},
				map[string]any{"label": "住宿", "value": 2},
			}},
		},
	}}}
	issues := validateTemplateData(tpl, recipe)
	if !hasIssueCode(issues, "data_contract_minimum") {
		t.Fatalf("nested shot contract was not enforced: %#v", issues)
	}
	if !strings.Contains(asString(issues[0]["message"]), "data.storyboard[0].data.categories[0].value") {
		t.Fatalf("nested contract issue path is not precise: %#v", issues[0])
	}
}
