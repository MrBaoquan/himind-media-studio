package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"himind-plugin/media-visual/internal/himindjsonrpc"
)

type contactCell struct {
	Index int     `json:"index"`
	Time  float64 `json:"time_seconds"`
	Row   int     `json:"row"`
	Col   int     `json:"col"`
}

func videoContactSheetCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	videoPath, err := workspaceFile(root, in.VideoPath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("video_path: " + err.Error())
	}
	info, statErr := os.Stat(videoPath)
	if statErr != nil || info.IsDir() || info.Size() == 0 {
		return nil, himindjsonrpc.InvalidParams("video_path must be a non-empty file")
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, himindjsonrpc.InternalError("ffmpeg not found in PATH")
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, himindjsonrpc.InternalError("ffprobe not found in PATH")
	}
	probe, probeErr := probeVideo(ffprobe, videoPath)
	if probeErr != nil {
		return nil, himindjsonrpc.InvalidParams("probe video: " + probeErr.Error())
	}
	times := parseFrameTimes(in.Times)
	if len(times) == 0 {
		times = defaultFrameTimes(probe.Duration, in.SampleCount)
	}
	if len(times) == 0 {
		return nil, himindjsonrpc.InvalidParams("no valid sample times")
	}
	if len(times) > 16 {
		times = times[:16]
	}
	cols := in.Columns
	if cols <= 0 {
		cols = 3
	}
	if cols > 6 {
		cols = 6
	}
	cellWidth := in.CellWidth
	if cellWidth <= 0 {
		cellWidth = 320
	}
	if cellWidth < 80 {
		cellWidth = 80
	}
	if cellWidth > 960 {
		cellWidth = 960
	}
	tempDir := filepath.Join(root, ".himind-media", "qa", fmt.Sprintf("sheet-%d", time.Now().UnixNano()))
	if mkErr := os.MkdirAll(tempDir, 0755); mkErr != nil {
		return nil, himindjsonrpc.InternalError(mkErr.Error())
	}
	defer os.RemoveAll(tempDir)
	images := make([]image.Image, 0, len(times))
	for index, at := range times {
		framePath := filepath.Join(tempDir, fmt.Sprintf("frame-%02d.png", index+1))
		if extractErr := extractVideoFrame(ffmpeg, videoPath, at, framePath); extractErr != nil {
			return nil, himindjsonrpc.InternalError(fmt.Sprintf("%.2fs 抽帧失败: %v", at, extractErr))
		}
		img, decodeErr := decodeImage(framePath)
		if decodeErr != nil {
			return nil, himindjsonrpc.InternalError(fmt.Sprintf("%.2fs 抽帧无法解码: %v", at, decodeErr))
		}
		images = append(images, img)
	}
	sheet, cells, cellHeight := composeContactSheet(images, times, cols, cellWidth)
	outputPath := in.OutputPath
	if outputPath == "" {
		outputPath = filepath.ToSlash(filepath.Join(".himind-media", "qa", fmt.Sprintf("contact-sheet-%d.png", time.Now().UnixNano())))
	}
	resolvedOutput, err := workspaceFile(root, outputPath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("output_path: " + err.Error())
	}
	if mkErr := os.MkdirAll(filepath.Dir(resolvedOutput), 0755); mkErr != nil {
		return nil, himindjsonrpc.InternalError(mkErr.Error())
	}
	file, createErr := os.Create(resolvedOutput)
	if createErr != nil {
		return nil, himindjsonrpc.InternalError(createErr.Error())
	}
	if encodeErr := png.Encode(file, sheet); encodeErr != nil {
		_ = file.Close()
		return nil, himindjsonrpc.InternalError(encodeErr.Error())
	}
	if closeErr := file.Close(); closeErr != nil {
		return nil, himindjsonrpc.InternalError(closeErr.Error())
	}
	hash, _ := fileSHA256(resolvedOutput)
	fileInfo, _ := os.Stat(resolvedOutput)
	size := int64(0)
	if fileInfo != nil {
		size = fileInfo.Size()
	}
	return map[string]any{
		"state": "ready", "ok": true, "probe": probe,
		"path": workspaceRelative(root, resolvedOutput), "sha256": hash, "size": size,
		"width": sheet.Bounds().Dx(), "height": sheet.Bounds().Dy(),
		"cols": cols, "rows": (len(times) + cols - 1) / cols, "cell_width": cellWidth, "cell_height": cellHeight,
		"cells": cells,
		"note":  "联系表用于一张图审全片；cells 给出每个格子的时间映射，可按索引回溯原片",
	}, nil
}

func extractVideoFrame(ffmpeg, videoPath string, at float64, outPath string) error {
	command := exec.Command(ffmpeg, "-hide_banner", "-loglevel", "error",
		"-ss", fmt.Sprintf("%.3f", at), "-i", videoPath,
		"-frames:v", "1", "-y", outPath)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}
	return nil
}

func composeContactSheet(frames []image.Image, times []float64, cols, cellWidth int) (*image.RGBA, []contactCell, int) {
	cellHeight := 0
	if len(frames) > 0 {
		bounds := frames[0].Bounds()
		cellHeight = int(float64(cellWidth) * float64(bounds.Dy()) / float64(bounds.Dx()))
	}
	if cellHeight < 1 {
		cellHeight = 1
	}
	rows := (len(frames) + cols - 1) / cols
	sheet := image.NewRGBA(image.Rect(0, 0, cols*cellWidth, rows*cellHeight))
	for y := 0; y < sheet.Bounds().Dy(); y++ {
		for x := 0; x < sheet.Bounds().Dx(); x++ {
			sheet.Set(x, y, color.RGBA{R: 12, G: 14, B: 18, A: 255})
		}
	}
	cells := make([]contactCell, 0, len(frames))
	for index, frame := range frames {
		col := index % cols
		row := index / cols
		bounds := frame.Bounds()
		scale := float64(cellWidth) / float64(bounds.Dx())
		drawW := cellWidth
		drawH := int(float64(bounds.Dy()) * scale)
		if drawH > cellHeight {
			drawH = cellHeight
			drawW = int(float64(bounds.Dx()) * float64(cellHeight) / float64(bounds.Dy()))
		}
		x0 := col*cellWidth + (cellWidth-drawW)/2
		y0 := row*cellHeight + (cellHeight-drawH)/2
		drawScaled(sheet, frame, x0, y0, drawW, drawH)
		cells = append(cells, contactCell{Index: index + 1, Time: times[index], Row: row + 1, Col: col + 1})
	}
	return sheet, cells, cellHeight
}

func drawScaled(dst *image.RGBA, src image.Image, x0, y0, width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	bounds := src.Bounds()
	for y := 0; y < height; y++ {
		sy := bounds.Min.Y + y*bounds.Dy()/height
		for x := 0; x < width; x++ {
			sx := bounds.Min.X + x*bounds.Dx()/width
			dst.Set(x0+x, y0+y, src.At(sx, sy))
		}
	}
}
