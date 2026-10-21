package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type timelineCommandLogEntry struct {
	CommandID string          `json:"command_id,omitempty"`
	Op        string          `json:"op"`
	Revision  int             `json:"revision"`
	Command   timelineCommand `json:"command"`
	CreatedAt string          `json:"created_at"`
}

func timelineCommandLogPath(root, projectID string) string {
	return filepath.Join(root, metadataDir, projectDir, projectID, "timeline-commands.jsonl")
}

func appendTimelineCommandLog(root, projectID string, commands []timelineCommand, revision int) error {
	path := timelineCommandLogPath(root, projectID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for index, command := range commands {
		entry := timelineCommandLogEntry{CommandID: command.CommandID, Op: command.Op, Revision: revision + index, Command: command, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		if _, err := writer.Write(append(data, byte(10))); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func loadTimelineCommandLog(root, projectID string) ([]timelineCommandLogEntry, error) {
	path := timelineCommandLogPath(root, projectID)
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return []timelineCommandLogEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	entries := make([]timelineCommandLogEntry, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var entry timelineCommandLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
			entries = append(entries, entry)
		}
	}
	return entries, scanner.Err()
}

func timelineCommandLogCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	entries, err := loadTimelineCommandLog(root, item.ID)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"state": "ready", "project_id": item.ID, "items": entries, "total": len(entries)}, nil
}
