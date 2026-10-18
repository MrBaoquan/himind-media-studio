package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// dataset.go — 随扩展分发的示例数据资产。
//
// 形态参考 ComfyUI 的 example_workflows：能力自带一个**能跑的示例**，
// 让 AI 通过读示例学会怎么用，而不是靠读文档。

type transitStation struct {
	Name     string     `json:"name"`
	LonLat   [2]float64 `json:"lonLat"`
	IsMain   bool       `json:"isMain"`
	LabelPos string     `json:"labelPos"`
}

type transitLine struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Color           string           `json:"color"`
	Speed           int              `json:"speed"`
	OpenYear        int              `json:"openYear"`
	Priority        int              `json:"priority"`
	LengthKm        float64          `json:"lengthKm"`
	PassingCities   []string         `json:"passingCities"`
	LabelConfig     map[string]any   `json:"labelConfig"`
	Stations        []transitStation `json:"stations"`
	RouteCoords     [][2]float64     `json:"routeCoords"`
	ExtensionBefore [][2]float64     `json:"extensionBefore"`
	ExtensionAfter  [][2]float64     `json:"extensionAfter"`
	RepairedCoords  [][2]float64     `json:"repairedCoords"`
}

type transitDataset struct {
	Schema     string         `json:"schema"`
	Region     map[string]any `json:"region"`
	Provenance map[string]any `json:"provenance"`
	Lines      []transitLine  `json:"lines"`
}

// bestCoords 返回该线路可用的几何：优先已修复坐标，其次手绘坐标。
func (l transitLine) bestCoords() [][2]float64 {
	if len(l.RepairedCoords) >= 2 {
		return l.RepairedCoords
	}
	return l.RouteCoords
}

// asRouteInput 把数据资产条目转成 route.* 能力入参。
func (l transitLine) asRouteInput() routeInput {
	waypoints := make([]waypointInput, 0, len(l.Stations))
	for _, s := range l.Stations {
		waypoints = append(waypoints, waypointInput{Name: s.Name, LonLat: s.LonLat})
	}
	return routeInput{ID: l.ID, Coords: l.bestCoords(), Waypoints: waypoints}
}

const transitDatasetRelPath = "catalog/data/geo/anhui-transit-lines.json"

var (
	transitDatasetOnce sync.Once
	transitDatasetMemo transitDataset
	transitDatasetErr  error
)

// loadTransitDataset 读取随包分发的示例数据集（带进程内缓存）。
func loadTransitDataset() (transitDataset, error) {
	transitDatasetOnce.Do(func() {
		path := filepath.Join(pluginRoot(), filepath.FromSlash(transitDatasetRelPath))
		content, err := os.ReadFile(path)
		if err != nil {
			transitDatasetErr = err
			return
		}
		if err := json.Unmarshal(content, &transitDatasetMemo); err != nil {
			transitDatasetErr = err
		}
	})
	return transitDatasetMemo, transitDatasetErr
}
