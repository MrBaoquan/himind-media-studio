package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// capability_catalog.go — 可复用视觉能力的声明、查询与审计。
//
// 一个能力不是"一个 JSX 文件"，而是 Shot + Technique + Contract + Template +
// Fixture + Metadata。这里把这六项变成可被 AI 查询和机械审计的资产，避免
// "写了一条视频"被误当成"沉淀了一个能力"。

const (
	capabilitiesRelDir = "catalog/capabilities"
	capabilitySchema   = "himind.video.capability-descriptor.v1"
)

type capabilityTemplateRef struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	Composition  string `json:"composition"`
	DataContract string `json:"data_contract"`
}

type capabilitySources struct {
	Shot       string   `json:"shot"`
	Registry   string   `json:"registry"`
	Techniques []string `json:"techniques"`
}

type capabilityFixture struct {
	ManifestID string `json:"manifest_id"`
	Recipe     string `json:"recipe"`
	Frames     []int  `json:"frames"`
}

type videoCapability struct {
	Schema     string                `json:"schema"`
	ID         string                `json:"id"`
	Version    string                `json:"version"`
	Name       string                `json:"name"`
	Status     string                `json:"status"`
	Structure  string                `json:"structure"`
	Summary    string                `json:"summary"`
	Tags       []string              `json:"tags,omitempty"`
	ReuseFor   []string              `json:"reuse_for"`
	Template   capabilityTemplateRef `json:"template"`
	Sources    capabilitySources     `json:"sources"`
	Fixture    capabilityFixture     `json:"fixture"`
	Acceptance []string              `json:"acceptance"`
	Evidence   map[string]any        `json:"evidence,omitempty"`
	Authoring  map[string]string     `json:"authoring,omitempty"`
	CreatedAt  string                `json:"created_at,omitempty"`
	UpdatedAt  string                `json:"updated_at,omitempty"`
}

type capabilityCheck struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

type capabilityIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

func loadCapabilities() ([]videoCapability, error) {
	dir := filepath.Join(pluginRoot(), filepath.FromSlash(capabilitiesRelDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []videoCapability{}, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	items := make([]videoCapability, 0, len(names))
	for _, name := range names {
		var item videoCapability
		if err := readJSON(filepath.Join(dir, name), &item); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func findCapability(id, version string) (videoCapability, bool) {
	items, err := loadCapabilities()
	if err != nil {
		return videoCapability{}, false
	}
	for _, item := range items {
		if item.ID == id && (version == "" || item.Version == version) {
			return item, true
		}
	}
	return videoCapability{}, false
}

func capabilityList(in requestInput) (any, *himindjsonrpc.Error) {
	items, err := loadCapabilities()
	if err != nil {
		return nil, himindjsonrpc.InternalError("无法读取能力目录: " + err.Error())
	}
	filtered := make([]videoCapability, 0, len(items))
	for _, item := range items {
		if in.Structure != "" && item.Structure != in.Structure {
			continue
		}
		if in.Status != "" && item.Status != in.Status {
			continue
		}
		if in.Family != "" && item.Structure != in.Family {
			continue
		}
		filtered = append(filtered, item)
	}
	return map[string]any{
		"capabilities": filtered,
		"total":        len(filtered),
		"note":         "只有 status=ready 的能力允许直接用于正式成片；candidate/validated 需继续通过门禁。",
	}, nil
}

func capabilityDescribe(in requestInput) (any, *himindjsonrpc.Error) {
	id := strings.TrimSpace(in.CapabilityID)
	if id == "" {
		return nil, himindjsonrpc.InvalidParams("capability_id is required")
	}
	item, ok := findCapability(id, strings.TrimSpace(in.Version))
	if !ok {
		return nil, himindjsonrpc.InvalidParams("capability_id or version not found")
	}
	return map[string]any{"capability": item, "immutable": true}, nil
}

// capabilityAudit 是"能力沉淀"的机械门禁：描述里声明的每个文件都必须真实存在，
// 且 Template、Contract、Fixture、Renderer 注册之间必须互相一致。
func capabilityAudit(in requestInput) (any, *himindjsonrpc.Error) {
	id := strings.TrimSpace(in.CapabilityID)
	if id == "" {
		return nil, himindjsonrpc.InvalidParams("capability_id is required")
	}
	item, ok := findCapability(id, strings.TrimSpace(in.Version))
	if !ok {
		return nil, himindjsonrpc.InvalidParams("capability_id or version not found")
	}

	checks := make([]capabilityCheck, 0, 12)
	issues := make([]capabilityIssue, 0)
	check := func(id string, ok bool, detail string, code, message, hint string) {
		checks = append(checks, capabilityCheck{ID: id, OK: ok, Detail: detail})
		if !ok {
			issues = append(issues, capabilityIssue{Code: code, Message: message, Hint: hint})
		}
	}

	check("schema", item.Schema == capabilitySchema, item.Schema, "capability_schema_invalid",
		"能力描述 schema 不是 "+capabilitySchema, "使用 schemas/capability-descriptor.schema.json")

	template, templateOK := findTemplate(item.Template.ID)
	check("template", templateOK, item.Template.ID, "capability_template_missing",
		"模板不存在: "+item.Template.ID, "先注册 catalog/templates.json")
	if templateOK {
		check("template_version", item.Template.Version == "" || template.Version == item.Template.Version,
			item.Template.Version+" -> "+template.Version, "capability_template_version_mismatch",
			"能力描述与模板版本不一致", "同步 template.version 和 descriptor.template.version")
		check("composition", template.Composition == item.Template.Composition,
			item.Template.Composition+" -> "+template.Composition, "capability_composition_mismatch",
			"能力描述与模板 Composition 不一致", "同步 descriptor.template.composition")
		check("data_contract", template.DataContract == item.Template.DataContract,
			item.Template.DataContract+" -> "+template.DataContract, "capability_contract_mismatch",
			"能力描述与模板 data_contract 不一致", "同步 descriptor.template.data_contract")
	}

	indexPath := filepath.Join(pluginRoot(), "renderer", "remotion", "index.jsx")
	indexRaw, indexErr := os.ReadFile(indexPath)
	indexOK := indexErr == nil && strings.Contains(string(indexRaw), `id="`+item.Template.Composition+`"`)
	check("index_registration", indexOK, item.Template.Composition, "capability_not_registered",
		"Composition 未注册到 renderer/remotion/index.jsx", "在 index.jsx 中注册 <Composition id=\""+item.Template.Composition+"\">")

	shotPath, shotErr := pluginRelativeFile(item.Sources.Shot)
	shotOK := shotErr == nil
	if shotOK {
		_, statErr := os.Stat(shotPath)
		shotOK = statErr == nil
	}
	check("shot_source", shotOK, item.Sources.Shot, "capability_shot_missing",
		"Shot 源文件不存在", "检查 descriptor.sources.shot")

	registryPath, registryErr := pluginRelativeFile(item.Sources.Registry)
	registryOK := registryErr == nil
	if registryOK {
		raw, readErr := os.ReadFile(registryPath)
		registryOK = readErr == nil && strings.Contains(string(raw), item.Template.Composition)
	}
	check("shot_registry", registryOK, item.Sources.Registry, "capability_registry_missing",
		"Shot 未注册到镜头注册表", "在 renderer/remotion/shots/registry.js 中注册模板 id 与组件")

	for _, technique := range item.Sources.Techniques {
		path, pathErr := pluginRelativeFile(technique)
		ok := pathErr == nil
		if ok {
			_, statErr := os.Stat(path)
			ok = statErr == nil
		}
		check("technique:"+technique, ok, technique, "capability_technique_missing",
			"Technique 源文件不存在: "+technique, "检查 descriptor.sources.techniques")
	}

	contractPath, contractErr := pluginRelativeFile(item.Template.DataContract)
	contractOK := contractErr == nil
	if contractOK {
		_, statErr := os.Stat(contractPath)
		contractOK = statErr == nil
	}
	check("contract_file", contractOK, item.Template.DataContract, "capability_contract_file_missing",
		"数据契约文件不存在", "先写 catalog/contracts/<capability>.schema.json")

	fixture, fixtureOK := findFixture(item.Fixture.ManifestID)
	check("fixture_manifest", fixtureOK, item.Fixture.ManifestID, "capability_fixture_missing",
		"Fixture 未登记到 manifest.json", "在 catalog/fixtures/manifest.json 登记配方与基线帧")
	if fixtureOK {
		check("fixture_template", fixture.Template == item.Template.ID,
			fixture.Template+" -> "+item.Template.ID, "capability_fixture_template_mismatch",
			"Fixture 绑定的模板与能力不一致", "同步 fixture.template")
		check("fixture_composition", fixture.Composition == item.Template.Composition,
			fixture.Composition+" -> "+item.Template.Composition, "capability_fixture_composition_mismatch",
			"Fixture 绑定的 Composition 与能力不一致", "同步 fixture.composition")
		check("fixture_recipe", fixture.Recipe == item.Fixture.Recipe,
			fixture.Recipe+" -> "+item.Fixture.Recipe, "capability_fixture_recipe_mismatch",
			"Fixture 配方路径与能力描述不一致", "同步 descriptor.fixture.recipe")
		check("fixture_frames", equalInts(fixture.Frames, item.Fixture.Frames),
			fmt.Sprintf("%v -> %v", fixture.Frames, item.Fixture.Frames), "capability_fixture_frames_mismatch",
			"Fixture 基线帧与能力描述不一致", "同步 descriptor.fixture.frames")
	}

	recipePath, recipeErr := pluginRelativeFile(filepath.ToSlash(filepath.Join("catalog", "fixtures", item.Fixture.Recipe)))
	recipeOK := recipeErr == nil
	if recipeOK {
		_, statErr := os.Stat(recipePath)
		recipeOK = statErr == nil
	}
	check("fixture_recipe_file", recipeOK, item.Fixture.Recipe, "capability_fixture_recipe_file_missing",
		"Fixture 配方文件不存在", "检查 descriptor.fixture.recipe")

	for _, frame := range item.Fixture.Frames {
		name := fmt.Sprintf("frame-%04d.png", frame)
		path, pathErr := pluginRelativeFile(filepath.ToSlash(filepath.Join("catalog", "fixtures", filepath.Dir(item.Fixture.Recipe), name)))
		ok := pathErr == nil
		if ok {
			_, statErr := os.Stat(path)
			ok = statErr == nil
		}
		check("fixture_frame:"+name, ok, name, "capability_fixture_frame_missing",
			"Fixture 基线帧不存在: "+name, "运行 dev/regress-stills.ps1 -Update 生成基线")
	}

	if item.Status == "ready" {
		check("acceptance", len(item.Acceptance) > 0, fmt.Sprintf("%d items", len(item.Acceptance)),
			"capability_acceptance_missing", "ready 能力必须有验收标准", "补充 descriptor.acceptance")
	}

	return map[string]any{
		"valid":      len(issues) == 0,
		"capability": item,
		"checks":     checks,
		"issues":     issues,
		"note":       "valid=false 时不得把能力标记为 ready。",
	}, nil
}

// pluginRelativeFile 拒绝绝对路径和越界路径，保证描述文件不能指向插件之外。
func pluginRelativeFile(rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("empty path")
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes plugin root: %s", rel)
	}
	root := filepath.Clean(pluginRoot())
	target := filepath.Join(root, clean)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes plugin root: %s", rel)
	}
	return target, nil
}

type fixtureEntry struct {
	ID          string
	Template    string
	Composition string
	Recipe      string
	Frames      []int
}

type fixtureManifestFile struct {
	Schema   string `json:"schema"`
	Fixtures []struct {
		ID          string `json:"id"`
		Template    string `json:"template"`
		Composition string `json:"composition"`
		Recipe      string `json:"recipe"`
		Frames      []int  `json:"frames"`
	} `json:"fixtures"`
}

func findFixture(id string) (fixtureEntry, bool) {
	var manifest fixtureManifestFile
	if err := readJSON(filepath.Join(pluginRoot(), "catalog", "fixtures", "manifest.json"), &manifest); err != nil {
		return fixtureEntry{}, false
	}
	for _, item := range manifest.Fixtures {
		if item.ID == id {
			return fixtureEntry{
				ID:          item.ID,
				Template:    item.Template,
				Composition: item.Composition,
				Recipe:      item.Recipe,
				Frames:      item.Frames,
			}, true
		}
	}
	return fixtureEntry{}, false
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
