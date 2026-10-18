package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// data_ingest.go —— 把用户数据文件变成可绑定的 items。
//
// 这是数据入口，不是领域模板：它只负责读取、识别列、数值化和基础校验。
// 真正的排名/地图/占比布局仍由 Shot 和 Recipe 负责。

type ingestedItem struct {
	Name   string  `json:"name"`
	Value  float64 `json:"value"`
	Unit   string  `json:"unit,omitempty"`
	Period string  `json:"period,omitempty"`
	Source string  `json:"source,omitempty"`
}

func dataIngest(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if strings.TrimSpace(in.Path) == "" {
		return nil, himindjsonrpc.InvalidParams("path is required")
	}
	path, err := workspaceFile(root, in.Path)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("read data file: " + err.Error())
	}
	ext := strings.ToLower(filepath.Ext(path))
	var items []ingestedItem
	issues := make([]map[string]any, 0)
	switch ext {
	case ".json":
		items, issues = ingestJSON(raw, in)
	case ".csv", ".tsv":
		items, issues = ingestCSV(raw, in, ext == ".tsv")
	default:
		return nil, himindjsonrpc.InvalidParams("data file must be .csv, .tsv or .json")
	}
	if len(issues) > 0 {
		return map[string]any{"valid": false, "issues": issues, "count": len(items), "items": items}, nil
	}
	ranked := append([]ingestedItem(nil), items...)
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].Value > ranked[j].Value })
	return map[string]any{
		"valid":        true,
		"count":        len(items),
		"items":        items,
		"ranked_items": ranked,
		"next":         "把 items 作为 data.items 传给 generate.run，或让 AI 选择 leaderboard/ranking 镜头",
	}, nil
}

func ingestCSV(raw []byte, in requestInput, tab bool) ([]ingestedItem, []map[string]any) {
	reader := csv.NewReader(strings.NewReader(string(raw)))
	if tab {
		reader.Comma = '\t'
	}
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, []map[string]any{issue("data_ingest_parse_failed", "input", "CSV 解析失败: "+err.Error(), "检查分隔符、引号和编码")}
	}
	if len(records) < 2 {
		return nil, []map[string]any{issue("data_ingest_empty", "input", "CSV 至少需要表头和一行数据", "补充数据行")}
	}
	header := make([]string, len(records[0]))
	for i, cell := range records[0] {
		header[i] = strings.TrimSpace(strings.TrimPrefix(cell, "\ufeff"))
	}
	nameAt := columnIndex(header, in.NameColumn, []string{"name", "city", "城市", "名称", "label", "省份"})
	valueAt := columnIndex(header, in.ValueColumn, []string{"value", "gdp", "GDP", "数值", "总量", "地区生产总值"})
	if nameAt < 0 || valueAt < 0 {
		return nil, []map[string]any{issue("data_ingest_columns_missing", "input", "无法识别名称或数值列", "用 name_column / value_column 指定表头，或使用 name/value 表头")}
	}
	unitAt := columnIndex(header, "", []string{"unit", "单位"})
	periodAt := columnIndex(header, "", []string{"period", "year", "年份", "年度"})
	sourceAt := columnIndex(header, "", []string{"source", "来源"})
	items := make([]ingestedItem, 0, len(records)-1)
	issues := make([]map[string]any, 0)
	for index, record := range records[1:] {
		if len(record) == 0 || strings.TrimSpace(strings.Join(record, "")) == "" {
			continue
		}
		row := index + 2
		name := cellAt(record, nameAt)
		valueText := strings.ReplaceAll(cellAt(record, valueAt), ",", "")
		valueText = strings.TrimSpace(strings.TrimSuffix(valueText, "%"))
		value, parseErr := strconv.ParseFloat(valueText, 64)
		if name == "" {
			issues = append(issues, issue("data_ingest_name_missing", "input", fmt.Sprintf("第 %d 行缺少名称", row), "补名称或删除空行"))
			continue
		}
		if parseErr != nil {
			issues = append(issues, issue("data_ingest_value_invalid", "input", fmt.Sprintf("第 %d 行数值无效: %s", row, valueText), "数值列必须是数字"))
			continue
		}
		items = append(items, ingestedItem{
			Name:   name,
			Value:  value,
			Unit:   firstNonEmpty(cellAt(record, unitAt), in.Unit),
			Period: firstNonEmpty(cellAt(record, periodAt), in.Period),
			Source: firstNonEmpty(cellAt(record, sourceAt), in.Source),
		})
	}
	if len(items) > 500 {
		items = items[:500]
	}
	return items, issues
}

func ingestJSON(raw []byte, in requestInput) ([]ingestedItem, []map[string]any) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, []map[string]any{issue("data_ingest_parse_failed", "input", "JSON 解析失败: "+err.Error(), "检查 JSON 语法")}
	}
	var rawItems []any
	switch typed := value.(type) {
	case []any:
		rawItems = typed
	case map[string]any:
		if list, ok := typed["items"].([]any); ok {
			rawItems = list
		} else {
			for name, rawValue := range typed {
				rawItems = append(rawItems, map[string]any{"name": name, "value": rawValue})
			}
		}
	default:
		return nil, []map[string]any{issue("data_ingest_shape_invalid", "input", "JSON 必须是数组、{items: []} 或 {名称: 数值}", "调整数据文件结构")}
	}
	items := make([]ingestedItem, 0, len(rawItems))
	issues := make([]map[string]any, 0)
	for index, rawItem := range rawItems {
		object, ok := rawItem.(map[string]any)
		if !ok {
			issues = append(issues, issue("data_ingest_item_invalid", "input", fmt.Sprintf("第 %d 项不是对象", index+1), "每项必须是 {name,value}"))
			continue
		}
		name := firstNonEmpty(asString(object["name"]), asString(object["label"]), asString(object["city"]), asString(object["城市"]))
		number, ok := schemaNumber(object["value"])
		if !ok {
			number, ok = schemaNumber(object["gdp"])
		}
		if name == "" || !ok {
			issues = append(issues, issue("data_ingest_item_invalid", "input", fmt.Sprintf("第 %d 项缺少名称或数值", index+1), "每项必须包含 name 和 value"))
			continue
		}
		items = append(items, ingestedItem{
			Name:   name,
			Value:  number,
			Unit:   firstNonEmpty(asString(object["unit"]), in.Unit),
			Period: firstNonEmpty(asString(object["period"]), in.Period),
			Source: firstNonEmpty(asString(object["source"]), in.Source),
		})
	}
	if len(items) > 500 {
		items = items[:500]
	}
	return items, issues
}

func columnIndex(header []string, explicit string, aliases []string) int {
	if explicit != "" {
		for i, cell := range header {
			if strings.EqualFold(strings.TrimSpace(cell), strings.TrimSpace(explicit)) {
				return i
			}
		}
		return -1
	}
	for i, cell := range header {
		normalized := strings.ToLower(strings.TrimSpace(cell))
		for _, alias := range aliases {
			if normalized == strings.ToLower(alias) {
				return i
			}
		}
	}
	return -1
}

func cellAt(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
