package main

import (
	"fmt"
	"os"

	"himind-plugin/audio-intelligence/internal/himindjsonrpc"
)

type requestInput struct {
	WorkspaceRoot string        `json:"workspace_root"`
	CreativeID    string        `json:"creative_id"`
	AlignmentPath string        `json:"alignment_path,omitempty"`
	Script        string        `json:"script,omitempty"`
	Transcript    string        `json:"transcript,omitempty"`
	VoiceClips    []alignClip   `json:"voice_clips,omitempty"`
	Subtitles     []subtitleCue `json:"subtitles,omitempty"`
	MaxChars      int           `json:"max_chars,omitempty"`
}

func main() {
	if err := himindjsonrpc.Serve(os.Stdin, os.Stdout, handle); err != nil {
		fmt.Fprintln(os.Stderr, "audio intelligence plugin stopped:", err)
	}
}

func handle(request himindjsonrpc.Request) (any, *himindjsonrpc.Error) {
	var in requestInput
	if rpcError := himindjsonrpc.DecodeParams(request, &in); rpcError != nil {
		return nil, rpcError
	}
	switch request.Method {
	case "audio.align":
		return audioAlignCapability(in)
	case "audio.alignment.get":
		return audioAlignmentGetCapability(in)
	case "audio.subtitle.from-alignment":
		return audioSubtitleFromAlignmentCapability(in)
	case "audio.qa.sync":
		return audioQASyncCapability(in)
	case "audio.qa.intelligibility":
		return audioQAIntelligibilityCapability(in)
	default:
		return nil, &himindjsonrpc.Error{Code: -32601, Message: "method not found: " + request.Method}
	}
}
