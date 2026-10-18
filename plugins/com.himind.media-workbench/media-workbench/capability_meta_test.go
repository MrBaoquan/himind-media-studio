package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"
)

// TestCapabilityDescriptionsFollowMetaguide 守住能力描述长度。
//
// 原因不只是规范：内核会把**能力描述**直接用作审批弹窗标题
// （`受控操作审批：<descriptor.name>`，插件的 name 回退到 description）。
// 描述写成功能清单，弹窗标题就会长到看不清重点。
//
// 上限：48 字（扩展仓 metaguide 的 CapabilityDescription 上限）。
func TestCapabilityDescriptionsFollowMetaguide(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), "plugin.json"))
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}
	var manifest struct {
		Capabilities []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
			RiskLevel   string `json:"risk_level"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse plugin.json: %v", err)
	}

	const limit = 48
	over := 0
	for _, cap := range manifest.Capabilities {
		n := utf8.RuneCountInString(cap.Description)
		if n == 0 {
			t.Errorf("%s: description is empty", cap.ID)
			continue
		}
		if n > limit {
			over++
			t.Errorf("%s: description is %d chars (> %d); it becomes the approval dialog title", cap.ID, n, limit)
		}
	}
	if over == 0 {
		t.Logf("%d capabilities, all descriptions within %d chars", len(manifest.Capabilities), limit)
	}
}
