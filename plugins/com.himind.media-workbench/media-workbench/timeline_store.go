package main

import (
	"os"
	"path/filepath"
	"strconv"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func timelineStorePath(root, projectID string) string {
	return filepath.Join(root, metadataDir, projectDir, projectID, "timeline.json")
}

func timelineGetCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	path := timelineStorePath(root, item.ID)
	if _, statErr := os.Stat(path); statErr != nil {
		return map[string]any{"state": "missing", "project_id": item.ID, "timeline": nil}, nil
	}
	var doc timelineDoc
	if err := readJSON(path, &doc); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"state": "ready", "project_id": item.ID, "timeline": doc, "path": filepath.ToSlash(path)}, nil
}

func timelineSaveCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if blockers := validateTimeline(in.Timeline); len(blockers) > 0 {
		return map[string]any{"state": "blocked", "ok": false, "blockers": blockers}, nil
	}
	path := timelineStorePath(root, item.ID)
	existing := timelineDoc{}
	_ = readJSON(path, &existing)
	if in.ExpectedRevision > 0 && existing.Revision != in.ExpectedRevision {
		return map[string]any{"state": "blocked", "ok": false, "blockers": []string{"REVISION_CONFLICT: expected " + strconv.Itoa(in.ExpectedRevision) + ", current " + strconv.Itoa(existing.Revision)}}, nil
	}
	doc := in.Timeline
	if doc.Revision <= existing.Revision {
		doc.Revision = existing.Revision + 1
	}
	if err := writeJSON(path, doc); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"state": "ready", "ok": true, "revision": doc.Revision, "timeline": doc, "path": filepath.ToSlash(path)}, nil
}
