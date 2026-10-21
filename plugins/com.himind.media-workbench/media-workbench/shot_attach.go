package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func shotAttachAsset(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadCreative(root, in.CreativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	shotID := strings.TrimSpace(in.ShotID)
	source := strings.TrimSpace(in.Path)
	if shotID == "" || source == "" {
		return nil, himindjsonrpc.InvalidParams("shot_id and path are required")
	}
	shots := asMapSlice(asMap(item.Storyboard)["shots"])
	selected := -1
	for index, shot := range shots {
		if strings.TrimSpace(asString(shot["id"])) == shotID {
			selected = index
			break
		}
	}
	if selected < 0 {
		return nil, himindjsonrpc.InvalidParams("shot not found: " + shotID)
	}

	sourcePath := source
	if !filepath.IsAbs(sourcePath) {
		sourcePath, err = workspaceFile(root, sourcePath)
		if err != nil {
			return nil, himindjsonrpc.InvalidParams(err.Error())
		}
	}
	info, err := os.Stat(sourcePath)
	if err != nil || info.IsDir() {
		return nil, himindjsonrpc.InvalidParams("asset file not found: " + sourcePath)
	}
	if info.Size() > 512*1024*1024 {
		return nil, himindjsonrpc.InvalidParams("asset exceeds 512MB")
	}
	ext := strings.ToLower(filepath.Ext(sourcePath))
	allowed := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".svg": true, ".mp4": true, ".webm": true, ".mov": true}
	if !allowed[ext] {
		return nil, himindjsonrpc.InvalidParams("unsupported asset extension: " + ext)
	}

	relative, err := filepath.Rel(root, sourcePath)
	if err != nil || strings.HasPrefix(relative, "..") || filepath.IsAbs(relative) {
		// 资产在视频工作区之外（例如 agent media 目录）：复制进创意资产目录，保留工作区可移植性。
		seed := sha256.Sum256([]byte(sourcePath + "|" + shotID + "|" + fmt.Sprint(info.ModTime().UnixNano())))
		name := "real-" + hex.EncodeToString(seed[:4]) + "-" + filepath.Base(sourcePath)
		targetRel := filepath.ToSlash(filepath.Join(".himind-media", "creatives", item.ID, "assets", shotID, name))
		targetAbs, targetErr := workspaceFile(root, targetRel)
		if targetErr != nil {
			return nil, himindjsonrpc.InvalidParams(targetErr.Error())
		}
		if mkErr := os.MkdirAll(filepath.Dir(targetAbs), 0755); mkErr != nil {
			return nil, himindjsonrpc.InternalError(mkErr.Error())
		}
		if copyErr := copyShotAsset(sourcePath, targetAbs); copyErr != nil {
			return nil, himindjsonrpc.InternalError(copyErr.Error())
		}
		sourcePath = targetAbs
		relative = targetRel
	}
	relative = filepath.ToSlash(relative)
	sum := strings.TrimSpace(in.SHA256)
	if sum == "" {
		if computed, hashErr := hashShotAsset(sourcePath); hashErr == nil {
			sum = computed
		}
	}
	shot := shots[selected]
	shot["provider"] = firstNonEmpty(strings.TrimSpace(in.Provider), "ai.image")
	shot["status"] = firstNonEmpty(strings.TrimSpace(in.Status), "generated")
	shot["mock"] = in.Mock
	shot["produced_at"] = time.Now().UTC().Format(time.RFC3339)
	refs := asStringSlice(shot["asset_refs"])
	refs = appendUnique(refs, relative)
	shot["asset_refs"] = refs
	shot["asset_meta"] = map[string]any{
		"path": relative, "media_id": strings.TrimSpace(in.MediaID), "sha256": sum,
		"width": in.Width, "height": in.Height, "model": strings.TrimSpace(in.Model), "source": "external",
	}
	shots[selected] = shot
	rawShots := make([]any, len(shots))
	for index, value := range shots {
		rawShots[index] = value
	}
	storyboard := asMap(item.Storyboard)
	storyboard["shots"] = rawShots
	item.Storyboard = storyboard
	item.Revision++
	item.Status = "draft"
	item.Approval = map[string]any{"approved_revision": 0}
	if err := writeCreative(root, &item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{
		"state":    "attached",
		"creative": item,
		"shot":     shot,
		"asset":    describeShotAsset(root, relative),
		"next":     []string{"调用 media.workbench.shot.status 查看资产", "调用 creative.validate 后批准"},
	}, nil
}

func copyShotAsset(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func hashShotAsset(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
