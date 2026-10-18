package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// recipe_refs.go —— 配方里的数据资产引用。
//
// 大块数据（GeoJSON 等）不应内联进每个项目。配方可以写：
//
//	"geo": {"$ref": "plugin:catalog/data/geo/anhui-cities.geojson"}
//	"values": {"$ref": "workspace:assets/values.json"}
//
// **物化发生在渲染/预览前**：作业快照里永远是完整数据，
// 渲染器不需要知道引用这件事，"同配方同输出"也不受影响。
//
// 边界：plugin: 只能读插件包内，workspace: 只能读 workspace_root 内。

const maxRecipeRefDepth = 3

func resolveRecipeRef(ref string, root string, depth int) (any, error) {
	if depth > maxRecipeRefDepth {
		return nil, fmt.Errorf("引用嵌套超过 %d 层：%s", maxRecipeRefDepth, ref)
	}

	trimmed := strings.TrimSpace(ref)
	var base string
	switch {
	case strings.HasPrefix(trimmed, "plugin:"):
		base = pluginRoot()
		trimmed = strings.TrimPrefix(trimmed, "plugin:")
	case strings.HasPrefix(trimmed, "workspace:"):
		base = root
		trimmed = strings.TrimPrefix(trimmed, "workspace:")
	default:
		return nil, fmt.Errorf("引用 %q 必须以 plugin: 或 workspace: 开头", ref)
	}

	path, err := workspaceFile(base, trimmed)
	if err != nil {
		return nil, fmt.Errorf("引用 %q 越界或无效: %w", ref, err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("引用 %q 读取失败: %w", ref, err)
	}
	var value any
	if err := json.Unmarshal(content, &value); err != nil {
		return nil, fmt.Errorf("引用 %q 不是合法 JSON: %w", ref, err)
	}
	return materializeRefs(value, root, depth+1)
}

func materializeRefs(value any, root string, depth int) (any, error) {
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 1 {
			if raw, ok := typed["$ref"]; ok {
				ref, isString := raw.(string)
				if !isString {
					return nil, fmt.Errorf("$ref 必须是字符串")
				}
				return resolveRecipeRef(ref, root, depth)
			}
		}
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			resolved, err := materializeRefs(child, root, depth)
			if err != nil {
				return nil, err
			}
			out[key] = resolved
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for index, child := range typed {
			resolved, err := materializeRefs(child, root, depth)
			if err != nil {
				return nil, err
			}
			out[index] = resolved
		}
		return out, nil
	default:
		return value, nil
	}
}

// materializeRecipe 返回一份把所有 $ref 展开后的新配方。
func materializeRecipe(root string, recipe map[string]any) (map[string]any, error) {
	resolved, err := materializeRefs(recipe, root, 0)
	if err != nil {
		return nil, err
	}
	out, ok := resolved.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("recipe 必须是对象")
	}
	return out, nil
}
