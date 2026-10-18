package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// code_project.go — Browser-Code 项目模式。
//
// 模板项目回答"用哪个 Shot / Recipe"；代码项目回答"AI 写了一个可持续修改的
// 视频工程"。两者共享项目、修订、预览、产物和反馈，但渲染契约不同。
//
// 代码项目只接受固定的 node render.mjs CLI：
//   --sheet / --strip / --stills / --frames / --encode
// 插件不执行任意 shell，也不接受 AI 临时拼接的命令。

type codeStylePack struct {
	Schema          string  `json:"schema"`
	ID              string  `json:"id"`
	Version         string  `json:"version"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Renderer        string  `json:"renderer"`
	Entry           string  `json:"entry"`
	VideoOut        string  `json:"video_out"`
	DurationSeconds float64 `json:"duration_seconds"`
	Preview         struct {
		SheetOut  string `json:"sheet_out"`
		StripOut  string `json:"strip_out"`
		StillsOut string `json:"stills_out"`
	} `json:"preview"`
	License     string `json:"license"`
	Attribution string `json:"attribution"`
	Trust       string `json:"trust,omitempty"`
	Network     struct {
		Mode  string   `json:"mode"`
		Allow []string `json:"allow"`
	} `json:"network"`
	Runtime struct {
		MaxWorkers int `json:"max_workers"`
	} `json:"runtime"`
	Quality struct {
		TextAudit struct {
			Enabled bool   `json:"enabled"`
			Page    string `json:"page"`
			Times   string `json:"times"`
		} `json:"text_audit"`
		BandingGate *bool `json:"banding_gate"`
	} `json:"quality"`
	Delivery struct {
		Web         bool `json:"web"`
		WebWidth    int  `json:"web_width"`
		WebCRF      int  `json:"web_crf"`
		MaxMasterMB int  `json:"max_master_mb"`
	} `json:"delivery"`
}

type codeWorkerSpec struct {
	Kind            string  `json:"kind"`
	WorkspaceRoot   string  `json:"workspace_root"`
	JobPath         string  `json:"job_path"`
	ProjectID       string  `json:"project_id"`
	SourceDir       string  `json:"source_dir"`
	Entry           string  `json:"entry"`
	ArtifactID      string  `json:"artifact_id"`
	ArtifactPath    string  `json:"artifact_path"`
	Workers         int     `json:"workers"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	Chrome          string  `json:"chrome"`
	AudioPath       string  `json:"audio_path,omitempty"`
	WebCopy         bool    `json:"web_copy,omitempty"`
	WebWidth        int     `json:"web_width,omitempty"`
	WebCRF          int     `json:"web_crf,omitempty"`
	MaxMasterMB     int     `json:"max_master_mb,omitempty"`
}

const codeProjectSchema = "himind.video.code-project.v1"

var codeSourceSkipDirs = map[string]bool{
	"node_modules":  true,
	"out":           true,
	".cache":        true,
	".git":          true,
	".himind-media": true,
	"dist":          true,
}

func projectMode(item project) string {
	mode := strings.ToLower(strings.TrimSpace(item.Mode))
	if mode == "" {
		return "template"
	}
	return mode
}

func readCodeStylePack(sourceRoot string) (codeStylePack, error) {
	var pack codeStylePack
	path := filepath.Join(sourceRoot, "style-pack.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return pack, fmt.Errorf("代码项目缺少 style-pack.json: %w", err)
	}
	if err := json.Unmarshal(raw, &pack); err != nil {
		return pack, fmt.Errorf("解析 style-pack.json: %w", err)
	}
	if pack.ID == "" {
		return pack, fmt.Errorf("style-pack.json 缺少 id")
	}
	if pack.Version == "" {
		pack.Version = "1.0.0"
	}
	if pack.Renderer == "" {
		pack.Renderer = "browser-code"
	}
	if pack.Renderer != "browser-code" {
		return pack, fmt.Errorf("当前只支持 renderer=browser-code")
	}
	if pack.Entry == "" {
		pack.Entry = "render.mjs"
	}
	if pack.VideoOut == "" {
		pack.VideoOut = "out/video.mp4"
	}
	if pack.DurationSeconds <= 0 {
		pack.DurationSeconds = 10
	}
	if pack.Preview.SheetOut == "" {
		pack.Preview.SheetOut = "out/check/sheet.jpg"
	}
	if pack.Preview.StripOut == "" {
		pack.Preview.StripOut = "out/check/strip.jpg"
	}
	if pack.Preview.StillsOut == "" {
		pack.Preview.StillsOut = "out/stills"
	}
	if pack.Network.Mode == "" {
		pack.Network.Mode = "none"
	}
	if pack.Delivery.WebWidth == 0 {
		pack.Delivery.WebWidth = 1280
	}
	if pack.Delivery.WebCRF == 0 {
		pack.Delivery.WebCRF = 23
	}
	return pack, nil
}

func sourceRootForProject(root string, item project) (string, error) {
	if projectMode(item) != "code" {
		return "", fmt.Errorf("项目不是代码项目")
	}
	if strings.TrimSpace(item.SourceDir) == "" {
		return "", fmt.Errorf("代码项目缺少 source_dir")
	}
	path, err := workspaceFile(root, item.SourceDir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("source_dir 不存在或不是目录: %s", item.SourceDir)
	}
	return path, nil
}

// hashCodeSource 只哈希会影响画面的源码与素材；out/、node_modules/ 等生成物
// 不参与，避免一次预览就让项目和预览永久"过期"。
func hashCodeSource(sourceRoot string) (string, error) {
	files := make([]string, 0, 128)
	err := filepath.Walk(sourceRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == sourceRoot {
			return nil
		}
		rel, relErr := filepath.Rel(sourceRoot, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			if codeSourceSkipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if info.Name() == ".DS_Store" || info.Name() == "package-lock.json" || info.Name() == "npm-shrinkwrap.json" || strings.HasSuffix(info.Name(), ".tmp") {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)
	hash := sha256.New()
	for _, rel := range files {
		content, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		_, _ = hash.Write([]byte(rel))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(content)
		_, _ = hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func projectImport(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	mode := strings.ToLower(strings.TrimSpace(in.OpenMode))
	if mode != "" && mode != "code" {
		return nil, himindjsonrpc.InvalidParams("project.import currently supports mode=code only")
	}
	sourceRoot, err := workspaceFile(root, in.SourceDir)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("source_dir: " + err.Error())
	}
	info, err := os.Stat(sourceRoot)
	if err != nil || !info.IsDir() {
		return nil, himindjsonrpc.InvalidParams("source_dir 不存在或不是目录")
	}
	pack, err := readCodeStylePack(sourceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	sourceDirRel, relErr := filepath.Rel(root, sourceRoot)
	if relErr != nil || strings.HasPrefix(sourceDirRel, "..") {
		return nil, himindjsonrpc.InvalidParams("source_dir 必须位于 workspace_root 内")
	}
	sourceDirRel = filepath.ToSlash(sourceDirRel)
	styleID := firstNonEmpty(in.StylePack, pack.ID)
	styleVersion := firstNonEmpty(in.StyleVersion, pack.Version)
	entry := firstNonEmpty(in.Entry, pack.Entry, "render.mjs")
	entryPath, err := workspaceFile(sourceRoot, entry)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("entry: " + err.Error())
	}
	if stat, statErr := os.Stat(entryPath); statErr != nil || stat.IsDir() {
		return nil, himindjsonrpc.InvalidParams("entry 不存在: " + entry)
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = firstNonEmpty(pack.Name, filepath.Base(sourceRoot))
	}
	id := projectIDForName(name)
	projectRoot := filepath.Join(root, metadataDir, projectDir, id)
	if _, statErr := os.Stat(projectRoot); statErr == nil {
		existing, loadErr := loadProject(root, id)
		if loadErr == nil && projectMode(existing) == "code" && existing.SourceDir == sourceDirRel {
			return map[string]any{"project": existing, "workspace_root": root, "project_path": projectRoot, "existing": true, "next": []string{"调用 project.sync", "调用 preview.review"}}, nil
		}
		return nil, himindjsonrpc.InvalidParams("project already exists; choose another name")
	}
	hash, err := hashCodeSource(sourceRoot)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	nowValue := now()
	item := project{
		ID: id, Name: name, Description: in.Description, CreativeBrief: strings.TrimSpace(in.CreativeBrief),
		Mode: "code", Renderer: "browser-code", StylePack: styleID, StyleVersion: styleVersion,
		SourceDir: sourceDirRel, Entry: filepath.ToSlash(entry),
		SourceHash: hash, PreviewStatus: "not_generated", VideoOut: pack.VideoOut, DurationSeconds: pack.DurationSeconds,
		SheetOut: pack.Preview.SheetOut, StripOut: pack.Preview.StripOut, StillsOut: pack.Preview.StillsOut,
		CreatedAt: nowValue, UpdatedAt: nowValue, Revision: 1,
	}
	if err := writeJSON(filepath.Join(projectRoot, "project.json"), item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if item.CreativeBrief != "" {
		if err := os.WriteFile(filepath.Join(projectRoot, "brief.md"), []byte(item.CreativeBrief+"\n"), 0644); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
	}
	return map[string]any{
		"project": item, "workspace_root": root, "project_path": projectRoot,
		"source_path": sourceRoot, "source_hash": hash,
		"next": []string{"调用 preview.review 生成审查图", "根据审查图修改代码", "调用 project.sync 后重新 preview.review", "通过 quality.check 后 render.start"},
	}, nil
}

func projectSync(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if projectMode(item) != "code" {
		return nil, himindjsonrpc.InvalidParams("project.sync is only for code projects")
	}
	sourceRoot, err := sourceRootForProject(root, item)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	hash, err := hashCodeSource(sourceRoot)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	changed := hash != item.SourceHash
	if changed {
		item.SourceHash = hash
		item.Revision++
		item.PreviewStatus = "stale"
		item.DeterminismStatus = "stale"
		item.DeterminismHash = ""
		item.UpdatedAt = now()
		if err := writeJSON(filepath.Join(root, metadataDir, projectDir, item.ID, "project.json"), item); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
	}
	storyboard := filepath.Join(sourceRoot, "STORYBOARD.md")
	storyboardExists := fileExists(storyboard)
	return map[string]any{
		"project": item, "changed": changed, "source_hash": hash,
		"storyboard_exists": storyboardExists, "preview_status": item.PreviewStatus,
		"next": []string{"源码变化后调用 preview.review 重新生成审查图"},
	}, nil
}

func projectContext(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item.WorkspaceRoot = root
	contextPayload := map[string]any{
		"project":        item,
		"mode":           projectMode(item),
		"workspace_root": root,
		"jobs":           projectJobs(root, item.ID),
		"artifacts":      projectArtifacts(root, item.ID),
		"latest_video":   latestArtifact(projectArtifacts(root, item.ID), "video/"),
		"latest_preview": latestArtifact(projectArtifacts(root, item.ID), "preview/"),
	}
	if projectMode(item) == "code" {
		if sourceRoot, sourceErr := sourceRootForProject(root, item); sourceErr == nil {
			contextPayload["source_path"] = sourceRoot
			contextPayload["storyboard"] = readTextExcerpt(filepath.Join(sourceRoot, "STORYBOARD.md"), 6000)
			if hash, hashErr := hashCodeSource(sourceRoot); hashErr == nil {
				contextPayload["current_source_hash"] = hash
				contextPayload["source_in_sync"] = hash == item.SourceHash
			}
		}
	}
	feedbackItems := feedbackEntriesWithStatus(root, item.ID)
	pendingFeedback := make([]map[string]any, 0)
	for _, entry := range feedbackItems {
		if entry["status"] == "open" {
			pendingFeedback = append(pendingFeedback, entry)
		}
	}
	contextPayload["feedback"] = feedbackItems
	contextPayload["pending_feedback"] = pendingFeedback
	contextPayload["next_actions"] = codeProjectNextActions(item)
	return contextPayload, nil
}

func codeProjectNextActions(item project) []string {
	if projectMode(item) != "code" {
		return []string{"project.get", "plan.validate", "preview.start", "quality.check", "render.start"}
	}
	actions := []string{}
	if item.PreviewStatus != "ready" {
		actions = append(actions, "preview.review")
	}
	if item.PreviewHash != "" && item.PreviewHash != item.SourceHash {
		actions = append(actions, "project.sync", "preview.review")
	}
	actions = append(actions, "quality.check", "render.start")
	return actions
}

func previewReview(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if projectMode(item) != "code" {
		return nil, himindjsonrpc.InvalidParams("preview.review is only for code projects")
	}
	sourceRoot, err := sourceRootForProject(root, item)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if _, err := exec.LookPath("node"); err != nil {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("renderer_runtime_missing", "runtime", "缺少 Node.js", "安装 Node.js 后重试")}, "retryable": true}, nil
	}
	chrome := findRemotionBrowser()
	if chrome == "" {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("chrome_missing", "runtime", "找不到 Chrome/Edge", "安装 Chrome 或设置 CHROME_PATH")}, "retryable": true}, nil
	}
	if err := ensureCodeDependencies(sourceRoot); err != nil {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("code_dependencies_missing", "runtime", err.Error(), "在代码项目目录运行 npm install")}, "retryable": true}, nil
	}
	hash, err := hashCodeSource(sourceRoot)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if hash != item.SourceHash {
		item.SourceHash = hash
		item.Revision++
		item.PreviewStatus = "stale"
	}
	kind := strings.ToLower(strings.TrimSpace(in.ReviewKind))
	if kind == "" {
		kind = "sheet"
	}
	width := in.Width
	if width <= 0 {
		width = 540
	}
	cols := in.Cols
	if cols <= 0 {
		cols = 3
	}
	absSource := sourceRoot
	args := []string{filepath.Join(absSource, filepath.FromSlash(item.Entry))}
	artifactPath := ""
	artifactKind := ""
	switch kind {
	case "sheet":
		times := firstNonEmpty(in.Times, "1,3,5,7,9")
		artifactPath = filepath.Join(absSource, filepath.FromSlash(item.SheetOut))
		args = append(args, "--sheet="+times, fmt.Sprintf("--cols=%d", cols), fmt.Sprintf("--w=%d", width), "--chrome="+chrome, "--out="+artifactPath)
		artifactKind = "preview/sheet"
	case "strip":
		rangeValue := firstNonEmpty(in.Range, "2:3")
		artifactPath = filepath.Join(absSource, filepath.FromSlash(item.StripOut))
		args = append(args, "--strip="+rangeValue, fmt.Sprintf("--cols=%d", maxInt(cols, 6)), fmt.Sprintf("--w=%d", maxInt(width, 320)), "--chrome="+chrome, "--out="+artifactPath)
		artifactKind = "preview/strip"
	case "stills":
		times := firstNonEmpty(in.Times, "2,5,8")
		artifactPath = filepath.Join(absSource, filepath.FromSlash(item.StillsOut))
		args = append(args, "--stills="+times, "--chrome="+chrome, "--out="+artifactPath)
	default:
		return nil, himindjsonrpc.InvalidParams("kind must be sheet, strip or stills")
	}
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil && kind != "stills" {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	started := time.Now()
	output, runErr := runCodeNode(sourceRoot, args, 10*time.Minute)
	if runErr != nil {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("code_preview_failed", "preview", conciseProcessOutput(output, runErr), "检查 render.mjs、Chrome 路径与依赖")}, "retryable": true, "output": string(output)}, nil
	}
	artifacts := make([]artifact, 0, 8)
	if kind == "stills" {
		entries, readErr := os.ReadDir(artifactPath)
		if readErr != nil {
			return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("code_preview_missing", "preview", "render.mjs 未生成 stills 目录", "检查 --stills 输出")}, "retryable": true}, nil
		}
		for _, entry := range entries {
			if entry.IsDir() || !isImageFile(entry.Name()) {
				continue
			}
			id := newID("preview-still")
			archived, archiveErr := archiveVideoArtifact(root, id, "preview/still", filepath.Join(artifactPath, entry.Name()))
			if archiveErr != nil {
				continue
			}
			itemArtifact, makeErr := makeArtifact(id, "preview/still", archived, item.ID, "browser-code", true)
			if makeErr == nil {
				artifacts = append(artifacts, itemArtifact)
			}
		}
	} else {
		info, statErr := os.Stat(artifactPath)
		if statErr != nil || info.Size() == 0 {
			return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("code_preview_missing", "preview", "render.mjs 未生成审查图", "检查 --sheet/--strip 输出路径")}, "retryable": true}, nil
		}
		id := newID("preview-" + kind)
		archived, archiveErr := archiveVideoArtifact(root, id, artifactKind, artifactPath)
		if archiveErr != nil {
			return nil, himindjsonrpc.InternalError(archiveErr.Error())
		}
		itemArtifact, makeErr := makeArtifact(id, artifactKind, archived, item.ID, "browser-code", true)
		if makeErr != nil {
			return nil, himindjsonrpc.InternalError(makeErr.Error())
		}
		artifacts = append(artifacts, itemArtifact)
	}
	item.PreviewStatus = "ready"
	item.PreviewHash = hash
	item.UpdatedAt = now()
	if err := writeJSON(filepath.Join(root, metadataDir, projectDir, item.ID, "project.json"), item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{
		"project": item, "kind": kind, "artifacts": artifacts,
		"source_hash": hash, "preview_status": item.PreviewStatus,
		"duration_ms": time.Since(started).Milliseconds(),
		"output":      string(output),
		"next":        []string{"在创作台查看审查图并记录反馈", "修改代码后调用 project.sync", "重新 preview.review", "通过 quality.check 后 render.start"},
	}, nil
}

func ensureCodeDependencies(sourceRoot string) error {
	if fileExists(filepath.Join(sourceRoot, "node_modules", "puppeteer-core", "package.json")) {
		return nil
	}
	npmPath, err := lookPathAny("npm.cmd", "npm")
	if err != nil {
		return fmt.Errorf("缺少 npm")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, npmPath, "install", "--no-audit", "--no-fund")
	command.Dir = sourceRoot
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install: %s", conciseProcessOutput(output, err))
	}
	return nil
}

func runCodeNode(sourceRoot string, args []string, timeout time.Duration) ([]byte, error) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return nil, err
	}
	pack, _ := readCodeStylePack(sourceRoot)
	networkMode := strings.ToLower(strings.TrimSpace(pack.Network.Mode))
	if networkMode == "" {
		networkMode = "none"
	}
	nodeArgs := append([]string{}, args...)
	if networkMode != "full" {
		guard := filepath.Join(pluginRoot(), "renderer", "browser-code", "network_guard.cjs")
		if fileExists(guard) {
			nodeArgs = append([]string{"--require", guard}, nodeArgs...)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, nodePath, nodeArgs...)
	command.Dir = sourceRoot
	env := append([]string{}, os.Environ()...)
	env = append(env, "HIMIND_NETWORK_MODE="+networkMode)
	env = append(env, "HIMIND_ALLOWED_HOSTS="+strings.Join(pack.Network.Allow, ","))
	chromeArgs := []string{}
	if networkMode == "none" {
		chromeArgs = append(chromeArgs, "--host-resolver-rules=MAP * ~NOTFOUND, EXCLUDE 127.0.0.1")
	} else if networkMode == "declared" {
		rules := []string{"MAP * ~NOTFOUND", "EXCLUDE 127.0.0.1"}
		for _, host := range pack.Network.Allow {
			host = strings.TrimSpace(host)
			if host != "" {
				rules = append(rules, "EXCLUDE "+host)
			}
		}
		chromeArgs = append(chromeArgs, "--host-resolver-rules="+strings.Join(rules, ", "))
	}
	if raw, marshalErr := json.Marshal(chromeArgs); marshalErr == nil {
		env = append(env, "HIMIND_CHROME_ARGS_JSON="+string(raw))
	}
	command.Env = env
	return command.CombinedOutput()
}

func qualityCheckCode(root string, item project) (any, *himindjsonrpc.Error) {
	sourceRoot, err := sourceRootForProject(root, item)
	if err != nil {
		return map[string]any{"passed": false, "checks": []map[string]any{}, "issues": []map[string]any{issue("code_source_missing", "project", err.Error(), "检查 source_dir")}, "project_id": item.ID}, nil
	}
	hash, _ := hashCodeSource(sourceRoot)
	previews := filterArtifacts(projectArtifacts(root, item.ID), "preview/")
	videos := filterArtifacts(projectArtifacts(root, item.ID), "video/")
	storyboard := fileExists(filepath.Join(sourceRoot, "STORYBOARD.md"))
	entry := fileExists(filepath.Join(sourceRoot, filepath.FromSlash(item.Entry)))
	checks := []map[string]any{
		{"id": "source_dir", "passed": true},
		{"id": "entry", "passed": entry},
		{"id": "storyboard", "passed": storyboard},
		{"id": "source_in_sync", "passed": hash == item.SourceHash},
		{"id": "preview_ready", "passed": item.PreviewStatus == "ready" && len(previews) > 0},
		{"id": "determinism", "passed": item.DeterminismStatus == "passed" && item.DeterminismHash == hash},
		{"id": "video", "passed": len(videos) > 0, "optional": true},
	}
	issues := []map[string]any{}
	if !entry {
		issues = append(issues, issue("code_entry_missing", "project", "入口文件不存在", "检查 entry 与 source_dir"))
	}
	if !storyboard {
		issues = append(issues, issue("code_storyboard_missing", "project", "缺少 STORYBOARD.md", "先写分镜再渲染"))
	}
	if hash != item.SourceHash {
		issues = append(issues, issue("code_source_stale", "preview", "源码已变化，预览已过期", "调用 project.sync 与 preview.review"))
	}
	if item.PreviewStatus != "ready" || len(previews) == 0 {
		issues = append(issues, issue("code_preview_required", "preview", "最终渲染前必须先完成审查图", "调用 preview.review"))
	}
	if item.DeterminismStatus != "passed" || item.DeterminismHash != hash {
		issues = append(issues, issue("code_determinism_required", "determinism", "必须通过同帧集重复渲染哈希检查", "调用 preview.determinism"))
	}
	passed := len(issues) == 0
	return map[string]any{"passed": passed, "checks": checks, "issues": issues, "project_id": item.ID, "source_hash": hash, "next": map[string]any{"render": "passed 时可调用 render.start", "fix": "根据 issues 修改代码或重新预览"}}, nil
}

func startCodeRender(root string, item project, in requestInput) (any, *himindjsonrpc.Error) {
	sourceRoot, err := sourceRootForProject(root, item)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	hash, err := hashCodeSource(sourceRoot)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if item.PreviewStatus != "ready" || item.PreviewHash == "" || item.PreviewHash != hash {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("code_preview_required", "preview", "最终渲染前必须先完成并审查最新预览", "调用 preview.review 后重试")}, "retryable": true}, nil
	}
	if item.DeterminismStatus != "passed" || item.DeterminismHash != hash {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("code_determinism_required", "determinism", "最终渲染前必须通过同帧集重复渲染哈希检查", "调用 preview.determinism 后重试")}, "retryable": true}, nil
	}
	if _, err := exec.LookPath("node"); err != nil {
		return rendererRuntimeBlocked("Node.js")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("ffmpeg_missing", "runtime", "缺少 ffmpeg", "安装 ffmpeg 并加入 PATH")}, "retryable": true}, nil
	}
	chrome := findRemotionBrowser()
	if chrome == "" {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("chrome_missing", "runtime", "找不到 Chrome/Edge", "安装 Chrome 或设置 CHROME_PATH")}, "retryable": true}, nil
	}
	jobRoot := filepath.Join(root, metadataDir, jobDir)
	artifactRoot := filepath.Join(root, metadataDir, artifactDir)
	if err := os.MkdirAll(jobRoot, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if err := os.MkdirAll(artifactRoot, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	jobID := newID("render")
	artifactID := newID("video-artifact")
	artifactPath := filepath.Join(artifactRoot, artifactID+".mp4")
	jobPath := filepath.Join(jobRoot, jobID+".json")
	jobItem := job{ID: jobID, ProjectID: item.ID, Stage: "render", Status: "queued", Progress: 0, Message: "等待 Browser-Code 渲染", CreatedAt: now(), UpdatedAt: now()}
	if err := writeJSON(jobPath, jobItem); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	pack, _ := readCodeStylePack(sourceRoot)
	workers := effectiveWorkerCount(in.Workers, pack.Runtime.MaxWorkers)
	spec := codeWorkerSpec{Kind: "browser-code", WorkspaceRoot: root, JobPath: jobPath, ProjectID: item.ID, SourceDir: item.SourceDir, Entry: item.Entry, ArtifactID: artifactID, ArtifactPath: artifactPath, Workers: workers, DurationSeconds: item.DurationSeconds, Chrome: chrome, AudioPath: in.AudioPath, WebCopy: pack.Delivery.Web, WebWidth: pack.Delivery.WebWidth, WebCRF: pack.Delivery.WebCRF, MaxMasterMB: pack.Delivery.MaxMasterMB}
	specPath := filepath.Join(jobRoot, jobID+".worker.json")
	if err := writeJSON(specPath, spec); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	logFile, err := os.OpenFile(filepath.Join(jobRoot, jobID+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	command := exec.Command(executable, "--render-job", specPath)
	command.Dir = root
	command.Stdout = logFile
	command.Stderr = logFile
	if err := command.Start(); err != nil {
		_ = logFile.Close()
		return nil, himindjsonrpc.InternalError("无法启动 Browser-Code 渲染任务: " + err.Error())
	}
	_ = logFile.Close()
	jobItem.Status = "preparing"
	jobItem.Progress = 5
	jobItem.WorkerPID = command.Process.Pid
	jobItem.Message = "正在启动 Browser-Code 渲染器"
	jobItem.UpdatedAt = now()
	_ = writeJSON(jobPath, jobItem)
	_ = command.Process.Release()
	return map[string]any{"job": jobItem, "adapter": "browser-code", "next": "调用 render.status 查询后台任务；完成后使用 artifact.export 或 artifact.open"}, nil
}

func effectiveWorkerCount(requested, maxWorkers int) int {
	workers := maxInt(1, requested)
	if maxWorkers > 0 && workers > maxWorkers {
		workers = maxWorkers
	}
	return workers
}

func webCopyScaleFilter(longEdge int) string {
	if longEdge <= 0 {
		longEdge = 1280
	}
	return fmt.Sprintf("scale='if(gte(iw,ih),min(%d,iw),-2)':'if(gte(iw,ih),-2,min(%d,ih))'", longEdge, longEdge)
}

func runCodeWorker(specPath string) error {
	var spec codeWorkerSpec
	if err := readJSON(specPath, &spec); err != nil {
		return err
	}
	root, err := workspace(spec.WorkspaceRoot)
	if err != nil {
		return err
	}
	for _, candidate := range []string{spec.JobPath, spec.ArtifactPath, specPath} {
		if _, err := workspaceFile(root, candidate); err != nil {
			return err
		}
	}
	var jobItem job
	if err := readJSON(spec.JobPath, &jobItem); err != nil {
		return err
	}
	update := func(status string, progress int, message string) {
		jobItem.Status = status
		jobItem.Progress = progress
		jobItem.Message = message
		jobItem.UpdatedAt = now()
		_ = writeJSON(spec.JobPath, jobItem)
	}
	fail := func(err error) error {
		update("failed", 100, conciseProcessOutput([]byte(err.Error()), err))
		return err
	}
	sourceRoot, err := workspaceFile(root, spec.SourceDir)
	if err != nil {
		return fail(err)
	}
	entryPath, err := workspaceFile(sourceRoot, spec.Entry)
	if err != nil {
		return fail(err)
	}
	update("rendering", 10, "正在渲染帧")
	frameArgs := []string{entryPath, "--frames", fmt.Sprintf("--workers=%d", maxInt(1, spec.Workers)), "--chrome=" + spec.Chrome}
	if spec.DurationSeconds > 0 {
		frameArgs = append(frameArgs, fmt.Sprintf("--duration=%g", spec.DurationSeconds))
	}
	framesOutput, frameErr := runCodeNode(sourceRoot, frameArgs, 30*time.Minute)
	if frameErr != nil {
		return fail(fmt.Errorf("frames: %s", conciseProcessOutput(framesOutput, frameErr)))
	}
	update("encoding", 80, "正在编码 MP4")
	encodeArgs := []string{entryPath, "--encode", "--chrome=" + spec.Chrome, "--out=" + spec.ArtifactPath}
	if spec.DurationSeconds > 0 {
		encodeArgs = append(encodeArgs, fmt.Sprintf("--duration=%g", spec.DurationSeconds))
	}
	if strings.TrimSpace(spec.AudioPath) != "" {
		encodeArgs = append(encodeArgs, "--audio="+spec.AudioPath)
	}
	encodeOutput, encodeErr := runCodeNode(sourceRoot, encodeArgs, 30*time.Minute)
	if encodeErr != nil {
		return fail(fmt.Errorf("encode: %s", conciseProcessOutput(encodeOutput, encodeErr)))
	}
	info, statErr := os.Stat(spec.ArtifactPath)
	if statErr != nil || info.Size() == 0 {
		return fail(fmt.Errorf("Browser-Code 未生成有效 MP4"))
	}
	artifactItem, err := makeArtifact(spec.ArtifactID, "video/mp4", spec.ArtifactPath, spec.ProjectID, "browser-code", true)
	if err != nil {
		return fail(err)
	}
	message := "Browser-Code 视频已生成"
	if spec.MaxMasterMB > 0 && info.Size() > int64(spec.MaxMasterMB)*1024*1024 {
		message += fmt.Sprintf("；母版 %.1f MB 超过建议上限 %d MB", float64(info.Size())/1048576, spec.MaxMasterMB)
	}
	if spec.WebCopy {
		ffmpegPath, ffmpegErr := exec.LookPath("ffmpeg")
		if ffmpegErr != nil {
			message += "；Web copy 跳过：缺少 ffmpeg"
		} else {
			webID := newID("video-web")
			webPath := filepath.Join(filepath.Dir(spec.ArtifactPath), webID+"-web.mp4")
			width := spec.WebWidth
			if width <= 0 {
				width = 1280
			}
			crf := spec.WebCRF
			if crf <= 0 {
				crf = 23
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			command := exec.CommandContext(ctx, ffmpegPath, "-y", "-hide_banner", "-loglevel", "error", "-i", spec.ArtifactPath, "-vf", webCopyScaleFilter(width), "-c:v", "libx264", "-preset", "medium", "-crf", strconv.Itoa(crf), "-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart", webPath)
			output, runErr := command.CombinedOutput()
			cancel()
			if runErr != nil {
				message += "；Web copy 失败：" + conciseProcessOutput(output, runErr)
			} else if webInfo, statErr := os.Stat(webPath); statErr == nil && webInfo.Size() > 0 {
				if webArtifact, makeErr := makeArtifact(webID, "video/web", webPath, spec.ProjectID, "browser-code-web", true); makeErr == nil {
					message += fmt.Sprintf("；Web copy 长边 %dpx %.1f MB", width, float64(webArtifact.Size)/1048576)
				}
			}
		}
	}
	update("quality", 95, "正在执行视觉 QA")
	qualityValue, qualityErr := qualityVisual(requestInput{WorkspaceRoot: spec.WorkspaceRoot, ProjectID: spec.ProjectID, VideoPath: spec.ArtifactPath})
	if qualityErr != nil {
		jobItem.QualityStatus = "error"
		jobItem.QualityReport = map[string]any{"ok": false, "error": qualityErr.Message}
		message += "；视觉 QA 失败"
	} else if report, ok := qualityValue.(visualReport); ok {
		applyVisualQuality(&jobItem, report)
		switch jobItem.QualityStatus {
		case "passed":
			message += "；视觉 QA 通过"
		case "warning":
			message += fmt.Sprintf("；视觉 QA 通过（%d warning）", len(report.Warnings))
		default:
			message += "；视觉 QA 失败"
		}
	}
	artifactItem.QualityStatus = jobItem.QualityStatus
	artifactItem.QualityReport = jobItem.QualityReport
	_ = writeJSON(spec.ArtifactPath+".json", artifactItem)
	jobItem.Status = "completed"
	jobItem.Progress = 100
	jobItem.ArtifactID = artifactItem.ID
	jobItem.ArtifactPath = artifactItem.Path
	jobItem.Message = message
	jobItem.UpdatedAt = now()
	return writeJSON(spec.JobPath, jobItem)
}

func filterArtifacts(items []artifact, prefix string) []artifact {
	out := make([]artifact, 0, len(items))
	for _, item := range items {
		if strings.HasPrefix(item.Kind, prefix) && item.Ready {
			out = append(out, item)
		}
	}
	return out
}

func readTextExcerpt(path string, limit int) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text := string(raw)
	runes := []rune(text)
	if len(runes) > limit {
		return string(runes[:limit]) + "\n…"
	}
	return text
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isImageFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		return true
	default:
		return false
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func inValueOr(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

// runRenderJob dispatches the fixed worker entrypoint by spec kind. Remotion
// specs historically omitted kind, so empty means remotion.
func runRenderJob(specPath string) error {
	var head struct {
		Kind string `json:"kind"`
	}
	_ = readJSON(specPath, &head)
	if head.Kind == "browser-code" {
		return runCodeWorker(specPath)
	}
	return runRemotionWorker(specPath)
}

// previewDeterminism renders the same still times twice and compares the
// resulting files. It catches unseeded randomness and time/state leakage
// before a Style Pack is promoted or a final MP4 is trusted.
func previewDeterminism(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if projectMode(item) != "code" {
		return nil, himindjsonrpc.InvalidParams("preview.determinism is only for code projects")
	}
	sourceRoot, err := sourceRootForProject(root, item)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if _, err := exec.LookPath("node"); err != nil {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("renderer_runtime_missing", "runtime", "缺少 Node.js", "安装 Node.js 后重试")}, "retryable": true}, nil
	}
	chrome := findRemotionBrowser()
	if chrome == "" {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("chrome_missing", "runtime", "找不到 Chrome/Edge", "安装 Chrome 或设置 CHROME_PATH")}, "retryable": true}, nil
	}
	if err := ensureCodeDependencies(sourceRoot); err != nil {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("code_dependencies_missing", "runtime", err.Error(), "在代码项目目录运行 npm install")}, "retryable": true}, nil
	}
	times := firstNonEmpty(in.Times, "1,5,9")
	jobRoot := filepath.Join(root, metadataDir, jobDir)
	runID := newID("determinism")
	firstDir := filepath.Join(jobRoot, runID+"-a")
	secondDir := filepath.Join(jobRoot, runID+"-b")
	entryPath := filepath.Join(sourceRoot, filepath.FromSlash(item.Entry))
	started := time.Now()
	firstOut, firstErr := runCodeNode(sourceRoot, []string{entryPath, "--stills=" + times, "--chrome=" + chrome, "--out=" + firstDir}, 5*time.Minute)
	if firstErr != nil {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("code_determinism_render_failed", "determinism", conciseProcessOutput(firstOut, firstErr), "检查 render.mjs --stills")}, "retryable": true}, nil
	}
	secondOut, secondErr := runCodeNode(sourceRoot, []string{entryPath, "--stills=" + times, "--chrome=" + chrome, "--out=" + secondDir}, 5*time.Minute)
	if secondErr != nil {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("code_determinism_render_failed", "determinism", conciseProcessOutput(secondOut, secondErr), "检查 render.mjs --stills")}, "retryable": true}, nil
	}
	files := []map[string]any{}
	valid := true
	firstNames := imageNames(firstDir)
	secondNames := imageNames(secondDir)
	if len(firstNames) == 0 || len(firstNames) != len(secondNames) {
		valid = false
		files = append(files, map[string]any{"first": len(firstNames), "second": len(secondNames)})
	} else {
		sort.Strings(firstNames)
		sort.Strings(secondNames)
		for i, name := range firstNames {
			firstHash, hashErr1 := fileSHA256(filepath.Join(firstDir, name))
			secondHash, hashErr2 := fileSHA256(filepath.Join(secondDir, secondNames[i]))
			ok := hashErr1 == nil && hashErr2 == nil && firstHash == secondHash
			if !ok {
				valid = false
			}
			files = append(files, map[string]any{"name": name, "first": firstHash, "second": secondHash, "match": ok})
		}
	}
	_ = os.RemoveAll(firstDir)
	_ = os.RemoveAll(secondDir)
	if hash, hashErr := hashCodeSource(sourceRoot); hashErr == nil {
		if valid {
			item.DeterminismStatus = "passed"
		} else {
			item.DeterminismStatus = "failed"
		}
		item.DeterminismHash = hash
		item.UpdatedAt = now()
		_ = writeJSON(filepath.Join(root, metadataDir, projectDir, item.ID, "project.json"), item)
	}
	return map[string]any{
		"valid": valid, "project_id": item.ID, "times": times,
		"files": files, "duration_ms": time.Since(started).Milliseconds(),
		"next": []string{"determinism.valid=true 后再执行最终渲染", "不一致时检查随机种子、时间、跨帧状态和外部资源"},
	}, nil
}

func imageNames(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && isImageFile(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	return names
}

// projectDataWrite writes the normalized data/brief inputs consumed by a
// data-driven Style Pack. Paths are fixed inside source_dir/input so an AI
// cannot turn this into an arbitrary file-write primitive.
func projectDataWrite(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if projectMode(item) != "code" {
		return nil, himindjsonrpc.InvalidParams("project.data.write is only for code projects")
	}
	if len(in.Data) == 0 {
		return nil, himindjsonrpc.InvalidParams("data is required")
	}
	sourceRoot, err := sourceRootForProject(root, item)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	inputRoot := filepath.Join(sourceRoot, "input")
	if err := os.MkdirAll(inputRoot, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	dataPath := filepath.Join(inputRoot, "data.json")
	if err := writeJSON(dataPath, in.Data); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	briefPath := filepath.Join(inputRoot, "brief.json")
	if len(in.Brief) > 0 {
		if err := writeJSON(briefPath, in.Brief); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
	}
	hash, err := hashCodeSource(sourceRoot)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	item.SourceHash = hash
	item.PreviewStatus = "stale"
	item.DeterminismStatus = "stale"
	item.DeterminismHash = ""
	item.Revision++
	item.UpdatedAt = now()
	if err := writeJSON(filepath.Join(root, metadataDir, projectDir, item.ID, "project.json"), item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{
		"project": item, "data_path": dataPath, "brief_path": briefPath,
		"source_hash": hash, "preview_status": item.PreviewStatus,
		"next": []string{"调用 preview.review 重新生成审查图", "调用 preview.determinism", "通过 quality.check 后 render.start"},
	}, nil
}
