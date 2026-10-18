package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// plan.go —— 生成器的规划层。
//
// Plan 是 AI 与确定性执行之间的契约：
//   Brief → NarrativePlan → StoryboardPlan → Presentation
//
// 这一层刻意不调用 LLM，也不直接渲染。AI 客户端负责产出计划，
// 插件负责规范化、校验、估算和编译成 Recipe。

const planVersion = "himind.video.plan.v1"

type CreativeBriefPlan struct {
	Goal            string   `json:"goal"`
	Audience        string   `json:"audience,omitempty"`
	Platform        string   `json:"platform,omitempty"`
	DurationSeconds float64  `json:"duration_seconds,omitempty"`
	Tone            string   `json:"tone,omitempty"`
	MustInclude     []string `json:"must_include,omitempty"`
	MustAvoid       []string `json:"must_avoid,omitempty"`
	DataRefs        []string `json:"data_refs,omitempty"`
}

type KeyPointPlan struct {
	Claim       string `json:"claim"`
	EvidenceRef string `json:"evidence_ref,omitempty"`
}

type NarrativePlan struct {
	Thesis       string         `json:"thesis"`
	Structure    string         `json:"structure"`
	KeyPoints    []KeyPointPlan `json:"key_points,omitempty"`
	RequiredData []string       `json:"required_data,omitempty"`
	Constraints  []string       `json:"constraints,omitempty"`
	RiskNotes    []string       `json:"risk_notes,omitempty"`
}

type ShotPlan struct {
	Shot            string         `json:"shot"`
	Role            string         `json:"role,omitempty"`
	DurationSeconds float64        `json:"duration_seconds"`
	DataBinding     string         `json:"data_binding,omitempty"`
	StyleOverrides  map[string]any `json:"style_overrides,omitempty"`
	Transition      string         `json:"transition,omitempty"`
	Rationale       string         `json:"rationale,omitempty"`
	Title           string         `json:"title,omitempty"`
	Subtitle        string         `json:"subtitle,omitempty"`
	Data            map[string]any `json:"data,omitempty"`
}

type StoryboardPlan struct {
	Shots []ShotPlan `json:"shots"`
}

type PresentationPlan struct {
	Title            string `json:"title,omitempty"`
	Subtitle         string `json:"subtitle,omitempty"`
	Source           string `json:"source,omitempty"`
	StyleID          string `json:"style_id,omitempty"`
	TransitionFrames int    `json:"transition_frames,omitempty"`
}

type VideoPlan struct {
	Version      string            `json:"version"`
	Brief        CreativeBriefPlan `json:"brief"`
	Narrative    NarrativePlan     `json:"narrative"`
	Storyboard   StoryboardPlan    `json:"storyboard"`
	Presentation PresentationPlan  `json:"presentation"`
}

type planEstimate struct {
	DurationSeconds float64  `json:"duration_seconds"`
	ShotCount       int      `json:"shot_count"`
	Templates       []string `json:"templates"`
	RequiredData    []string `json:"required_data,omitempty"`
	CostClass       string   `json:"cost_class"`
	RiskNotes       []string `json:"risk_notes,omitempty"`
}

func planNormalize(in requestInput) (any, *himindjsonrpc.Error) {
	planMap := planFromInput(in)
	if planMap == nil {
		return nil, himindjsonrpc.InvalidParams("plan or plan_step is required")
	}
	plan, err := normalizePlan(planMap)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	issues := validatePlan(plan)
	return map[string]any{
		"valid":  len(issues) == 0,
		"plan":   plan,
		"issues": issues,
		"next":   "调用 plan.validate，然后调用 recipe.compile",
	}, nil
}

func planValidate(in requestInput) (any, *himindjsonrpc.Error) {
	planMap := planFromInput(in)
	if planMap == nil {
		return nil, himindjsonrpc.InvalidParams("plan or plan_step is required")
	}
	plan, err := normalizePlan(planMap)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	issues := validatePlan(plan)
	return map[string]any{
		"valid":   len(issues) == 0,
		"plan":    plan,
		"issues":  issues,
		"summary": summarizePlan(plan),
	}, nil
}

func planEstimateCapability(in requestInput) (any, *himindjsonrpc.Error) {
	planMap := planFromInput(in)
	if planMap == nil {
		return nil, himindjsonrpc.InvalidParams("plan or plan_step is required")
	}
	plan, err := normalizePlan(planMap)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	issues := validatePlan(plan)
	return map[string]any{
		"valid":    len(issues) == 0,
		"issues":   issues,
		"estimate": estimatePlan(plan),
	}, nil
}

func planFromRecipeCapability(in requestInput) (any, *himindjsonrpc.Error) {
	recipe := in.Recipe
	if recipe == nil && strings.TrimSpace(in.ProjectID) != "" {
		root, err := workspace(in.WorkspaceRoot)
		if err != nil {
			return nil, himindjsonrpc.InvalidParams(err.Error())
		}
		item, err := loadProject(root, in.ProjectID)
		if err != nil {
			return nil, himindjsonrpc.InvalidParams(err.Error())
		}
		recipe = item.Recipe
	}
	if recipe == nil {
		return nil, himindjsonrpc.InvalidParams("recipe or project_id is required")
	}
	plan, err := planFromRecipe(recipe)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	return map[string]any{
		"plan":   plan,
		"issues": validatePlan(plan),
		"next":   "调用 plan.normalize 后可用于新的生成流程",
	}, nil
}

func recipeCompile(in requestInput) (any, *himindjsonrpc.Error) {
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
		return map[string]any{"state": "blocked", "blockers": issues, "plan": plan, "retryable": true}, nil
	}
	recipe, compileIssues := compilePlan(plan, in.Data)
	if len(compileIssues) > 0 {
		return map[string]any{"state": "blocked", "blockers": compileIssues, "plan": plan, "retryable": true}, nil
	}
	return map[string]any{
		"state":  "compiled",
		"plan":   plan,
		"recipe": recipe,
		"next":   "调用 recipe.validate 检查数据契约与安全区",
	}, nil
}

func normalizePlan(raw map[string]any) (VideoPlan, error) {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return VideoPlan{}, err
	}
	var plan VideoPlan
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return VideoPlan{}, fmt.Errorf("invalid plan: %w", err)
	}
	if plan.Version == "" {
		plan.Version = planVersion
	}
	if plan.Storyboard.Shots == nil {
		plan.Storyboard.Shots = []ShotPlan{}
	}
	for index := range plan.Storyboard.Shots {
		shot := &plan.Storyboard.Shots[index]
		if shot.Role == "" {
			shot.Role = "shot"
		}
		if shot.Transition == "" {
			shot.Transition = "cut"
		}
		if shot.DataBinding == "" {
			shot.DataBinding = "data"
		}
		if shot.DurationSeconds <= 0 {
			if templateItem, ok := findTemplate(shot.Shot); ok {
				shot.DurationSeconds = float64(templateItem.DurationSeconds)
			} else {
				shot.DurationSeconds = 4
			}
		}
	}
	if plan.Presentation.StyleID == "" {
		for _, shot := range plan.Storyboard.Shots {
			if templateItem, ok := findTemplate(shot.Shot); ok && len(templateItem.CompatibleStyle) > 0 {
				plan.Presentation.StyleID = templateItem.CompatibleStyle[0]
				break
			}
		}
	}
	if plan.Presentation.Title == "" {
		plan.Presentation.Title = plan.Narrative.Thesis
	}
	if plan.Narrative.Thesis == "" {
		plan.Narrative.Thesis = plan.Presentation.Title
	}
	if plan.Narrative.Structure == "" {
		if len(plan.Storyboard.Shots) > 1 {
			plan.Narrative.Structure = "storyboard"
		} else {
			plan.Narrative.Structure = "single-shot"
		}
	}
	if plan.Brief.Goal == "" {
		plan.Brief.Goal = plan.Presentation.Title
	}
	if plan.Brief.DurationSeconds <= 0 {
		plan.Brief.DurationSeconds = totalShotSeconds(plan.Storyboard.Shots)
	}
	if plan.Brief.DurationSeconds <= 0 {
		plan.Brief.DurationSeconds = 10
	}
	if plan.Narrative.RequiredData == nil {
		plan.Narrative.RequiredData = []string{}
	}
	return plan, nil
}

func validatePlan(plan VideoPlan) []map[string]any {
	issues := make([]map[string]any, 0)
	if plan.Version != planVersion {
		issues = append(issues, issue("plan_version_invalid", "plan", "plan.version 必须是 "+planVersion, "使用当前版本的 Plan schema"))
	}
	if len(plan.Storyboard.Shots) == 0 {
		issues = append(issues, issue("plan_shots_missing", "plan", "Plan 至少需要一个镜头", "在 storyboard.shots 中提供镜头"))
	}
	if strings.TrimSpace(plan.Presentation.Title) == "" {
		issues = append(issues, issue("plan_title_missing", "plan", "Plan 缺少标题", "在 presentation.title 或 narrative.thesis 中提供标题"))
	}
	styleID := plan.Presentation.StyleID
	if styleID == "" {
		issues = append(issues, issue("plan_style_missing", "plan", "Plan 缺少风格", "在 presentation.style_id 中提供兼容风格"))
	} else if _, ok := findStyle(styleID); !ok {
		issues = append(issues, issue("plan_style_not_found", "plan", "Plan 引用了不存在的风格", "调用 style.list 选择已注册风格"))
	}

	total := 0.0
	for index, shot := range plan.Storyboard.Shots {
		path := fmt.Sprintf("storyboard.shots[%d]", index)
		if strings.TrimSpace(shot.Shot) == "" {
			issues = append(issues, issue("plan_shot_missing", "plan", path+" 缺少 shot", "填写模板/镜头 id"))
			continue
		}
		templateItem, ok := findTemplate(shot.Shot)
		if !ok {
			issues = append(issues, issue("plan_shot_not_found", "plan", path+" 引用了不存在的镜头", "调用 template.list 选择 ready 模板"))
		} else {
			if templateItem.Status != "ready" {
				issues = append(issues, issue("plan_shot_not_ready", "plan", path+" 的镜头还不是 ready", "改用 ready 模板，或先完成该镜头的实现"))
			}
			if styleID != "" && !contains(templateItem.CompatibleStyle, styleID) {
				issues = append(issues, issue("plan_style_incompatible", "plan", path+" 与当前风格不兼容", "改用兼容风格或更换镜头"))
			}
		}
		if shot.DurationSeconds < 1 || shot.DurationSeconds > 60 {
			issues = append(issues, issue("plan_shot_duration_invalid", "plan", path+" 时长必须在 1–60 秒", "调整 duration_seconds"))
		}
		total += shot.DurationSeconds
		if strings.TrimSpace(shot.DataBinding) == "" {
			issues = append(issues, issue("plan_data_binding_missing", "plan", path+" 缺少数据绑定", "填写 data_binding，例如 data"))
		}
	}
	if total < 1 || total > 300 {
		issues = append(issues, issue("plan_duration_invalid", "plan", "Plan 总时长必须在 1–300 秒", "调整镜头时长"))
	}
	if plan.Presentation.TransitionFrames < 0 || plan.Presentation.TransitionFrames > 60 {
		issues = append(issues, issue("plan_transition_invalid", "plan", "transition_frames 必须在 0–60", "调整转场帧数"))
	}
	return issues
}

func summarizePlan(plan VideoPlan) map[string]any {
	templates := make([]string, 0, len(plan.Storyboard.Shots))
	for _, shot := range plan.Storyboard.Shots {
		templates = append(templates, shot.Shot)
	}
	return map[string]any{
		"title":            plan.Presentation.Title,
		"style_id":         plan.Presentation.StyleID,
		"shot_count":       len(plan.Storyboard.Shots),
		"duration_seconds": totalShotSeconds(plan.Storyboard.Shots),
		"templates":        templates,
		"required_data":    plan.Narrative.RequiredData,
		"structure":        plan.Narrative.Structure,
	}
}

func estimatePlan(plan VideoPlan) planEstimate {
	templates := make([]string, 0, len(plan.Storyboard.Shots))
	for _, shot := range plan.Storyboard.Shots {
		templates = append(templates, shot.Shot)
	}
	duration := totalShotSeconds(plan.Storyboard.Shots)
	cost := "low"
	if duration > 15 || len(plan.Storyboard.Shots) > 3 {
		cost = "medium"
	}
	if duration > 45 || len(plan.Storyboard.Shots) > 8 {
		cost = "high"
	}
	return planEstimate{
		DurationSeconds: duration,
		ShotCount:       len(plan.Storyboard.Shots),
		Templates:       templates,
		RequiredData:    plan.Narrative.RequiredData,
		CostClass:       cost,
		RiskNotes:       plan.Narrative.RiskNotes,
	}
}

func totalShotSeconds(shots []ShotPlan) float64 {
	total := 0.0
	for _, shot := range shots {
		total += shot.DurationSeconds
	}
	return total
}

func planFromRecipe(recipe map[string]any) (VideoPlan, error) {
	templateID := asString(recipe["template_id"])
	if templateID == "" {
		return VideoPlan{}, fmt.Errorf("recipe.template_id is required")
	}
	styleID := asString(recipe["style_id"])
	title := asString(recipe["title"])
	subtitle := asString(recipe["subtitle"])
	source := asString(recipe["source"])
	duration := numberFloat(recipe["duration_seconds"])

	shots := make([]ShotPlan, 0)
	if templateID == "storyboard" {
		if data, ok := recipe["data"].(map[string]any); ok {
			if rawShots, ok := data["storyboard"].([]any); ok {
				for _, raw := range rawShots {
					shotMap, ok := raw.(map[string]any)
					if !ok {
						continue
					}
					shotData, _ := shotMap["data"].(map[string]any)
					shots = append(shots, ShotPlan{
						Shot:            asString(shotMap["shot"]),
						Role:            "shot",
						DurationSeconds: numberFloat(shotMap["duration_seconds"]),
						DataBinding:     "data",
						Transition:      "cut",
						Title:           asString(shotMap["title"]),
						Subtitle:        asString(shotMap["subtitle"]),
						Data:            shotData,
					})
				}
			}
		}
	} else {
		shotData, _ := recipe["data"].(map[string]any)
		shots = append(shots, ShotPlan{
			Shot:            templateID,
			Role:            "shot",
			DurationSeconds: duration,
			DataBinding:     "data",
			Transition:      "cut",
			Data:            shotData,
		})
	}

	if len(shots) == 0 {
		return VideoPlan{}, fmt.Errorf("recipe has no renderable shots")
	}
	dataKeys := []string{}
	if data, ok := recipe["data"].(map[string]any); ok {
		for key := range data {
			dataKeys = append(dataKeys, key)
		}
		sort.Strings(dataKeys)
	}
	raw := map[string]any{
		"version": planVersion,
		"brief": map[string]any{
			"goal":             title,
			"duration_seconds": duration,
		},
		"narrative": map[string]any{
			"thesis":        title,
			"structure":     map[bool]string{true: "storyboard", false: "single-shot"}[len(shots) > 1],
			"required_data": dataKeys,
		},
		"storyboard": map[string]any{"shots": shots},
		"presentation": map[string]any{
			"title":    title,
			"subtitle": subtitle,
			"source":   source,
			"style_id": styleID,
		},
	}
	return normalizePlan(raw)
}

func compilePlan(plan VideoPlan, data map[string]any) (map[string]any, []map[string]any) {
	styleID := plan.Presentation.StyleID
	if len(plan.Storyboard.Shots) == 1 {
		shot := plan.Storyboard.Shots[0]
		if shot.DataBinding != "" && shot.DataBinding != "data" {
			return nil, []map[string]any{issue("compile_binding_unsupported", "compile", "单镜头编译目前只支持 data 绑定", "改用 data_binding=data，或在 Storyboard 中拆分数据")}
		}
		renderData := data
		if renderData == nil {
			renderData = shot.Data
		}
		recipe := defaultRecipe(shot.Shot, styleID, renderData)
		applyPlanPresentation(recipe, plan)
		recipe["duration_seconds"] = roundSeconds(shot.DurationSeconds)
		return recipe, nil
	}

	storyboard := make([]any, 0, len(plan.Storyboard.Shots))
	issues := make([]map[string]any, 0)
	for index, shot := range plan.Storyboard.Shots {
		path := fmt.Sprintf("storyboard.shots[%d]", index)
		if shot.Data == nil {
			issues = append(issues, issue("compile_shot_data_missing", "compile", path+" 缺少 shot.data", "为每个镜头提供自己的 data"))
			continue
		}
		if templateItem, ok := findTemplate(shot.Shot); ok && templateItem.DataContract != "" {
			contractIssues := validateTemplateData(templateItem, map[string]any{"data": shot.Data})
			for _, contractIssue := range contractIssues {
				contractIssue["message"] = path + ": " + asString(contractIssue["message"])
				issues = append(issues, contractIssue)
			}
		}
		entry := map[string]any{
			"shot":             shot.Shot,
			"duration_seconds": roundSeconds(shot.DurationSeconds),
			"data":             shot.Data,
		}
		if shot.Title != "" {
			entry["title"] = shot.Title
		}
		if shot.Subtitle != "" {
			entry["subtitle"] = shot.Subtitle
		}
		storyboard = append(storyboard, entry)
	}
	if len(issues) > 0 {
		return nil, issues
	}
	renderData := map[string]any{"storyboard": storyboard}
	if plan.Presentation.TransitionFrames > 0 {
		renderData["transition_frames"] = plan.Presentation.TransitionFrames
	}
	recipe := defaultRecipe("storyboard", styleID, renderData)
	applyPlanPresentation(recipe, plan)
	recipe["duration_seconds"] = roundSeconds(totalShotSeconds(plan.Storyboard.Shots))
	return recipe, nil
}

func applyPlanPresentation(recipe map[string]any, plan VideoPlan) {
	if plan.Presentation.Title != "" {
		recipe["title"] = plan.Presentation.Title
	}
	if plan.Presentation.Subtitle != "" {
		recipe["subtitle"] = plan.Presentation.Subtitle
	}
	if plan.Presentation.Source != "" {
		recipe["source"] = plan.Presentation.Source
	}
}

func roundSeconds(value float64) int {
	seconds := int(math.Round(value))
	if seconds < 1 {
		return 1
	}
	return seconds
}

func numberFloat(value any) float64 {
	if number, ok := schemaNumber(value); ok {
		return number
	}
	return 0
}
