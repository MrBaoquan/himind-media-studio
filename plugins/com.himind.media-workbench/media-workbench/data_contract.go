package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

// data_contract.go —— 把模板声明的 data_contract 变成真实执行点。
//
// 这里实现的是模板契约所需的 JSON Schema 子集：
// type / required / properties / items / minItems / maxItems /
// minimum / maximum / minProperties / additionalProperties。
//
// 不追求通用 JSON Schema 全集；目标是让创作台能在 recipe.validate
// 阶段拦住"分类值为负、步骤超限、字段名写错"这类会直接毁掉画面的错误。

const maxContractIssues = 12

func validateTemplateData(templateItem videoTemplate, recipe map[string]any) []map[string]any {
	return validateTemplateDataAt(templateItem, recipe, "data")
}

// validateTemplateDataAt 允许把同一份契约复用到 Storyboard 的嵌套镜头：
// 路径会变成 data.storyboard[i].data.*，错误能直接指到具体镜头和字段。
func validateTemplateDataAt(templateItem videoTemplate, recipe map[string]any, rootPath string) []map[string]any {
	if templateItem.DataContract == "" {
		return nil
	}
	data, ok := recipe["data"].(map[string]any)
	if !ok {
		return []map[string]any{issue("data_contract_invalid", "recipe", rootPath+" 必须是对象", "把渲染数据放在 Recipe.data 中")}
	}
	path := filepath.Join(pluginRoot(), filepath.FromSlash(templateItem.DataContract))
	raw, err := os.ReadFile(path)
	if err != nil {
		return []map[string]any{issue("data_contract_missing", "recipe", "模板声明的数据契约不存在", "检查 catalog/contracts 下的 schema 文件")}
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return []map[string]any{issue("data_contract_invalid_schema", "recipe", "模板数据契约不是合法 JSON", "修复 catalog/contracts 下的 schema 文件")}
	}
	issues := make([]map[string]any, 0)
	validateSchemaValue(data, schema, rootPath, &issues)
	if templateItem.Family == "storyboard" {
		validateStoryboardShotContracts(data, rootPath, &issues)
	}
	return issues
}

// validateStoryboardShotContracts 是组合层的第二道契约：
// storyboard.schema 只约束镜头外壳，镜头内部 data 仍必须通过各自模板的契约。
func validateStoryboardShotContracts(data map[string]any, rootPath string, issues *[]map[string]any) {
	items, _ := data["storyboard"].([]any)
	for index, rawItem := range items {
		item, _ := rawItem.(map[string]any)
		shotID := asString(item["shot"])
		if shotID == "" {
			continue // 外层 schema 已经报 required
		}
		shotTemplate, ok := findTemplate(shotID)
		if !ok {
			appendContractIssue(issues, "storyboard_shot_unknown", fmt.Sprintf("%s.storyboard[%d].shot", rootPath, index), fmt.Sprintf("storyboard[%d] 使用了未知镜头 %s", index, shotID))
			continue
		}
		if !shotTemplate.StoryboardShot {
			appendContractIssue(issues, "storyboard_shot_invalid", fmt.Sprintf("%s.storyboard[%d].shot", rootPath, index), fmt.Sprintf("%s 不能作为 Storyboard 镜头", shotID))
			continue
		}
		shotData, _ := item["data"].(map[string]any)
		if shotData == nil {
			shotData = map[string]any{}
		}
		nested := validateTemplateDataAt(shotTemplate, map[string]any{"data": shotData}, fmt.Sprintf("%s.storyboard[%d].data", rootPath, index))
		*issues = append(*issues, nested...)
		if len(*issues) >= maxContractIssues {
			return
		}
	}
}

func validateSchemaValue(value any, schema map[string]any, path string, issues *[]map[string]any) {
	if schema == nil || len(*issues) >= maxContractIssues {
		return
	}
	if expected := asString(schema["type"]); expected != "" && !matchesSchemaType(value, expected) {
		appendContractIssue(issues, "data_contract_type", path, fmt.Sprintf("%s 必须是 %s", path, expected))
		return
	}

	switch typed := value.(type) {
	case map[string]any:
		required := schemaStringList(schema["required"])
		for _, key := range required {
			if _, ok := typed[key]; !ok {
				appendContractIssue(issues, "data_contract_required", path+"."+key, fmt.Sprintf("%s.%s 是必填字段", path, key))
			}
		}
		properties, _ := schema["properties"].(map[string]any)
		for key, childSchema := range properties {
			child, ok := typed[key]
			if !ok {
				continue
			}
			childMap, _ := childSchema.(map[string]any)
			validateSchemaValue(child, childMap, path+"."+key, issues)
		}
		if allowed, ok := schema["additionalProperties"].(bool); ok && !allowed {
			for key := range typed {
				if _, known := properties[key]; !known {
					appendContractIssue(issues, "data_contract_additional_property", path+"."+key, fmt.Sprintf("%s.%s 不是契约允许的字段", path, key))
				}
			}
		}
		if minimum, ok := schemaNumber(schema["minProperties"]); ok && float64(len(typed)) < minimum {
			appendContractIssue(issues, "data_contract_min_properties", path, fmt.Sprintf("%s 至少需要 %d 个字段", path, int(minimum)))
		}
	case []any:
		if minimum, ok := schemaNumber(schema["minItems"]); ok && float64(len(typed)) < minimum {
			appendContractIssue(issues, "data_contract_min_items", path, fmt.Sprintf("%s 至少需要 %d 项", path, int(minimum)))
		}
		if maximum, ok := schemaNumber(schema["maxItems"]); ok && float64(len(typed)) > maximum {
			appendContractIssue(issues, "data_contract_max_items", path, fmt.Sprintf("%s 最多允许 %d 项", path, int(maximum)))
		}
		if itemSchema, ok := schema["items"].(map[string]any); ok {
			for index, child := range typed {
				validateSchemaValue(child, itemSchema, fmt.Sprintf("%s[%d]", path, index), issues)
				if len(*issues) >= maxContractIssues {
					break
				}
			}
		}
	case float64, float32, int, int64, int32, json.Number:
		number, _ := schemaNumber(value)
		if minimum, ok := schemaNumber(schema["minimum"]); ok && number < minimum {
			appendContractIssue(issues, "data_contract_minimum", path, fmt.Sprintf("%s 必须 ≥ %v", path, minimum))
		}
		if maximum, ok := schemaNumber(schema["maximum"]); ok && number > maximum {
			appendContractIssue(issues, "data_contract_maximum", path, fmt.Sprintf("%s 必须 ≤ %v", path, maximum))
		}
	}
}

func matchesSchemaType(value any, expected string) bool {
	switch expected {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := schemaNumber(value)
		return ok
	case "integer":
		number, ok := schemaNumber(value)
		return ok && math.Trunc(number) == number
	case "boolean":
		_, ok := value.(bool)
		return ok
	default:
		return true
	}
}

func schemaNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case json.Number:
		number, err := typed.Float64()
		return number, err == nil
	default:
		return 0, false
	}
}

func schemaStringList(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func appendContractIssue(issues *[]map[string]any, code, path, message string) {
	if len(*issues) >= maxContractIssues {
		return
	}
	*issues = append(*issues, issue(code, "recipe", message, fmt.Sprintf("按 data_contract 修改 %s", path)))
}
