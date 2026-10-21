package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitSubtitleTextKeepsCueLengthBounded(t *testing.T) {
	parts := splitSubtitleText("在合肥，有一轮人造太阳。它不挂在天空，而是在一座实验装置里。", 12)
	if len(parts) == 0 {
		t.Fatal("expected subtitle parts")
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			t.Fatal("empty subtitle part")
		}
		if len([]rune(part)) > 12 {
			t.Fatalf("part too long: %q", part)
		}
	}
}

func TestBuildSubtitlesStaysInsideBeat(t *testing.T) {
	beats := []audioBeat{
		{ID: "b1", Start: 0, DurationSeconds: 5, End: 5, Narration: "第一句。第二句。"},
		{ID: "b2", Start: 5, DurationSeconds: 4, End: 9, Narration: "第三句。"},
	}
	cues := buildSubtitles(beats, map[string]float64{"b1": 4.8, "b2": 3.2}, 24, 0.9)
	if len(cues) == 0 {
		t.Fatal("expected cues")
	}
	for _, cue := range cues {
		if cue.BeatID == "" {
			t.Fatalf("cue missing beat id: %+v", cue)
		}
		if cue.Start < 0 || cue.End > 9.0001 {
			t.Fatalf("cue outside timeline: %+v", cue)
		}
		if cue.Text == "" {
			t.Fatal("empty cue text")
		}
	}
}

func TestApplyFitPolicyCapsTempo(t *testing.T) {
	clip := voiceClip{DurationSeconds: 7, WindowSeconds: 5}
	applyFitPolicy(&clip)
	if clip.Tempo <= 1 || clip.Tempo > 1.5 {
		t.Fatalf("unexpected tempo: %v", clip.Tempo)
	}
	if clip.EffectiveDuration > clip.WindowSeconds+0.03 {
		t.Fatalf("clip should be fitted close to window: %v", clip.EffectiveDuration)
	}
	overflow := voiceClip{DurationSeconds: 12, WindowSeconds: 5}
	applyFitPolicy(&overflow)
	if overflow.FitWarning == "" {
		t.Fatal("expected overflow warning when 1.5x is still too long")
	}
}

func TestParseLoudnormJSON(t *testing.T) {
	raw := `[Parsed_loudnorm_0] {
  "input_i" : "-16.28",
  "input_tp" : "-2.49",
  "input_lra" : "5.70"
}`
	values := parseLoudnormJSON(raw)
	if math.Abs(values["input_i"]+16.28) > 0.001 {
		t.Fatalf("input_i = %v", values["input_i"])
	}
	if math.Abs(values["input_tp"]+2.49) > 0.001 {
		t.Fatalf("input_tp = %v", values["input_tp"])
	}
}

func TestAudioRuntimeCatalogPinsArtifacts(t *testing.T) {
	runtimeItem, err := findAudioRuntime("piper")
	if err != nil {
		t.Fatal(err)
	}
	if runtimeItem.License == "" || len(runtimeItem.Artifacts) == 0 || len(runtimeItem.Voices) == 0 {
		t.Fatalf("incomplete runtime: %+v", runtimeItem)
	}
	for _, artifact := range runtimeItem.Artifacts {
		if len(artifact.SHA256) != 64 {
			t.Fatalf("artifact hash not pinned: %+v", artifact)
		}
	}
}

func TestPiperRuntimeStatusReportsMissingInEmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	status, err := buildPiperRuntimeStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if status.Installed {
		t.Fatal("empty workspace must not report piper installed")
	}
	if len(status.Missing) == 0 {
		t.Fatal("expected missing runtime files")
	}
}

func TestRestrictedVoiceRequiresExplicitConfirmation(t *testing.T) {
	root := t.TempDir()
	runtimeItem, err := findAudioRuntime("piper")
	if err != nil {
		t.Fatal(err)
	}
	dir := audioRuntimeRoot(root, runtimeItem)
	if err := os.MkdirAll(filepath.Join(dir, "piper"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "piper", "piper.exe"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, voice := range runtimeItem.Voices {
		model := filepath.Join(dir, filepath.FromSlash(voice.Model))
		config := filepath.Join(dir, filepath.FromSlash(voice.Config))
		_ = os.MkdirAll(filepath.Dir(model), 0755)
		_ = os.WriteFile(model, []byte("x"), 0644)
		_ = os.WriteFile(config, []byte("{}"), 0644)
	}
	_, _, _, _, err = resolvePiperVoice(root, "zh_CN-huayan-medium", false)
	if err == nil {
		t.Fatal("restricted voice should require confirmation")
	}
	if !strings.Contains(err.Error(), "allow_restricted_voice") {
		t.Fatalf("unexpected error: %v", err)
	}
	var voice audioVoicePack
	_, voice, _, _, err = resolvePiperVoice(root, "zh_CN-huayan-medium", true)
	if err != nil {
		t.Fatalf("confirmed restricted voice should resolve: %v", err)
	}
	if voice.ID == "" {
		t.Fatal("expected voice pack")
	}
}
