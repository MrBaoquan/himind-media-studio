package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type timelineComment struct {
	ID         string `json:"id"`
	ProjectID  string `json:"project_id"`
	Frame      int    `json:"frame"`
	Timecode   string `json:"timecode"`
	Text       string `json:"text"`
	Category   string `json:"category,omitempty"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	ResolvedAt string `json:"resolved_at,omitempty"`
}

func timelineCommentPath(root, projectID string) string {
	return filepath.Join(root, metadataDir, projectDir, projectID, "timeline-comments.json")
}

func loadTimelineComments(root, projectID string) []timelineComment {
	var items []timelineComment
	_ = readJSON(timelineCommentPath(root, projectID), &items)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Frame < items[j].Frame })
	return items
}

func saveTimelineComments(root, projectID string, items []timelineComment) error {
	path := timelineCommentPath(root, projectID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return writeJSON(path, items)
}

func formatTimecode(frame, fps int) string {
	if fps <= 0 {
		fps = 30
	}
	seconds := frame / fps
	return fmt.Sprintf("%02d:%02d:%02d:%02d", seconds/3600, (seconds%3600)/60, seconds%60, frame%fps)
}

func timelineCommentListCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	items := loadTimelineComments(root, item.ID)
	return map[string]any{"state": "ready", "project_id": item.ID, "items": items, "total": len(items)}, nil
}

func timelineCommentAddCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	text := strings.TrimSpace(in.Note)
	if text == "" {
		return nil, himindjsonrpc.InvalidParams("note is required")
	}
	fps := in.Timeline.Timebase.Num
	if fps <= 0 {
		doc := timelineDoc{}
		if readJSON(timelineStorePath(root, item.ID), &doc) == nil && doc.Timebase.Num > 0 {
			fps = doc.Timebase.Num
		}
	}
	if fps <= 0 {
		fps = 30
	}
	frame := int(in.TimeSeconds * float64(fps))
	comment := timelineComment{ID: newID("tl-comment"), ProjectID: item.ID, Frame: frame, Timecode: formatTimecode(frame, fps), Text: text, Category: firstNonEmpty(in.Category, "note"), Status: "open", CreatedAt: now()}
	items := append(loadTimelineComments(root, item.ID), comment)
	if err := saveTimelineComments(root, item.ID, items); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"state": "ready", "comment": comment, "total": len(items)}, nil
}

func timelineCommentResolveCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	items := loadTimelineComments(root, item.ID)
	found := false
	for index := range items {
		if items[index].ID == in.FeedbackID {
			items[index].Status = firstNonEmpty(in.Status, "resolved")
			items[index].ResolvedAt = now()
			found = true
		}
	}
	if !found {
		return map[string]any{"state": "missing", "blockers": []string{"COMMENT_NOT_FOUND"}}, nil
	}
	if err := saveTimelineComments(root, item.ID, items); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"state": "ready", "items": items, "total": len(items)}, nil
}
