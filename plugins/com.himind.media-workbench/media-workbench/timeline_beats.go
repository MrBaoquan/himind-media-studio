package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os/exec"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func detectBeats(path string) ([]float64, error) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found in PATH")
	}
	command := exec.Command(ffmpeg, "-hide_banner", "-loglevel", "error", "-i", path, "-ac", "1", "-ar", "22050", "-f", "f32le", "-")
	output, err := command.Output()
	if err != nil {
		return nil, err
	}
	if len(output) < 4 {
		return []float64{}, nil
	}
	frameSamples := 441
	hopSamples := 220
	energies := make([]float64, 0)
	for offset := 0; offset+frameSamples <= len(output)/4; offset += hopSamples {
		energy := 0.0
		for sample := 0; sample < frameSamples; sample++ {
			bits := binary.LittleEndian.Uint32(output[(offset+sample)*4:])
			value := float64(math.Float32frombits(bits))
			energy += value * value
		}
		energies = append(energies, energy/float64(frameSamples))
	}
	beats := make([]float64, 0)
	window := 100
	lastBeat := -1.0
	for index, energy := range energies {
		if index < 8 {
			continue
		}
		start := index - window
		if start < 0 {
			start = 0
		}
		average := 0.0
		for cursor := start; cursor < index; cursor++ {
			average += energies[cursor]
		}
		if index-start > 0 {
			average /= float64(index - start)
		}
		timeSeconds := float64(index*hopSamples) / 22050.0
		if energy > average*2.2 && energy > 0.00002 && timeSeconds-lastBeat > 0.22 && lastBeat < 0 {
			beats = append(beats, timeSeconds)
			lastBeat = timeSeconds
			continue
		}
		if energy > average*2.2 && energy > 0.00002 && timeSeconds-lastBeat > 0.22 {
			beats = append(beats, timeSeconds)
			lastBeat = timeSeconds
		}
	}
	return beats, nil
}

func timelineBeatDetectCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	path, err := workspaceFile(root, in.Path)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("path: " + err.Error())
	}
	beats, err := detectBeats(path)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"state": "ready", "beats": beats, "total": len(beats), "path": filepathToSlash(in.Path)}, nil
}

func timelineBeatMarkersCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	path, err := workspaceFile(root, in.Path)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("path: " + err.Error())
	}
	beats, err := detectBeats(path)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	fps := in.Timeline.Timebase.Num
	if fps <= 0 {
		fps = 30
	}
	markers := make([]map[string]any, 0, len(beats))
	for _, beat := range beats {
		markers = append(markers, map[string]any{"frame": int(math.Round(beat * float64(fps))), "label": "beat", "kind": "beat", "strength": 1})
	}
	doc := in.Timeline
	doc.Markers = append(doc.Markers, markers...)
	return map[string]any{"state": "ready", "markers": markers, "total": len(markers), "timeline": doc}, nil
}

func filepathToSlash(value string) string { return strings.ReplaceAll(value, "\\", "/") }
