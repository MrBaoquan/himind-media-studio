package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"himind-plugin/audio-production/internal/himindjsonrpc"
)

// audio_piper.go — 可校验的离线神经 TTS 运行时。
//
// 运行时与 voice pack 都是资产：下载地址、版本、SHA-256、许可证与兼容性
// 全部登记在 catalog/audio-runtimes.json。插件只负责安装、校验与调用。

func audioRuntimesRelPath() string {
	return filepath.Join("catalog", "audio-runtimes.json")
}

func loadAudioRuntimes() ([]audioRuntime, error) {
	path := filepath.Join(pluginRoot(), audioRuntimesRelPath())
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("缺少音频运行时目录: %w", err)
	}
	var catalog audioRuntimeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, fmt.Errorf("catalog/audio-runtimes.json: %w", err)
	}
	if catalog.Schema != "himind.audio-runtimes.v1" {
		return nil, fmt.Errorf("unexpected audio runtime schema: %s", catalog.Schema)
	}
	return catalog.Runtimes, nil
}

func findAudioRuntime(id string) (audioRuntime, error) {
	runtimes, err := loadAudioRuntimes()
	if err != nil {
		return audioRuntime{}, err
	}
	for _, runtimeItem := range runtimes {
		if runtimeItem.ID == id {
			return runtimeItem, nil
		}
	}
	return audioRuntime{}, fmt.Errorf("未知音频运行时: %s", id)
}

func audioRuntimeRoot(root string, runtimeItem audioRuntime) string {
	return filepath.Join(root, metadataDir, "runtimes", runtimeItem.ID+"-"+runtimeItem.Version)
}

func buildPiperRuntimeStatus(root string) (audioRuntimeStatus, error) {
	runtimeItem, err := findAudioRuntime("piper")
	if err != nil {
		return audioRuntimeStatus{}, err
	}
	dir := audioRuntimeRoot(root, runtimeItem)
	exe := filepath.Join(dir, "piper", "piper.exe")
	status := audioRuntimeStatus{
		ID: runtimeItem.ID, Version: runtimeItem.Version, Path: dir,
		Executable: exe, License: runtimeItem.License, Voices: runtimeItem.Voices,
	}
	if info, statErr := os.Stat(exe); statErr == nil && !info.IsDir() {
		status.Installed = true
	} else {
		status.Missing = append(status.Missing, "piper/piper.exe")
	}
	for index := range status.Voices {
		voice := &status.Voices[index]
		modelPath := filepath.Join(dir, filepath.FromSlash(voice.Model))
		configPath := filepath.Join(dir, filepath.FromSlash(voice.Config))
		if _, statErr := os.Stat(modelPath); statErr != nil {
			voice.Available = false
			status.Missing = append(status.Missing, voice.Model)
		}
		if _, statErr := os.Stat(configPath); statErr != nil {
			voice.Available = false
			status.Missing = append(status.Missing, voice.Config)
		}
	}
	if !status.Installed {
		status.Next = "调用 media.workbench.audio.runtime.install（runtime_id=piper）安装运行时与声音"
	}
	return status, nil
}

func resolvePiperVoice(root, requested string, allowRestricted bool) (audioRuntimeStatus, audioVoicePack, string, string, error) {
	status, err := buildPiperRuntimeStatus(root)
	if err != nil {
		return status, audioVoicePack{}, "", "", err
	}
	if !status.Installed {
		return status, audioVoicePack{}, "", "", fmt.Errorf("Piper 运行时尚未安装；先调用 media.workbench.audio.runtime.install")
	}
	requested = strings.TrimSpace(strings.TrimPrefix(requested, "piper:"))
	selected := -1
	if requested == "" {
		for index, voice := range status.Voices {
			if voice.Available {
				selected = index
				break
			}
		}
	} else {
		for index, voice := range status.Voices {
			if strings.EqualFold(voice.ID, requested) || strings.EqualFold(voice.Name, requested) {
				selected = index
				break
			}
		}
	}
	if selected < 0 {
		return status, audioVoicePack{}, "", "", fmt.Errorf("未找到 Piper 声音: %s", requested)
	}
	voice := status.Voices[selected]
	if !voice.Available {
		return status, voice, "", "", fmt.Errorf("声音 %s 当前不可用: %s", voice.ID, voice.Note)
	}
	if !voice.CommercialUse && !allowRestricted {
		return status, voice, "", "", fmt.Errorf("声音 %s 许可受限（%s）。仅开发/演示可用；确认后传 allow_restricted_voice=true", voice.ID, voice.License)
	}
	dir := audioRuntimeRoot(root, audioRuntime{ID: "piper", Version: status.Version})
	modelPath := filepath.Join(dir, filepath.FromSlash(voice.Model))
	configPath := filepath.Join(dir, filepath.FromSlash(voice.Config))
	return status, voice, modelPath, configPath, nil
}

func synthesizeVoice(root, text, output, provider, voice string, rate, volume int, allowRestricted bool) (voiceClip, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "piper":
		status, pack, modelPath, configPath, err := resolvePiperVoice(root, voice, allowRestricted)
		if err != nil {
			return voiceClip{}, err
		}
		return synthesizePiper(root, text, output, status, pack, modelPath, configPath, rate)
	case "windows-sapi":
		sapiVoice, err := resolveVoice(root, voice, "zh-CN")
		if err != nil {
			return voiceClip{}, err
		}
		clip, err := synthesizeSAPI(root, text, output, sapiVoice, rate, volume)
		if err != nil {
			return voiceClip{}, err
		}
		clip.License = "Windows SAPI system voice"
		clip.CommercialUse = true
		return clip, nil
	default:
		return voiceClip{}, fmt.Errorf("TTS provider 尚未接入: %s", provider)
	}
}

func synthesizePiper(root, text, output string, status audioRuntimeStatus, voice audioVoicePack, modelPath, configPath string, rate int) (voiceClip, error) {
	if _, err := workspaceFile(root, output); err != nil {
		return voiceClip{}, err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return voiceClip{}, err
	}
	lengthScale := 1.0 - float64(rate)*0.02
	if lengthScale < 0.75 {
		lengthScale = 0.75
	}
	if lengthScale > 1.35 {
		lengthScale = 1.35
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, status.Executable,
		"--model", modelPath,
		"--output_file", output,
		"--length_scale", fmt.Sprintf("%.3f", lengthScale),
		"--noise_scale", "0.667",
		"--noise_w", "0.8",
	)
	command.Dir = filepath.Dir(status.Executable)
	command.Stdin = strings.NewReader(text)
	outputBytes, err := command.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return voiceClip{}, fmt.Errorf("Piper 合成超时")
	}
	if err != nil {
		return voiceClip{}, fmt.Errorf("Piper 合成失败: %v\n%s", err, conciseProcessText(outputBytes))
	}
	info, err := os.Stat(output)
	if err != nil || info.Size() == 0 {
		return voiceClip{}, fmt.Errorf("Piper 未生成有效 WAV")
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
		Text: text, Path: workspaceRelativePath(root, output), Provider: "piper", Voice: voice.ID, Rate: rate,
		DurationSeconds: duration, EffectiveDuration: duration, Tempo: 1,
		License: voice.License, CommercialUse: voice.CommercialUse, SHA256: hash,
	}, nil
}

type audioRuntimeInstallManifest struct {
	ID          string                 `json:"id"`
	Version     string                 `json:"version"`
	InstalledAt string                 `json:"installed_at"`
	Root        string                 `json:"root"`
	License     string                 `json:"license"`
	Artifacts   []audioRuntimeArtifact `json:"artifacts"`
	Voices      []audioVoicePack       `json:"voices"`
}

func audioRuntimeInstallCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	runtimeID := firstNonEmpty(strings.TrimSpace(in.RuntimeID), "piper")
	runtimeItem, err := findAudioRuntime(runtimeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	dir := audioRuntimeRoot(root, runtimeItem)
	plan := map[string]any{
		"runtime_id": runtimeItem.ID,
		"version":    runtimeItem.Version,
		"root":       dir,
		"license":    runtimeItem.License,
		"artifacts":  runtimeItem.Artifacts,
		"voices":     runtimeItem.Voices,
	}
	if in.DryRun {
		return map[string]any{"state": "planned", "plan": plan, "dry_run": true}, nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	downloaded := make([]audioRuntimeArtifact, 0, len(runtimeItem.Artifacts))
	for _, artifact := range runtimeItem.Artifacts {
		if strings.TrimSpace(artifact.URL) == "" || strings.TrimSpace(artifact.SHA256) == "" {
			return nil, himindjsonrpc.InternalError("runtime artifact 缺少 url 或 sha256")
		}
		if artifact.Kind == "zip" {
			archive := filepath.Join(dir, ".download", "runtime.zip")
			if err := downloadArtifact(archive, artifact.URL, artifact.SHA256); err != nil {
				return nil, himindjsonrpc.InternalError(err.Error())
			}
			if err := extractZipToDir(archive, dir); err != nil {
				return nil, himindjsonrpc.InternalError(err.Error())
			}
			_ = os.Remove(archive)
		} else {
			dest := filepath.Join(dir, filepath.FromSlash(artifact.Dest))
			if !in.Force {
				if hash, hashErr := fileSHA256(dest); hashErr == nil && strings.EqualFold(hash, artifact.SHA256) {
					downloaded = append(downloaded, artifact)
					continue
				}
			}
			if err := downloadArtifact(dest, artifact.URL, artifact.SHA256); err != nil {
				return nil, himindjsonrpc.InternalError(err.Error())
			}
		}
		downloaded = append(downloaded, artifact)
	}
	manifest := audioRuntimeInstallManifest{
		ID: runtimeItem.ID, Version: runtimeItem.Version, InstalledAt: now(), Root: dir,
		License: runtimeItem.License, Artifacts: downloaded, Voices: runtimeItem.Voices,
	}
	if err := writeJSON(filepath.Join(dir, "runtime.json"), manifest); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	status, err := buildPiperRuntimeStatus(root)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"state": "ready", "runtime": status, "manifest": manifest}, nil
}

func downloadArtifact(destination, url, expectedSHA256 string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	tempPath := destination + ".part"
	_ = os.Remove(tempPath)
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "HiMind-Media-Workbench/0.17.1")
	client := &http.Client{Timeout: 12 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("下载失败 %s: %w", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("下载失败 %s: HTTP %d", url, response.StatusCode)
	}
	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(file, hash), response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("写入失败 %s: %w", destination, copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return closeErr
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, expectedSHA256) {
		_ = os.Remove(tempPath)
		return fmt.Errorf("SHA-256 校验失败 %s: 期望 %s 实际 %s", url, expectedSHA256, actual)
	}
	if err := os.Rename(tempPath, destination); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	_ = size
	return nil
}

func extractZipToDir(archive, destination string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	root, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	for _, file := range reader.File {
		clean := filepath.Clean(filepath.FromSlash(file.Name))
		if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			continue
		}
		target := filepath.Join(root, clean)
		rel, err := filepath.Rel(root, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("zip 条目越界: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		source, err := file.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			_ = source.Close()
			return err
		}
		_, copyErr := io.Copy(output, source)
		_ = source.Close()
		_ = output.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}
