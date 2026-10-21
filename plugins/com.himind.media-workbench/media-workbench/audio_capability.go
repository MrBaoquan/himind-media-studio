package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// audio_capability.go — 旁白、字幕、BGM 与混音底座。
//
// 设计边界：
//   - TTS 只负责“文本 -> 音频文件”，不进入画面渲染器；
//   - 字幕是独立时间轨，由渲染器按时间绘制；
//   - BGM 是通用音频资产，不绑定某个视频题材；
//   - 混音由 ffmpeg 完成，渲染器只负责把混音轨 mux 进 MP4。
//
// P0 提供两条 TTS 适配器：Piper（离线神经 TTS，推荐）与 Windows SAPI。
// 两者都只产出标准 WAV；Provider/voice/license 写入 manifest，便于替换与审计。

const audioManifestName = "audio-manifest.json"

type audioVoiceInfo struct {
	Name    string `json:"name"`
	Culture string `json:"culture"`
	Gender  string `json:"gender"`
	Age     string `json:"age"`
	Default bool   `json:"default"`
}

type audioRuntimeArtifact struct {
	Kind   string `json:"kind"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Dest   string `json:"dest"`
	Size   int64  `json:"size,omitempty"`
}

type audioVoicePack struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Language      string `json:"language"`
	Quality       string `json:"quality"`
	Model         string `json:"model"`
	Config        string `json:"config"`
	License       string `json:"license"`
	LicenseURL    string `json:"license_url,omitempty"`
	CommercialUse bool   `json:"commercial_use"`
	DemoOnly      bool   `json:"demo_only,omitempty"`
	Available     bool   `json:"available"`
	Note          string `json:"note,omitempty"`
}

type audioRuntime struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	License     string                 `json:"license"`
	Description string                 `json:"description"`
	Artifacts   []audioRuntimeArtifact `json:"artifacts"`
	Voices      []audioVoicePack       `json:"voices"`
}

type audioRuntimeCatalog struct {
	Schema   string         `json:"schema"`
	Runtimes []audioRuntime `json:"runtimes"`
}

type audioRuntimeStatus struct {
	ID         string           `json:"id"`
	Version    string           `json:"version"`
	Installed  bool             `json:"installed"`
	Path       string           `json:"path,omitempty"`
	Executable string           `json:"executable,omitempty"`
	License    string           `json:"license"`
	Voices     []audioVoicePack `json:"voices"`
	Missing    []string         `json:"missing,omitempty"`
	Next       string           `json:"next,omitempty"`
}

type bgmTone struct {
	Frequency float64 `json:"frequency"`
	Volume    float64 `json:"volume"`
	Lowpass   int     `json:"lowpass,omitempty"`
}

type bgmNoise struct {
	Color     string  `json:"color"`
	Amplitude float64 `json:"amplitude"`
	Lowpass   int     `json:"lowpass,omitempty"`
	Highpass  int     `json:"highpass,omitempty"`
	Volume    float64 `json:"volume,omitempty"`
}

type bgmTremolo struct {
	Frequency float64 `json:"frequency"`
	Depth     float64 `json:"depth"`
}

type bgmEcho struct {
	InGain  float64 `json:"in_gain"`
	OutGain float64 `json:"out_gain"`
	Delays  string  `json:"delays"`
	Decays  string  `json:"decays"`
}

type bgmPreset struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Mood        []string    `json:"mood"`
	License     string      `json:"license"`
	Loop        bool        `json:"loop"`
	Tones       []bgmTone   `json:"tones"`
	Noise       *bgmNoise   `json:"noise,omitempty"`
	Tremolo     *bgmTremolo `json:"tremolo,omitempty"`
	Echo        *bgmEcho    `json:"echo,omitempty"`
	Gain        float64     `json:"gain"`
	FadeIn      float64     `json:"fade_in"`
	FadeOut     float64     `json:"fade_out"`
}

type audioBeat struct {
	ID              string  `json:"id"`
	Start           float64 `json:"start"`
	DurationSeconds float64 `json:"duration_seconds"`
	End             float64 `json:"end"`
	Narration       string  `json:"narration"`
	OnScreenText    string  `json:"on_screen_text,omitempty"`
	Purpose         string  `json:"purpose,omitempty"`
}

type voiceClip struct {
	BeatID            string  `json:"beat_id"`
	Text              string  `json:"text"`
	Path              string  `json:"path"`
	Provider          string  `json:"provider"`
	Voice             string  `json:"voice"`
	VoiceLicense      string  `json:"voice_license,omitempty"`
	CommercialVoice   bool    `json:"commercial_voice"`
	Rate              int     `json:"rate"`
	Start             float64 `json:"start"`
	WindowSeconds     float64 `json:"window_seconds"`
	DurationSeconds   float64 `json:"duration_seconds"`
	EffectiveDuration float64 `json:"effective_duration_seconds"`
	Tempo             float64 `json:"tempo,omitempty"`
	License           string  `json:"license,omitempty"`
	CommercialUse     bool    `json:"commercial_use"`
	SHA256            string  `json:"sha256"`
	FitWarning        string  `json:"fit_warning,omitempty"`
}

type voiceClipInput struct {
	Path          string  `json:"path"`
	Start         float64 `json:"start"`
	WindowSeconds float64 `json:"window_seconds,omitempty"`
	Tempo         float64 `json:"tempo,omitempty"`
	FitToWindow   bool    `json:"fit_to_window,omitempty"`
	Label         string  `json:"label,omitempty"`
}

type subtitleCue struct {
	Start  float64 `json:"start"`
	End    float64 `json:"end"`
	Text   string  `json:"text"`
	BeatID string  `json:"beat_id,omitempty"`
}

type bgmAsset struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Path            string   `json:"path"`
	DurationSeconds float64  `json:"duration_seconds"`
	SHA256          string   `json:"sha256"`
	License         string   `json:"license"`
	Loop            bool     `json:"loop"`
	Mood            []string `json:"mood,omitempty"`
	Generated       bool     `json:"generated"`
}

type audioQAResult struct {
	OK              bool     `json:"ok"`
	Status          string   `json:"status"`
	Path            string   `json:"path"`
	Codec           string   `json:"codec,omitempty"`
	Channels        int      `json:"channels,omitempty"`
	SampleRate      int      `json:"sample_rate,omitempty"`
	DurationSeconds float64  `json:"duration_seconds"`
	ExpectedSeconds float64  `json:"expected_duration_seconds,omitempty"`
	DurationDrift   float64  `json:"duration_drift_seconds,omitempty"`
	IntegratedLUFS  float64  `json:"integrated_lufs,omitempty"`
	TruePeakDBTP    float64  `json:"true_peak_dbtp,omitempty"`
	LoudnessRange   float64  `json:"loudness_range,omitempty"`
	MaxVolumeDB     float64  `json:"max_volume_db,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	Blockers        []string `json:"blockers,omitempty"`
}

type audioManifest struct {
	Version         string         `json:"version"`
	CreativeID      string         `json:"creative_id"`
	ProjectID       string         `json:"project_id,omitempty"`
	GeneratedAt     string         `json:"generated_at"`
	VoiceProvider   string         `json:"voice_provider"`
	Voice           string         `json:"voice"`
	VoiceLicense    string         `json:"voice_license,omitempty"`
	CommercialVoice bool           `json:"commercial_voice"`
	Rate            int            `json:"rate"`
	VoiceGainDB     float64        `json:"voice_gain_db"`
	BGMPreset       string         `json:"bgm_preset"`
	BGMGainDB       float64        `json:"bgm_gain_db"`
	Ducking         bool           `json:"ducking"`
	DurationSeconds float64        `json:"duration_seconds"`
	Clips           []voiceClip    `json:"clips"`
	Subtitles       []subtitleCue  `json:"subtitles"`
	BGM             *bgmAsset      `json:"bgm,omitempty"`
	MixPath         string         `json:"mix_path"`
	SubtitlesPath   string         `json:"subtitles_path"`
	QA              *audioQAResult `json:"qa,omitempty"`
	Source          string         `json:"source"`
}

func audioCatalogCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	piper, piperErr := buildPiperRuntimeStatus(root)
	if piperErr != nil {
		return nil, himindjsonrpc.InternalError(piperErr.Error())
	}
	sapiVoices, sapiWarning := listSAPIVoices(root)
	presets, err := loadBGMPresets()
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	defaultVoice := ""
	if piper.Installed && len(piper.Voices) > 0 {
		for _, voice := range piper.Voices {
			if voice.Available {
				defaultVoice = voice.ID
				break
			}
		}
	}
	warnings := make([]string, 0)
	if sapiWarning != "" {
		warnings = append(warnings, "Windows SAPI 不可用: "+sapiWarning)
	}
	if !piper.Installed {
		warnings = append(warnings, "Piper 运行时尚未安装；调用 media.workbench.audio.runtime.install 安装后即可离线合成中文旁白")
	}
	return map[string]any{
		"providers": []map[string]any{
			{"id": "piper", "name": "Piper 离线神经 TTS", "available": piper.Installed, "network": false, "offline": true, "license": piper.License, "runtime": piper},
			{"id": "windows-sapi", "name": "Windows SAPI", "available": len(sapiVoices) > 0, "network": false, "offline": true, "license": "system"},
			{"id": "ark-speech", "name": "火山方舟语音", "available": false, "network": true, "note": "需要先在 Agent Plan 开启语音模型抵扣"},
			{"id": "openai-speech", "name": "OpenAI Speech", "available": false, "network": true},
		},
		"voices":      sapiVoices,
		"bgm_presets": presets,
		"defaults": map[string]any{
			"provider":      "piper",
			"voice":         defaultVoice,
			"rate":          4,
			"volume":        100,
			"bgm_preset":    "documentary-drone",
			"voice_gain_db": 0,
			"bgm_gain_db":   -20,
			"ducking":       true,
		},
		"warnings": warnings,
	}, nil
}

func audioSpeechCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	provider := firstNonEmpty(strings.TrimSpace(in.VoiceProvider), "piper")
	rate := 4
	if in.VoiceRate != nil {
		rate = *in.VoiceRate
	}
	volume := 100
	if in.VoiceVolume != nil {
		volume = *in.VoiceVolume
	}

	if strings.TrimSpace(in.CreativeID) == "" {
		text := strings.TrimSpace(asString(in.Value["text"]))
		if text == "" {
			return nil, himindjsonrpc.InvalidParams("creative_id or value.text is required")
		}
		output := strings.TrimSpace(in.Output)
		if output == "" {
			output = filepath.ToSlash(filepath.Join(metadataDir, "audio", "speech-"+strconv.FormatInt(time.Now().UnixNano(), 10)+".wav"))
		}
		path, err := workspaceFile(root, output)
		if err != nil {
			return nil, himindjsonrpc.InvalidParams("output: " + err.Error())
		}
		clip, err := synthesizeVoice(root, text, path, provider, in.Voice, rate, volume, in.AllowRestrictedVoice)
		if err != nil {
			return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("AUDIO_SPEECH_FAILED", "audio", err.Error(), "检查 TTS 运行时/voice/license 后重试")}, "retryable": true}, nil
		}
		return map[string]any{"state": "ready", "clip": clip, "provider": provider}, nil
	}

	beats, item, err := loadAudioBeats(root, in.CreativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	audioDir := filepath.Join(creativeRoot(root, item.ID), "audio")
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	clips := make([]voiceClip, 0, len(beats))
	warnings := make([]string, 0)
	for _, beat := range beats {
		if strings.TrimSpace(beat.Narration) == "" {
			continue
		}
		output := filepath.Join(audioDir, "voice-"+beat.ID+".wav")
		clip, err := synthesizeVoice(root, beat.Narration, output, provider, in.Voice, rate, volume, in.AllowRestrictedVoice)
		if err != nil {
			return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("AUDIO_SPEECH_FAILED", "audio", fmt.Sprintf("beat %s 合成失败: %v", beat.ID, err), "检查 TTS 运行时/voice/license 后重试")}, "retryable": true}, nil
		}
		clip.BeatID = beat.ID
		clip.Start = beat.Start
		clip.WindowSeconds = beat.DurationSeconds
		applyFitPolicy(&clip)
		if clip.FitWarning != "" {
			warnings = append(warnings, clip.FitWarning)
		}
		clips = append(clips, clip)
	}
	return map[string]any{
		"state": "ready", "provider": provider, "total": len(clips), "clips": clips, "warnings": warnings,
		"next": "调用 subtitle.generate 生成字幕轨，再调用 audio.mix 混音",
	}, nil
}

func subtitleGenerateCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if strings.TrimSpace(in.CreativeID) == "" {
		return nil, himindjsonrpc.InvalidParams("creative_id is required")
	}
	beats, item, err := loadAudioBeats(root, in.CreativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	manifest, _ := loadAudioManifest(root, item.ID)
	durations := map[string]float64{}
	if manifest != nil {
		for _, clip := range manifest.Clips {
			durations[clip.BeatID] = clip.EffectiveDuration
		}
	}
	maxChars := in.MaxChars
	if maxChars <= 0 {
		maxChars = 24
	}
	minCue := in.MinCueSeconds
	if minCue <= 0 {
		minCue = 0.9
	}
	cues := buildSubtitles(beats, durations, maxChars, minCue)
	output := strings.TrimSpace(in.Output)
	if output == "" {
		output = filepath.ToSlash(filepath.Join(metadataDir, creativeDir, item.ID, "audio", "subtitles.json"))
	}
	path, err := workspaceFile(root, output)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("output: " + err.Error())
	}
	if err := writeJSON(path, map[string]any{
		"version": "himind.subtitles.v1", "creative_id": item.ID,
		"generated_at": now(), "subtitles": cues,
	}); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"state": "ready", "subtitles": cues, "total": len(cues), "path": path, "next": "把 subtitles 写入项目 data.json 后重新预览"}, nil
}

func audioBGMCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	presetID := firstNonEmpty(strings.TrimSpace(in.BGMPreset), "documentary-drone")
	duration := in.DurationSeconds
	if duration <= 0 {
		duration = 72
	}
	output := strings.TrimSpace(in.Output)
	if output == "" {
		if strings.TrimSpace(in.CreativeID) == "" {
			return nil, himindjsonrpc.InvalidParams("output is required when creative_id is absent")
		}
		output = filepath.ToSlash(filepath.Join(metadataDir, creativeDir, in.CreativeID, "audio", "bgm-"+presetID+".wav"))
	}
	path, err := workspaceFile(root, output)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("output: " + err.Error())
	}
	asset, err := generateBGM(path, presetID, duration)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"state": "ready", "bgm": asset, "provider": "ffmpeg-lavfi", "license": asset.License}, nil
}

func audioMixCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	clips := make([]voiceClipInput, 0, len(in.VoiceClips))
	for _, clip := range in.VoiceClips {
		if strings.TrimSpace(clip.Path) == "" {
			continue
		}
		clips = append(clips, clip)
	}
	if len(clips) == 0 {
		return nil, himindjsonrpc.InvalidParams("voice_clips is required")
	}
	duration := in.DurationSeconds
	if duration <= 0 {
		duration = maxAudioEnd(clips)
	}
	if duration <= 0 {
		return nil, himindjsonrpc.InvalidParams("duration_seconds is required")
	}
	bgmPath := strings.TrimSpace(in.BGMPath)
	output := strings.TrimSpace(in.Output)
	if output == "" {
		if strings.TrimSpace(in.CreativeID) == "" {
			return nil, himindjsonrpc.InvalidParams("output is required when creative_id is absent")
		}
		output = filepath.ToSlash(filepath.Join(metadataDir, creativeDir, in.CreativeID, "audio", "mix.m4a"))
	}
	outPath, err := workspaceFile(root, output)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("output: " + err.Error())
	}
	if bgmPath != "" {
		if _, err := workspaceFile(root, bgmPath); err != nil {
			return nil, himindjsonrpc.InvalidParams("bgm_path: " + err.Error())
		}
	}
	voiceGain := 0.0
	if in.VoiceGainDB != nil {
		voiceGain = *in.VoiceGainDB
	}
	bgmGain := -20.0
	if in.BGMGainDB != nil {
		bgmGain = *in.BGMGainDB
	}
	ducking := true
	if in.Ducking != nil {
		ducking = *in.Ducking
	}
	if err := mixAudio(root, clips, bgmPath, duration, voiceGain, bgmGain, ducking, outPath); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	qa, _ := analyzeAudio(outPath, duration)
	return map[string]any{"state": "ready", "mix": map[string]any{"path": outPath, "duration_seconds": duration, "voice_gain_db": voiceGain, "bgm_gain_db": bgmGain, "ducking": ducking}, "qa": qa}, nil
}

func audioQACapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	pathText := strings.TrimSpace(in.Path)
	if pathText == "" && strings.TrimSpace(in.CreativeID) != "" {
		pathText = filepath.ToSlash(filepath.Join(metadataDir, creativeDir, in.CreativeID, "audio", "mix.m4a"))
	}
	if pathText == "" {
		return nil, himindjsonrpc.InvalidParams("path or creative_id is required")
	}
	path, err := workspaceFile(root, pathText)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("path: " + err.Error())
	}
	expected := in.ExpectedSeconds
	if expected <= 0 {
		expected = in.DurationSeconds
	}
	qa, err := analyzeAudio(path, expected)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"qa": qa}, nil
}

func audioStatusCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if strings.TrimSpace(in.CreativeID) == "" {
		return nil, himindjsonrpc.InvalidParams("creative_id is required")
	}
	manifest, err := loadAudioManifest(root, in.CreativeID)
	if err != nil || manifest == nil {
		return map[string]any{"state": "missing", "creative_id": in.CreativeID, "next": "调用 audio.build 生成旁白、字幕、BGM 与混音"}, nil
	}
	return map[string]any{"state": "ready", "manifest": manifest}, nil
}

// audioBuildCapability 是创作台/AI 使用的主入口：一次完成旁白、字幕、BGM、
// 混音与 QA，并把字幕轨写回项目数据。它是编排，不替代底层单项能力。
func audioBuildCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if strings.TrimSpace(in.CreativeID) == "" {
		return nil, himindjsonrpc.InvalidParams("creative_id is required")
	}
	beats, item, err := loadAudioBeats(root, in.CreativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if len(beats) == 0 {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("AUDIO_BEATS_MISSING", "script", "脚本没有 beats，无法生成旁白", "先在创作包补齐 script.beats")}, "retryable": true}, nil
	}
	provider := firstNonEmpty(strings.TrimSpace(in.VoiceProvider), "piper")
	rate := 4
	if in.VoiceRate != nil {
		rate = *in.VoiceRate
	}
	volume := 100
	if in.VoiceVolume != nil {
		volume = *in.VoiceVolume
	}
	audioDir := filepath.Join(creativeRoot(root, item.ID), "audio")
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}

	totalDuration := 0.0
	for _, beat := range beats {
		totalDuration = math.Max(totalDuration, beat.End)
	}
	if in.DurationSeconds > 0 {
		totalDuration = in.DurationSeconds
	}
	if totalDuration <= 0 {
		totalDuration = 72
	}

	clips := make([]voiceClip, 0, len(beats))
	warnings := make([]string, 0)
	for _, beat := range beats {
		text := strings.TrimSpace(beat.Narration)
		if text == "" {
			continue
		}
		output := filepath.Join(audioDir, "voice-"+beat.ID+".wav")
		clip, err := synthesizeVoice(root, text, output, provider, in.Voice, rate, volume, in.AllowRestrictedVoice)
		if err != nil {
			return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("AUDIO_SPEECH_FAILED", "audio", fmt.Sprintf("beat %s 合成失败: %v", beat.ID, err), "检查 TTS 运行时/voice/license 后重试")}, "retryable": true}, nil
		}
		clip.BeatID = beat.ID
		clip.Start = beat.Start
		clip.WindowSeconds = beat.DurationSeconds
		applyFitPolicy(&clip)
		if clip.FitWarning != "" {
			warnings = append(warnings, clip.FitWarning)
		}
		clips = append(clips, clip)
	}
	if len(clips) == 0 {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("AUDIO_NARRATION_EMPTY", "script", "所有 beat 都没有 narration", "补齐旁白文本")}, "retryable": true}, nil
	}

	maxChars := in.MaxChars
	if maxChars <= 0 {
		maxChars = 24
	}
	minCue := in.MinCueSeconds
	if minCue <= 0 {
		minCue = 0.9
	}
	effective := map[string]float64{}
	for _, clip := range clips {
		effective[clip.BeatID] = clip.EffectiveDuration
	}
	cues := buildSubtitles(beats, effective, maxChars, minCue)
	subtitlesPath := filepath.Join(audioDir, "subtitles.json")
	if err := writeJSON(subtitlesPath, map[string]any{
		"version": "himind.subtitles.v1", "creative_id": item.ID, "generated_at": now(), "subtitles": cues,
	}); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}

	presetID := firstNonEmpty(strings.TrimSpace(in.BGMPreset), "documentary-drone")
	bgmPath := filepath.Join(audioDir, "bgm-"+presetID+".wav")
	bgm, err := generateBGM(bgmPath, presetID, totalDuration)
	if err != nil {
		return nil, himindjsonrpc.InternalError("BGM 生成失败: " + err.Error())
	}

	clipInputs := make([]voiceClipInput, 0, len(clips))
	for _, clip := range clips {
		clipInputs = append(clipInputs, voiceClipInput{
			Path: clip.Path, Start: clip.Start, WindowSeconds: clip.WindowSeconds,
			Tempo: clip.Tempo, FitToWindow: true, Label: clip.BeatID,
		})
	}
	mixPath := filepath.Join(audioDir, "mix.m4a")
	voiceGain := 0.0
	if in.VoiceGainDB != nil {
		voiceGain = *in.VoiceGainDB
	}
	bgmGain := -20.0
	if in.BGMGainDB != nil {
		bgmGain = *in.BGMGainDB
	}
	ducking := true
	if in.Ducking != nil {
		ducking = *in.Ducking
	}
	if err := mixAudio(root, clipInputs, bgmPath, totalDuration, voiceGain, bgmGain, ducking, mixPath); err != nil {
		return nil, himindjsonrpc.InternalError("混音失败: " + err.Error())
	}
	qa, qaErr := analyzeAudio(mixPath, totalDuration)
	if qaErr != nil {
		warnings = append(warnings, "audio QA 未完成: "+qaErr.Error())
	}
	if qa != nil {
		qa.Path = workspaceRelativePath(root, qa.Path)
		warnings = append(warnings, qa.Warnings...)
		warnings = append(warnings, qa.Blockers...)
	}

	manifest := audioManifest{
		Version: "himind.audio-manifest.v1", CreativeID: item.ID, ProjectID: strings.TrimSpace(in.ProjectID),
		GeneratedAt: now(), VoiceProvider: provider, Voice: clips[0].Voice, Rate: rate,
		VoiceLicense: clips[0].License, CommercialVoice: clips[0].CommercialUse,
		VoiceGainDB: voiceGain, BGMPreset: presetID, BGMGainDB: bgmGain, Ducking: ducking,
		DurationSeconds: totalDuration, Clips: clips, Subtitles: cues, BGM: &bgm,
		MixPath: workspaceRelativePath(root, mixPath), SubtitlesPath: workspaceRelativePath(root, subtitlesPath), QA: qa,
		Source: provider + " + ffmpeg-lavfi; voice license=" + clips[0].License + "; BGM generated original",
	}
	if err := writeJSON(filepath.Join(audioDir, audioManifestName), manifest); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}

	attachPath := ""
	if strings.TrimSpace(in.ProjectID) != "" {
		attachPath, err = attachSubtitlesToProject(root, in.ProjectID, cues, mixPath)
		if err != nil {
			warnings = append(warnings, "字幕未写入项目: "+err.Error())
		}
	}
	return map[string]any{
		"state": "ready", "manifest": manifest, "warnings": warnings, "attached_path": attachPath,
		"next": []string{"若项目源码变化，先调用 preview.review", "调用 preview.determinism", "调用 render.start 并传 audio_path=" + filepath.ToSlash(mixPath)},
	}, nil
}

// audioAttachCapability 把音频插件的 mix 与字幕轨挂到项目数据上。
// 音频生成归 audio-production；项目挂载归创作台。
func audioAttachCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	projectID := strings.TrimSpace(in.ProjectID)
	creativeID := strings.TrimSpace(in.CreativeID)
	if projectID == "" || creativeID == "" {
		return nil, himindjsonrpc.InvalidParams("project_id and creative_id are required")
	}
	audioPath := strings.TrimSpace(in.AudioPath)
	if audioPath == "" {
		return nil, himindjsonrpc.InvalidParams("audio_path is required")
	}
	resolvedAudio, err := workspaceFile(root, audioPath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("audio_path: " + err.Error())
	}
	item, err := loadCreative(root, creativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	cues := in.Subtitles
	if len(cues) == 0 {
		beats, _, beatErr := loadAudioBeats(root, creativeID)
		if beatErr != nil {
			return nil, himindjsonrpc.InvalidParams(beatErr.Error())
		}
		durations := map[string]float64{}
		if manifest, manifestErr := loadAudioManifest(root, item.ID); manifestErr == nil && manifest != nil {
			for _, clip := range manifest.Clips {
				durations[clip.BeatID] = clip.EffectiveDuration
			}
		}
		cues = buildSubtitles(beats, durations, 24, 0.9)
	}
	attached, err := attachSubtitlesToProject(root, projectID, cues, resolvedAudio)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{
		"state": "ready", "attached_path": attached, "subtitles": cues,
		"next": []string{"重新调用 preview.review", "重新调用 preview.determinism", "render.start 时传 audio_path"},
	}, nil
}
func loadAudioBeats(root, creativeID string) ([]audioBeat, creativePackage, error) {
	item, err := loadCreative(root, creativeID)
	if err != nil {
		return nil, creativePackage{}, err
	}
	rawBeats := asMapSlice(asMap(item.Script)["beats"])
	beats := make([]audioBeat, 0, len(rawBeats))
	cursor := 0.0
	for index, raw := range rawBeats {
		id := strings.TrimSpace(asString(raw["id"]))
		if id == "" {
			id = fmt.Sprintf("b%d", index+1)
		}
		duration := numberFloat(raw["duration_seconds"])
		if duration <= 0 {
			duration = 5
		}
		start := numberFloat(raw["start"])
		if start <= 0 && index > 0 {
			start = cursor
		}
		beats = append(beats, audioBeat{
			ID: id, Start: start, DurationSeconds: duration, End: start + duration,
			Narration:    strings.TrimSpace(asString(raw["narration"])),
			OnScreenText: strings.TrimSpace(asString(raw["on_screen_text"])),
			Purpose:      strings.TrimSpace(asString(raw["purpose"])),
		})
		cursor = start + duration
	}
	return beats, item, nil
}

func listSAPIVoices(root string) ([]audioVoiceInfo, string) {
	if runtime.GOOS != "windows" {
		return []audioVoiceInfo{}, "当前平台没有 Windows SAPI"
	}
	script := `$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Speech
$s = New-Object System.Speech.Synthesis.SpeechSynthesizer
try {
  $items = @($s.GetInstalledVoices() | ForEach-Object {
    $i = $_.VoiceInfo
    [pscustomobject]@{name=$i.Name; culture=$i.Culture.Name; gender=[string]$i.Gender; age=[string]$i.Age}
  })
  $items | ConvertTo-Json -Depth 3
} finally { $s.Dispose() }`
	output, err := runPowerShellFile(root, "voices.ps1", script, nil, 30*time.Second)
	if err != nil {
		return []audioVoiceInfo{}, err.Error()
	}
	output = strings.TrimSpace(output)
	if output == "" || output == "null" {
		return []audioVoiceInfo{}, "没有检测到已安装的 SAPI 语音"
	}
	var voices []audioVoiceInfo
	if strings.HasPrefix(output, "[") {
		if err := json.Unmarshal([]byte(output), &voices); err != nil {
			return []audioVoiceInfo{}, err.Error()
		}
	} else {
		var single audioVoiceInfo
		if err := json.Unmarshal([]byte(output), &single); err != nil {
			return []audioVoiceInfo{}, err.Error()
		}
		voices = append(voices, single)
	}
	defaultSet := false
	for index := range voices {
		if strings.EqualFold(voices[index].Culture, "zh-CN") && !defaultSet {
			voices[index].Default = true
			defaultSet = true
		}
	}
	sort.SliceStable(voices, func(i, j int) bool {
		if voices[i].Culture != voices[j].Culture {
			return voices[i].Culture < voices[j].Culture
		}
		return voices[i].Name < voices[j].Name
	})
	return voices, ""
}

func resolveVoice(root, requested, language string) (string, error) {
	voices, warning := listSAPIVoices(root)
	if len(voices) == 0 {
		return "", fmt.Errorf("没有可用的 SAPI 语音: %s", warning)
	}
	requested = strings.TrimSpace(requested)
	if requested != "" {
		for _, voice := range voices {
			if strings.EqualFold(voice.Name, requested) {
				return voice.Name, nil
			}
		}
		for _, voice := range voices {
			if strings.Contains(strings.ToLower(voice.Name), strings.ToLower(requested)) {
				return voice.Name, nil
			}
		}
		return "", fmt.Errorf("未找到语音: %s", requested)
	}
	for _, voice := range voices {
		if strings.EqualFold(voice.Culture, language) {
			return voice.Name, nil
		}
	}
	return voices[0].Name, nil
}

func synthesizeSAPI(root, text, output, voice string, rate, volume int) (voiceClip, error) {
	if runtime.GOOS != "windows" {
		return voiceClip{}, fmt.Errorf("Windows SAPI 仅支持 Windows")
	}
	if _, err := workspaceFile(root, output); err != nil {
		return voiceClip{}, err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return voiceClip{}, err
	}
	rate = maxInt(-10, minInt(10, rate))
	volume = maxInt(0, minInt(100, volume))
	script := `$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Speech
$s = New-Object System.Speech.Synthesis.SpeechSynthesizer
try {
  $voices = @($s.GetInstalledVoices() | ForEach-Object { $_.VoiceInfo })
  $match = $null
  if ($env:HIMIND_TTS_VOICE) { $match = $voices | Where-Object { $_.Name -eq $env:HIMIND_TTS_VOICE } | Select-Object -First 1 }
  if (-not $match) { $match = $voices | Where-Object { $_.Culture.Name -eq 'zh-CN' } | Select-Object -First 1 }
  if ($match) { try { $s.SelectVoice($match.Name) } catch { Write-Output ("voice-debug env=[" + $env:HIMIND_TTS_VOICE + "] match=[" + $match.Name + "] default=[" + $s.Voice.Name + "] ps=" + $PSHOME + " x64=" + [Environment]::Is64BitProcess + " arch=" + $env:PROCESSOR_ARCHITECTURE + " user=" + $env:USERNAME + " profile=" + $env:USERPROFILE + " session=" + $env:SESSIONNAME); throw } }
  $s.Rate = [Math]::Max(-10,[Math]::Min(10,[int]$env:HIMIND_TTS_RATE))
  $s.Volume = [Math]::Max(0,[Math]::Min(100,[int]$env:HIMIND_TTS_VOLUME))
  $s.SetOutputToWaveFile($env:HIMIND_TTS_OUTPUT)
  $s.Speak($env:HIMIND_TTS_TEXT)
} finally { $s.Dispose() }`
	env := []string{
		"HIMIND_TTS_TEXT=" + text,
		"HIMIND_TTS_OUTPUT=" + output,
		"HIMIND_TTS_VOICE=" + voice,
		"HIMIND_TTS_RATE=" + strconv.Itoa(rate),
		"HIMIND_TTS_VOLUME=" + strconv.Itoa(volume),
	}
	if _, err := runPowerShellFile(root, "tts.ps1", script, env, 120*time.Second); err != nil {
		return voiceClip{}, err
	}
	info, err := os.Stat(output)
	if err != nil || info.Size() == 0 {
		return voiceClip{}, fmt.Errorf("SAPI 未生成有效 WAV")
	}
	duration, err := probeAudioDuration(output)
	if err != nil {
		return voiceClip{}, err
	}
	hash, err := fileSHA256(output)
	if err != nil {
		return voiceClip{}, err
	}
	return voiceClip{
		Text: text, Path: workspaceRelativePath(root, output), Provider: "windows-sapi", Voice: voice, Rate: rate,
		DurationSeconds: duration, EffectiveDuration: duration, Tempo: 1, SHA256: hash,
	}, nil
}

func runPowerShellFile(root, name, script string, extraEnv []string, timeout time.Duration) (string, error) {
	dir := filepath.Join(root, metadataDir, "tools")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0644); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", path)
	command.Env = append(os.Environ(), extraEnv...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("PowerShell 执行超时")
	}
	if err != nil {
		return "", fmt.Errorf("PowerShell 执行失败: %v\nstdout=%s\nstderr=%s", err, conciseProcessText(stdout.Bytes()), conciseProcessText(stderr.Bytes()))
	}
	return stdout.String(), nil
}

func runPowerShellScript(script string, extraEnv []string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-EncodedCommand", encodePowerShell(script))
	command.Env = append(os.Environ(), extraEnv...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("PowerShell 执行超时")
	}
	if err != nil {
		return "", fmt.Errorf("PowerShell 执行失败: %v\nstdout=%s\nstderr=%s", err, conciseProcessText(stdout.Bytes()), conciseProcessText(stderr.Bytes()))
	}
	return stdout.String(), nil
}

func encodePowerShell(script string) string {
	units := utf16.Encode([]rune(script))
	buf := make([]byte, len(units)*2)
	for index, unit := range units {
		binary.LittleEndian.PutUint16(buf[index*2:], unit)
	}
	return base64.StdEncoding.EncodeToString(buf)
}

func conciseProcessText(data []byte) string {
	text := strings.TrimSpace(string(data))
	if len([]rune(text)) > 1200 {
		r := []rune(text)
		return string(r[len(r)-1200:])
	}
	return text
}

func applyFitPolicy(clip *voiceClip) {
	if clip.DurationSeconds <= 0 {
		return
	}
	if clip.WindowSeconds <= 0 {
		clip.EffectiveDuration = clip.DurationSeconds
		clip.Tempo = 1
		return
	}
	target := math.Max(0.4, clip.WindowSeconds-0.12)
	if clip.DurationSeconds <= target {
		clip.EffectiveDuration = clip.DurationSeconds
		clip.Tempo = 1
		return
	}
	tempo := clip.DurationSeconds / target
	if tempo > 1.5 {
		tempo = 1.5
	}
	clip.Tempo = math.Round(tempo*1000) / 1000
	clip.EffectiveDuration = clip.DurationSeconds / clip.Tempo
	if clip.EffectiveDuration > clip.WindowSeconds+0.02 {
		clip.FitWarning = fmt.Sprintf("%s: 旁白 %.2fs 超过镜头 %.2fs，即使 1.5x 仍会溢出 %.2fs", clip.BeatID, clip.DurationSeconds, clip.WindowSeconds, clip.EffectiveDuration-clip.WindowSeconds)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func loadBGMPresets() ([]bgmPreset, error) {
	path := filepath.Join(pluginRoot(), "catalog", "audio-presets.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return defaultBGMPresets(), nil
	}
	var payload struct {
		Schema  string      `json:"schema"`
		Presets []bgmPreset `json:"presets"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("catalog/audio-presets.json: %w", err)
	}
	if len(payload.Presets) == 0 {
		return defaultBGMPresets(), nil
	}
	return payload.Presets, nil
}

func defaultBGMPresets() []bgmPreset {
	return []bgmPreset{{
		ID: "documentary-drone", Name: "纪录片低频", Description: "克制的低频铺底与极轻粉噪，适合知识解释。",
		Mood: []string{"documentary", "calm", "scientific"}, License: "generated-original", Loop: true,
		Tones:   []bgmTone{{Frequency: 55, Volume: 0.09, Lowpass: 220}, {Frequency: 82.41, Volume: 0.055, Lowpass: 300}, {Frequency: 110, Volume: 0.035, Lowpass: 380}},
		Noise:   &bgmNoise{Color: "pink", Amplitude: 0.035, Lowpass: 1100, Highpass: 45, Volume: 0.55},
		Tremolo: &bgmTremolo{Frequency: 0.1, Depth: 0.28}, Echo: &bgmEcho{InGain: 0.8, OutGain: 0.55, Delays: "700|1100", Decays: "0.22|0.13"},
		Gain: 0.58, FadeIn: 2, FadeOut: 2.5,
	}}
}

func findBGMPreset(id string) (bgmPreset, error) {
	presets, err := loadBGMPresets()
	if err != nil {
		return bgmPreset{}, err
	}
	for _, preset := range presets {
		if preset.ID == id {
			return preset, nil
		}
	}
	ids := make([]string, 0, len(presets))
	for _, preset := range presets {
		ids = append(ids, preset.ID)
	}
	return bgmPreset{}, fmt.Errorf("未知 BGM preset: %s (可选: %s)", id, strings.Join(ids, ", "))
}

func generateBGM(output, presetID string, duration float64) (bgmAsset, error) {
	preset, err := findBGMPreset(presetID)
	if err != nil {
		return bgmAsset{}, err
	}
	if duration <= 0 {
		return bgmAsset{}, fmt.Errorf("BGM 时长必须大于 0")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return bgmAsset{}, err
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return bgmAsset{}, fmt.Errorf("缺少 ffmpeg")
	}
	args := []string{"-y", "-hide_banner", "-loglevel", "error"}
	filterParts := make([]string, 0)
	inputIndex := 0
	labels := make([]string, 0)
	for _, tone := range preset.Tones {
		args = append(args, "-f", "lavfi", "-i", fmt.Sprintf("sine=frequency=%.3f:sample_rate=48000:duration=%.3f", tone.Frequency, duration))
		lowpass := ""
		if tone.Lowpass > 0 {
			lowpass = fmt.Sprintf(",lowpass=f=%d", tone.Lowpass)
		}
		label := fmt.Sprintf("t%d", inputIndex)
		filterParts = append(filterParts, fmt.Sprintf("[%d:a]volume=%.5f%s[%s]", inputIndex, tone.Volume, lowpass, label))
		labels = append(labels, "["+label+"]")
		inputIndex++
	}
	if preset.Noise != nil {
		color := firstNonEmpty(preset.Noise.Color, "pink")
		args = append(args, "-f", "lavfi", "-i", fmt.Sprintf("anoisesrc=color=%s:sample_rate=48000:duration=%.3f:amplitude=%.5f", color, duration, preset.Noise.Amplitude))
		volume := preset.Noise.Volume
		if volume <= 0 {
			volume = 1
		}
		label := fmt.Sprintf("n%d", inputIndex)
		filterParts = append(filterParts, buildNoiseFilter(inputIndex, preset.Noise.Highpass, preset.Noise.Lowpass, volume, label))
		labels = append(labels, "["+label+"]")
		inputIndex++
	}
	if len(labels) == 0 {
		return bgmAsset{}, fmt.Errorf("BGM preset 没有音源")
	}
	chain := fmt.Sprintf("%samix=inputs=%d:normalize=0", strings.Join(labels, ""), len(labels))
	if preset.Tremolo != nil && preset.Tremolo.Frequency > 0 {
		frequency := math.Max(0.1, preset.Tremolo.Frequency)
		chain += fmt.Sprintf(",tremolo=f=%.4f:d=%.3f", frequency, preset.Tremolo.Depth)
	}
	if preset.Echo != nil && preset.Echo.Delays != "" {
		chain += fmt.Sprintf(",aecho=%.3f:%.3f:%s:%s", preset.Echo.InGain, preset.Echo.OutGain, preset.Echo.Delays, preset.Echo.Decays)
	}
	gain := preset.Gain
	if gain <= 0 {
		gain = 0.6
	}
	fadeIn := preset.FadeIn
	if fadeIn <= 0 {
		fadeIn = 2
	}
	fadeOut := preset.FadeOut
	if fadeOut <= 0 {
		fadeOut = 2
	}
	chain += fmt.Sprintf(",atrim=0:%.3f,aformat=sample_fmts=s16:channel_layouts=stereo,alimiter=limit=0.85:level=disabled,afade=t=in:st=0:d=%.3f,afade=t=out:st=%.3f:d=%.3f,volume=%.3f[out]",
		duration, math.Min(fadeIn, duration/3), math.Max(0, duration-fadeOut), math.Min(fadeOut, duration/3), gain)
	filterParts = append(filterParts, chain)
	args = append(args, "-filter_complex", strings.Join(filterParts, ";"), "-map", "[out]", "-c:a", "pcm_s16le", "-ar", "48000", "-ac", "2", output)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, ffmpeg, args...)
	outputBytes, err := command.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return bgmAsset{}, fmt.Errorf("BGM 生成超时")
	}
	if err != nil {
		return bgmAsset{}, fmt.Errorf("ffmpeg BGM: %v\n%s", err, conciseProcessText(outputBytes))
	}
	actualDuration, err := probeAudioDuration(output)
	if err != nil {
		return bgmAsset{}, err
	}
	hash, err := fileSHA256(output)
	if err != nil {
		return bgmAsset{}, err
	}
	return bgmAsset{
		ID: preset.ID, Name: preset.Name, Path: output, DurationSeconds: actualDuration,
		SHA256: hash, License: preset.License, Loop: preset.Loop, Mood: preset.Mood, Generated: true,
	}, nil
}

func buildNoiseFilter(index, highpass, lowpass int, volume float64, label string) string {
	parts := make([]string, 0, 3)
	if highpass > 0 {
		parts = append(parts, fmt.Sprintf("highpass=f=%d", highpass))
	}
	if lowpass > 0 {
		parts = append(parts, fmt.Sprintf("lowpass=f=%d", lowpass))
	}
	parts = append(parts, fmt.Sprintf("volume=%.4f", volume))
	return fmt.Sprintf("[%d:a]%s[%s]", index, strings.Join(parts, ","), label)
}

func mixAudio(root string, clips []voiceClipInput, bgmPath string, duration, voiceGain, bgmGain float64, ducking bool, output string) error {
	if duration <= 0 {
		return fmt.Errorf("duration_seconds must be > 0")
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("缺少 ffmpeg")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	args := []string{"-y", "-hide_banner", "-loglevel", "error"}
	filterParts := make([]string, 0, len(clips)+8)
	voiceLabels := make([]string, 0, len(clips))
	inputIndex := 0
	for index, clip := range clips {
		path, err := workspaceFile(root, clip.Path)
		if err != nil {
			return fmt.Errorf("voice clip %d: %w", index, err)
		}
		if info, statErr := os.Stat(path); statErr != nil || info.IsDir() {
			return fmt.Errorf("voice clip 不存在: %s", clip.Path)
		}
		args = append(args, "-i", path)
		label := fmt.Sprintf("v%d", inputIndex)
		chain := "aformat=sample_fmts=fltp:sample_rates=48000:channel_layouts=stereo"
		tempo := clip.Tempo
		if clip.FitToWindow && clip.WindowSeconds > 0 {
			d, dErr := probeAudioDuration(path)
			if dErr == nil && d > clip.WindowSeconds-0.12 {
				tempo = d / math.Max(0.4, clip.WindowSeconds-0.12)
			}
		}
		if tempo > 1.001 {
			atempo := math.Min(2.0, tempo)
			chain += fmt.Sprintf(",atempo=%.4f", atempo)
		}
		if voiceGain != 0 {
			chain += fmt.Sprintf(",volume=%.3fdB", voiceGain)
		}
		chain += fmt.Sprintf(",adelay=delays=%d:all=1", int(math.Round(math.Max(0, clip.Start)*1000)))
		filterParts = append(filterParts, fmt.Sprintf("[%d:a]%s[%s]", inputIndex, chain, label))
		voiceLabels = append(voiceLabels, "["+label+"]")
		inputIndex++
	}
	bgmIndex := -1
	if strings.TrimSpace(bgmPath) != "" {
		path, err := workspaceFile(root, bgmPath)
		if err != nil {
			return fmt.Errorf("bgm: %w", err)
		}
		args = append(args, "-i", path)
		bgmIndex = inputIndex
		inputIndex++
	}
	voiceMix := ""
	if len(voiceLabels) > 0 {
		voiceMix = fmt.Sprintf("%samix=inputs=%d:duration=longest:normalize=0,alimiter=limit=0.90:level=disabled", strings.Join(voiceLabels, ""), len(voiceLabels))
		if ducking && bgmIndex >= 0 {
			filterParts = append(filterParts, voiceMix+",asplit=2[voice][sc]")
		} else {
			filterParts = append(filterParts, voiceMix+"[voice]")
		}
	}
	bgmLabel := ""
	if bgmIndex >= 0 {
		fadeOut := math.Max(0, duration-2)
		chain := fmt.Sprintf("aformat=sample_fmts=fltp:sample_rates=48000:channel_layouts=stereo,volume=%.3fdB,atrim=0:%.3f,afade=t=in:st=0:d=1.5,afade=t=out:st=%.3f:d=2", bgmGain, duration, fadeOut)
		filterParts = append(filterParts, fmt.Sprintf("[%d:a]%s[bgm]", bgmIndex, chain))
		bgmLabel = "[bgm]"
	}
	switch {
	case voiceMix != "" && bgmLabel != "" && ducking:
		filterParts = append(filterParts, fmt.Sprintf("%s[sc]sidechaincompress=threshold=0.025:ratio=6:attack=25:release=600[bgmduck]", bgmLabel))
		filterParts = append(filterParts, fmt.Sprintf("[voice][bgmduck]amix=inputs=2:duration=longest:normalize=0,apad=whole_dur=%.3f,atrim=0:%.3f,loudnorm=I=-16:TP=-2.5:LRA=11,alimiter=limit=0.78:level=disabled[aout]", duration, duration))
	case voiceMix != "" && bgmLabel != "":
		filterParts = append(filterParts, fmt.Sprintf("[voice]%samix=inputs=2:duration=longest:normalize=0,apad=whole_dur=%.3f,atrim=0:%.3f,loudnorm=I=-16:TP=-2.5:LRA=11,alimiter=limit=0.78:level=disabled[aout]", bgmLabel, duration, duration))
	case voiceMix != "":
		filterParts = append(filterParts, fmt.Sprintf("[voice]apad=whole_dur=%.3f,atrim=0:%.3f,loudnorm=I=-16:TP=-2.5:LRA=11,alimiter=limit=0.78:level=disabled[aout]", duration, duration))
	case bgmLabel != "":
		filterParts = append(filterParts, fmt.Sprintf("%sapad=whole_dur=%.3f,atrim=0:%.3f,loudnorm=I=-16:TP=-2.5:LRA=11,alimiter=limit=0.78:level=disabled[aout]", bgmLabel, duration, duration))
	default:
		return fmt.Errorf("没有可混音的音轨")
	}
	args = append(args, "-filter_complex", strings.Join(filterParts, ";"), "-map", "[aout]",
		"-c:a", "aac", "-b:a", "192k", "-ar", "48000", "-ac", "2", "-movflags", "+faststart", output)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, ffmpeg, args...)
	outputBytes, err := command.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("混音超时")
	}
	if err != nil {
		return fmt.Errorf("ffmpeg mix: %v\n%s", err, conciseProcessText(outputBytes))
	}
	if info, statErr := os.Stat(output); statErr != nil || info.Size() == 0 {
		return fmt.Errorf("混音未生成有效文件")
	}
	return nil
}

func maxAudioEnd(clips []voiceClipInput) float64 {
	maxEnd := 0.0
	for _, clip := range clips {
		maxEnd = math.Max(maxEnd, clip.Start+clip.WindowSeconds)
	}
	return maxEnd
}

func probeAudioDuration(path string) (float64, error) {
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return 0, fmt.Errorf("缺少 ffprobe")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, ffprobe, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	output, err := command.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration: %v\n%s", err, conciseProcessText(output))
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration invalid: %s", strings.TrimSpace(string(output)))
	}
	return value, nil
}

func analyzeAudio(path string, expected float64) (*audioQAResult, error) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("音频文件不存在: %s", path)
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("缺少 ffprobe")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	probeArgs := []string{"-v", "error", "-show_entries", "format=duration:stream=codec_name,channels,sample_rate", "-of", "json", path}
	probe := exec.CommandContext(ctx, ffprobe, probeArgs...)
	probeRaw, err := probe.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffprobe: %v\n%s", err, conciseProcessText(probeRaw))
	}
	var parsed struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecName  string `json:"codec_name"`
			Channels   int    `json:"channels"`
			SampleRate string `json:"sample_rate"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(probeRaw, &parsed); err != nil {
		return nil, err
	}
	result := &audioQAResult{Path: path, Status: "passed"}
	if len(parsed.Streams) == 0 {
		result.Blockers = append(result.Blockers, "没有音频流")
	}
	for _, stream := range parsed.Streams {
		if stream.CodecName != "" {
			result.Codec = stream.CodecName
			result.Channels = stream.Channels
			result.SampleRate, _ = strconv.Atoi(stream.SampleRate)
			break
		}
	}
	result.DurationSeconds, _ = strconv.ParseFloat(parsed.Format.Duration, 64)
	if expected > 0 {
		result.ExpectedSeconds = expected
		result.DurationDrift = math.Abs(result.DurationSeconds - expected)
		if result.DurationDrift > 0.2 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("音频时长 %.2fs 与期望 %.2fs 相差 %.2fs", result.DurationSeconds, expected, result.DurationDrift))
		}
	}

	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("缺少 ffmpeg")
	}
	loudCtx, loudCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer loudCancel()
	loud := exec.CommandContext(loudCtx, ffmpeg, "-hide_banner", "-nostats", "-i", path, "-af", "loudnorm=I=-16:TP=-1.5:LRA=11:print_format=json", "-f", "null", "-")
	loudRaw, _ := loud.CombinedOutput()
	values := parseLoudnormJSON(string(loudRaw))
	result.IntegratedLUFS = values["input_i"]
	result.TruePeakDBTP = values["input_tp"]
	result.LoudnessRange = values["input_lra"]
	if result.IntegratedLUFS != 0 && (result.IntegratedLUFS < -18 || result.IntegratedLUFS > -14) {
		result.Warnings = append(result.Warnings, fmt.Sprintf("整体响度 %.1f LUFS 不在 -16±2 范围内", result.IntegratedLUFS))
	}
	if result.TruePeakDBTP > -1.0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("真峰值 %.1f dBTP 接近削波", result.TruePeakDBTP))
	}

	volCtx, volCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer volCancel()
	vol := exec.CommandContext(volCtx, ffmpeg, "-hide_banner", "-nostats", "-i", path, "-af", "volumedetect", "-f", "null", "-")
	volRaw, _ := vol.CombinedOutput()
	result.MaxVolumeDB = parseMaxVolume(string(volRaw))
	if result.MaxVolumeDB > -0.1 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("采样峰值 %.1f dB，存在削波风险", result.MaxVolumeDB))
	}
	if len(result.Blockers) > 0 {
		result.Status = "failed"
	} else if len(result.Warnings) > 0 {
		result.Status = "warning"
	}
	result.OK = result.Status != "failed"
	return result, nil
}

func parseLoudnormJSON(text string) map[string]float64 {
	result := map[string]float64{}
	matches := regexp.MustCompile(`(?s)\{[^{}]*"input_i"[^{}]*\}`).FindAllString(text, -1)
	if len(matches) == 0 {
		return result
	}
	var payload struct {
		InputI   string `json:"input_i"`
		InputTP  string `json:"input_tp"`
		InputLRA string `json:"input_lra"`
	}
	if err := json.Unmarshal([]byte(matches[len(matches)-1]), &payload); err != nil {
		return result
	}
	result["input_i"], _ = strconv.ParseFloat(payload.InputI, 64)
	result["input_tp"], _ = strconv.ParseFloat(payload.InputTP, 64)
	result["input_lra"], _ = strconv.ParseFloat(payload.InputLRA, 64)
	return result
}

func parseMaxVolume(text string) float64 {
	matches := regexp.MustCompile(`max_volume:\s*(-?[0-9.]+)\s*dB`).FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return 0
	}
	value, _ := strconv.ParseFloat(matches[len(matches)-1][1], 64)
	return value
}

func loadAudioManifest(root, creativeID string) (*audioManifest, error) {
	path := filepath.Join(creativeRoot(root, creativeID), "audio", audioManifestName)
	var manifest audioManifest
	if err := readJSON(path, &manifest); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return &manifest, nil
}

func attachSubtitlesToProject(root, projectID string, cues []subtitleCue, mixPath string) (string, error) {
	item, err := loadProject(root, projectID)
	if err != nil {
		return "", err
	}
	if projectMode(item) == "code" {
		sourceRoot, err := sourceRootForProject(root, item)
		if err != nil {
			return "", err
		}
		dataPath := filepath.Join(sourceRoot, "input", "data.json")
		data := map[string]any{}
		if raw, readErr := os.ReadFile(dataPath); readErr == nil {
			_ = json.Unmarshal(raw, &data)
		}
		if data == nil {
			data = map[string]any{}
		}
		data["subtitles"] = cues
		data["audio"] = map[string]any{
			"mix_path":  workspaceRelativePath(root, mixPath),
			"has_voice": true,
			"has_bgm":   true,
		}
		if err := writeJSON(dataPath, data); err != nil {
			return "", err
		}
		return dataPath, nil
	}
	recipe := item.Recipe
	if recipe == nil {
		recipe = map[string]any{}
	}
	data, _ := recipe["data"].(map[string]any)
	if data == nil {
		data = map[string]any{}
	}
	data["subtitles"] = cues
	data["audio"] = map[string]any{"mix_path": workspaceRelativePath(root, mixPath), "has_voice": true, "has_bgm": true}
	recipe["data"] = data
	item.Recipe = recipe
	item.Revision++
	item.UpdatedAt = now()
	if err := writeJSON(filepath.Join(root, metadataDir, projectDir, item.ID, "project.json"), item); err != nil {
		return "", err
	}
	return filepath.Join(root, metadataDir, projectDir, item.ID, "project.json"), nil
}

// ---------- 字幕切分 ----------

var subtitleClausePattern = regexp.MustCompile(`[^。！？!?；;，,、：:]+[。！？!?；;，,、：:]?`)

func splitSubtitleText(text string, maxChars int) []string {
	if maxChars < 6 {
		maxChars = 6
	}
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r", ""))
	if text == "" {
		return nil
	}
	clauses := subtitleClausePattern.FindAllString(text, -1)
	segments := make([]string, 0, len(clauses))
	for _, clause := range clauses {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}
		runes := []rune(clause)
		if len(runes) <= maxChars {
			segments = append(segments, clause)
			continue
		}
		for len(runes) > 0 {
			cut := maxChars
			if cut > len(runes) {
				cut = len(runes)
			}
			segments = append(segments, strings.TrimSpace(string(runes[:cut])))
			runes = runes[cut:]
		}
	}
	packed := make([]string, 0, len(segments))
	for _, segment := range segments {
		if len(packed) == 0 {
			packed = append(packed, segment)
			continue
		}
		current := []rune(packed[len(packed)-1])
		next := []rune(segment)
		if len(current)+len(next) <= maxChars {
			packed[len(packed)-1] = packed[len(packed)-1] + segment
		} else {
			packed = append(packed, segment)
		}
	}
	return packed
}

func buildSubtitles(beats []audioBeat, effective map[string]float64, maxChars int, minCue float64) []subtitleCue {
	cues := make([]subtitleCue, 0, len(beats)*2)
	for _, beat := range beats {
		text := strings.TrimSpace(beat.Narration)
		if text == "" {
			continue
		}
		segments := splitSubtitleText(text, maxChars)
		if len(segments) == 0 {
			continue
		}
		audioDuration := effective[beat.ID]
		if audioDuration <= 0 {
			audioDuration = beat.DurationSeconds
		}
		end := math.Min(beat.End, beat.Start+audioDuration+0.55)
		if end-beat.Start < 1.0 {
			end = math.Min(beat.End, beat.Start+1.0)
		}
		if end <= beat.Start {
			end = beat.End
		}
		for len(segments) > 1 && float64(len(segments))*minCue > end-beat.Start {
			last := segments[len(segments)-1]
			segments = append(segments[:len(segments)-2], segments[len(segments)-2]+last)
		}
		totalRunes := 0
		for _, segment := range segments {
			totalRunes += len([]rune(segment))
		}
		if totalRunes == 0 {
			continue
		}
		cursor := beat.Start
		for index, segment := range segments {
			weight := float64(len([]rune(segment))) / float64(totalRunes)
			next := cursor + (end-beat.Start)*weight
			if index == len(segments)-1 {
				next = end
			}
			if next <= cursor {
				next = cursor + 0.2
			}
			cues = append(cues, subtitleCue{Start: roundMillis(cursor), End: roundMillis(math.Min(next, beat.End)), Text: segment, BeatID: beat.ID})
			cursor = next
		}
	}
	return cues
}

func workspaceRelativePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func roundMillis(value float64) float64 {
	return math.Round(value*1000) / 1000
}
