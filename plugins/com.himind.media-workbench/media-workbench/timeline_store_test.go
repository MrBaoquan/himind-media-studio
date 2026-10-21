package main

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTimelineProject(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, metadataDir, projectDir, "p1")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.json"), []byte(`{"id":"p1","name":"p1"}`), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestTimelineStoreSaveGet(t *testing.T) {
	root := t.TempDir()
	setupTimelineProject(t, root)
	doc := timelineFixture()
	doc.Revision = 0
	saved, rpcErr := timelineSaveCapability(requestInput{WorkspaceRoot: root, ProjectID: "p1", Timeline: doc})
	if rpcErr != nil {
		t.Fatalf("save: %+v", rpcErr)
	}
	savedMap := saved.(map[string]any)
	if savedMap["ok"] != true {
		t.Fatalf("save failed: %+v", savedMap)
	}
	if savedMap["revision"] != 1 {
		t.Fatalf("revision = %v", savedMap["revision"])
	}
	loaded, rpcErr := timelineGetCapability(requestInput{WorkspaceRoot: root, ProjectID: "p1"})
	if rpcErr != nil {
		t.Fatalf("get: %+v", rpcErr)
	}
	loadedMap := loaded.(map[string]any)
	if loadedMap["state"] != "ready" {
		t.Fatalf("state = %v", loadedMap["state"])
	}
	if len(loadedMap["timeline"].(timelineDoc).Tracks) != 1 {
		t.Fatalf("timeline not restored")
	}
}

func TestTimelineStoreRevisionConflict(t *testing.T) {
	root := t.TempDir()
	setupTimelineProject(t, root)
	doc := timelineFixture()
	_, _ = timelineSaveCapability(requestInput{WorkspaceRoot: root, ProjectID: "p1", Timeline: doc})
	conflict, rpcErr := timelineSaveCapability(requestInput{WorkspaceRoot: root, ProjectID: "p1", Timeline: doc, ExpectedRevision: 99})
	if rpcErr != nil {
		t.Fatalf("save: %+v", rpcErr)
	}
	blockers := conflict.(map[string]any)["blockers"].([]string)
	if len(blockers) == 0 || blockers[0][:18] != "REVISION_CONFLICT:" {
		t.Fatalf("expected conflict, got %v", blockers)
	}
}
