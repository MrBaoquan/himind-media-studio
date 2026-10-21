package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type subtitleIssue struct {
	Code    string `json:"code"`
	Index   int    `json:"index"`
	Message string `json:"message"`
}

type subtitleTrackPayload struct {
	State           string          `json:"state"`
	DataPath        string          `json:"data_path"`
	DurationSeconds float64         `json:"duration_seconds"`
	Subtitles       []subtitleCue   `json:"subtitles"`
	Issues          []subtitleIssue `json:"issues"`
	Count           int             `json:"count"`
}

func subtitleDataPath(root string, item project) (string, error) {
	sourceRoot, err := sourceRootForProject(root, item)
	if err != nil {
		return "", err
	}
	candidates := []string{
		filepath.Join(sourceRoot, "input", "data.json"),
		filepath.Join(sourceRoot, "data.json"),
	}
	for _, candidate := range candidates {
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("project has no input/data.json")
}

func readSubtitlePayload(path string) (map[string]any, []subtitleCue, error) {
	payload := map[string]any{}
	data, err := os.ReadFile(path)
	if err != nil {
		return payload, nil, err
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return payload, nil, err
	}
	raw, _ := payload["subtitles"].([]any)
	cues := make([]subtitleCue, 0, len(raw))
	for _, entry := range raw {
		item, _ := entry.(map[string]any)
		if item == nil {
			continue
		}
		cue := subtitleCue{
			Start: numberFloat(item["start"]),
			End:   numberFloat(item["end"]),
			Text:  strings.TrimSpace(stringValue(item["text"])),
		}
		cues = append(cues, cue)
	}
	return payload, cues, nil
}

func stringValue(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func evaluateSubtitles(cues []subtitleCue) []subtitleIssue {
	issues := make([]subtitleIssue, 0)
	for index, cue := range cues {
		if cue.End <= cue.Start {
			issues = append(issues, subtitleIssue{Code: "CUE_INVALID_RANGE", Index: index, Message: fmt.Sprintf("第 %d 条字幕结束时间不大于开始时间", index+1)})
		}
		if cue.Text == "" {
			issues = append(issues, subtitleIssue{Code: "CUE_EMPTY_TEXT", Index: index, Message: fmt.Sprintf("第 %d 条字幕为空", index+1)})
		}
		if index > 0 && cue.Start < cues[index-1].End-0.001 {
			issues = append(issues, subtitleIssue{Code: "CUE_OVERLAP", Index: index, Message: fmt.Sprintf("第 %d 条与上一条重叠", index+1)})
		}
		if index > 0 && cue.Start-cues[index-1].End > 1.5 {
			issues = append(issues, subtitleIssue{Code: "CUE_GAP", Index: index, Message: fmt.Sprintf("第 %d 条前有 %.2fs 空档", index+1, cue.Start-cues[index-1].End)})
		}
	}
	return issues
}

func subtitleList(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	path, err := subtitleDataPath(root, item)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	payload, cues, err := readSubtitlePayload(path)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	duration := numberFloat(payload["duration"])
	if duration <= 0 {
		duration = float64(item.DurationSeconds)
	}
	issues := evaluateSubtitles(cues)
	state := "ready"
	if len(issues) > 0 {
		state = "warning"
	}
	return subtitleTrackPayload{
		State: state, DataPath: path, DurationSeconds: duration,
		Subtitles: cues, Issues: issues, Count: len(cues),
	}, nil
}

func subtitleUpdate(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	path, err := subtitleDataPath(root, item)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	payload, _, err := readSubtitlePayload(path)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	cues := make([]subtitleCue, 0, len(in.Subtitles))
	for _, cue := range in.Subtitles {
		text := strings.TrimSpace(cue.Text)
		if text == "" {
			return nil, himindjsonrpc.InvalidParams("subtitle text must not be empty")
		}
		if cue.End <= cue.Start {
			return nil, himindjsonrpc.InvalidParams("subtitle end must be greater than start")
		}
		cues = append(cues, subtitleCue{Start: cue.Start, End: cue.End, Text: text})
	}
	sort.SliceStable(cues, func(left, right int) bool { return cues[left].Start < cues[right].Start })
	serialized := make([]any, 0, len(cues))
	for _, cue := range cues {
		serialized = append(serialized, map[string]any{"start": cue.Start, "end": cue.End, "text": cue.Text})
	}
	payload["subtitles"] = serialized
	if err := writeJSON(path, payload); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"state": "ready", "saved": len(cues), "data_path": path}, nil
}
