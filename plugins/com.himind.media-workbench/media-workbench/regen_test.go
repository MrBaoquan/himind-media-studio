package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestRegenerateExampleDataset 用本扩展自己的 pipeline.geo.route 重新生成示例数据资产。
//
// 这是**资产构建步骤**而不是回归测试，因此默认跳过；需要时显式开启：
//
//	$env:HIMIND_REGEN_GEO=1; go test -run TestRegenerateExampleDataset -v
//
// 之所以做成可复跑的构建而不是一次性脚本：
// 资产必须能由扩展自身的能力重新产出，否则"沉淀"就变成了手工搬运。
func TestRegenerateExampleDataset(t *testing.T) {
	if os.Getenv("HIMIND_REGEN_GEO") != "1" {
		t.Skip("set HIMIND_REGEN_GEO=1 to regenerate catalog/data/geo/anhui-transit-lines.json")
	}

	path := filepath.Join(pluginRoot(), filepath.FromSlash(transitDatasetRelPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read dataset: %v", err)
	}
	var dataset transitDataset
	if err := json.Unmarshal(raw, &dataset); err != nil {
		t.Fatalf("parse dataset: %v", err)
	}

	policy := defaultRoutePolicy()
	changed := 0
	for i := range dataset.Lines {
		line := &dataset.Lines[i]
		if len(line.RepairedCoords) < 2 {
			continue
		}
		result := repairRoute(line.asRouteInput(), policy)
		if len(result.Coords) != len(line.RepairedCoords) {
			changed++
			t.Logf("%-14s %d -> %d points (loops removed: %d)", line.ID, len(line.RepairedCoords), len(result.Coords), result.LoopsRemoved)
		}
		line.RepairedCoords = result.Coords
	}

	if dataset.Provenance == nil {
		dataset.Provenance = map[string]any{}
	}
	dataset.Provenance["regenerated_by"] = "pipeline.geo.route.repair @ plugin " + pluginVersion()
	dataset.Provenance["repair_policy"] = policy

	encoded, err := json.MarshalIndent(dataset, "", " ")
	if err != nil {
		t.Fatalf("encode dataset: %v", err)
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		t.Fatalf("write dataset: %v", err)
	}
	t.Logf("regenerated %s (%d lines, %d changed)", transitDatasetRelPath, len(dataset.Lines), changed)

	// 确定性：再序列化一次必须逐字节相同
	again, _ := json.MarshalIndent(dataset, "", " ")
	if string(again) != string(encoded) {
		t.Fatal("dataset serialization is not deterministic")
	}
}

// pluginVersion 读取插件清单版本。
func pluginVersion() string {
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), "plugin.json"))
	if err != nil {
		return "unknown"
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(raw, &manifest) != nil {
		return "unknown"
	}
	return manifest.Version
}
