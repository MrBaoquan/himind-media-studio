package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// manifest_test.go — 守住"扩展自身契约"的测试。
//
// 这些断言覆盖的是扩展要遵守的**外部约束**（ADR-0119 命名、宿主超时语义、
// 目录与兜底一致性）以及"新增模板真的能出片"这件事。

func TestManifestDeclaresNamespaceAndDistributionTargets(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), "plugin.json"))
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}
	var manifest struct {
		CapabilityNamespace string   `json:"capability_namespace"`
		DistributionTargets []string `json:"distribution_targets"`
		Capabilities        []struct {
			ID             string `json:"id"`
			TimeoutSeconds int    `json:"timeout_seconds"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse plugin.json: %v", err)
	}

	// ADR-0119：多段命名空间必须在清单中登记，否则能力命名门禁会拒绝新版本。
	if manifest.CapabilityNamespace != "media.workbench" {
		t.Errorf("capability_namespace must be \"media.workbench\", got %q", manifest.CapabilityNamespace)
	}
	if len(manifest.DistributionTargets) == 0 {
		t.Error("distribution_targets must be declared explicitly")
	}

	// 能力 ID 必须落在声明的命名空间下。
	for _, cap := range manifest.Capabilities {
		if !strings.HasPrefix(cap.ID, "media.workbench.") {
			t.Errorf("capability %q is outside the declared namespace", cap.ID)
		}
	}

	// 宿主默认 30s，且调用方传入的 timeout 会被 clamp 到声明值。
	// preview.player 首次要 npm install（≤5min）+ esbuild（≤3min），必须显式提高。
	for _, cap := range manifest.Capabilities {
		if cap.ID == "media.workbench.preview.player" && cap.TimeoutSeconds < 600 {
			t.Errorf("preview.player declares timeout_seconds=%d; first run needs >600s", cap.TimeoutSeconds)
		}
	}
}

func TestFallbackTemplatesMatchCatalog(t *testing.T) {
	var catalog []videoTemplate
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), "catalog", "templates.json"))
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatalf("parse catalog: %v", err)
	}
	fallback := loadTemplatesFromFallback()
	if len(fallback) != len(catalog) {
		t.Fatalf("fallback has %d templates, catalog has %d", len(fallback), len(catalog))
	}
	for i := range catalog {
		if fallback[i].ID != catalog[i].ID {
			t.Errorf("fallback[%d] = %s, catalog[%d] = %s", i, fallback[i].ID, i, catalog[i].ID)
		}
		if fallback[i].Composition != catalog[i].Composition {
			t.Errorf("%s: fallback composition %q != catalog %q", catalog[i].ID, fallback[i].Composition, catalog[i].Composition)
		}
	}
}

func TestRouteGrowthCompositionIsRegistered(t *testing.T) {
	renderer := filepath.Join(pluginRoot(), "renderer", "remotion")
	index, err := os.ReadFile(filepath.Join(renderer, "index.jsx"))
	if err != nil {
		t.Fatalf("read index.jsx: %v", err)
	}
	player, err := os.ReadFile(filepath.Join(renderer, "player.jsx"))
	if err != nil {
		t.Fatalf("read player.jsx: %v", err)
	}

	// 每个模板都必须声明 status，且 status 必须与"渲染器里有没有实现"一致。
	valid := map[string]bool{"ready": true, "placeholder": true, "planned": true}
	for _, item := range loadTemplates() {
		if !valid[item.Status] {
			t.Errorf("template %s has invalid status %q", item.ID, item.Status)
			continue
		}
		registered := strings.Contains(string(index), `id="`+item.Composition+`"`)
		if registered && item.Status == "planned" {
			t.Errorf("template %s is marked planned but composition %q is registered", item.ID, item.Composition)
		}
		if !registered && item.Status != "planned" {
			t.Errorf("template %s is marked %q but composition %q is not registered in index.jsx", item.ID, item.Status, item.Composition)
		}
	}

	// 预览与渲染必须共用同一份注册表 —— 参考实现用三元表达式硬编码映射，
	// 未知 composition 会静默退化成 LeaderboardTech。
	if !strings.Contains(string(index), "export const COMPOSITIONS") {
		t.Error("index.jsx must export a COMPOSITIONS registry")
	}
	if !strings.Contains(string(player), "COMPOSITIONS[") {
		t.Error("player.jsx must resolve the component through the registry")
	}
	if strings.Contains(string(player), "? MapStory\n    : LeaderboardTech") {
		t.Error("player.jsx still contains the silent-fallback ternary")
	}
}

func TestRendererSourcesIncludeSubdirectories(t *testing.T) {
	// 渲染器现在有 techniques/ 与 compositions/ 子目录；
	// 复制逻辑必须递归，否则工作区跑的是旧代码（表现是"改了没生效"）。
	dir := filepath.Join(pluginRoot(), "renderer", "remotion", "techniques")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("techniques/ must ship with the plugin: %v", err)
	}
	comps, err := os.ReadDir(filepath.Join(pluginRoot(), "renderer", "remotion", "compositions"))
	if err != nil || len(comps) == 0 {
		t.Fatalf("compositions/ must ship with the plugin: %v", err)
	}
}

// TestDefaultRecipeForRouteGrowthPassesInvariants 是这一轮扩展的"验收断言"：
// **新建项目时由扩展自己生成的配方，必须直接通过轨迹不变量校验。**
// 这一条把"产出合格的视频"从口号变成可执行的检查。
func TestDefaultRecipeForRouteGrowthPassesInvariants(t *testing.T) {
	recipe := defaultRecipe("route-growth", "tech-blue", nil)

	if recipe["template_id"] != "route-growth" {
		t.Fatalf("unexpected recipe template: %v", recipe["template_id"])
	}
	if recipe["adapter"] != "remotion" {
		t.Fatalf("route-growth must render on remotion, got %v", recipe["adapter"])
	}
	if _, ok := recipe["style_tokens"]; !ok {
		t.Error("recipe must carry style_tokens so the renderer can consume a style without code changes")
	}

	data, _ := recipe["data"].(map[string]any)
	if data == nil {
		t.Fatal("recipe.data is missing")
	}
	lines, _ := data["lines"].([]any)
	if len(lines) < 10 {
		t.Fatalf("expected the example dataset to populate recipe.data.lines, got %d", len(lines))
	}

	// 把配方里的线路还原成 route.* 入参，重跑不变量校验。
	routes := make([]routeInput, 0, len(lines))
	for _, raw := range lines {
		line, _ := raw.(map[string]any)
		if line == nil {
			continue
		}
		coords := toCoordSlice(line["coords"])
		waypoints := make([]waypointInput, 0)
		if stationList, ok := line["stations"].([]any); ok {
			for _, rawStation := range stationList {
				station, _ := rawStation.(map[string]any)
				if station == nil {
					continue
				}
				ll := toCoordSlice([]any{station["lonLat"]})
				if len(ll) != 1 {
					continue
				}
				name, _ := station["name"].(string)
				waypoints = append(waypoints, waypointInput{Name: name, LonLat: ll[0]})
			}
		}
		id, _ := line["id"].(string)
		routes = append(routes, routeInput{ID: id, Coords: coords, Waypoints: waypoints})
	}

	policy := defaultRoutePolicy()
	failures := 0
	for _, route := range routes {
		result := validateRoute(route, policy)
		if !result.OK {
			failures++
			t.Errorf("route %s in the generated recipe fails invariants:", route.ID)
			for _, f := range result.Findings {
				t.Errorf("  [%s] %s %s", f.Severity, f.Code, f.Message)
			}
		}
	}
	if failures > 0 {
		t.Fatalf("%d/%d routes in the generated recipe are not qualified", failures, len(routes))
	}
	t.Logf("recipe carries %d lines; all pass station-anchoring / ratio / spike / loop invariants", len(routes))
}

func toCoordSlice(raw any) [][2]float64 {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([][2]float64, 0, len(list))
	for _, entry := range list {
		pair, ok := entry.([]any)
		if !ok || len(pair) != 2 {
			continue
		}
		lon, lonOK := pair[0].(float64)
		lat, latOK := pair[1].(float64)
		if !lonOK || !latOK {
			continue
		}
		out = append(out, [2]float64{lon, lat})
	}
	return out
}
