package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// series.go —— 同一叙事结构换数据，批量生成多个项目。
//
// M2 的第一版只做单镜头系列：
//   一个 Plan + 多组 item data → 多个项目
// 批量渲染、抽样验收和系列级进度放到后续迭代。

type seriesItem struct {
	Name     string         `json:"name"`
	Title    string         `json:"title,omitempty"`
	Subtitle string         `json:"subtitle,omitempty"`
	Data     map[string]any `json:"data"`
}

type seriesPlanItem struct {
	Name            string   `json:"name"`
	ProjectName     string   `json:"project_name"`
	Title           string   `json:"title"`
	Subtitle        string   `json:"subtitle,omitempty"`
	DurationSeconds float64  `json:"duration_seconds"`
	DataKeys        []string `json:"data_keys"`
}

func seriesPlanCapability(in requestInput) (any, *himindjsonrpc.Error) {
	plan, _, previewItems, issues := prepareSeries(in)
	result := map[string]any{
		"valid":   len(issues) == 0,
		"issues":  issues,
		"summary": summarizeSeries(plan, previewItems),
		"items":   previewItems,
	}
	if plan.Version == "" {
		result["plan"] = nil
	}
	return result, nil
}

func seriesGenerateCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	plan, items, previewItems, issues := prepareSeries(in)
	if len(issues) > 0 {
		return map[string]any{"state": "blocked", "blockers": issues, "items": previewItems, "retryable": true}, nil
	}
	previewEnabled := false
	if in.Preview != nil {
		previewEnabled = *in.Preview
	}
	created := make([]any, 0, len(items))
	failures := make([]any, 0)
	for index, item := range items {
		itemPlan := clonePlan(plan)
		if item.Title != "" {
			itemPlan.Presentation.Title = item.Title
		}
		if item.Subtitle != "" {
			itemPlan.Presentation.Subtitle = item.Subtitle
		}
		if len(itemPlan.Storyboard.Shots) == 1 {
			itemPlan.Storyboard.Shots[0].Data = item.Data
		}
		planMap, err := planToMap(itemPlan)
		if err != nil {
			failures = append(failures, map[string]any{"name": item.Name, "error": err.Error()})
			continue
		}
		itemPreview := previewEnabled
		result, rpcErr := generateRun(requestInput{
			WorkspaceRoot: root,
			Name:          previewItems[index].ProjectName,
			CreativeBrief: plan.Brief.Goal,
			Plan:          planMap,
			Data:          item.Data,
			Preview:       &itemPreview,
		})
		if rpcErr != nil {
			failures = append(failures, map[string]any{"name": item.Name, "error": rpcErr.Message})
			continue
		}
		payload, _ := result.(map[string]any)
		if asString(payload["state"]) != "created" {
			failures = append(failures, map[string]any{"name": item.Name, "result": payload})
			continue
		}
		created = append(created, map[string]any{
			"name":             item.Name,
			"project":          payload["project"],
			"project_path":     payload["project_path"],
			"preview":          payload["preview"],
			"duration_seconds": previewItems[index].DurationSeconds,
		})
	}
	state := "completed"
	if len(failures) > 0 && len(created) == 0 {
		state = "failed"
	} else if len(failures) > 0 {
		state = "partial"
	}
	return map[string]any{
		"state":    state,
		"summary":  summarizeSeries(plan, previewItems),
		"items":    previewItems,
		"created":  created,
		"failures": failures,
	}, nil
}

func prepareSeries(in requestInput) (VideoPlan, []seriesItem, []seriesPlanItem, []map[string]any) {
	issues := make([]map[string]any, 0)
	planMap := planFromInput(in)
	if planMap == nil {
		issues = append(issues, issue("series_plan_missing", "series", "系列生成需要 Plan", "先调用 video-director 生成 Plan"))
		return VideoPlan{}, nil, nil, issues
	}
	plan, err := normalizePlan(planMap)
	if err != nil {
		issues = append(issues, issue("series_plan_invalid", "series", err.Error(), "修复 Plan 结构后重试"))
		return VideoPlan{}, nil, nil, issues
	}
	issues = append(issues, validatePlan(plan)...)
	if len(plan.Storyboard.Shots) != 1 {
		issues = append(issues, issue("series_multi_shot_not_supported", "series", "当前系列生成只支持单镜头 Plan", "先拆分系列，或等待多镜头系列能力"))
	}
	if len(in.Items) == 0 {
		issues = append(issues, issue("series_items_missing", "series", "至少需要一个系列 item", "在 items 中提供 name 和 data"))
	}
	seen := map[string]bool{}
	items := make([]seriesItem, 0, len(in.Items))
	previewItems := make([]seriesPlanItem, 0, len(in.Items))
	baseName := strings.TrimSpace(in.NamePrefix)
	if baseName == "" {
		baseName = "系列"
	}
	for index, raw := range in.Items {
		item := raw
		if strings.TrimSpace(item.Name) == "" {
			item.Name = fmt.Sprintf("第 %d 项", index+1)
		}
		projectName := baseName + " · " + item.Name
		if seen[projectName] {
			issues = append(issues, issue("series_name_duplicate", "series", "系列项目名重复："+projectName, "为每个 item 使用不同的 name"))
			continue
		}
		seen[projectName] = true
		if item.Data == nil {
			issues = append(issues, issue("series_item_data_missing", "series", item.Name+" 缺少 data", "为每个 item 提供自己的数据"))
		}
		if item.Title == "" {
			item.Title = plan.Presentation.Title
		}
		items = append(items, item)
		dataKeys := make([]string, 0, len(item.Data))
		for key := range item.Data {
			dataKeys = append(dataKeys, key)
		}
		sort.Strings(dataKeys)
		previewItems = append(previewItems, seriesPlanItem{
			Name:            item.Name,
			ProjectName:     projectName,
			Title:           item.Title,
			Subtitle:        item.Subtitle,
			DurationSeconds: totalShotSeconds(plan.Storyboard.Shots),
			DataKeys:        dataKeys,
		})
	}
	return plan, items, previewItems, issues
}

func summarizeSeries(plan VideoPlan, items []seriesPlanItem) map[string]any {
	total := 0.0
	for _, item := range items {
		total += item.DurationSeconds
	}
	return map[string]any{
		"count":         len(items),
		"total_seconds": total,
		"structure":     plan.Narrative.Structure,
		"template":      firstShotID(plan),
		"style_id":      plan.Presentation.StyleID,
	}
}

func firstShotID(plan VideoPlan) string {
	if len(plan.Storyboard.Shots) == 0 {
		return ""
	}
	return plan.Storyboard.Shots[0].Shot
}

func clonePlan(plan VideoPlan) VideoPlan {
	raw, _ := json.Marshal(plan)
	var out VideoPlan
	_ = json.Unmarshal(raw, &out)
	return out
}

func planToMap(plan VideoPlan) (map[string]any, error) {
	raw, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
