package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// architecture_test.go —— 架构适应度函数（Architecture Fitness Functions）。
//
// 这些测试不验证"功能对不对"，而验证"架构有没有走样"：
//   1. 约束清单里的每条规则都必须指向一个真实存在的执行点
//   2. 能力必须落在 boundaries.json 声明的归属里；切换 stage 会强制完成拆分
//
// 它们的作用是把"重构"从"靠自觉"变成"机械步骤"。

// ── 约束清单 ──

func TestConstraintManifestRulesAreEnforced(t *testing.T) {
	manifest, err := loadConstraints()
	if err != nil {
		t.Fatalf("load constraints: %v", err)
	}
	if manifest.Schema != "himind.video.constraints.v1" {
		t.Errorf("unexpected schema: %s", manifest.Schema)
	}
	if len(manifest.Rules) == 0 {
		t.Fatal("constraint manifest is empty")
	}

	capabilities, external := declaredCapabilities()
	tests := declaredTestNames()
	seen := map[string]bool{}

	for _, rule := range manifest.Rules {
		if rule.Code == "" || rule.Message == "" || rule.Stage == "" {
			t.Errorf("rule %+v is missing code/message/stage", rule)
		}
		if seen[rule.Code] {
			t.Errorf("duplicate rule code: %s", rule.Code)
		}
		seen[rule.Code] = true

		switch rule.Level {
		case "error", "warn", "gate", "info":
		default:
			t.Errorf("%s: invalid level %q", rule.Code, rule.Level)
		}

		ok, detail := resolveEnforcement(rule, capabilities, external, tests)
		if !ok {
			t.Errorf("%s: enforcement point is not real (%s/%s): %s",
				rule.Code, rule.EnforcedBy.Kind, rule.EnforcedBy.ID, detail)
		}
	}

	// 至少要有覆盖各阶段的规则，避免清单退化成"只有软约束"
	stages := map[string]bool{}
	for _, rule := range manifest.Rules {
		stages[rule.Stage] = true
	}
	for _, required := range []string{"input", "plan", "series", "recipe", "geometry", "render", "output", "extension"} {
		if !stages[required] {
			t.Errorf("constraint manifest has no rule for stage %q", required)
		}
	}
	t.Logf("%d rules, all enforcement points resolve", len(manifest.Rules))
}

func TestConstraintListCapabilityReportsResolved(t *testing.T) {
	result, rpcErr := constraintList(requestInput{})
	if rpcErr != nil {
		t.Fatalf("constraint.list: %s", rpcErr.Message)
	}
	payload, _ := result.(map[string]any)
	summary, _ := payload["summary"].(map[string]any)
	if summary == nil {
		t.Fatal("missing summary")
	}
	unresolved, _ := payload["unresolved"].([]string)
	if len(unresolved) != 0 {
		t.Fatalf("constraint.list reports unresolved rules: %v", unresolved)
	}
	if summary["rules"].(int) == 0 {
		t.Fatal("constraint.list returned no rules")
	}
}

// ── 能力边界 ──

type boundaryManifest struct {
	Schema  string   `json:"schema"`
	Stage   string   `json:"stage"`
	Stages  []string `json:"stages"`
	Current struct {
		PluginID    string   `json:"plugin_id"`
		DisplayName string   `json:"display_name"`
		Owns        []string `json:"owns"`
	} `json:"current"`
	Planned []struct {
		Stage       string   `json:"stage"`
		PluginID    string   `json:"plugin_id"`
		DisplayName string   `json:"display_name"`
		Prefixes    []string `json:"prefixes"`
		Rationale   string   `json:"rationale"`
	} `json:"planned"`
}

func loadBoundaries(t *testing.T) boundaryManifest {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), "catalog", "boundaries.json"))
	if err != nil {
		t.Fatalf("read boundaries.json: %v", err)
	}
	var b boundaryManifest
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatalf("parse boundaries.json: %v", err)
	}
	if b.Schema != "himind.video.boundaries.v1" {
		t.Fatalf("unexpected boundaries schema: %s", b.Schema)
	}
	return b
}

func stageIndex(stages []string, value string) int {
	for i, s := range stages {
		if s == value {
			return i
		}
	}
	return -1
}

func hasPrefix(id string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(id, p) {
			return true
		}
	}
	return false
}

// TestCapabilityBoundaries 是边界适应度函数。
//
// 当前 stage=p0（尚未拆分）：所有能力都应属于当前插件声明的前缀。
// 把 boundaries.json 的 stage 改成 p1/p2 后，本测试会立即要求把
// render.* / artifact.* / runtime.* / route.* 搬出创作台 ——
// 重构变成机械步骤，而不是靠自觉。
func TestCapabilityBoundaries(t *testing.T) {
	b := loadBoundaries(t)
	capabilities := declaredCapabilityIDs()
	if len(capabilities) == 0 {
		t.Fatal("no capabilities declared")
	}

	stageAt := stageIndex(b.Stages, b.Stage)
	if stageAt < 0 {
		t.Fatalf("unknown stage %q", b.Stage)
	}

	// 1. 所有能力必须落在当前插件声明的归属前缀内
	for id := range capabilities {
		if !hasPrefix(id, b.Current.Owns) {
			t.Errorf("capability %q is outside the declared ownership of %s (%v)",
				id, b.Current.PluginID, b.Current.Owns)
		}
	}

	// 2. 已经"到期"的拆分计划，必须真的拆走
	for _, plan := range b.Planned {
		planAt := stageIndex(b.Stages, plan.Stage)
		if planAt < 0 {
			t.Errorf("planned entry %s has unknown stage %q", plan.PluginID, plan.Stage)
			continue
		}
		if planAt > stageAt {
			continue // 尚未到期
		}
		for id := range capabilities {
			if hasPrefix(id, plan.Prefixes) {
				t.Errorf("stage is %s but %q still lives in %s; it must move to %s (%v)",
					b.Stage, id, b.Current.PluginID, plan.PluginID, plan.Prefixes)
			}
		}
		if plan.Rationale == "" {
			t.Errorf("planned split %s is missing a rationale", plan.PluginID)
		}
	}

	// 3. 显示名必须与重新定位一致
	var manifest struct {
		Name string `json:"name"`
	}
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), "plugin.json"))
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse plugin.json: %v", err)
	}
	if manifest.Name != b.Current.DisplayName {
		t.Errorf("plugin display name %q does not match boundaries.json (%q)",
			manifest.Name, b.Current.DisplayName)
	}
}

func TestConstraintEvidenceCoversPackagedTests(t *testing.T) {
	manifest, err := loadConstraints()
	if err != nil {
		t.Fatalf("load constraints: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), "catalog", "constraint-evidence.json"))
	if err != nil {
		t.Fatalf("read constraint evidence: %v", err)
	}
	var evidence constraintEvidence
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatalf("parse constraint evidence: %v", err)
	}
	if evidence.Schema != "himind.video.constraint-evidence.v1" {
		t.Fatalf("unexpected evidence schema: %s", evidence.Schema)
	}
	available := map[string]bool{}
	for _, name := range evidence.Tests {
		available[name] = true
	}
	for _, rule := range manifest.Rules {
		if rule.EnforcedBy.Kind != "test" {
			continue
		}
		if !available[rule.EnforcedBy.ID] {
			t.Errorf("%s: packaged evidence is missing %s", rule.Code, rule.EnforcedBy.ID)
		}
	}
}

func TestExternalCapabilityPatterns(t *testing.T) {
	patterns := []string{"audio.*", "visual.reference.validate", "media.job.*"}
	cases := map[string]bool{
		"audio.qa.sync":                true,
		"visual.reference.validate":    true,
		"visual.image.qa":              false,
		"media.job.get":                true,
		"media.workbench.render.start": false,
		"system.health":                false,
	}
	for id, want := range cases {
		if got := matchesExternalCapability(id, patterns); got != want {
			t.Errorf("matchesExternalCapability(%q) = %v, want %v", id, got, want)
		}
	}
}
