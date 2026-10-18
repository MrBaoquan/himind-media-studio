package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// generate.go —— 一键生成的项目写入与预览编排。
//
// generate.run 仍然遵守两阶段模型：
//   Plan → 校验 → Recipe → 项目落盘 → 预览
// 它不会自动渲染，也不会绕过审批。用户确认预览后再调用 render.start。

func generateRun(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, himindjsonrpc.InvalidParams("name is required")
	}
	planMap := planFromInput(in)
	if planMap == nil {
		return nil, himindjsonrpc.InvalidParams("plan or plan_step is required")
	}
	plan, err := normalizePlan(planMap)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	issues := validatePlan(plan)
	if len(issues) > 0 {
		return map[string]any{"state": "blocked_plan", "blockers": issues, "retryable": true}, nil
	}
	recipe, compileIssues := compilePlan(plan, dataFromInput(in))
	if len(compileIssues) > 0 {
		return map[string]any{"state": "blocked_recipe", "blockers": compileIssues, "retryable": true}, nil
	}

	templateID := "storyboard"
	if len(plan.Storyboard.Shots) == 1 {
		templateID = plan.Storyboard.Shots[0].Shot
	}
	projectID := projectIDForName(in.Name)
	projectRoot := filepath.Join(root, metadataDir, projectDir, projectID)
	if _, statErr := os.Stat(projectRoot); statErr == nil {
		return nil, himindjsonrpc.InvalidParams("project already exists; use its project_id")
	}
	createdAt := now()
	creativeBrief := strings.TrimSpace(in.CreativeBrief)
	if creativeBrief == "" {
		creativeBrief = plan.Brief.Goal
	}
	item := project{
		ID:            projectID,
		Name:          in.Name,
		Description:   in.Description,
		CreativeBrief: creativeBrief,
		Template:      templateID,
		Style:         plan.Presentation.StyleID,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
		Revision:      1,
		Recipe:        recipe,
	}
	if err := writeJSON(filepath.Join(projectRoot, "project.json"), item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if err := writeJSON(filepath.Join(projectRoot, "recipe.json"), recipe); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if err := writeJSON(filepath.Join(projectRoot, "plan.json"), plan); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if err := writeJSON(filepath.Join(projectRoot, "storyboard.json"), plan.Storyboard); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if creativeBrief != "" {
		if err := os.WriteFile(filepath.Join(projectRoot, "brief.md"), []byte(creativeBrief+"\n"), 0644); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
	}

	result := map[string]any{
		"state":          "created",
		"project":        item,
		"workspace_root": root,
		"project_path":   projectRoot,
		"plan":           plan,
		"recipe":         recipe,
		"next":           "确认 Plan 后调用 preview.player；渲染前调用 render.start 并完成审批",
	}
	previewEnabled := true
	if in.Preview != nil {
		previewEnabled = *in.Preview
	}
	if previewEnabled {
		preview, previewErr := previewPlayer(requestInput{WorkspaceRoot: root, ProjectID: projectID})
		if previewErr != nil {
			result["preview"] = map[string]any{"state": "blocked", "message": previewErr.Message}
		} else {
			result["preview"] = preview
		}
	}
	return result, nil
}

// projectIDForName 保留可读 slug；名称含中文或 slug 过短时追加稳定哈希，
// 避免“系列A · 2024”和“系列B · 2024”都生成项目 ID 2024。
func projectIDForName(name string) string {
	normalized := strings.TrimSpace(name)
	base := slug(normalized)
	if base == "" || !idPattern.MatchString(base) {
		base = "video-project"
	}
	if hasNonASCII(normalized) || len(base) < 3 {
		base = base + "-" + stableNameHash(normalized)
	}
	if len(base) > 63 {
		base = base[:63]
	}
	return base
}

func stableNameHash(name string) string {
	sum := sha256.Sum256([]byte(name))
	return hex.EncodeToString(sum[:4])
}

func hasNonASCII(value string) bool {
	for _, r := range value {
		if r > 127 {
			return true
		}
	}
	return false
}
