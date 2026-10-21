package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// shotProvider 描述一种镜头生产方式。它只声明契约，不直接调用模型：
// 程序化能力可以内置，AI 图片/视频等由 External Pack 提供。
type shotProvider struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Kind            string   `json:"kind"`   // programmatic | image | video | stock
	Source          string   `json:"source"` // workbench | agent | external
	Description     string   `json:"description"`
	CostTier        string   `json:"cost_tier"`
	LatencyTier     string   `json:"latency_tier"`
	Network         bool     `json:"network"`
	Builtin         bool     `json:"builtin"`
	Available       bool     `json:"available"`
	RequiresPack    string   `json:"requires_pack,omitempty"`
	Capability      string   `json:"capability,omitempty"`
	RequiresService string   `json:"requires_service,omitempty"`
	Async           bool     `json:"async"`
	Inputs          []string `json:"inputs"`
	Outputs         []string `json:"outputs"`
}

type shotProviderCatalog struct {
	Schema    string         `json:"schema"`
	Providers []shotProvider `json:"providers"`
}

func loadShotProviders() ([]shotProvider, error) {
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), "catalog", "shot-providers.json"))
	if err != nil {
		return nil, err
	}
	var catalog shotProviderCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, err
	}
	if catalog.Schema != "himind.shot-providers.v1" {
		return nil, fmt.Errorf("unexpected shot provider schema: %s", catalog.Schema)
	}
	return catalog.Providers, nil
}

func shotProviderByID(providers []shotProvider, id string) (shotProvider, bool) {
	for _, provider := range providers {
		if provider.ID == id {
			return provider, true
		}
	}
	return shotProvider{}, false
}

func shotProvidersCapability(in requestInput) (any, *himindjsonrpc.Error) {
	providers, err := loadShotProviders()
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{
		"items":   providers,
		"total":   len(providers),
		"default": "programmatic.browser-code",
		"note":    "内置 provider 可直接使用；requires_pack 非空的 provider 需要安装对应 External Pack。",
	}, nil
}

func shotAssign(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadCreative(root, in.CreativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	shotID := strings.TrimSpace(in.ShotID)
	if shotID == "" {
		return nil, himindjsonrpc.InvalidParams("shot_id is required")
	}
	providerID := strings.TrimSpace(in.Provider)
	providers, err := loadShotProviders()
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if providerID != "" {
		if _, ok := shotProviderByID(providers, providerID); !ok {
			return nil, himindjsonrpc.InvalidParams("unknown shot provider: " + providerID)
		}
	}
	storyboard := asMap(item.Storyboard)
	shots := asMapSlice(storyboard["shots"])
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
	shot := shots[selected]
	if providerID != "" {
		shot["provider"] = providerID
	}
	shot["technique"] = strings.TrimSpace(in.Technique)
	shot["prompt"] = strings.TrimSpace(in.Prompt)
	shot["fallback_provider"] = strings.TrimSpace(in.FallbackProvider)
	if in.ReferenceAssets != nil {
		shot["reference_assets"] = in.ReferenceAssets
	}
	if strings.TrimSpace(asString(shot["status"])) == "" {
		shot["status"] = "planned"
	}
	shots[selected] = shot
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
	return map[string]any{
		"creative": item,
		"shot":     shot,
		"next":     []string{"调用 media.workbench.timeline.plan 检查能力覆盖与 blockers", "补齐 prompt/资产后由 External Pack 执行生成"},
	}, nil
}

func timelinePlan(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	item, err := loadCreative(root, in.CreativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	providers, err := loadShotProviders()
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	providerMap := map[string]shotProvider{}
	for _, provider := range providers {
		providerMap[provider.ID] = provider
	}

	shots := asMapSlice(asMap(item.Storyboard)["shots"])
	beats := asMapSlice(asMap(item.Script)["beats"])
	beatIDs := map[string]bool{}
	for _, beat := range beats {
		if id := strings.TrimSpace(asString(beat["id"])); id != "" {
			beatIDs[id] = true
		}
	}

	defaultProvider := "programmatic.browser-code"
	planShots := make([]map[string]any, 0, len(shots))
	blockers := make([]map[string]any, 0)
	warnings := make([]map[string]any, 0)
	coverage := map[string]int{}
	totalDuration := 0.0

	for index, shot := range shots {
		shotID := strings.TrimSpace(asString(shot["id"]))
		if shotID == "" {
			shotID = fmt.Sprintf("shot%d", index+1)
		}
		providerID := strings.TrimSpace(asString(shot["provider"]))
		if providerID == "" {
			providerID = defaultProvider
			warnings = append(warnings, issue("SHOT_PROVIDER_DEFAULT", "shot", fmt.Sprintf("%s 未指定生成方式，按程序化处理", shotID), "在分镜中设置 provider"))
		}
		provider, ok := providerMap[providerID]
		if !ok {
			blockers = append(blockers, issue("SHOT_PROVIDER_UNKNOWN", "shot", fmt.Sprintf("%s 使用了未知 provider: %s", shotID, providerID), "调用 media.workbench.shot.providers 查看可用能力"))
			provider = shotProvider{ID: providerID, Name: providerID, Kind: "unknown"}
		} else if !provider.Available {
			remediation := "安装对应 External Pack 后重试"
			if provider.RequiresPack != "" {
				remediation = "安装 " + provider.RequiresPack + " 后重新生成计划"
			}
			blockers = append(blockers, issue("SHOT_PROVIDER_UNAVAILABLE", "shot", fmt.Sprintf("%s 的生成方式 %s 尚未安装", shotID, provider.Name), remediation))
		}
		duration := numberFloat(shot["duration_seconds"])
		if duration <= 0 {
			blockers = append(blockers, issue("SHOT_DURATION_INVALID", "shot", fmt.Sprintf("%s 时长无效", shotID), "设置大于 0 的时长"))
		}
		totalDuration += duration
		prompt := strings.TrimSpace(asString(shot["prompt"]))
		if (provider.Kind == "image" || provider.Kind == "video") && prompt == "" {
			blockers = append(blockers, issue("SHOT_PROMPT_MISSING", "shot", fmt.Sprintf("%s 是 %s 镜头但没有 prompt", shotID, provider.Kind), "补充镜头提示词"))
		}
		beatID := strings.TrimSpace(asString(shot["beat_id"]))
		if beatID != "" && !beatIDs[beatID] {
			warnings = append(warnings, issue("SHOT_BEAT_MISSING", "shot", fmt.Sprintf("%s 引用了不存在的 beat: %s", shotID, beatID), "检查 beat_id"))
		}
		coverage[providerID]++
		planShots = append(planShots, map[string]any{
			"id":                shotID,
			"beat_id":           beatID,
			"duration_seconds":  duration,
			"provider":          providerID,
			"provider_name":     provider.Name,
			"provider_kind":     provider.Kind,
			"provider_source":   provider.Source,
			"capability":        provider.Capability,
			"requires_service":  provider.RequiresService,
			"async":             provider.Async,
			"available":         provider.Available,
			"technique":         strings.TrimSpace(asString(shot["technique"])),
			"prompt":            prompt,
			"status":            firstNonEmpty(strings.TrimSpace(asString(shot["status"])), "planned"),
			"fallback_provider": strings.TrimSpace(asString(shot["fallback_provider"])),
		})
	}

	target := numberFloat(asMap(item.Production)["duration_seconds"])
	if target > 0 && math.Abs(totalDuration-target) > 1 {
		warnings = append(warnings, issue("SHOT_DURATION_DRIFT", "timeline", fmt.Sprintf("镜头总时长 %.1fs 与制作规格 %.1fs 不一致", totalDuration, target), "调整镜头时长或制作规格"))
	}
	state := "ready"
	if len(blockers) > 0 {
		state = "blocked"
	}
	return map[string]any{
		"creative_id":             item.ID,
		"revision":                item.Revision,
		"title":                   item.Title,
		"duration_seconds":        totalDuration,
		"target_duration_seconds": target,
		"shots":                   planShots,
		"coverage":                coverage,
		"blockers":                blockers,
		"warnings":                warnings,
		"state":                   state,
		"next":                    []string{"补齐 blockers 后由 External Pack 执行镜头生成", "程序化镜头可直接进入 Remotion 时间线组装", "生成完成后调用 render.start"},
	}, nil
}
