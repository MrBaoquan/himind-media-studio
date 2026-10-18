package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type bandingLine struct {
	Label    string           `json:"label"`
	Unique   int              `json:"unique_levels"`
	MaxRunPX int              `json:"max_flat_run_px"`
	OK       bool             `json:"ok"`
	Runs     []map[string]any `json:"runs,omitempty"`
}

type bandingReport struct {
	OK          bool             `json:"ok"`
	Input       string           `json:"input,omitempty"`
	Width       int              `json:"width"`
	Height      int              `json:"height"`
	Time        float64          `json:"time,omitempty"`
	Thresholds  map[string]any   `json:"thresholds"`
	Lines       []bandingLine    `json:"lines"`
	Findings    []map[string]any `json:"findings,omitempty"`
	Remediation string           `json:"remediation,omitempty"`
}

// qualityBanding quantifies dark-gradient banding on an image or a video frame.
// It complements the renderer's PNG default: the renderer prevents the bug, the
// check proves the delivered frame does not contain flat color bands.
func qualityBanding(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	imagePath := strings.TrimSpace(in.ImagePath)
	videoPath := strings.TrimSpace(in.VideoPath)
	if imagePath == "" && videoPath == "" {
		return nil, himindjsonrpc.InvalidParams("image_path or video_path is required")
	}
	var img image.Image
	input := ""
	seconds := in.TimeSeconds
	if imagePath != "" {
		path, pathErr := workspaceFile(root, imagePath)
		if pathErr != nil {
			return nil, himindjsonrpc.InvalidParams("image_path: " + pathErr.Error())
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return nil, himindjsonrpc.InvalidParams("image_path: " + openErr.Error())
		}
		defer file.Close()
		decoded, _, decodeErr := image.Decode(file)
		if decodeErr != nil {
			return nil, himindjsonrpc.InvalidParams("decode image: " + decodeErr.Error())
		}
		img = decoded
		input = path
	} else {
		path, pathErr := workspaceFile(root, videoPath)
		if pathErr != nil {
			return nil, himindjsonrpc.InvalidParams("video_path: " + pathErr.Error())
		}
		if seconds <= 0 {
			seconds = 9
		}
		decoded, decodeErr := decodeVideoFrame(path, seconds)
		if decodeErr != nil {
			return nil, himindjsonrpc.InvalidParams("decode video frame: " + decodeErr.Error())
		}
		img = decoded
		input = path
	}
	report := analyzeBanding(img, in.MaxRunRatio, in.MinLevels)
	report.Input = input
	report.Time = seconds
	return report, nil
}

func decodeVideoFrame(path string, seconds float64) (image.Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-ss", strconv.FormatFloat(seconds, 'f', 3, 64),
		"-i", path,
		"-frames:v", "1",
		"-f", "image2pipe",
		"-vcodec", "png",
		"-",
	)
	output, err := command.Output()
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(output))
	return img, err
}

func analyzeBanding(img image.Image, ratio float64, minLevels int) bandingReport {
	if ratio <= 0 || ratio > 1 {
		ratio = 0.08
	}
	if minLevels <= 0 {
		minLevels = 12
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	maxRunLimit := int(math.Max(24, math.Round(float64(width)*ratio)))
	lines := make([]bandingLine, 0, 4)
	appendLine := func(label string, points [][2]int) {
		values := make([]float64, 0, len(points))
		for _, point := range points {
			values = append(values, pixelLuma(img, point[0], point[1]))
		}
		lines = append(lines, analyzeBandingLine(label, values, maxRunLimit, minLevels))
	}
	for _, y := range []int{
		bounds.Min.Y + int(float64(height)*0.26),
		bounds.Min.Y + int(float64(height)*0.74),
	} {
		points := make([][2]int, 0, width)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			points = append(points, [2]int{x, y})
		}
		appendLine(fmt.Sprintf("y=%d", y), points)
	}
	for _, x := range []int{
		bounds.Min.X + int(float64(width)*0.02),
		bounds.Max.X - 1 - int(float64(width)*0.02),
	} {
		points := make([][2]int, 0, height)
		start := bounds.Min.Y + int(float64(height)*0.05)
		end := bounds.Max.Y - int(float64(height)*0.05)
		for y := start; y < end; y++ {
			points = append(points, [2]int{x, y})
		}
		appendLine(fmt.Sprintf("x=%d", x), points)
	}
	findings := make([]map[string]any, 0)
	for _, line := range lines {
		if line.OK {
			continue
		}
		findings = append(findings, issue(
			"DARK_GRADIENT_BANDING",
			"output",
			fmt.Sprintf("%s 出现 %d 级亮度、最大平坦段 %dpx（上限 %dpx）", line.Label, line.Unique, line.MaxRunPX, maxRunLimit),
			"最终帧改用 PNG；移除不必要的 CSS contrast/vignette；暗部加 dither；提高 Web copy 码率",
		))
	}
	report := bandingReport{
		OK:       len(findings) == 0,
		Width:    width,
		Height:   height,
		Lines:    lines,
		Findings: findings,
		Thresholds: map[string]any{
			"max_run_ratio":    ratio,
			"max_run_limit_px": maxRunLimit,
			"min_levels":       minLevels,
		},
	}
	if !report.OK {
		report.Remediation = "最终帧必须使用 PNG；暗部渐变需要 dither；Web copy 码率只能兜底，不能替代母版修复。"
	}
	return report
}

func analyzeBandingLine(label string, values []float64, maxRunLimit, minLevels int) bandingLine {
	unique := make(map[int]struct{}, len(values))
	runs := make([]map[string]any, 0)
	maxRun := 0
	start := 0
	for i := 1; i <= len(values); i++ {
		if i < len(values) && math.Abs(values[i]-values[start]) <= 0.5 {
			continue
		}
		length := i - start
		if length > maxRun {
			maxRun = length
		}
		unique[int(math.Round(values[start]))] = struct{}{}
		if length >= 20 {
			runs = append(runs, map[string]any{
				"value":  math.Round(values[start]*100) / 100,
				"length": length,
			})
		}
		start = i
	}
	if len(runs) > 12 {
		runs = runs[:12]
	}
	return bandingLine{
		Label:    label,
		Unique:   len(unique),
		MaxRunPX: maxRun,
		OK:       len(unique) >= minLevels && maxRun <= maxRunLimit,
		Runs:     runs,
	}
}

func pixelLuma(img image.Image, x, y int) float64 {
	r, g, b, _ := img.At(x, y).RGBA()
	return 0.2126*float64(r>>8) + 0.7152*float64(g>>8) + 0.0722*float64(b>>8)
}
