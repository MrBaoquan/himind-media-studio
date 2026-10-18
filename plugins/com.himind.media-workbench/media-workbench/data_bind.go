package main

import (
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// data_bind.go —— 把通用 items 映射为目标模板的数据契约。
//
// data.ingest 只负责“读成 items”；data.bind 负责“按模板变成 categories / steps /
// events / values / items”。这层不知道业务事实，只做确定性的结构映射。

func dataBind(in requestInput) (any, *himindjsonrpc.Error) {
	templateID := strings.TrimSpace(in.TemplateID)
	planMap := planFromInput(in)
	if templateID == "" && planMap != nil {
		plan, err := normalizePlan(planMap)
		if err == nil && len(plan.Storyboard.Shots) > 0 {
			templateID = plan.Storyboard.Shots[0].Shot
		}
	}
	if templateID == "" {
		return nil, himindjsonrpc.InvalidParams("template_id or plan_step is required")
	}
	templateItem, ok := findTemplate(templateID)
	if !ok {
		return nil, himindjsonrpc.InvalidParams("template not found: " + templateID)
	}
	raw := dataFromInput(in)
	if raw == nil {
		return map[string]any{"valid": false, "issues": []map[string]any{issue("data_bind_missing", "recipe", "没有可绑定的数据", "先调用 data.ingest 或传入 data")}, "template_id": templateID}, nil
	}
	bound, issues := bindDataForTemplate(templateID, raw)
	if len(issues) == 0 && templateItem.DataContract != "" {
		issues = append(issues, validateTemplateData(templateItem, map[string]any{"data": bound})...)
	}
	if len(issues) > 0 {
		return map[string]any{"valid": false, "issues": issues, "template_id": templateID, "data": bound}, nil
	}
	return map[string]any{
		"valid":       true,
		"template_id": templateID,
		"data":        bound,
		"next":        "把 data 作为 generate.run 的 data 或 recipe.compile 的 data",
	}, nil
}

func bindDataForTemplate(templateID string, raw map[string]any) (map[string]any, []map[string]any) {
	items, _ := raw["items"].([]any)
	base := cloneAnyMap(raw)
	switch templateID {
	case "leaderboard-tech":
		if len(items) == 0 {
			return base, []map[string]any{issue("data_bind_items_missing", "recipe", "排行榜需要 items", "提供 name/value 数据")}
		}
		base["items"] = normalizeItems(items)
		return base, nil
	case "bar-race":
		if len(items) == 0 {
			return base, []map[string]any{issue("data_bind_items_missing", "recipe", "时序竞速需要 items", "提供带 valueByYear 的数据")}
		}
		for _, item := range items {
			object, _ := item.(map[string]any)
			if _, ok := object["valueByYear"]; !ok {
				return base, []map[string]any{issue("data_bind_value_by_year_missing", "recipe", "bar-race 的每个 item 需要 valueByYear", "补充按年份的数值映射")}
			}
		}
		base["items"] = items
		return base, nil
	case "composition":
		if categories, ok := raw["categories"].([]any); ok {
			base["categories"] = categories
			return base, nil
		}
		if len(items) < 2 || len(items) > 8 {
			return base, []map[string]any{issue("data_bind_categories_invalid", "recipe", "composition 需要 2–8 个分类", "提供 2–8 条 name/value 数据")}
		}
		categories := make([]any, 0, len(items))
		for _, item := range items {
			object, _ := item.(map[string]any)
			name := firstNonEmpty(asString(object["name"]), asString(object["label"]))
			value, ok := schemaNumber(object["value"])
			if name == "" || !ok {
				return base, []map[string]any{issue("data_bind_item_invalid", "recipe", "分类项需要 name 和数值 value", "检查数据行")}
			}
			categories = append(categories, map[string]any{"label": name, "value": value})
		}
		base["categories"] = categories
		return base, nil
	case "sankey-flow":
		if links, ok := raw["links"].([]any); ok {
			base["links"] = links
			return base, nil
		}
		if len(items) < 2 || len(items) > 8 {
			return base, []map[string]any{issue("data_bind_links_invalid", "recipe", "sankey-flow 需要 2–8 条流向", "提供 2–8 条 name/value 数据")}
		}
		totalLabel := firstNonEmpty(asString(raw["total_label"]), "总量")
		links := make([]any, 0, len(items))
		for _, item := range items {
			object, _ := item.(map[string]any)
			name := firstNonEmpty(asString(object["name"]), asString(object["label"]))
			value, ok := schemaNumber(object["value"])
			if name == "" || !ok {
				return base, []map[string]any{issue("data_bind_link_invalid", "recipe", "流向项需要目标名称和数值", "检查数据行")}
			}
			links = append(links, map[string]any{"source": totalLabel, "target": name, "value": value})
		}
		base["total_label"] = totalLabel
		base["links"] = links
		return base, nil
	case "flow":
		if steps, ok := raw["steps"].([]any); ok {
			base["steps"] = steps
			return base, nil
		}
		if len(items) < 2 || len(items) > 6 {
			return base, []map[string]any{issue("data_bind_steps_invalid", "recipe", "flow 需要 2–6 个步骤", "提供 2–6 条 name/description 数据")}
		}
		steps := make([]any, 0, len(items))
		for _, item := range items {
			object, _ := item.(map[string]any)
			label := firstNonEmpty(asString(object["name"]), asString(object["label"]))
			if label == "" {
				return base, []map[string]any{issue("data_bind_step_invalid", "recipe", "步骤缺少名称", "检查数据行")}
			}
			entry := map[string]any{"label": label}
			if description := firstNonEmpty(asString(object["description"]), asString(object["note"])); description != "" {
				entry["description"] = description
			}
			steps = append(steps, entry)
		}
		base["steps"] = steps
		return base, nil
	case "timeline":
		if events, ok := raw["events"].([]any); ok {
			base["events"] = events
			return base, nil
		}
		events := make([]any, 0, len(items))
		for _, item := range items {
			object, _ := item.(map[string]any)
			title := firstNonEmpty(asString(object["title"]), asString(object["name"]))
			year, ok := schemaNumber(object["year"])
			if title == "" || !ok {
				return base, []map[string]any{issue("data_bind_event_invalid", "recipe", "timeline 每项需要 year 和 title", "提供 year/title 数据")}
			}
			entry := map[string]any{"year": int(year), "title": title}
			if description := firstNonEmpty(asString(object["description"]), asString(object["note"])); description != "" {
				entry["description"] = description
			}
			events = append(events, entry)
		}
		base["events"] = events
		return base, nil
	case "choropleth":
		if values, ok := raw["values"].(map[string]any); ok {
			base["values"] = values
			return base, nil
		}
		values := map[string]any{}
		for _, item := range items {
			object, _ := item.(map[string]any)
			name := firstNonEmpty(asString(object["name"]), asString(object["region"]))
			value, ok := schemaNumber(object["value"])
			if name == "" || !ok {
				return base, []map[string]any{issue("data_bind_region_invalid", "recipe", "choropleth 每项需要地区名和数值", "提供 region/value 数据")}
			}
			values[name] = value
		}
		if len(values) == 0 {
			return base, []map[string]any{issue("data_bind_values_missing", "recipe", "choropleth 需要 values", "提供地区数据")}
		}
		base["values"] = values
		return base, nil
	case "storyboard":
		if storyboard, ok := raw["storyboard"].([]any); ok {
			base["storyboard"] = storyboard
			return base, nil
		}
		return base, []map[string]any{issue("data_bind_storyboard_missing", "recipe", "storyboard 需要镜头数组", "直接提供 storyboard 数据")}
	case "route-growth":
		if lines, ok := raw["lines"].([]any); ok {
			base["lines"] = lines
			return base, nil
		}
		return base, []map[string]any{issue("data_bind_lines_missing", "recipe", "route-growth 需要 lines 与站点/坐标", "提供结构化线路数据或省略 data 使用模板默认示例数据")}
	default:
		return base, nil
	}
}

func normalizeItems(items []any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		object, _ := item.(map[string]any)
		name := firstNonEmpty(asString(object["name"]), asString(object["label"]))
		value, ok := schemaNumber(object["value"])
		if name == "" || !ok {
			continue
		}
		entry := map[string]any{"name": name, "value": value}
		if sub := firstNonEmpty(asString(object["sub_value"]), asString(object["change"])); sub != "" {
			entry["sub_value"] = sub
		}
		if icon := asString(object["icon"]); icon != "" {
			entry["icon"] = icon
		}
		out = append(out, entry)
	}
	return out
}

func cloneAnyMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
