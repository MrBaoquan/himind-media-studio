package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// constraints.go — Harness 约束清单的读取与**执行点解析**。
//
// 设计原则：每条约束都必须指向一个可执行的执行点；找不到执行点的约束
// 不写进清单。否则 harness 会退化成"AGENTS.md 里的一段劝导"。
//
// 这个能力让 AI 能直接查询"我被什么约束着、每条由谁执行"。

type constraintEnforcement struct {
	Kind string `json:"kind"` // capability | schema | workflow | test | runtime | script
	ID   string `json:"id"`
}

type constraintRule struct {
	Code       string                `json:"code"`
	Level      string                `json:"level"` // error | warn | gate | info
	Stage      string                `json:"stage"`
	Message    string                `json:"message"`
	EnforcedBy constraintEnforcement `json:"enforced_by"`
}

type constraintManifest struct {
	Schema      string           `json:"schema"`
	ID          string           `json:"id"`
	Version     string           `json:"version"`
	Description string           `json:"description"`
	Note        string           `json:"note"`
	Rules       []constraintRule `json:"rules"`
}

type resolvedConstraint struct {
	constraintRule
	Resolved bool   `json:"resolved"`
	Detail   string `json:"detail,omitempty"`
}

const constraintsRelPath = "catalog/constraints.json"
const constraintEvidenceRelPath = "catalog/constraint-evidence.json"

// 允许的强制类型。新增类型必须同时更新这里的白名单与解析逻辑。
var enforcementKinds = map[string]bool{
	"capability": true,
	"schema":     true,
	"workflow":   true,
	"test":       true,
	"runtime":    true,
	"script":     true,
}

// 由内核或宿主环境强制、无法在本仓内自证的执行点。
var runtimeEnforcements = map[string]string{
	"capability.input_schema": "内核在调用前按 input_schema 校验并拒绝非法入参",
	"workspace_file_guard":    "插件所有写入经 workspaceFile 校验，禁止越出 workspace_root",
	"approval_center":         "内核审批中心：risk_level != read_only 的插件能力自动需要人工确认",
}

func loadConstraints() (constraintManifest, error) {
	var manifest constraintManifest
	content, err := os.ReadFile(filepath.Join(pluginRoot(), filepath.FromSlash(constraintsRelPath)))
	if err != nil {
		return manifest, err
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		return manifest, err
	}
	return manifest, nil
}

// declaredCapabilityIDs 从 plugin.json 读取本插件能力 ID 集合。
func declaredCapabilityIDs() map[string]bool {
	ids, _ := declaredCapabilities()
	return ids
}

// declaredCapabilities 同时返回本插件能力与显式声明的外部可选能力模式
// （external_capabilities，支持精确 ID 或尾部 .*）。
func declaredCapabilities() (map[string]bool, []string) {
	ids := map[string]bool{}
	external := []string{}
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), "plugin.json"))
	if err != nil {
		return ids, external
	}
	var manifest struct {
		Capabilities []struct {
			ID string `json:"id"`
		} `json:"capabilities"`
		ExternalCapabilities []string `json:"external_capabilities"`
		Contributes          struct {
			ExternalCapabilities []string `json:"external_capabilities"`
			Views                []struct {
				ExternalCapabilities []string `json:"external_capabilities"`
			} `json:"views"`
		} `json:"contributes"`
	}
	if json.Unmarshal(raw, &manifest) != nil {
		return ids, external
	}
	for _, cap := range manifest.Capabilities {
		ids[cap.ID] = true
	}
	external = append(external, manifest.ExternalCapabilities...)
	external = append(external, manifest.Contributes.ExternalCapabilities...)
	for _, view := range manifest.Contributes.Views {
		external = append(external, view.ExternalCapabilities...)
	}
	return ids, external
}

func matchesExternalCapability(id string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if pattern == id {
			return true
		}
		if strings.HasSuffix(pattern, ".*") && strings.HasPrefix(id, strings.TrimSuffix(pattern, "*")) {
			return true
		}
	}
	return false
}

var testFuncPattern = regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\(`)

// declaredTestNames 扫描包内的 *_test.go，收集测试函数名。
func declaredTestNames() map[string]bool {
	names := map[string]bool{}
	entries, err := os.ReadDir(pluginRoot())
	if err != nil {
		return names
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		content, readErr := os.ReadFile(filepath.Join(pluginRoot(), entry.Name()))
		if readErr != nil {
			continue
		}
		for _, match := range testFuncPattern.FindAllStringSubmatch(string(content), -1) {
			names[match[1]] = true
		}
	}
	return names
}

type constraintEvidence struct {
	Schema string   `json:"schema"`
	Tests  []string `json:"tests"`
}

// declaredPackagedTestNames adds the release evidence manifest. The plugin
// package intentionally excludes Go source and *_test.go files, so source
// scanning alone would make the installed UI report eleven false "unresolved"
// constraints. Evidence is generated only from tests that run in the source
// tree; TestConstraintEvidenceCoversPackagedTests keeps the two in sync.
func declaredPackagedTestNames() map[string]bool {
	names := declaredTestNames()
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), filepath.FromSlash(constraintEvidenceRelPath)))
	if err != nil {
		return names
	}
	var evidence constraintEvidence
	if json.Unmarshal(raw, &evidence) != nil || evidence.Schema != "himind.video.constraint-evidence.v1" {
		return names
	}
	for _, name := range evidence.Tests {
		if strings.TrimSpace(name) != "" {
			names[strings.TrimSpace(name)] = true
		}
	}
	return names
}

// resolveEnforcement 判断某条规则的执行点是否真实存在。
func resolveEnforcement(rule constraintRule, capabilities map[string]bool, external []string, tests map[string]bool) (bool, string) {
	if !enforcementKinds[rule.EnforcedBy.Kind] {
		return false, fmt.Sprintf("未知的 enforced_by.kind: %q", rule.EnforcedBy.Kind)
	}
	switch rule.EnforcedBy.Kind {
	case "capability":
		if capabilities[rule.EnforcedBy.ID] {
			return true, "能力已声明"
		}
		if matchesExternalCapability(rule.EnforcedBy.ID, external) {
			return true, "外部可选能力：未安装时该约束降级为警告"
		}
		return false, "能力未在 plugin.json 中声明"
	case "test":
		if tests[rule.EnforcedBy.ID] {
			return true, "测试存在"
		}
		return false, "找不到该测试函数"
	case "script":
		path := filepath.Join(pluginRoot(), filepath.FromSlash(rule.EnforcedBy.ID))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return true, "脚本存在"
		}
		return false, "找不到该脚本"
	case "runtime":
		if detail, ok := runtimeEnforcements[rule.EnforcedBy.ID]; ok {
			return true, detail
		}
		return false, "未登记的运行时执行点"
	case "schema":
		if rule.EnforcedBy.ID == "capability.input_schema" || strings.HasSuffix(rule.EnforcedBy.ID, ".schema.json") {
			return true, "由 JSON Schema 校验"
		}
		return false, "未识别的 schema 执行点"
	case "workflow":
		return false, "本仓暂无 workflow 包；该规则需在工作流落地后启用"
	}
	return false, "未处理的执行点类型"
}

// constraintList 返回约束清单及每条规则的执行点解析结果。
func constraintList(in requestInput) (any, *himindjsonrpc.Error) {
	manifest, err := loadConstraints()
	if err != nil {
		return nil, himindjsonrpc.InternalError("无法读取约束清单: " + err.Error())
	}
	capabilities, external := declaredCapabilities()
	tests := declaredPackagedTestNames()

	resolved := make([]resolvedConstraint, 0, len(manifest.Rules))
	unresolved := make([]string, 0)
	byLevel := map[string]int{}
	byKind := map[string]int{}
	for _, rule := range manifest.Rules {
		ok, detail := resolveEnforcement(rule, capabilities, external, tests)
		byLevel[rule.Level]++
		byKind[rule.EnforcedBy.Kind]++
		if !ok {
			unresolved = append(unresolved, rule.Code)
		}
		resolved = append(resolved, resolvedConstraint{constraintRule: rule, Resolved: ok, Detail: detail})
	}
	sort.Strings(unresolved)

	// stage 过滤：只返回某一阶段的规则
	if strings.TrimSpace(in.Family) != "" {
		stage := strings.TrimSpace(in.Family)
		filtered := make([]resolvedConstraint, 0, len(resolved))
		for _, item := range resolved {
			if item.Stage == stage {
				filtered = append(filtered, item)
			}
		}
		resolved = filtered
	}

	return map[string]any{
		"constraints": manifest,
		"resolved":    resolved,
		"summary": map[string]any{
			"rules":      len(manifest.Rules),
			"resolved":   len(manifest.Rules) - len(unresolved),
			"unresolved": len(unresolved),
			"by_level":   byLevel,
			"by_kind":    byKind,
		},
		"unresolved": unresolved,
		"note":       "unresolved 非空说明清单里存在无法自证的约束，应修复或删除该条，而不是忽略。",
	}, nil
}
