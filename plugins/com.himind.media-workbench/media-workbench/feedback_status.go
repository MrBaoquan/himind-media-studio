package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type feedbackStatus struct {
	Status     string `json:"status"` // open | resolved | ignored
	Note       string `json:"note,omitempty"`
	ResolvedAt string `json:"resolved_at,omitempty"`
	SourceHash string `json:"source_hash,omitempty"`
}

func feedbackStatusPath(root, projectID string) string {
	return filepath.Join(root, metadataDir, projectDir, projectID, "feedback-status.json")
}

func loadFeedbackStatuses(root, projectID string) map[string]feedbackStatus {
	items := map[string]feedbackStatus{}
	_ = readJSON(feedbackStatusPath(root, projectID), &items)
	return items
}

// feedbackEntriesWithStatus merges append-only feedback.jsonl with the mutable
// open/resolved status file. The JSONL remains the source of truth for content;
// status is deliberately separate so resolving never rewrites history.
func feedbackEntriesWithStatus(root, projectID string) []map[string]any {
	path := filepath.Join(root, metadataDir, projectDir, projectID, "feedback.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		return []map[string]any{}
	}
	statuses := loadFeedbackStatuses(root, projectID)
	items := make([]map[string]any, 0)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry map[string]any
		if json.Unmarshal([]byte(line), &entry) != nil {
			continue
		}
		id := asString(entry["id"])
		status, ok := statuses[id]
		if !ok {
			status = feedbackStatus{Status: "open"}
		}
		if status.Status == "" {
			status.Status = "open"
		}
		entry["status"] = status.Status
		entry["status_note"] = status.Note
		entry["resolved_at"] = status.ResolvedAt
		entry["status_source_hash"] = status.SourceHash
		items = append(items, entry)
	}
	return items
}

func feedbackResolve(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !idPattern.MatchString(in.ProjectID) || strings.TrimSpace(in.FeedbackID) == "" {
		return nil, himindjsonrpc.InvalidParams("project_id and feedback_id are required")
	}
	status := strings.ToLower(strings.TrimSpace(in.Status))
	if status == "" {
		status = "resolved"
	}
	if status != "resolved" && status != "ignored" && status != "open" {
		return nil, himindjsonrpc.InvalidParams("status must be open, resolved or ignored")
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	entries := feedbackEntriesWithStatus(root, item.ID)
	found := false
	for _, entry := range entries {
		if asString(entry["id"]) == in.FeedbackID {
			found = true
			break
		}
	}
	if !found {
		return nil, himindjsonrpc.InvalidParams("feedback_id not found")
	}
	statuses := loadFeedbackStatuses(root, item.ID)
	statusItem := feedbackStatus{Status: status, Note: strings.TrimSpace(in.Note)}
	if status == "resolved" || status == "ignored" {
		statusItem.ResolvedAt = now()
		if projectMode(item) == "code" {
			statusItem.SourceHash = item.SourceHash
		}
	}
	statuses[in.FeedbackID] = statusItem
	if err := writeJSON(feedbackStatusPath(root, item.ID), statuses); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	pending := 0
	for _, entry := range feedbackEntriesWithStatus(root, item.ID) {
		if asString(entry["status"]) == "open" {
			pending++
		}
	}
	return map[string]any{
		"updated": true, "feedback_id": in.FeedbackID, "status": statusItem,
		"pending": pending, "project_id": item.ID,
		"next": []string{"重新调用 project.context 检查 pending_feedback", "继续修改代码或进入最终渲染"},
	}, nil
}
