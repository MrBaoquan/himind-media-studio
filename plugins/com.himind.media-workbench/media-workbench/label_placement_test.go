package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type transitFixtureLine struct {
	ID       string      `json:"id"`
	Path     [][]float64 `json:"path"`
	Stations []string    `json:"stations"`
}

type transitFixtureStation struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

type transitFixture struct {
	Lines    []transitFixtureLine    `json:"lines"`
	Stations []transitFixtureStation `json:"stations"`
}

func loadTransitFixture(t *testing.T) transitFixture {
	t.Helper()
	path := filepath.Join("..", "..", "..", "style-packs", "transit-network", "template", "input", "data.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var data transitFixture
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return data
}

func TestLabelPlacementHefeiFixture(t *testing.T) {
	data := loadTransitFixture(t)
	lineCount := map[string]int{}
	terminals := map[string]bool{}
	for _, line := range data.Lines {
		for _, id := range line.Stations {
			lineCount[id]++
		}
		if len(line.Stations) > 0 {
			terminals[line.Stations[0]] = true
			terminals[line.Stations[len(line.Stations)-1]] = true
		}
	}
	labels := make([]labelInput, 0, len(data.Stations))
	for _, station := range data.Stations {
		priority := 20
		size := 22.0
		if lineCount[station.ID] > 1 {
			priority = 100
			size = 26
		} else if terminals[station.ID] {
			priority = 60
			size = 24
		}
		hideable := false
		leader := true
		labels = append(labels, labelInput{
			ID: station.ID, Text: station.Name, X: station.X, Y: station.Y,
			Priority: priority, FontSize: size, Hideable: &hideable, Leader: &leader,
		})
	}
	obstacles := []labelObstacle{}
	for _, line := range data.Lines {
		obstacles = append(obstacles, labelObstacle{Kind: "line", Points: line.Path, Width: 12})
	}
	for _, station := range data.Stations {
		radius := 7.0
		if lineCount[station.ID] > 1 {
			radius = 13
		}
		obstacles = append(obstacles, labelObstacle{Kind: "circle", X: station.X, Y: station.Y, R: radius})
	}
	vp := &labelViewport{Width: 1080, Height: 1920, SafeArea: []float64{60, 330, 960, 1150}}
	result, rpcErr := labelPlace(requestInput{Labels: labels, Obstacles: obstacles, Viewport: vp})
	if rpcErr != nil {
		t.Fatalf("label.place: %v", rpcErr)
	}
	payload, _ := json.Marshal(result)
	var parsed struct {
		Placements []labelPlacement `json:"placements"`
		Metrics    map[string]any   `json:"metrics"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		t.Fatalf("decode placements: %v", err)
	}
	visible := 0
	inputs := []labelPlacementInput{}
	for _, placement := range parsed.Placements {
		if !placement.Visible {
			continue
		}
		visible++
		inputs = append(inputs, labelPlacementInput{ID: placement.ID, Box: placement.Box, Leader: len(placement.Leader) > 0})
	}
	if visible != len(labels) {
		t.Fatalf("visible=%d, want %d; metrics=%v", visible, len(labels), parsed.Metrics)
	}
	validation, rpcErr := labelValidate(requestInput{PlacementInputs: inputs, Obstacles: obstacles, Viewport: vp})
	if rpcErr != nil {
		t.Fatalf("label.validate: %v", rpcErr)
	}
	report, _ := json.Marshal(validation)
	var validated struct {
		OK       bool           `json:"ok"`
		Findings []labelFinding `json:"findings"`
	}
	if err := json.Unmarshal(report, &validated); err != nil {
		t.Fatalf("decode validation: %v", err)
	}
	if !validated.OK || len(validated.Findings) != 0 {
		t.Fatalf("validation failed: %+v", validated.Findings)
	}
}

func TestLabelValidateDetectsOverlap(t *testing.T) {
	vp := &labelViewport{Width: 1080, Height: 1920, SafeArea: []float64{0, 0, 1080, 1920}}
	result, rpcErr := labelValidate(requestInput{
		Viewport: vp,
		PlacementInputs: []labelPlacementInput{
			{ID: "a", Box: []float64{100, 100, 120, 30}},
			{ID: "b", Box: []float64{150, 110, 120, 30}},
		},
	})
	if rpcErr != nil {
		t.Fatalf("label.validate: %v", rpcErr)
	}
	payload, _ := json.Marshal(result)
	var parsed struct {
		OK       bool           `json:"ok"`
		Findings []labelFinding `json:"findings"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if parsed.OK {
		t.Fatal("expected overlap to fail")
	}
	found := false
	for _, finding := range parsed.Findings {
		if finding.Code == "LABEL_COLLISION" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing LABEL_COLLISION: %+v", parsed.Findings)
	}
}
