package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultRecipesMatchDeclaredContracts 守住"模板声明的 data 契约"。
//
// 每个模板可以声明 data_contract（JSON Schema）。这里读取 schema 的 required，
// 断言新建项目时生成的默认数据包含这些键 —— 否则用户新建项目就会看到一个
// 数据不完整的空壳。
func TestDefaultRecipesMatchDeclaredContracts(t *testing.T) {
	checked := 0
	for _, tpl := range loadTemplates() {
		if tpl.DataContract == "" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(pluginRoot(), filepath.FromSlash(tpl.DataContract)))
		if err != nil {
			t.Errorf("%s: cannot read data contract %s: %v", tpl.ID, tpl.DataContract, err)
			continue
		}
		var schema struct {
			Required []string `json:"required"`
		}
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Errorf("%s: invalid data contract JSON: %v", tpl.ID, err)
			continue
		}

		recipe := defaultRecipe(tpl.ID, "tech-blue", nil)
		data, _ := recipe["data"].(map[string]any)
		if data == nil {
			t.Errorf("%s: default recipe has no data", tpl.ID)
			continue
		}
		for _, key := range schema.Required {
			if _, ok := data[key]; !ok {
				t.Errorf("%s: default recipe data is missing required key %q", tpl.ID, key)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no template declares a data_contract; contracts would go unchecked")
	}
	t.Logf("%d templates verified against their declared data contracts", checked)
}
