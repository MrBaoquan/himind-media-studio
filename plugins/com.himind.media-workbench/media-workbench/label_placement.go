package main

import (
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type labelViewport struct {
	Width    float64   `json:"width"`
	Height   float64   `json:"height"`
	SafeArea []float64 `json:"safe_area"`
}

type labelInput struct {
	ID          string  `json:"id"`
	Text        string  `json:"text"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Priority    int     `json:"priority"`
	FontSize    float64 `json:"font_size"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	MaxDistance float64 `json:"max_distance"`
	Leader      *bool   `json:"leader"`
	Hideable    *bool   `json:"hideable"`
}

type labelObstacle struct {
	Kind   string      `json:"kind"`
	X      float64     `json:"x"`
	Y      float64     `json:"y"`
	W      float64     `json:"w"`
	H      float64     `json:"h"`
	R      float64     `json:"r"`
	Width  float64     `json:"width"`
	Points [][]float64 `json:"points"`
}

type labelPlacementInput struct {
	ID      string    `json:"id"`
	Box     []float64 `json:"box"`
	Visible *bool     `json:"visible"`
	Leader  bool      `json:"leader"`
}

type labelBox struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

type labelCandidate struct {
	DX     float64
	DY     float64
	Anchor string
	Align  string
}

type labelChoice struct {
	Box    labelBox
	Anchor string
	Align  string
	TextX  float64
	TextY  float64
}

type labelPlacement struct {
	ID      string      `json:"id"`
	Text    string      `json:"text"`
	Box     []float64   `json:"box"`
	Anchor  string      `json:"anchor"`
	Visible bool        `json:"visible"`
	Leader  [][]float64 `json:"leader,omitempty"`
	Score   float64     `json:"score"`
	Reason  string      `json:"reason,omitempty"`
}

type labelFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	LabelID  string `json:"label_id,omitempty"`
	Message  string `json:"message"`
}

func normalizeLabelViewport(vp *labelViewport) labelViewport {
	out := labelViewport{Width: 1080, Height: 1920}
	if vp != nil {
		if vp.Width > 0 {
			out.Width = vp.Width
		}
		if vp.Height > 0 {
			out.Height = vp.Height
		}
		if len(vp.SafeArea) == 4 {
			out.SafeArea = append([]float64(nil), vp.SafeArea...)
		}
	}
	if len(out.SafeArea) != 4 {
		out.SafeArea = []float64{0, 0, out.Width, out.Height}
	}
	return out
}

func safeAreaBounds(vp labelViewport) (float64, float64, float64, float64) {
	x, y, w, h := vp.SafeArea[0], vp.SafeArea[1], vp.SafeArea[2], vp.SafeArea[3]
	if w <= 0 {
		w = vp.Width - x
	}
	if h <= 0 {
		h = vp.Height - y
	}
	return x, y, x + w, y + h
}

func labelRuneWidth(r rune, size float64) float64 {
	if unicode.IsSpace(r) {
		return size * 0.35
	}
	if r < 128 {
		return size * 0.62
	}
	return size
}

func labelSize(label labelInput) (float64, float64) {
	size := label.FontSize
	if size <= 0 {
		size = 22
	}
	width := label.Width
	if width <= 0 {
		width = 0
		for _, r := range label.Text {
			width += labelRuneWidth(r, size)
		}
		width = math.Ceil(width)
	}
	height := label.Height
	if height <= 0 {
		height = math.Ceil(size * 1.35)
	}
	return width, height
}

func candidateBox(label labelInput, width, height float64, c labelCandidate) labelChoice {
	textX := label.X + c.DX
	textY := label.Y + c.DY
	x := textX
	if c.Align == "right" {
		x = textX - width
	}
	if c.Align == "center" {
		x = textX - width/2
	}
	return labelChoice{
		Box:    labelBox{X: x, Y: textY - height/2, W: width, H: height},
		Anchor: c.Anchor,
		Align:  c.Align,
		TextX:  textX,
		TextY:  textY,
	}
}

func labelCandidates(maxDistance float64) []labelCandidate {
	d := 16.0
	if maxDistance > 0 && maxDistance < d {
		d = maxDistance
	}
	return []labelCandidate{
		{d, 0, "right", "left"},
		{-d, 0, "left", "right"},
		{0, -24, "top", "center"},
		{0, 26, "bottom", "center"},
		{12, -18, "top-right", "left"},
		{12, 18, "bottom-right", "left"},
		{-12, -18, "top-left", "right"},
		{-12, 18, "bottom-left", "right"},
	}
}

func fallbackCandidates() []labelCandidate {
	return []labelCandidate{
		{0, -64, "top-far", "center"},
		{0, 64, "bottom-far", "center"},
		{-56, 0, "left-far", "right"},
		{56, 0, "right-far", "left"},
		{-50, -58, "top-left-far", "right"},
		{50, -58, "top-right-far", "left"},
		{-50, 58, "bottom-left-far", "right"},
		{50, 58, "bottom-right-far", "left"},
	}
}

func boxesOverlap(a, b labelBox, pad float64) bool {
	return !(a.X+a.W+pad < b.X || b.X+b.W+pad < a.X || a.Y+a.H+pad < b.Y || b.Y+b.H+pad < a.Y)
}

func pointInRect(x, y float64, b labelBox) bool {
	return x >= b.X && x <= b.X+b.W && y >= b.Y && y <= b.Y+b.H
}

func segmentIntersectsRect(x1, y1, x2, y2 float64, b labelBox) bool {
	if pointInRect(x1, y1, b) || pointInRect(x2, y2, b) {
		return true
	}
	edges := [][4]float64{
		{b.X, b.Y, b.X + b.W, b.Y},
		{b.X + b.W, b.Y, b.X + b.W, b.Y + b.H},
		{b.X + b.W, b.Y + b.H, b.X, b.Y + b.H},
		{b.X, b.Y + b.H, b.X, b.Y},
	}
	for _, e := range edges {
		ax, ay, bx, by := e[0], e[1], e[2], e[3]
		d1 := (x2-x1)*(ay-y1) - (y2-y1)*(ax-x1)
		d2 := (x2-x1)*(by-y1) - (y2-y1)*(bx-x1)
		d3 := (bx-ax)*(y1-ay) - (by-ay)*(x1-ax)
		d4 := (bx-ax)*(y2-ay) - (by-ay)*(x2-ax)
		if ((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) && ((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0)) {
			return true
		}
	}
	return false
}

func circleIntersectsRect(cx, cy, radius float64, b labelBox) bool {
	nx := math.Max(b.X, math.Min(cx, b.X+b.W))
	ny := math.Max(b.Y, math.Min(cy, b.Y+b.H))
	dx := cx - nx
	dy := cy - ny
	return dx*dx+dy*dy < radius*radius
}

func labelBoxSafe(box labelBox, placed []labelBox, obstacles []labelObstacle, vp labelViewport, ignoreLines bool) bool {
	x0, y0, x1, y1 := safeAreaBounds(vp)
	if box.X < x0+6 || box.Y < y0+6 || box.X+box.W > x1-6 || box.Y+box.H > y1-6 {
		return false
	}
	for _, other := range placed {
		if boxesOverlap(box, other, 6) {
			return false
		}
	}
	for _, obstacle := range obstacles {
		switch strings.ToLower(obstacle.Kind) {
		case "rect":
			if boxesOverlap(box, labelBox{X: obstacle.X, Y: obstacle.Y, W: obstacle.W, H: obstacle.H}, 3) {
				return false
			}
		case "circle":
			if circleIntersectsRect(obstacle.X, obstacle.Y, obstacle.R+3, box) {
				return false
			}
		case "line":
			if ignoreLines {
				continue
			}
			for i := 1; i < len(obstacle.Points); i++ {
				a, b := obstacle.Points[i-1], obstacle.Points[i]
				if len(a) < 2 || len(b) < 2 {
					continue
				}
				expanded := labelBox{X: box.X - 3, Y: box.Y - 3, W: box.W + 6, H: box.H + 6}
				if segmentIntersectsRect(a[0], a[1], b[0], b[1], expanded) {
					return false
				}
			}
		}
	}
	return true
}

func leaderPoints(label labelInput, box labelBox) [][]float64 {
	boxCenter := box.X + box.W/2
	edgeX := box.X
	if label.X > boxCenter {
		edgeX = box.X + box.W
	}
	edgeY := math.Max(box.Y, math.Min(label.Y, box.Y+box.H))
	return [][]float64{{label.X, label.Y}, {edgeX, edgeY}}
}

func labelPlace(in requestInput) (any, *himindjsonrpc.Error) {
	if len(in.Labels) == 0 {
		return nil, himindjsonrpc.InvalidParams("labels is required")
	}
	started := time.Now()
	vp := normalizeLabelViewport(in.Viewport)
	labels := append([]labelInput(nil), in.Labels...)
	sort.SliceStable(labels, func(i, j int) bool {
		if labels[i].Priority != labels[j].Priority {
			return labels[i].Priority > labels[j].Priority
		}
		if labels[i].Y != labels[j].Y {
			return labels[i].Y < labels[j].Y
		}
		return labels[i].X < labels[j].X
	})
	placed := []labelBox{}
	placements := []labelPlacement{}
	hidden := 0
	for _, label := range labels {
		w, h := labelSize(label)
		var chosen *labelChoice
		for _, c := range labelCandidates(label.MaxDistance) {
			choice := candidateBox(label, w, h, c)
			if labelBoxSafe(choice.Box, placed, in.Obstacles, vp, false) {
				chosen = &choice
				break
			}
		}
		hideable := true
		if label.Hideable != nil {
			hideable = *label.Hideable
		}
		if chosen == nil && !hideable {
			for _, c := range fallbackCandidates() {
				choice := candidateBox(label, w, h, c)
				if labelBoxSafe(choice.Box, placed, in.Obstacles, vp, true) {
					chosen = &choice
					break
				}
			}
		}
		if chosen == nil {
			hidden++
			placements = append(placements, labelPlacement{ID: label.ID, Text: label.Text, Visible: false, Reason: "NO_SAFE_POSITION"})
			continue
		}
		placed = append(placed, chosen.Box)
		placement := labelPlacement{
			ID: label.ID, Text: label.Text,
			Box:    []float64{chosen.Box.X, chosen.Box.Y, chosen.Box.W, chosen.Box.H},
			Anchor: chosen.Anchor, Visible: true, Score: 1,
		}
		wantLeader := true
		if label.Leader != nil {
			wantLeader = *label.Leader
		}
		if wantLeader && math.Hypot(chosen.TextX-label.X, chosen.TextY-label.Y) > 30 {
			placement.Leader = leaderPoints(label, chosen.Box)
		}
		placements = append(placements, placement)
	}
	return map[string]any{
		"ok":         true,
		"viewport":   vp,
		"placements": placements,
		"metrics": map[string]any{
			"labels": len(labels), "visible": len(labels) - hidden, "hidden": hidden,
			"collisions": 0, "out_of_bounds": 0,
			"duration_ms": time.Since(started).Milliseconds(),
		},
	}, nil
}

func labelValidate(in requestInput) (any, *himindjsonrpc.Error) {
	if len(in.PlacementInputs) == 0 {
		return nil, himindjsonrpc.InvalidParams("placements is required")
	}
	vp := normalizeLabelViewport(in.Viewport)
	findings := []labelFinding{}
	boxes := []labelBox{}
	ids := []string{}
	visible := 0
	for _, item := range in.PlacementInputs {
		if item.Visible != nil && !*item.Visible {
			continue
		}
		visible++
		if len(item.Box) != 4 {
			findings = append(findings, labelFinding{Code: "LABEL_BOX_INVALID", Severity: "error", LabelID: item.ID, Message: "box must be [x,y,w,h]"})
			continue
		}
		box := labelBox{X: item.Box[0], Y: item.Box[1], W: item.Box[2], H: item.Box[3]}
		x0, y0, x1, y1 := safeAreaBounds(vp)
		if box.X < x0 || box.Y < y0 || box.X+box.W > x1 || box.Y+box.H > y1 {
			findings = append(findings, labelFinding{Code: "LABEL_OUT_OF_BOUNDS", Severity: "error", LabelID: item.ID, Message: "label box exceeds safe area"})
		}
		for i, other := range boxes {
			if boxesOverlap(box, other, 0) {
				findings = append(findings, labelFinding{Code: "LABEL_COLLISION", Severity: "error", LabelID: item.ID, Message: "label collides with " + ids[i]})
			}
		}
		for _, obstacle := range in.Obstacles {
			switch strings.ToLower(obstacle.Kind) {
			case "rect":
				if boxesOverlap(box, labelBox{X: obstacle.X, Y: obstacle.Y, W: obstacle.W, H: obstacle.H}, 0) {
					findings = append(findings, labelFinding{Code: "LABEL_OBSTACLE", Severity: "error", LabelID: item.ID, Message: "label overlaps rect obstacle"})
				}
			case "circle":
				if circleIntersectsRect(obstacle.X, obstacle.Y, obstacle.R, box) {
					findings = append(findings, labelFinding{Code: "LABEL_OBSTACLE", Severity: "error", LabelID: item.ID, Message: "label overlaps circle obstacle"})
				}
			case "line":
				if item.Leader {
					continue
				}
				for i := 1; i < len(obstacle.Points); i++ {
					a, b := obstacle.Points[i-1], obstacle.Points[i]
					if len(a) < 2 || len(b) < 2 {
						continue
					}
					if segmentIntersectsRect(a[0], a[1], b[0], b[1], box) {
						findings = append(findings, labelFinding{Code: "LABEL_OBSTACLE", Severity: "error", LabelID: item.ID, Message: "label crosses line obstacle"})
						break
					}
				}
			}
		}
		boxes = append(boxes, box)
		ids = append(ids, item.ID)
	}
	ok := true
	for _, f := range findings {
		if f.Severity == "error" {
			ok = false
			break
		}
	}
	return map[string]any{
		"ok":       ok,
		"findings": findings,
		"metrics":  map[string]any{"placements": len(in.PlacementInputs), "visible": visible, "findings": len(findings)},
	}, nil
}

func labelCount(text string) int { return utf8.RuneCountInString(text) }
