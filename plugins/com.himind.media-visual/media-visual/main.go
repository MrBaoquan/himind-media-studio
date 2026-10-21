package main

import (
	"fmt"
	"os"

	"himind-plugin/media-visual/internal/himindjsonrpc"
)

type requestInput struct {
	WorkspaceRoot   string   `json:"workspace_root"`
	PackID          string   `json:"pack_id,omitempty"`
	Version         string   `json:"version,omitempty"`
	Name            string   `json:"name,omitempty"`
	Description     string   `json:"description,omitempty"`
	ReferenceImages []string `json:"reference_images,omitempty"`
	Role            string   `json:"role,omitempty"`
	Character       string   `json:"character,omitempty"`
	Prompt          string   `json:"prompt,omitempty"`
	NegativePrompt  string   `json:"negative_prompt,omitempty"`
	Seed            int64    `json:"seed,omitempty"`
	License         string   `json:"license,omitempty"`
	LicenseURL      string   `json:"license_url,omitempty"`
	CommercialUse   bool     `json:"commercial_use,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	PackPath        string   `json:"pack_path,omitempty"`
	ReferenceAssets []string `json:"reference_assets,omitempty"`
	ReferencePath   string   `json:"reference_path,omitempty"`
	CandidatePath   string   `json:"candidate_path,omitempty"`
	CandidatePaths  []string `json:"candidate_paths,omitempty"`
	ImagePath       string   `json:"image_path,omitempty"`
	VideoPath       string   `json:"video_path,omitempty"`
	Times           string   `json:"times,omitempty"`
	SampleCount     int      `json:"sample_count,omitempty"`
	KeepFrames      bool     `json:"keep_frames,omitempty"`
	SceneThreshold  float64  `json:"scene_threshold,omitempty"`
	MaxCuts         int      `json:"max_cuts,omitempty"`
	OutputPath      string   `json:"output_path,omitempty"`
	Columns         int      `json:"columns,omitempty"`
	CellWidth       int      `json:"cell_width,omitempty"`
	Scale           float64  `json:"scale,omitempty"`
	Width           int      `json:"width,omitempty"`
	Height          int      `json:"height,omitempty"`
	CRF             int      `json:"crf,omitempty"`
	Preset          string   `json:"preset,omitempty"`
	Mute            bool     `json:"mute,omitempty"`
	Threshold       float64  `json:"threshold,omitempty"`
}

func main() {
	if err := himindjsonrpc.Serve(os.Stdin, os.Stdout, handle); err != nil {
		fmt.Fprintln(os.Stderr, "media visual plugin stopped:", err)
	}
}

func handle(request himindjsonrpc.Request) (any, *himindjsonrpc.Error) {
	var in requestInput
	if rpcError := himindjsonrpc.DecodeParams(request, &in); rpcError != nil {
		return nil, rpcError
	}
	switch request.Method {
	case "visual.reference.pack":
		return referencePackCapability(in)
	case "visual.reference.validate":
		return referenceValidateCapability(in)
	case "visual.reference.character.consistency":
		return referenceConsistencyCapability(in)
	case "visual.image.compare":
		return imageCompareCapability(in)
	case "visual.image.qa":
		return imageQACapability(in)
	case "visual.video.qa":
		return videoQACapability(in)
	case "visual.video.scenes":
		return videoScenesCapability(in)
	case "visual.video.contact-sheet":
		return videoContactSheetCapability(in)
	case "visual.upscale":
		return visualUpscaleCapability(in)
	default:
		return nil, &himindjsonrpc.Error{Code: -32601, Message: "method not found: " + request.Method}
	}
}
