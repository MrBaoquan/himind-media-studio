package main

import (
	"fmt"
	"os"

	"himind-plugin/audio-production/internal/himindjsonrpc"
)

type requestInput struct {
	WorkspaceRoot        string           `json:"workspace_root"`
	CreativeID           string           `json:"creative_id"`
	ProjectID            string           `json:"project_id,omitempty"`
	VoiceProvider        string           `json:"voice_provider,omitempty"`
	Voice                string           `json:"voice,omitempty"`
	VoiceRate            *int             `json:"voice_rate,omitempty"`
	VoiceVolume          *int             `json:"voice_volume,omitempty"`
	VoiceGainDB          *float64         `json:"voice_gain_db,omitempty"`
	BGMPreset            string           `json:"bgm_preset,omitempty"`
	BGMPath              string           `json:"bgm_path,omitempty"`
	BGMGainDB            *float64         `json:"bgm_gain_db,omitempty"`
	Ducking              *bool            `json:"ducking,omitempty"`
	DurationSeconds      float64          `json:"duration_seconds,omitempty"`
	MaxChars             int              `json:"max_chars,omitempty"`
	MinCueSeconds        float64          `json:"min_cue_seconds,omitempty"`
	ExpectedSeconds      float64          `json:"expected_duration_seconds,omitempty"`
	VoiceClips           []voiceClipInput `json:"voice_clips,omitempty"`
	Output               string           `json:"output,omitempty"`
	Path                 string           `json:"path,omitempty"`
	RuntimeID            string           `json:"runtime_id,omitempty"`
	DryRun               bool             `json:"dry_run,omitempty"`
	Force                bool             `json:"force,omitempty"`
	AllowRestrictedVoice bool             `json:"allow_restricted_voice,omitempty"`
	Value                map[string]any   `json:"value,omitempty"`
}

func main() {
	if err := himindjsonrpc.Serve(os.Stdin, os.Stdout, handle); err != nil {
		fmt.Fprintln(os.Stderr, "audio production plugin stopped:", err)
	}
}

func handle(request himindjsonrpc.Request) (any, *himindjsonrpc.Error) {
	var in requestInput
	if rpcError := himindjsonrpc.DecodeParams(request, &in); rpcError != nil {
		return nil, rpcError
	}
	switch request.Method {
	case "audio.catalog":
		return audioCatalogCapability(in)
	case "audio.runtime.install":
		return audioRuntimeInstallCapability(in)
	case "audio.speech":
		return audioSpeechCapability(in)
	case "audio.bgm.generate":
		return audioBGMCapability(in)
	case "audio.mix.render":
		return audioMixCapability(in)
	case "audio.qa.signal":
		return audioQACapability(in)
	case "audio.manifest.get":
		return audioStatusCapability(in)
	case "audio.pipeline.render":
		return audioBuildCapability(in)
	default:
		return nil, &himindjsonrpc.Error{Code: -32601, Message: "method not found: " + request.Method}
	}
}
