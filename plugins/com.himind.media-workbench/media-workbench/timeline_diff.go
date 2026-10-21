package main

import (
	"encoding/json"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type timelineDiffItem struct {
	TrackID string `json:"track_id"`
	ClipID  string `json:"clip_id"`
	Kind    string `json:"kind"`
}

func clipByID(doc timelineDoc) map[string]timelineClip {
	items := map[string]timelineClip{}
	for _, track := range doc.Tracks {
		for _, clip := range track.Clips {
			items[clip.ID] = clip
		}
	}
	return items
}

func clipJSON(clip timelineClip) string { data, _ := json.Marshal(clip); return string(data) }

func timelineDiffCapability(in requestInput) (any, *himindjsonrpc.Error) {
	base, target := in.BaseTimeline, in.TargetTimeline
	if base.Schema == "" || target.Schema == "" {
		return nil, himindjsonrpc.InvalidParams("base_timeline and target_timeline are required")
	}
	before, after := clipByID(base), clipByID(target)
	added, removed, changed := make([]timelineDiffItem, 0), make([]timelineDiffItem, 0), make([]timelineDiffItem, 0)
	for id, clip := range after {
		if _, ok := before[id]; !ok {
			added = append(added, timelineDiffItem{TrackID: clip.TrackID, ClipID: id, Kind: "clip"})
		}
	}
	for id, clip := range before {
		if _, ok := after[id]; !ok {
			removed = append(removed, timelineDiffItem{TrackID: clip.TrackID, ClipID: id, Kind: "clip"})
		} else if clipJSON(clip) != clipJSON(after[id]) {
			changed = append(changed, timelineDiffItem{TrackID: after[id].TrackID, ClipID: id, Kind: "clip"})
		}
	}
	return map[string]any{
		"state": "ready", "from_revision": base.Revision, "to_revision": target.Revision,
		"added": added, "removed": removed, "changed": changed,
		"summary": map[string]any{"added": len(added), "removed": len(removed), "changed": len(changed)},
	}, nil
}
