package main

import (
	"encoding/json"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type timelineHistory struct {
	Current timelineDoc   `json:"current"`
	Undo    []timelineDoc `json:"undo,omitempty"`
	Redo    []timelineDoc `json:"redo,omitempty"`
}

func cloneTimeline(doc timelineDoc) timelineDoc {
	data, _ := json.Marshal(doc)
	var cloned timelineDoc
	_ = json.Unmarshal(data, &cloned)
	return cloned
}

func (history *timelineHistory) apply(commands []timelineCommand) []string {
	snapshot := cloneTimeline(history.Current)
	next, blockers := applyTimelineCommands(history.Current, commands)
	if len(blockers) > 0 {
		return blockers
	}
	history.Undo = append(history.Undo, snapshot)
	if len(history.Undo) > 100 {
		history.Undo = history.Undo[len(history.Undo)-100:]
	}
	history.Current = next
	history.Redo = nil
	return nil
}

func (history *timelineHistory) undo() bool {
	if len(history.Undo) == 0 {
		return false
	}
	history.Redo = append(history.Redo, cloneTimeline(history.Current))
	history.Current = history.Undo[len(history.Undo)-1]
	history.Undo = history.Undo[:len(history.Undo)-1]
	return true
}

func (history *timelineHistory) redo() bool {
	if len(history.Redo) == 0 {
		return false
	}
	history.Undo = append(history.Undo, cloneTimeline(history.Current))
	history.Current = history.Redo[len(history.Redo)-1]
	history.Redo = history.Redo[:len(history.Redo)-1]
	return true
}

func timelineHistoryApplyCapability(in requestInput) (any, *himindjsonrpc.Error) {
	if len(in.Commands) == 0 {
		return nil, himindjsonrpc.InvalidParams("commands is required")
	}
	history := in.History
	blockers := history.apply(in.Commands)
	return map[string]any{"ok": len(blockers) == 0, "history": history, "blockers": blockers}, nil
}

func timelineHistoryUndoCapability(in requestInput) (any, *himindjsonrpc.Error) {
	history := in.History
	ok := history.undo()
	return map[string]any{"ok": ok, "history": history}, nil
}

func timelineHistoryRedoCapability(in requestInput) (any, *himindjsonrpc.Error) {
	history := in.History
	ok := history.redo()
	return map[string]any{"ok": ok, "history": history}, nil
}
