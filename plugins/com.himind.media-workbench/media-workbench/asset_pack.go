package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// asset_pack.go — 通用资产包注册表。
//
// 资产包不区分 video/audio domain：一个包可以同时包含模板、字体、音频、
// 数据、提示词和可选执行入口。创作台只按 resource.type/role 消费自己需要的部分。

type assetResource struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Format      string `json:"format,omitempty"`
	Mime        string `json:"mime,omitempty"`
	Role        string `json:"role,omitempty"`
	Runtime     string `json:"runtime,omitempty"`
	Entry       string `json:"entry,omitempty"`
	Description string `json:"description,omitempty"`
}

type assetEntrypoint struct {
	ID       string   `json:"id"`
	Resource string   `json:"resource"`
	Runtime  string   `json:"runtime"`
	Entry    string   `json:"entry"`
	Inputs   []string `json:"inputs,omitempty"`
	Outputs  []string `json:"outputs,omitempty"`
}

type assetPack struct {
	Schema      string            `json:"schema"`
	ID          string            `json:"id"`
	Version     string            `json:"version"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	License     string            `json:"license,omitempty"`
	Attribution string            `json:"attribution,omitempty"`
	Resources   []assetResource   `json:"resources"`
	Entrypoints []assetEntrypoint `json:"entrypoints,omitempty"`
	Permissions map[string]any    `json:"permissions,omitempty"`
	Provenance  map[string]any    `json:"provenance,omitempty"`
	Quality     map[string]any    `json:"quality,omitempty"`
	Consumers   []string          `json:"consumers,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Installed   bool              `json:"installed,omitempty"`
	InstallPath string            `json:"install_path,omitempty"`
	// Path/Source are response metadata, never persisted.
	Path   string `json:"path,omitempty"`
	Source string `json:"source,omitempty"`
}

type assetPackCandidate struct {
	dir    string
	source string
}

func assetPackRoots(workspaceRoot string) []assetPackCandidate {
	candidates := []assetPackCandidate{}
	appendRoot := func(root, source string) {
		if root == "" {
			return
		}
		for _, name := range []string{"asset-packs", "style-packs"} {
			dir := filepath.Join(root, name)
			if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
				candidates = append(candidates, assetPackCandidate{dir: dir, source: source})
			}
		}
	}
	appendRoot(workspaceRoot, "workspace")
	repoRoot := filepath.Clean(filepath.Join(pluginRoot(), "..", "..", ".."))
	appendRoot(repoRoot, "repository")
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		appendRoot(filepath.Join(local, "HiMindAgent"), "installed")
	}
	if custom := strings.TrimSpace(os.Getenv("HIMIND_ASSET_PACK_HOME")); custom != "" {
		if stat, err := os.Stat(custom); err == nil && stat.IsDir() {
			candidates = append(candidates, assetPackCandidate{dir: custom, source: "installed"})
		}
	}
	return candidates
}

func loadAssetPackAt(dir, source string) (assetPack, error) {
	var pack assetPack
	manifest := filepath.Join(dir, "asset-pack.json")
	if err := readJSON(manifest, &pack); err != nil {
		// Compatibility fallback: an old style-pack directory without a root
		// asset-pack.json is exposed as a template-only pack.
		var legacy codeStylePack
		legacyPath := filepath.Join(dir, "style-pack.json")
		legacyErr := readJSON(legacyPath, &legacy)
		if legacyErr != nil {
			legacyPath = filepath.Join(dir, "template", "style-pack.json")
			legacyErr = readJSON(legacyPath, &legacy)
		}
		if legacyErr != nil {
			return pack, err
		}
		pack = assetPack{
			Schema: "himind.asset-pack.v1", ID: legacy.ID, Version: legacy.Version, Name: legacy.Name,
			Description: legacy.Description, License: legacy.License, Attribution: legacy.Attribution,
			Resources: []assetResource{{
				ID: "template", Type: "template", Path: "template", Role: "render-template",
				Runtime: legacy.Renderer, Entry: legacy.Entry,
			}},
			Tags: []string{"video", "style"},
		}
	}
	if pack.Schema == "" {
		pack.Schema = "himind.asset-pack.v1"
	}
	if pack.Version == "" {
		pack.Version = "1.0.0"
	}
	pack.Path = dir
	pack.Source = source
	return pack, nil
}

func loadAssetPacks(workspaceRoot string) ([]assetPack, error) {
	seen := map[string]bool{}
	items := []assetPack{}
	installedPaths := installedAssetPackPaths()
	appendPack := func(pack assetPack) {
		key := pack.ID + "@" + pack.Version
		if path, ok := installedPaths[key]; ok {
			pack.Installed = true
			pack.InstallPath = path
		}
		if pack.ID == "" || seen[key] {
			return
		}
		seen[key] = true
		items = append(items, pack)
	}
	for _, candidate := range assetPackRoots(workspaceRoot) {
		entries, err := os.ReadDir(candidate.dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			base := filepath.Join(candidate.dir, entry.Name())
			if pack, loadErr := loadAssetPackAt(base, candidate.source); loadErr == nil {
				appendPack(pack)
				continue
			}
			// Installed packs use <pack-id>/<version>/ so multiple versions can
			// coexist and an older version can be selected explicitly.
			versions, versionErr := os.ReadDir(base)
			if versionErr != nil {
				continue
			}
			for _, versionEntry := range versions {
				if !versionEntry.IsDir() {
					continue
				}
				pack, loadErr := loadAssetPackAt(filepath.Join(base, versionEntry.Name()), candidate.source)
				if loadErr == nil {
					appendPack(pack)
				}
			}
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func findAssetPack(workspaceRoot, id, version string) (assetPack, bool) {
	packs, err := loadAssetPacks(workspaceRoot)
	if err != nil {
		return assetPack{}, false
	}
	for _, pack := range packs {
		if pack.ID == id && (version == "" || pack.Version == version) {
			return pack, true
		}
	}
	return assetPack{}, false
}

func assetList(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	packs, err := loadAssetPacks(root)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	filtered := make([]assetPack, 0, len(packs))
	for _, pack := range packs {
		if in.TypeFilter != "" && !assetPackHasType(pack, in.TypeFilter) {
			continue
		}
		filtered = append(filtered, pack)
	}
	return map[string]any{"items": filtered, "total": len(filtered), "workspace_root": root}, nil
}

func assetDescribe(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	pack, ok := findAssetPack(root, strings.TrimSpace(in.PackID), strings.TrimSpace(in.Version))
	if !ok {
		return nil, himindjsonrpc.InvalidParams("asset pack not found")
	}
	return map[string]any{"pack": pack, "immutable": true}, nil
}

func assetRead(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	pack, ok := findAssetPack(root, strings.TrimSpace(in.PackID), strings.TrimSpace(in.Version))
	if !ok {
		return nil, himindjsonrpc.InvalidParams("asset pack not found")
	}
	resource, ok := findAssetResource(pack, in.ResourceID)
	if !ok {
		return nil, himindjsonrpc.InvalidParams("asset resource not found")
	}
	path, err := packRelativeFile(pack, resource.Path)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil, himindjsonrpc.InvalidParams("asset resource is not a file")
	}
	if info.Size() > 20*1024*1024 {
		return nil, himindjsonrpc.InvalidParams("asset resource is too large for inline read")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"pack_id": pack.ID, "resource": resource, "size": len(raw), "base64": base64.StdEncoding.EncodeToString(raw)}, nil
}

func assetMaterialize(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	pack, ok := findAssetPack(root, strings.TrimSpace(in.PackID), strings.TrimSpace(in.Version))
	if !ok {
		return nil, himindjsonrpc.InvalidParams("asset pack not found")
	}
	resource, ok := findAssetResource(pack, in.ResourceID)
	if !ok {
		return nil, himindjsonrpc.InvalidParams("asset resource not found")
	}
	source, err := packRelativeFile(pack, resource.Path)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	targetDir := firstNonEmpty(in.TargetDir, "assets/"+pack.ID)
	target, err := workspaceFile(root, targetDir)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if err := copyAssetPath(source, target); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"pack_id": pack.ID, "resource_id": resource.ID, "target": target, "next": []string{"在插件/Domain Adapter 中解析该资源"}}, nil
}

func stylePackScaffold(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	pack, ok := findAssetPack(root, strings.TrimSpace(in.PackID), strings.TrimSpace(in.Version))
	if !ok {
		return nil, himindjsonrpc.InvalidParams("asset pack not found")
	}
	resource, ok := findAssetResource(pack, firstNonEmpty(in.ResourceID, "template"))
	if !ok || !strings.EqualFold(resource.Type, "template") {
		return nil, himindjsonrpc.InvalidParams("asset pack has no template resource")
	}
	source, err := packRelativeFile(pack, resource.Path)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = pack.Name
	}
	targetDir := firstNonEmpty(in.TargetDir, projectIDForName(name))
	target, err := workspaceFile(root, targetDir)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if err := copyAssetPath(source, target); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if !fileExists(filepath.Join(target, "style-pack.json")) {
		legacy := codeStylePack{
			Schema: "himind.video.style-pack.v1", ID: pack.ID, Version: pack.Version, Name: pack.Name,
			Renderer: firstNonEmpty(resource.Runtime, "browser-code"), Entry: firstNonEmpty(resource.Entry, "render.mjs"),
			VideoOut: "out/video.mp4", DurationSeconds: 10,
		}
		legacy.Preview.SheetOut = "out/check/sheet.jpg"
		legacy.Preview.StripOut = "out/check/strip.jpg"
		legacy.Preview.StillsOut = "out/stills"
		if err := writeJSON(filepath.Join(target, "style-pack.json"), legacy); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
	}
	importResult, importErr := projectImport(requestInput{
		WorkspaceRoot: root, Name: name, SourceDir: targetDir,
		Description: in.Description, CreativeBrief: in.CreativeBrief,
	})
	if importErr != nil {
		return nil, importErr
	}
	payload := importResult.(map[string]any)
	payload["pack_id"] = pack.ID
	payload["pack_version"] = pack.Version
	return payload, nil
}

func assetPackHasType(pack assetPack, typeFilter string) bool {
	for _, resource := range pack.Resources {
		if strings.EqualFold(resource.Type, typeFilter) {
			return true
		}
	}
	return false
}

func findAssetResource(pack assetPack, id string) (assetResource, bool) {
	for _, resource := range pack.Resources {
		if resource.ID == id {
			return resource, true
		}
	}
	return assetResource{}, false
}

func packRelativeFile(pack assetPack, rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("empty asset resource path")
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("asset resource escapes pack root")
	}
	root := filepath.Clean(pack.Path)
	target := filepath.Join(root, clean)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("asset resource escapes pack root")
	}
	return target, nil
}

func copyAssetPath(source, target string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyAssetDir(source, target)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.Create(target)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func copyAssetDir(source, target string) error {
	skip := map[string]bool{"node_modules": true, ".git": true, "out": true, ".cache": true, "dist": true}
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(target, 0755)
		}
		if skip[info.Name()] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		destination := filepath.Join(target, rel)
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return err
		}
		return copyAssetPath(path, destination)
	})
}
func assetPackValidate(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	pack, ok := findAssetPack(root, strings.TrimSpace(in.PackID), strings.TrimSpace(in.Version))
	if !ok {
		return nil, himindjsonrpc.InvalidParams("asset pack not found")
	}
	return validateAssetPack(pack), nil
}
