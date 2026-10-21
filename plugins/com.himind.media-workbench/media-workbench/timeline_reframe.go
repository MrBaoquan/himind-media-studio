package main

import (
	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func timelineReframeCapability(in requestInput) (any, *himindjsonrpc.Error) {
	doc := in.Timeline
	if len(doc.Tracks) == 0 {
		return nil, himindjsonrpc.InvalidParams("timeline is required")
	}
	source := doc.Canvas
	width, height := in.Width, in.Height
	if width <= 0 || height <= 0 {
		return nil, himindjsonrpc.InvalidParams("width and height are required")
	}
	fit := in.FitMode
	if fit == "" {
		fit = "cover"
	}
	if fit != "cover" && fit != "contain" {
		return nil, himindjsonrpc.InvalidParams("fit_mode must be cover or contain")
	}
	doc.Canvas = timelineCanvas{Width: width, Height: height}
	changed := 0
	for trackIndex := range doc.Tracks {
		if doc.Tracks[trackIndex].Kind != "video" {
			continue
		}
		for clipIndex := range doc.Tracks[trackIndex].Clips {
			clip := &doc.Tracks[trackIndex].Clips[clipIndex]
			transform := timelineTransform{}
			if clip.Transform != nil {
				transform = *clip.Transform
			}
			transform.Fit = fit
			transform.Scale = 1
			clip.Transform = &transform
			changed++
		}
	}
	return map[string]any{"state": "ready", "timeline": doc, "source_canvas": source, "canvas": doc.Canvas, "fit": fit, "clips": changed, "subject_tracking": "deterministic-center"}, nil
}
