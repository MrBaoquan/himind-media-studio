package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type producedShotAsset struct {
	ShotID         string `json:"shot_id"`
	Provider       string `json:"provider"`
	Path           string `json:"path"`
	SHA256         string `json:"sha256"`
	Size           int    `json:"size"`
	Mock           bool   `json:"mock"`
	PreviewDataURL string `json:"preview_data_url"`
}

func shotProduce(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadCreative(root, in.CreativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	shots := asMapSlice(asMap(item.Storyboard)["shots"])
	if len(shots) == 0 {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("SHOT_PRODUCE_NO_SHOTS", "shot", "创作包没有分镜", "先补充 storyboard.shots")}, "retryable": true}, nil
	}

	targets := selectProduceTargets(shots, strings.TrimSpace(in.ShotID), in.ShotIDs)
	if len(targets) == 0 {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("SHOT_PRODUCE_NO_TARGETS", "shot", "没有可生成的镜头；默认只生成 provider=ai.image 的镜头", "在分镜中设置 provider=ai.image，或传 shot_ids")}, "retryable": true}, nil
	}

	assets := make([]producedShotAsset, 0)
	blockers := make([]map[string]any, 0)
	warnings := make([]map[string]any, 0)
	now := time.Now().UTC().Format(time.RFC3339)
	for _, index := range targets {
		shot := shots[index]
		shotID := strings.TrimSpace(asString(shot["id"]))
		if shotID == "" {
			shotID = fmt.Sprintf("shot%d", index+1)
			shot["id"] = shotID
		}
		provider := strings.TrimSpace(asString(shot["provider"]))
		if provider == "" {
			provider = "programmatic.browser-code"
			warnings = append(warnings, issue("SHOT_PROVIDER_DEFAULT", "shot", fmt.Sprintf("%s 未指定生成方式，跳过 AI 生成", shotID), "设置 provider=ai.image 后再生成"))
		}
		switch provider {
		case "ai.image":
			prompt := strings.TrimSpace(asString(shot["prompt"]))
			if prompt == "" {
				warnings = append(warnings, issue("SHOT_PROMPT_MISSING", "shot", fmt.Sprintf("%s 没有 prompt，将生成占位图", shotID), "补充镜头提示词以获得更准确的模拟图"))
			}
			asset, err := writeMockShotSVG(root, item.ID, shotID, provider, prompt)
			if err != nil {
				blockers = append(blockers, issue("SHOT_PROVIDER_FAILED", "shot", fmt.Sprintf("%s 模拟图片生成失败: %v", shotID, err), "检查工作区写入权限"))
				continue
			}
			refs := asStringSlice(shot["asset_refs"])
			refs = appendUnique(refs, asset.Path)
			shot["provider"] = provider
			shot["status"] = "generated"
			shot["mock"] = true
			shot["asset_refs"] = refs
			shot["produced_at"] = now
			shots[index] = shot
			assets = append(assets, asset)
		case "ai.video":
			blockers = append(blockers, issue("SHOT_PROVIDER_ASYNC_NOT_IMPLEMENTED", "shot", fmt.Sprintf("%s 是 AI 视频镜头；异步生成尚未接入 shot.produce", shotID), "先用 ai.image + Ken Burns，或接入 media.video.generate 与 media.job.wait"))
		case "stock.video":
			blockers = append(blockers, issue("SHOT_PROVIDER_UNAVAILABLE", "shot", fmt.Sprintf("%s 需要素材库 External Pack", shotID), "安装 com.himind.media-stock-pack"))
		default:
			warnings = append(warnings, issue("SHOT_PROGRAMMATIC_PRODUCE_NOT_IMPLEMENTED", "shot", fmt.Sprintf("%s 是程序化镜头，暂不由 shot.produce 生成", shotID), "程序化镜头继续走整体 Browser-Code / Remotion 渲染"))
		}
	}

	if len(assets) > 0 {
		storyboard := asMap(item.Storyboard)
		rawShots := make([]any, len(shots))
		for index, shot := range shots {
			rawShots[index] = shot
		}
		storyboard["shots"] = rawShots
		item.Storyboard = storyboard
		item.Revision++
		item.Status = "draft"
		item.Approval = map[string]any{"approved_revision": 0}
		if err := writeCreative(root, &item); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
	}

	state := "blocked"
	if len(assets) > 0 && len(blockers) == 0 {
		state = "completed"
	} else if len(assets) > 0 {
		state = "partial"
	}
	return map[string]any{
		"state":    state,
		"mode":     "mock",
		"creative": item,
		"assets":   assets,
		"blockers": blockers,
		"warnings": warnings,
		"next":     []string{"调用 media.workbench.shot.status 查看资产", "调用 creative.validate 后批准", "下一步接入 timeline.assemble 与 Remotion 时间线"},
	}, nil
}

func shotStatus(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadCreative(root, in.CreativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	shots := asMapSlice(asMap(item.Storyboard)["shots"])
	items := make([]map[string]any, 0, len(shots))
	for index, shot := range shots {
		shotID := strings.TrimSpace(asString(shot["id"]))
		if shotID == "" {
			shotID = fmt.Sprintf("shot%d", index+1)
		}
		refs := asStringSlice(shot["asset_refs"])
		assets := make([]map[string]any, 0, len(refs))
		for _, ref := range refs {
			assets = append(assets, describeShotAsset(root, ref))
		}
		items = append(items, map[string]any{
			"shot_id":    shotID,
			"provider":   strings.TrimSpace(asString(shot["provider"])),
			"status":     firstNonEmpty(strings.TrimSpace(asString(shot["status"])), "planned"),
			"asset_refs": refs,
			"assets":     assets,
			"mock":       shot["mock"] == true,
		})
	}
	return map[string]any{"creative_id": item.ID, "revision": item.Revision, "items": items, "total": len(items)}, nil
}

func selectProduceTargets(shots []map[string]any, shotID string, shotIDs []string) []int {
	wanted := map[string]bool{}
	if shotID != "" {
		wanted[shotID] = true
	}
	for _, id := range shotIDs {
		if id = strings.TrimSpace(id); id != "" {
			wanted[id] = true
		}
	}
	targets := make([]int, 0)
	for index, shot := range shots {
		id := strings.TrimSpace(asString(shot["id"]))
		if len(wanted) > 0 {
			if wanted[id] {
				targets = append(targets, index)
			}
			continue
		}
		if strings.TrimSpace(asString(shot["provider"])) == "ai.image" {
			targets = append(targets, index)
		}
	}
	return targets
}

func writeMockShotSVG(root, creativeID, shotID, provider, prompt string) (producedShotAsset, error) {
	seed := sha256.Sum256([]byte(creativeID + "|" + shotID + "|" + prompt))
	hash := hex.EncodeToString(seed[:])
	relPath := filepath.ToSlash(filepath.Join(".himind-media", "creatives", creativeID, "assets", shotID, "mock-"+hash[:8]+".svg"))
	absPath, err := workspaceFile(root, relPath)
	if err != nil {
		return producedShotAsset{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return producedShotAsset{}, err
	}
	svg := mockShotSVG(shotID, provider, prompt, hash)
	if err := os.WriteFile(absPath, []byte(svg), 0644); err != nil {
		return producedShotAsset{}, err
	}
	sum := sha256.Sum256([]byte(svg))
	return producedShotAsset{
		ShotID: shotID, Provider: provider, Path: relPath,
		SHA256: hex.EncodeToString(sum[:]), Size: len(svg), Mock: true,
		PreviewDataURL: "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg)),
	}, nil
}

func mockShotSVG(shotID, provider, prompt, hash string) string {
	palette := [][2]string{
		{"#0f172a", "#1d4ed8"}, {"#111827", "#7c3aed"}, {"#1f2937", "#0ea5e9"},
		{"#2b1b17", "#d97706"}, {"#0b3b2e", "#10b981"}, {"#2a1030", "#db2777"},
	}
	bucket := int(hash[0]) % len(palette)
	colors := palette[bucket]
	lines := wrapRunes(firstNonEmpty(prompt, "（无 prompt）"), 20, 8)
	var text strings.Builder
	for index, line := range lines {
		fmt.Fprintf(&text, `<text x="100" y="%d" font-family="Microsoft YaHei, Noto Sans SC, sans-serif" font-size="46" fill="#ffffff" opacity=".92">%s</text>`, 560+index*64, html.EscapeString(line))
	}
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="1080" height="1920" viewBox="0 0 1080 1920">
  <defs>
    <linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s"/>
    </linearGradient>
  </defs>
  <rect width="1080" height="1920" fill="url(#g)"/>
  <circle cx="900" cy="220" r="260" fill="rgba(255,255,255,.08)"/>
  <circle cx="200" cy="1700" r="340" fill="rgba(255,255,255,.05)"/>
  <rect x="56" y="56" width="968" height="1808" rx="40" fill="rgba(0,0,0,.24)" stroke="rgba(255,255,255,.18)" stroke-width="3"/>
  <text x="100" y="180" font-family="Cascadia Mono, Consolas, monospace" font-size="30" fill="#ffffff" opacity=".72">HIMIND / SHOT MOCK</text>
  <text x="100" y="310" font-family="Microsoft YaHei, Noto Sans SC, sans-serif" font-size="84" font-weight="700" fill="#ffffff">MOCK AI IMAGE</text>
  <text x="100" y="390" font-family="Cascadia Mono, Consolas, monospace" font-size="32" fill="#ffffff" opacity=".68">%s · %s · 1080×1920</text>
  <line x1="100" y1="460" x2="980" y2="460" stroke="rgba(255,255,255,.22)" stroke-width="2"/>
  %s
  <text x="100" y="1740" font-family="Cascadia Mono, Consolas, monospace" font-size="28" fill="#ffffff" opacity=".58">MOCK ASSET · no model call · deterministic</text>
</svg>`, colors[0], colors[1], html.EscapeString(shotID), html.EscapeString(provider), text.String())
}

func wrapRunes(value string, width, maxLines int) []string {
	runes := []rune(value)
	lines := make([]string, 0, maxLines)
	for len(runes) > 0 && len(lines) < maxLines {
		n := width
		if len(runes) < n {
			n = len(runes)
		}
		lines = append(lines, string(runes[:n]))
		runes = runes[n:]
	}
	if len(runes) > 0 && len(lines) > 0 {
		lines[len(lines)-1] = strings.TrimSuffix(lines[len(lines)-1], " ") + "…"
	}
	return lines
}

func describeShotAsset(root, ref string) map[string]any {
	item := map[string]any{"path": ref, "exists": false}
	absPath, err := workspaceFile(root, ref)
	if err != nil {
		return item
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return item
	}
	item["exists"] = true
	item["size"] = int(info.Size())
	ext := strings.ToLower(filepath.Ext(absPath))
	switch ext {
	case ".svg", ".png", ".jpg", ".jpeg", ".webp":
		if info.Size() <= 6*1024*1024 {
			if raw, readErr := os.ReadFile(absPath); readErr == nil {
				mime := "image/svg+xml"
				switch ext {
				case ".png":
					mime = "image/png"
				case ".jpg", ".jpeg":
					mime = "image/jpeg"
				case ".webp":
					mime = "image/webp"
				}
				item["preview_data_url"] = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw)
			}
		}
	}
	return item
}

func appendUnique(values []string, value string) []string {
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}
