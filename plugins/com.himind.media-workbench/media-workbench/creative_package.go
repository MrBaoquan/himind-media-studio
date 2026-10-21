package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// creative_package.go — 创意开发阶段。
//
// CreativePackage 位于 Plan/Recipe 之上：它保存 Brief、Research、Script、
// Storyboard 和 ProductionSpec，审批通过后再 materialize 成创作台项目。
// 这一层不渲染、不执行 LLM；AI 负责产出内容，插件负责持久化与门禁。

const creativePackageVersion = "himind.video.creative-package.v1"

type creativePackage struct {
	Version    string         `json:"version"`
	ID         string         `json:"id"`
	Revision   int            `json:"revision"`
	Status     string         `json:"status"`
	Title      string         `json:"title"`
	Brief      map[string]any `json:"brief"`
	Research   map[string]any `json:"research"`
	Script     map[string]any `json:"script"`
	Storyboard map[string]any `json:"storyboard"`
	Production map[string]any `json:"production"`
	Approval   map[string]any `json:"approval"`
	Links      map[string]any `json:"links"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

func creativeRoot(root, id string) string {
	return filepath.Join(root, metadataDir, creativeDir, id)
}

func loadCreative(root, id string) (creativePackage, error) {
	if !strings.HasPrefix(id, "creative-") || !idPattern.MatchString(id) {
		return creativePackage{}, fmt.Errorf("invalid creative_id")
	}
	var item creativePackage
	if err := readJSON(filepath.Join(creativeRoot(root, id), "creative.json"), &item); err != nil {
		return creativePackage{}, fmt.Errorf("creative not found: %s", id)
	}
	return item, nil
}

func writeCreative(root string, item *creativePackage) error {
	dir := creativeRoot(root, item.ID)
	if err := os.MkdirAll(filepath.Join(dir, "revisions"), 0755); err != nil {
		return err
	}
	item.UpdatedAt = now()
	if item.CreatedAt == "" {
		item.CreatedAt = item.UpdatedAt
	}
	if err := writeJSON(filepath.Join(dir, "creative.json"), item); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, "revisions", fmt.Sprintf("r%04d.json", item.Revision)), item); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "brief.md"), []byte(creativeBriefMarkdown(*item)), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "script.md"), []byte(creativeScriptMarkdown(*item)), 0644); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, "storyboard.json"), item.Storyboard)
}

func creativeCreate(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	brief := in.Brief
	if brief == nil {
		brief = map[string]any{}
		if strings.TrimSpace(in.CreativeBrief) != "" {
			brief["goal"] = strings.TrimSpace(in.CreativeBrief)
		}
	}
	brief, _ = cloneMap(brief)
	if strings.TrimSpace(asString(brief["goal"])) == "" {
		return nil, himindjsonrpc.InvalidParams("brief.goal or creative_brief is required")
	}
	title := firstNonEmpty(in.Title, asString(brief["title"]), in.Name, "未命名创意")
	brief["title"] = title
	id := newID("creative")
	item := creativePackage{
		Version: creativePackageVersion, ID: id, Revision: 1, Status: "draft", Title: title,
		Brief:      brief,
		Research:   map[string]any{"mode": "none", "claims": []any{}, "sources": []any{}, "uncertainties": []any{}},
		Script:     map[string]any{"logline": "", "beats": []any{}},
		Storyboard: map[string]any{"shots": []any{}},
		Production: map[string]any{},
		Approval:   map[string]any{"approved_revision": 0},
		Links:      map[string]any{"project_id": "", "materialized_revision": 0},
		CreatedAt:  now(),
	}
	if err := writeCreative(root, &item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"creative": item, "workspace_root": root, "path": creativeRoot(root, id), "next": []string{"调用 creative.update 写入 research/script/storyboard/production", "调用 creative.validate"}}, nil
}

func creativeList(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	items := make([]map[string]any, 0)
	entries, readErr := os.ReadDir(filepath.Join(root, metadataDir, creativeDir))
	if readErr != nil && !os.IsNotExist(readErr) {
		return nil, himindjsonrpc.InternalError(readErr.Error())
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "creative-") {
			continue
		}
		item, loadErr := loadCreative(root, entry.Name())
		if loadErr != nil {
			continue
		}
		if strings.TrimSpace(in.Status) != "" && item.Status != in.Status {
			continue
		}
		items = append(items, map[string]any{
			"id": item.ID, "title": item.Title, "status": item.Status, "revision": item.Revision,
			"updated_at": item.UpdatedAt, "production": item.Production,
			"approved_revision": asInt(item.Approval["approved_revision"]),
			"project_id":        asString(item.Links["project_id"]),
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return asString(items[i]["updated_at"]) > asString(items[j]["updated_at"])
	})
	return map[string]any{"items": items, "total": len(items), "workspace_root": root}, nil
}

func creativeGet(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadCreative(root, in.CreativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	issues := validateCreative(root, item)
	return map[string]any{"creative": item, "validation": map[string]any{"valid": len(issues) == 0, "issues": issues}, "path": creativeRoot(root, item.ID)}, nil
}

func creativeUpdate(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadCreative(root, in.CreativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if expected := strings.TrimSpace(in.ExpectedUpdatedAt); expected != "" && expected != item.UpdatedAt {
		return nil, himindjsonrpc.InvalidParams("creative was changed; refresh creative.get before updating")
	}
	section := strings.ToLower(strings.TrimSpace(in.Section))
	if section == "" || in.Value == nil {
		return nil, himindjsonrpc.InvalidParams("section and value are required")
	}
	value, cloneErr := cloneMap(in.Value)
	if cloneErr != nil {
		return nil, himindjsonrpc.InvalidParams(cloneErr.Error())
	}
	switch section {
	case "brief":
		item.Brief = value
	case "research":
		item.Research = value
	case "script":
		item.Script = value
	case "storyboard":
		item.Storyboard = value
	case "production":
		item.Production = value
	default:
		return nil, himindjsonrpc.InvalidParams("section must be brief, research, script, storyboard or production")
	}
	item.Revision++
	item.Status = "draft"
	item.Approval = map[string]any{"approved_revision": 0}
	if err := writeCreative(root, &item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"creative": item, "next": []string{"调用 creative.validate", "通过后调用 creative.approve"}}, nil
}

func creativeValidate(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadCreative(root, in.CreativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	issues := validateCreative(root, item)
	return map[string]any{
		"valid": len(issues) == 0, "issues": issues, "creative_id": item.ID,
		"revision": item.Revision, "summary": creativeSummary(item),
		"next": "valid=true 后调用 creative.approve；批准后用 creative.materialize 创建项目",
	}, nil
}

func creativeApprove(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadCreative(root, in.CreativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	issues := validateCreative(root, item)
	if len(issues) > 0 {
		return map[string]any{"state": "blocked", "blockers": issues, "retryable": true}, nil
	}
	item.Status = "approved"
	item.Approval = map[string]any{
		"approved_revision": item.Revision,
		"approved_at":       now(),
		"approved_by":       "user",
		"note":              strings.TrimSpace(in.Note),
	}
	if err := writeCreative(root, &item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"state": "approved", "creative": item, "next": "调用 creative.materialize 创建项目"}, nil
}

func creativeMaterialize(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadCreative(root, in.CreativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	approved := asInt(item.Approval["approved_revision"])
	if item.Status != "approved" || approved != item.Revision {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("CREATIVE_NOT_APPROVED", "creative", "创作包尚未批准当前 revision", "先调用 creative.approve")}, "retryable": true}, nil
	}
	issues := validateCreative(root, item)
	if len(issues) > 0 {
		return map[string]any{"state": "blocked", "blockers": issues, "retryable": true}, nil
	}
	mode := strings.ToLower(strings.TrimSpace(asString(item.Production["mode"])))
	if mode == "" {
		if strings.TrimSpace(asString(item.Production["style_pack"])) != "" {
			mode = "code"
		} else {
			mode = "template"
		}
	}
	name := firstNonEmpty(in.Name, item.Title)
	switch mode {
	case "code":
		return materializeCreativeCode(root, item, name)
	case "template":
		return materializeCreativeTemplate(root, item, name)
	default:
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("CREATIVE_MODE_INVALID", "production", "production.mode 必须是 code 或 template", "修复 production.mode")}, "retryable": true}, nil
	}
}

func materializeCreativeCode(root string, item creativePackage, name string) (any, *himindjsonrpc.Error) {
	packID := strings.TrimSpace(asString(item.Production["style_pack"]))
	version := strings.TrimSpace(asString(item.Production["style_version"]))
	if packID == "" {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("CREATIVE_STYLE_PACK_MISSING", "production", "code 模式缺少 production.style_pack", "补充 style_pack")}, "retryable": true}, nil
	}
	scaffold, rpcErr := stylePackScaffold(requestInput{
		WorkspaceRoot: root, PackID: packID, Version: version, Name: name,
		TargetDir: projectIDForName(name), Description: item.Title,
		CreativeBrief: creativeBriefMarkdown(item),
	})
	if rpcErr != nil {
		return nil, rpcErr
	}
	payload, ok := scaffold.(map[string]any)
	if !ok {
		return nil, himindjsonrpc.InternalError("stylepack.scaffold returned invalid payload")
	}
	projectItem, ok := payload["project"].(project)
	if !ok {
		return nil, himindjsonrpc.InternalError("stylepack.scaffold did not return a project")
	}
	data := asMap(item.Production["data"])
	if len(data) == 0 {
		data = map[string]any{"title": item.Title, "source": asString(asMap(item.Brief)["source"]), "scenes": item.Storyboard["shots"]}
	}
	if _, dataErr := projectDataWrite(requestInput{WorkspaceRoot: root, ProjectID: projectItem.ID, Data: data, Brief: item.Brief}); dataErr != nil {
		return nil, dataErr
	}
	updated, err := loadProject(root, projectItem.ID)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	updated.SourceCreativeID = item.ID
	updated.SourceCreativeRevision = item.Revision
	if err := writeJSON(filepath.Join(root, metadataDir, projectDir, updated.ID, "project.json"), updated); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	item.Status = "materialized"
	item.Links = map[string]any{"project_id": updated.ID, "materialized_revision": item.Revision}
	if err := writeCreative(root, &item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{
		"state": "materialized", "creative": item, "project": updated, "workspace_root": root,
		"next": []string{"调用 preview.review", "根据审查图记录反馈", "通过 preview.determinism 后 render.start"},
	}, nil
}

func materializeCreativeTemplate(root string, item creativePackage, name string) (any, *himindjsonrpc.Error) {
	plan := creativeToPlan(item)
	preview := false
	result, rpcErr := generateRun(requestInput{
		WorkspaceRoot: root, Name: name, Plan: plan,
		Data: asMap(item.Production["data"]), Preview: &preview,
		CreativeBrief: creativeBriefMarkdown(item),
	})
	if rpcErr != nil {
		return nil, rpcErr
	}
	payload, ok := result.(map[string]any)
	if !ok {
		return nil, himindjsonrpc.InternalError("generate.run returned invalid payload")
	}
	projectItem, ok := payload["project"].(project)
	if !ok {
		return nil, himindjsonrpc.InternalError("generate.run did not return a project")
	}
	updated, err := loadProject(root, projectItem.ID)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	updated.SourceCreativeID = item.ID
	updated.SourceCreativeRevision = item.Revision
	if err := writeJSON(filepath.Join(root, metadataDir, projectDir, updated.ID, "project.json"), updated); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	item.Status = "materialized"
	item.Links = map[string]any{"project_id": updated.ID, "materialized_revision": item.Revision}
	if err := writeCreative(root, &item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"state": "materialized", "creative": item, "project": updated, "plan": plan, "next": "调用 preview.start 或 preview.player"}, nil
}

func creativeToPlan(item creativePackage) map[string]any {
	brief := cloneWithoutError(item.Brief)
	if _, ok := brief["goal"]; !ok {
		brief["goal"] = item.Title
	}
	narrative := map[string]any{
		"thesis":        firstNonEmpty(asString(asMap(item.Script)["logline"]), asString(brief["goal"])),
		"structure":     firstNonEmpty(asString(asMap(item.Script)["structure"]), "hook -> insight -> conclusion"),
		"required_data": item.Production["required_data"],
		"constraints":   brief["constraints"],
	}
	if claims := item.Research["claims"]; claims != nil {
		narrative["key_points"] = claims
	}
	presentation := map[string]any{
		"title": item.Title, "subtitle": asString(brief["subtitle"]),
		"source": asString(brief["source"]), "style_id": asString(item.Production["style_id"]),
	}
	return map[string]any{
		"version": planVersion, "brief": brief, "narrative": narrative,
		"storyboard": item.Storyboard, "presentation": presentation,
	}
}

func validateCreative(root string, item creativePackage) []map[string]any {
	issues := make([]map[string]any, 0)
	if item.Version != creativePackageVersion {
		issues = append(issues, issue("CREATIVE_VERSION_INVALID", "creative", "creative.version 必须是 "+creativePackageVersion, "使用当前 schema"))
	}
	if strings.TrimSpace(item.Title) == "" {
		issues = append(issues, issue("CREATIVE_TITLE_MISSING", "brief", "创作包缺少标题", "补充标题"))
	}
	brief := asMap(item.Brief)
	if strings.TrimSpace(asString(brief["goal"])) == "" {
		issues = append(issues, issue("CREATIVE_BRIEF_GOAL_MISSING", "brief", "Brief 缺少目标", "补充 brief.goal"))
	}
	researchMode := strings.ToLower(strings.TrimSpace(asString(item.Research["mode"])))
	if researchMode == "" {
		researchMode = "none"
	}
	if researchMode != "none" && researchMode != "optional" && researchMode != "required" {
		issues = append(issues, issue("CREATIVE_RESEARCH_MODE_INVALID", "research", "research.mode 必须是 none/optional/required", "修复 research.mode"))
	}
	claims := asMapSlice(item.Research["claims"])
	sources := asMapSlice(item.Research["sources"])
	sourceIDs := map[string]bool{}
	for _, source := range sources {
		if id := asString(source["id"]); id != "" {
			sourceIDs[id] = true
		}
	}
	if researchMode == "required" && (len(claims) == 0 || len(sources) == 0) {
		issues = append(issues, issue("CREATIVE_RESEARCH_REQUIRED", "research", "事实型内容必须包含 claims 和 sources", "补充事实与来源"))
	}
	for _, claim := range claims {
		if asString(claim["text"]) == "" {
			issues = append(issues, issue("CREATIVE_CLAIM_EMPTY", "research", "claim 缺少 text", "补充 claim.text"))
		}
		for _, ref := range asStringSlice(claim["evidence_refs"]) {
			if !sourceIDs[ref] {
				issues = append(issues, issue("CREATIVE_EVIDENCE_MISSING", "research", "claim 引用了不存在的 source: "+ref, "修正 evidence_refs"))
			}
		}
	}
	beats := asMapSlice(item.Script["beats"])
	beatIDs := map[string]bool{}
	beatSeconds := 0.0
	for _, beat := range beats {
		id := asString(beat["id"])
		if id == "" {
			issues = append(issues, issue("CREATIVE_BEAT_ID_MISSING", "script", "script beat 缺少 id", "补充 beat.id"))
		} else {
			beatIDs[id] = true
		}
		if asString(beat["narration"]) == "" && asString(beat["on_screen_text"]) == "" {
			issues = append(issues, issue("CREATIVE_BEAT_TEXT_MISSING", "script", "script beat 需要 narration 或 on_screen_text", "补充表述"))
		}
		duration := numberFloat(beat["duration_seconds"])
		if duration <= 0 {
			issues = append(issues, issue("CREATIVE_BEAT_DURATION_INVALID", "script", "script beat 时长必须大于 0", "修复 duration_seconds"))
		}
		beatSeconds += duration
	}
	if len(beats) == 0 {
		issues = append(issues, issue("CREATIVE_SCRIPT_EMPTY", "script", "创作包还没有脚本节拍", "补充 script.beats"))
	}
	shots := asMapSlice(item.Storyboard["shots"])
	shotSeconds := 0.0
	for _, shot := range shots {
		if asString(shot["id"]) == "" {
			issues = append(issues, issue("CREATIVE_SHOT_ID_MISSING", "storyboard", "storyboard shot 缺少 id", "补充 shot.id"))
		}
		if asString(shot["shot"]) == "" {
			issues = append(issues, issue("CREATIVE_SHOT_TYPE_MISSING", "storyboard", "storyboard shot 缺少 shot", "补充 shot"))
		}
		if beatID := asString(shot["beat_id"]); beatID != "" && len(beats) > 0 && !beatIDs[beatID] {
			issues = append(issues, issue("CREATIVE_SHOT_BEAT_MISSING", "storyboard", "shot 引用了不存在的 beat: "+beatID, "修正 beat_id"))
		}
		duration := numberFloat(shot["duration_seconds"])
		if duration <= 0 {
			issues = append(issues, issue("CREATIVE_SHOT_DURATION_INVALID", "storyboard", "shot 时长必须大于 0", "修复 duration_seconds"))
		}
		shotSeconds += duration
	}
	if len(shots) == 0 {
		issues = append(issues, issue("CREATIVE_STORYBOARD_EMPTY", "storyboard", "创作包还没有分镜", "补充 storyboard.shots"))
	}
	if len(beats) > 0 && len(shots) > 0 && mathAbs(beatSeconds-shotSeconds) > 1.5 {
		issues = append(issues, issue("CREATIVE_TIMELINE_MISMATCH", "storyboard", fmt.Sprintf("脚本 %.1fs 与分镜 %.1fs 不一致", beatSeconds, shotSeconds), "调整脚本或分镜时长"))
	}
	mode := strings.ToLower(strings.TrimSpace(asString(item.Production["mode"])))
	if mode == "" {
		mode = "code"
	}
	if mode != "code" && mode != "template" {
		issues = append(issues, issue("CREATIVE_MODE_INVALID", "production", "production.mode 必须是 code 或 template", "修复 production.mode"))
	}
	if mode == "code" {
		packID := strings.TrimSpace(asString(item.Production["style_pack"]))
		if packID == "" {
			issues = append(issues, issue("CREATIVE_STYLE_PACK_MISSING", "production", "code 模式缺少 style_pack", "补充 style_pack"))
		} else if _, ok := findAssetPack(root, packID, asString(item.Production["style_version"])); !ok {
			issues = append(issues, issue("CREATIVE_STYLE_PACK_NOT_FOUND", "production", "找不到 Style Pack: "+packID, "检查 style_pack/version"))
		}
	}
	if mode == "template" {
		if _, ok := findTemplate(asString(item.Production["template_id"])); !ok {
			issues = append(issues, issue("CREATIVE_TEMPLATE_NOT_FOUND", "production", "找不到 template_id", "检查模板"))
		}
		if _, ok := findStyle(asString(item.Production["style_id"])); !ok {
			issues = append(issues, issue("CREATIVE_STYLE_NOT_FOUND", "production", "找不到 style_id", "检查风格"))
		}
	}
	if numberFloat(item.Production["duration_seconds"]) <= 0 {
		issues = append(issues, issue("CREATIVE_DURATION_MISSING", "production", "production.duration_seconds 必须大于 0", "补充时长"))
	}
	return issues
}

func creativeSummary(item creativePackage) map[string]any {
	return map[string]any{
		"id": item.ID, "title": item.Title, "revision": item.Revision, "status": item.Status,
		"beats": len(asMapSlice(item.Script["beats"])), "shots": len(asMapSlice(item.Storyboard["shots"])),
		"production": item.Production,
	}
}

func creativeBriefMarkdown(item creativePackage) string {
	brief := asMap(item.Brief)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", item.Title)
	for _, key := range []string{"goal", "audience", "platform", "tone", "duration_seconds"} {
		if value := asString(brief[key]); value != "" {
			fmt.Fprintf(&b, "- **%s**: %s\n", key, value)
		}
	}
	if constraints := asStringSlice(brief["constraints"]); len(constraints) > 0 {
		b.WriteString("\n## Constraints\n\n")
		for _, value := range constraints {
			fmt.Fprintf(&b, "- %s\n", value)
		}
	}
	return b.String()
}

func creativeScriptMarkdown(item creativePackage) string {
	var b strings.Builder
	b.WriteString("# Script\n\n")
	logline := asString(asMap(item.Script)["logline"])
	if logline != "" {
		fmt.Fprintf(&b, "%s\n\n", logline)
	}
	for index, beat := range asMapSlice(item.Script["beats"]) {
		fmt.Fprintf(&b, "## Beat %d · %s\n\n", index+1, firstNonEmpty(asString(beat["purpose"]), asString(beat["id"])))
		if narration := asString(beat["narration"]); narration != "" {
			fmt.Fprintf(&b, "- Narration: %s\n", narration)
		}
		if text := asString(beat["on_screen_text"]); text != "" {
			fmt.Fprintf(&b, "- On screen: %s\n", text)
		}
		if visual := asString(beat["visual_intent"]); visual != "" {
			fmt.Fprintf(&b, "- Visual: %s\n", visual)
		}
		fmt.Fprintf(&b, "- Duration: %.1fs\n\n", numberFloat(beat["duration_seconds"]))
	}
	return b.String()
}

func asMap(value any) map[string]any {
	if mapValue, ok := value.(map[string]any); ok {
		return mapValue
	}
	return map[string]any{}
}

func asMapSlice(value any) []map[string]any {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if mapValue, ok := item.(map[string]any); ok {
			out = append(out, mapValue)
		}
	}
	return out
}

func asStringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		if values, ok := value.([]string); ok {
			return values
		}
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if text := strings.TrimSpace(asString(item)); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func asInt(value any) int { return numberValue(value) }

func mathAbs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func cloneWithoutError(input map[string]any) map[string]any {
	out, err := cloneMap(input)
	if err != nil {
		return map[string]any{}
	}
	return out
}
