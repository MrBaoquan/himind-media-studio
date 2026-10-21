package main

import (
	"fmt"
	"sort"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func timelinePIPCapability(in requestInput) (any, *himindjsonrpc.Error) {
	doc := in.Timeline
	var source *timelineClip
	var sourceTrack *timelineTrack
	for trackIndex := range doc.Tracks {
		if doc.Tracks[trackIndex].Kind != "video" {
			continue
		}
		for clipIndex := range doc.Tracks[trackIndex].Clips {
			if doc.Tracks[trackIndex].Clips[clipIndex].ID == in.ClipID {
				copyClip := doc.Tracks[trackIndex].Clips[clipIndex]
				source = &copyClip
				sourceTrack = &doc.Tracks[trackIndex]
				break
			}
		}
		if source != nil {
			break
		}
	}
	if source == nil || sourceTrack == nil {
		return nil, himindjsonrpc.InvalidParams("clip_id not found")
	}
	trackIndex := -1
	for index := range doc.Tracks {
		if doc.Tracks[index].ID == "V2" {
			trackIndex = index
			break
		}
	}
	if trackIndex < 0 {
		doc.Tracks = append(doc.Tracks, timelineTrack{ID: "V2", Kind: "video", Index: 2, Role: "overlay"})
		trackIndex = len(doc.Tracks) - 1
	}
	if doc.Tracks[trackIndex].Locked {
		return map[string]any{"state": "blocked", "blockers": []string{"TRACK_LOCKED: V2"}}, nil
	}
	pip := *source
	pip.ID = fmt.Sprintf("%s-pip", source.ID)
	pip.TrackID = "V2"
	pip.LinkedGroup = ""
	pip.Transform = &timelineTransform{Scale: 0.35, PositionX: 0.25, PositionY: -0.25, Fit: "cover"}
	pip.Opacity = 1
	doc.Tracks[trackIndex].Clips = append(doc.Tracks[trackIndex].Clips, pip)
	sort.SliceStable(doc.Tracks[trackIndex].Clips, func(i, j int) bool {
		return doc.Tracks[trackIndex].Clips[i].StartFrame < doc.Tracks[trackIndex].Clips[j].StartFrame
	})
	if blockers := validateTimeline(doc); len(blockers) > 0 {
		return map[string]any{"state": "blocked", "blockers": blockers, "timeline": doc}, nil
	}
	return map[string]any{"state": "ready", "timeline": doc, "clip_id": pip.ID, "track_id": "V2"}, nil
}
