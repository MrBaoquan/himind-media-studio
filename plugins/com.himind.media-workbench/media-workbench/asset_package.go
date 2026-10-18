package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// asset_package.go — Asset Pack 的可分发制品。
//
// .hmpack 是确定性的 ZIP：
//
//	package.json   资产包文件清单与 SHA-256
//	asset-pack.json 资产包 Manifest
//	...            资源文件
//
// 安装目录：
//
//	%LOCALAPPDATA%\HiMindAgent\asset-packs\<pack_id>\<version>\
//
// Asset Pack 仍然只是资产容器；安装不会执行包内代码。

const (
	assetPackageSchema       = "himind.asset-package.v1"
	assetPackageManifestName = "package.json"
	assetPackManifestName    = "asset-pack.json"
)

var (
	assetPackIDPattern      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,63}$`)
	assetPackVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
)

type assetPackageFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type assetPackageManifest struct {
	Schema  string             `json:"schema"`
	ID      string             `json:"id"`
	Version string             `json:"version"`
	Files   []assetPackageFile `json:"files"`
}

func validateAssetPack(pack assetPack) map[string]any {
	checks := make([]map[string]any, 0, 12)
	issues := make([]map[string]any, 0)
	check := func(id string, ok bool, detail, code, message, remediation string) {
		checks = append(checks, map[string]any{"id": id, "passed": ok, "detail": detail})
		if !ok {
			issues = append(issues, issue(code, "asset", message, remediation))
		}
	}
	check("schema", pack.Schema == "himind.asset-pack.v1", pack.Schema, "asset_pack_schema", "schema 必须为 himind.asset-pack.v1", "修复 asset-pack.json")
	check("license", strings.TrimSpace(pack.License) != "", pack.License, "asset_pack_license", "资产包必须声明 license", "补充 license")
	if pack.Provenance != nil {
		check("provenance_source", asString(pack.Provenance["source"]) != "", asString(pack.Provenance["source"]), "asset_pack_provenance", "provenance 缺少 source", "补充 provenance.source")
		check("provenance_license", asString(pack.Provenance["license"]) != "", asString(pack.Provenance["license"]), "asset_pack_provenance_license", "provenance 缺少 license", "补充 provenance.license")
	}
	templateFound := false
	previewFound := false
	for _, resource := range pack.Resources {
		path, pathErr := packRelativeFile(pack, resource.Path)
		exists := pathErr == nil
		if exists {
			_, statErr := os.Stat(path)
			exists = statErr == nil
		}
		check("resource:"+resource.ID, exists, resource.Path, "asset_pack_resource_missing", "资源不存在: "+resource.Path, "修复 resources[].path")
		if strings.EqualFold(resource.Type, "template") {
			templateFound = true
		}
		if strings.EqualFold(resource.Type, "image") && (resource.Role == "preview" || resource.ID == "preview") {
			previewFound = true
		}
	}
	check("template", templateFound, "template", "asset_pack_template", "资产包必须包含 template 资源", "添加 template 资源")
	check("preview", previewFound, "preview", "asset_pack_preview", "资产包必须包含 preview 图片", "添加 preview/sheet.jpg")
	for _, entry := range pack.Entrypoints {
		found := false
		for _, resource := range pack.Resources {
			if resource.ID == entry.Resource {
				found = true
				break
			}
		}
		check("entrypoint:"+entry.ID, found && strings.TrimSpace(entry.Runtime) != "", entry.Resource, "asset_pack_entrypoint", "entrypoint 引用了不存在的资源或 runtime", "修复 entrypoints[]")
	}
	return map[string]any{"valid": len(issues) == 0, "pack": pack, "checks": checks, "issues": issues}
}

func assetPackPackage(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	pack, ok := findAssetPack(root, strings.TrimSpace(in.PackID), strings.TrimSpace(in.Version))
	if !ok {
		return nil, himindjsonrpc.InvalidParams("asset pack not found")
	}
	validation := validateAssetPack(pack)
	if valid, _ := validation["valid"].(bool); !valid {
		return validation, nil
	}
	output := strings.TrimSpace(in.Output)
	if output == "" {
		output = filepath.Join("dist", "asset-packages", pack.ID+"-"+pack.Version+".hmpack")
	}
	if !strings.EqualFold(filepath.Ext(output), ".hmpack") {
		return nil, himindjsonrpc.InvalidParams("output must end with .hmpack")
	}
	target, err := workspaceFile(root, output)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("output: " + err.Error())
	}
	size, packageHash, manifest, err := writeAssetPackage(pack, target)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{
		"valid":          true,
		"state":          "packaged",
		"pack_id":        pack.ID,
		"version":        pack.Version,
		"package_path":   target,
		"package_sha256": packageHash,
		"size":           size,
		"files":          len(manifest.Files),
		"manifest":       manifest,
	}, nil
}

func writeAssetPackage(pack assetPack, target string) (int64, string, assetPackageManifest, error) {
	files, err := collectAssetPackFiles(pack.Path)
	if err != nil {
		return 0, "", assetPackageManifest{}, err
	}
	manifest := assetPackageManifest{
		Schema:  assetPackageSchema,
		ID:      pack.ID,
		Version: pack.Version,
		Files:   files,
	}
	rawManifest, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return 0, "", manifest, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return 0, "", manifest, err
	}
	if info, statErr := os.Stat(target); statErr == nil && info.IsDir() {
		return 0, "", manifest, fmt.Errorf("package output is a directory: %s", target)
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".hmpack-*")
	if err != nil {
		return 0, "", manifest, err
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()
	archive := zip.NewWriter(temp)
	fixedTime := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	writeEntry := func(name string, data []byte) error {
		header := &zip.FileHeader{Name: filepath.ToSlash(name), Method: zip.Deflate, Modified: fixedTime}
		header.SetMode(0644)
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		return err
	}
	if err := writeEntry(assetPackageManifestName, rawManifest); err != nil {
		return 0, "", manifest, err
	}
	for _, item := range files {
		data, readErr := os.ReadFile(filepath.Join(pack.Path, filepath.FromSlash(item.Path)))
		if readErr != nil {
			return 0, "", manifest, readErr
		}
		if err := writeEntry(item.Path, data); err != nil {
			return 0, "", manifest, err
		}
	}
	if err := archive.Close(); err != nil {
		return 0, "", manifest, err
	}
	if err := temp.Close(); err != nil {
		return 0, "", manifest, err
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return 0, "", manifest, fmt.Errorf("replace package output: %w", err)
	}
	if err := os.Rename(tempPath, target); err != nil {
		return 0, "", manifest, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return 0, "", manifest, err
	}
	hash, err := fileSHA256(target)
	if err != nil {
		return 0, "", manifest, err
	}
	return info.Size(), hash, manifest, nil
}

func collectAssetPackFiles(root string) ([]assetPackageFile, error) {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("asset pack root is not a directory: %s", root)
	}
	skipDirs := map[string]bool{"node_modules": true, ".git": true, "out": true, ".cache": true, "dist": true, "__pycache__": true}
	skipFiles := map[string]bool{".DS_Store": true, "Thumbs.db": true}
	files := make([]assetPackageFile, 0, 32)
	hasManifest := false
	err = filepath.Walk(root, func(path string, item os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if skipDirs[item.Name()] {
			if item.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if item.IsDir() {
			return nil
		}
		if !item.Mode().IsRegular() {
			return fmt.Errorf("asset pack contains unsupported file type: %s", rel)
		}
		if skipFiles[item.Name()] {
			return nil
		}
		hash, err := fileSHA256(path)
		if err != nil {
			return err
		}
		clean := filepath.ToSlash(filepath.Clean(rel))
		if clean == assetPackManifestName {
			hasManifest = true
		}
		files = append(files, assetPackageFile{Path: clean, Size: item.Size(), SHA256: hash})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !hasManifest {
		return nil, fmt.Errorf("asset pack is missing %s", assetPackManifestName)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func assetPackInstall(in requestInput) (any, *himindjsonrpc.Error) {
	packagePath := strings.TrimSpace(in.PackagePath)
	if packagePath == "" {
		return nil, himindjsonrpc.InvalidParams("package_path is required")
	}
	absolute, err := filepath.Abs(packagePath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !strings.EqualFold(filepath.Ext(absolute), ".hmpack") {
		return nil, himindjsonrpc.InvalidParams("package_path must end with .hmpack")
	}
	result, err := installAssetPackage(absolute, in.Force)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	return result, nil
}

func installAssetPackage(packagePath string, force bool) (map[string]any, error) {
	info, err := os.Stat(packagePath)
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("package not found: %s", packagePath)
	}
	installRoot, err := assetPackInstallRoot()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(installRoot, 0755); err != nil {
		return nil, err
	}
	archive, err := zip.OpenReader(packagePath)
	if err != nil {
		return nil, fmt.Errorf("open package: %w", err)
	}
	defer archive.Close()
	manifest, err := readAssetPackageManifest(archive)
	if err != nil {
		return nil, err
	}
	tempDir, err := os.MkdirTemp(installRoot, ".install-")
	if err != nil {
		return nil, err
	}
	keepTemp := false
	defer func() {
		if !keepTemp {
			_ = os.RemoveAll(tempDir)
		}
	}()
	entries := map[string]*zip.File{}
	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		clean, pathErr := safeAssetPackagePath(entry.Name)
		if pathErr != nil {
			return nil, pathErr
		}
		if _, exists := entries[clean]; exists {
			return nil, fmt.Errorf("duplicate package entry: %s", clean)
		}
		entries[clean] = entry
	}
	for _, file := range manifest.Files {
		clean, pathErr := safeAssetPackagePath(file.Path)
		if pathErr != nil {
			return nil, pathErr
		}
		entry, exists := entries[clean]
		if !exists {
			return nil, fmt.Errorf("package entry missing: %s", clean)
		}
		if entry.FileInfo().IsDir() {
			return nil, fmt.Errorf("package entry is a directory: %s", clean)
		}
		hash, size, copyErr := extractAssetPackageFile(entry, filepath.Join(tempDir, filepath.FromSlash(clean)))
		if copyErr != nil {
			return nil, copyErr
		}
		if size != file.Size || !strings.EqualFold(hash, file.SHA256) {
			return nil, fmt.Errorf("integrity mismatch: %s", clean)
		}
	}
	manifestPaths := map[string]bool{assetPackageManifestName: true}
	for _, file := range manifest.Files {
		clean, pathErr := safeAssetPackagePath(file.Path)
		if pathErr != nil {
			return nil, pathErr
		}
		manifestPaths[clean] = true
	}
	for name := range entries {
		if !manifestPaths[name] {
			return nil, fmt.Errorf("package contains undeclared entry: %s", name)
		}
	}
	pack, err := loadAssetPackAt(tempDir, "package")
	if err != nil {
		return nil, fmt.Errorf("read installed asset-pack.json: %w", err)
	}
	if pack.ID != manifest.ID || pack.Version != manifest.Version {
		return nil, fmt.Errorf("asset-pack.json identity does not match package manifest")
	}
	validation := validateAssetPack(pack)
	if valid, _ := validation["valid"].(bool); !valid {
		raw, _ := json.Marshal(validation["issues"])
		return nil, fmt.Errorf("asset pack validation failed: %s", string(raw))
	}
	target := filepath.Join(installRoot, manifest.ID, manifest.Version)
	if info, statErr := os.Stat(target); statErr == nil {
		if !force {
			return nil, fmt.Errorf("asset pack %s@%s is already installed", manifest.ID, manifest.Version)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("install target is not a directory: %s", target)
		}
		if err := os.RemoveAll(target); err != nil {
			return nil, fmt.Errorf("replace installed asset pack: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return nil, err
	}
	if err := os.Rename(tempDir, target); err != nil {
		return nil, fmt.Errorf("move installed asset pack: %w", err)
	}
	keepTemp = true
	packageHash, err := fileSHA256(packagePath)
	if err != nil {
		return nil, err
	}
	metadata := map[string]any{
		"schema":         "himind.asset-install.v1",
		"pack_id":        manifest.ID,
		"version":        manifest.Version,
		"package_path":   packagePath,
		"package_sha256": packageHash,
		"installed_at":   now(),
		"files":          len(manifest.Files),
	}
	if err := writeJSON(filepath.Join(target, "install.json"), metadata); err != nil {
		return nil, err
	}
	return map[string]any{
		"state":          "installed",
		"pack_id":        manifest.ID,
		"version":        manifest.Version,
		"installed_path": target,
		"package_sha256": packageHash,
		"files":          len(manifest.Files),
		"source":         packagePath,
	}, nil
}

func readAssetPackageManifest(archive *zip.ReadCloser) (assetPackageManifest, error) {
	var manifest assetPackageManifest
	var raw []byte
	for _, entry := range archive.File {
		clean, err := safeAssetPackagePath(entry.Name)
		if err != nil {
			return manifest, err
		}
		if clean == assetPackageManifestName {
			reader, err := entry.Open()
			if err != nil {
				return manifest, err
			}
			raw, err = io.ReadAll(io.LimitReader(reader, 4*1024*1024))
			_ = reader.Close()
			if err != nil {
				return manifest, err
			}
		}
	}
	if len(raw) == 0 {
		return manifest, fmt.Errorf("package is missing %s", assetPackageManifestName)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return manifest, fmt.Errorf("parse package manifest: %w", err)
	}
	if manifest.Schema != assetPackageSchema {
		return manifest, fmt.Errorf("package schema must be %s", assetPackageSchema)
	}
	if !assetPackIDPattern.MatchString(manifest.ID) {
		return manifest, fmt.Errorf("invalid package id: %s", manifest.ID)
	}
	if !assetPackVersionPattern.MatchString(manifest.Version) {
		return manifest, fmt.Errorf("invalid package version: %s", manifest.Version)
	}
	if len(manifest.Files) == 0 {
		return manifest, fmt.Errorf("package manifest has no files")
	}
	seen := map[string]bool{}
	for _, file := range manifest.Files {
		clean, err := safeAssetPackagePath(file.Path)
		if err != nil {
			return manifest, err
		}
		if seen[clean] {
			return manifest, fmt.Errorf("duplicate manifest entry: %s", clean)
		}
		seen[clean] = true
		if file.Size < 0 || strings.TrimSpace(file.SHA256) == "" {
			return manifest, fmt.Errorf("invalid manifest entry: %s", clean)
		}
	}
	if !seen[assetPackManifestName] {
		return manifest, fmt.Errorf("package manifest must include %s", assetPackManifestName)
	}
	return manifest, nil
}

func extractAssetPackageFile(entry *zip.File, target string) (string, int64, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", 0, err
	}
	reader, err := entry.Open()
	if err != nil {
		return "", 0, err
	}
	defer reader.Close()
	output, err := os.Create(target)
	if err != nil {
		return "", 0, err
	}
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(output, hash), reader)
	closeErr := output.Close()
	if copyErr != nil {
		return "", 0, copyErr
	}
	if closeErr != nil {
		return "", 0, closeErr
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

func installedAssetPackPaths() map[string]string {
	paths := map[string]string{}
	root, err := assetPackInstallRoot()
	if err != nil {
		return paths
	}
	packs, err := os.ReadDir(root)
	if err != nil {
		return paths
	}
	for _, packEntry := range packs {
		if !packEntry.IsDir() {
			continue
		}
		packDir := filepath.Join(root, packEntry.Name())
		versions, err := os.ReadDir(packDir)
		if err != nil {
			continue
		}
		for _, versionEntry := range versions {
			if !versionEntry.IsDir() {
				continue
			}
			dir := filepath.Join(packDir, versionEntry.Name())
			pack, loadErr := loadAssetPackAt(dir, "installed")
			if loadErr != nil {
				continue
			}
			paths[pack.ID+"@"+pack.Version] = dir
		}
	}
	return paths
}

func assetPackInstallRoot() (string, error) {
	if root := strings.TrimSpace(os.Getenv("HIMIND_ASSET_PACK_HOME")); root != "" {
		return filepath.Abs(root)
	}
	local := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
	if local == "" {
		return "", fmt.Errorf("LOCALAPPDATA is not set; cannot locate asset pack install root")
	}
	return filepath.Join(local, "HiMindAgent", "asset-packs"), nil
}

func assetPackUninstall(in requestInput) (any, *himindjsonrpc.Error) {
	id := strings.TrimSpace(in.PackID)
	version := strings.TrimSpace(in.Version)
	if !assetPackIDPattern.MatchString(id) {
		return nil, himindjsonrpc.InvalidParams("invalid pack_id")
	}
	if !assetPackVersionPattern.MatchString(version) {
		return nil, himindjsonrpc.InvalidParams("invalid version")
	}
	root, err := assetPackInstallRoot()
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	target := filepath.Join(root, id, version)
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, himindjsonrpc.InvalidParams("install target escapes asset pack root")
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		return nil, himindjsonrpc.InvalidParams("asset pack version is not installed")
	}
	if err := os.RemoveAll(target); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	_ = os.Remove(filepath.Join(root, id))
	return map[string]any{"state": "uninstalled", "pack_id": id, "version": version, "path": target}, nil
}

func safeAssetPackagePath(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("empty package path")
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") {
		return "", fmt.Errorf("unsafe package path: %s", name)
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe package path: %s", name)
	}
	return filepath.ToSlash(clean), nil
}
