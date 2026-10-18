package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// projectUpgradeStyle upgrades the renderer contract of a code project from the
// matching Style Pack. It intentionally only touches the renderer entry file:
// the project's reel code, data, storyboard and delivery settings are preserved.
func projectUpgradeStyle(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if projectMode(item) != "code" {
		return nil, himindjsonrpc.InvalidParams("project.upgrade_style is only for code projects")
	}
	sourceRoot, err := sourceRootForProject(root, item)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	targetVersion := firstNonEmpty(in.TargetVersion, item.StyleVersion)
	pack, ok := findAssetPack(root, item.StylePack, targetVersion)
	if !ok && targetVersion != "" {
		pack, ok = findAssetPack(root, item.StylePack, "")
		targetVersion = ""
	}
	if !ok {
		return nil, himindjsonrpc.InvalidParams("style pack not found: " + item.StylePack + " " + targetVersion)
	}
	if targetVersion == "" {
		targetVersion = pack.Version
	}
	resource, ok := findAssetResource(pack, "template")
	if !ok {
		return nil, himindjsonrpc.InvalidParams("style pack has no template resource: " + pack.ID)
	}
	templateRoot, err := packRelativeFile(pack, resource.Path)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	templateEntry := firstNonEmpty(resource.Entry, item.Entry, "render.mjs")
	templateEntryPath := filepath.Join(templateRoot, filepath.FromSlash(templateEntry))
	relative, relErr := filepath.Rel(templateRoot, templateEntryPath)
	if relErr != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, himindjsonrpc.InvalidParams("style pack entry escapes template root")
	}
	entryPath, err := workspaceFile(sourceRoot, item.Entry)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("project entry: " + err.Error())
	}
	oldRaw, err := os.ReadFile(entryPath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("read project entry: " + err.Error())
	}
	newRaw, err := os.ReadFile(templateEntryPath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("read style pack entry: " + err.Error())
	}
	oldSHA, _ := fileSHA256(entryPath)
	newSHA, _ := fileSHA256(templateEntryPath)
	changed := !bytes.Equal(oldRaw, newRaw)
	plan := map[string]any{
		"project_id":           item.ID,
		"style_pack":           pack.ID,
		"target_version":       targetVersion,
		"entry":                item.Entry,
		"changed":              []string{},
		"preserved":            []string{"reel/", "input/", "STORYBOARD.md", "package.json", "package-lock.json"},
		"old_sha256":           oldSHA,
		"new_sha256":           newSHA,
		"requires_review":      changed,
		"requires_determinism": changed,
		"requires_render":      changed,
		"next":                 []string{"preview.review", "preview.determinism", "render.start"},
	}
	if !changed {
		plan["state"] = "current"
		plan["project"] = item
		return plan, nil
	}
	if in.DryRun {
		plan["state"] = "planned"
		plan["changed"] = []string{item.Entry}
		plan["project"] = item
		return plan, nil
	}
	backupDir := filepath.Join(root, metadataDir, projectDir, item.ID, "renderer-backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	backupPath := filepath.Join(backupDir, filepath.Base(item.Entry)+"."+time.Now().UTC().Format("20060102-150405")+".bak")
	if err := os.WriteFile(backupPath, oldRaw, 0644); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if err := os.WriteFile(entryPath, newRaw, 0644); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	hash, err := hashCodeSource(sourceRoot)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	item.SourceHash = hash
	item.StyleVersion = targetVersion
	item.PreviewStatus = "stale"
	item.DeterminismStatus = "stale"
	item.DeterminismHash = ""
	item.Revision++
	item.UpdatedAt = now()
	if err := writeJSON(filepath.Join(root, metadataDir, projectDir, item.ID, "project.json"), item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	plan["state"] = "upgraded"
	plan["changed"] = []string{item.Entry}
	plan["backup"] = backupPath
	plan["project"] = item
	plan["message"] = fmt.Sprintf("已将 %s 升级到 %s %s；项目代码与数据保持不变", item.Name, pack.ID, targetVersion)
	return plan, nil
}
