package main

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"
)

type textAuditText struct {
	Text     string  `json:"text"`
	Left     float64 `json:"left"`
	Right    float64 `json:"right"`
	Top      float64 `json:"top"`
	Bottom   float64 `json:"bottom"`
	Alpha    float64 `json:"alpha"`
	FontSize float64 `json:"fontSize"`
}

type textAuditScene struct {
	Time    float64         `json:"time"`
	Title   string          `json:"title"`
	Caption string          `json:"caption"`
	Texts   []textAuditText `json:"texts"`
}

type textAuditPayload struct {
	Schema string           `json:"schema"`
	Width  int              `json:"width"`
	Height int              `json:"height"`
	Scenes []textAuditScene `json:"scenes"`
}

func textAuditChecks(root, projectID, times string) ([]visualCheck, []map[string]any, []string) {
	skipped := []visualCheck{
		{ID: "text_bounds", Status: "skipped", Detail: "需要 Style Pack text audit adapter"},
		{ID: "safe_area", Status: "skipped", Detail: "需要文字 bounding box 数据"},
		{ID: "text_overlap", Status: "skipped", Detail: "需要文字 bounding box 数据"},
		{ID: "title_hit", Status: "skipped", Detail: "需要场景/数据契约审计"},
	}
	if strings.TrimSpace(projectID) == "" {
		return skipped, nil, nil
	}
	item, err := loadProject(root, projectID)
	if err != nil {
		return []visualCheck{{ID: "text_audit", Status: "warning", Level: "warn", Detail: err.Error()}}, nil, []string{"文字审计跳过: " + err.Error()}
	}
	sourceRoot, err := sourceRootForProject(root, item)
	if err != nil {
		return []visualCheck{{ID: "text_audit", Status: "warning", Level: "warn", Detail: err.Error()}}, nil, []string{"文字审计跳过: " + err.Error()}
	}
	enabled, page, configuredTimes := textAuditConfig(root, item, sourceRoot)
	if !enabled {
		return skipped, nil, nil
	}
	if strings.TrimSpace(times) != "" {
		configuredTimes = times
	}
	script := filepath.Join(pluginRoot(), "renderer", "browser-code", "text-audit.mjs")
	if !fileExists(script) {
		return []visualCheck{{ID: "text_audit", Status: "warning", Level: "warn", Detail: "text-audit.mjs 不存在"}}, nil, []string{"文字审计脚本缺失"}
	}
	args := []string{script, "--source=" + sourceRoot, "--page=" + page, "--times=" + configuredTimes}
	output, runErr := runCodeNode(sourceRoot, args, 2*time.Minute)
	if runErr != nil {
		return []visualCheck{{ID: "text_audit", Status: "warning", Level: "warn", Detail: conciseProcessOutput(output, runErr)}}, nil, []string{"文字审计失败: " + conciseProcessOutput(output, runErr)}
	}
	payload, parseErr := parseTextAuditPayload(output)
	if parseErr != nil {
		return []visualCheck{{ID: "text_audit", Status: "warning", Level: "warn", Detail: parseErr.Error()}}, nil, []string{"文字审计输出无法解析: " + parseErr.Error()}
	}
	checks, findings, warnings := evaluateTextAudit(payload)
	return checks, findings, warnings
}

func textAuditConfig(root string, item project, sourceRoot string) (bool, string, string) {
	read := func(dir string) (bool, string, string) {
		pack, err := readCodeStylePack(dir)
		if err != nil || !pack.Quality.TextAudit.Enabled {
			return false, "", ""
		}
		page := firstNonEmpty(pack.Quality.TextAudit.Page, "reel/index.html")
		times := firstNonEmpty(pack.Quality.TextAudit.Times, "1.2,3.5,9,16,23,28")
		return true, page, times
	}
	if enabled, page, times := read(sourceRoot); enabled {
		return enabled, page, times
	}
	pack, ok := findAssetPack(root, item.StylePack, item.StyleVersion)
	if !ok {
		return false, "", ""
	}
	resource, ok := findAssetResource(pack, "template")
	if !ok {
		return false, "", ""
	}
	templateRoot, err := packRelativeFile(pack, resource.Path)
	if err != nil {
		return false, "", ""
	}
	return read(templateRoot)
}

func parseTextAuditPayload(output []byte) (textAuditPayload, error) {
	text := string(output)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return textAuditPayload{}, fmt.Errorf("no JSON object in text audit output")
	}
	var payload textAuditPayload
	if err := json.Unmarshal([]byte(text[start:end+1]), &payload); err != nil {
		return textAuditPayload{}, err
	}
	if payload.Width <= 0 {
		payload.Width = 1080
	}
	if payload.Height <= 0 {
		payload.Height = 1920
	}
	return payload, nil
}

func evaluateTextAudit(payload textAuditPayload) ([]visualCheck, []map[string]any, []string) {
	width, height := float64(payload.Width), float64(payload.Height)
	marginX, marginY := width*0.05, height*0.05
	boundsFailed, safeFailed, overlapFailed, overlapWarned, titleMissing := false, false, false, false, false
	hasTitle := false
	findings := make([]map[string]any, 0)
	warnings := make([]string, 0)
	textCount := 0
	for _, scene := range payload.Scenes {
		textCount += len(scene.Texts)
		for _, text := range scene.Texts {
			if text.Left < 0 || text.Right > width || text.Top < 0 || text.Bottom > height {
				boundsFailed = true
				findings = append(findings, issue("TEXT_OUT_OF_BOUNDS", "output", fmt.Sprintf("%.2fs 文本越界: %s", scene.Time, text.Text), "把文字收进画布边界"))
			}
			if text.FontSize >= 32 && (text.Left < marginX || text.Right > width-marginX || text.Top < marginY || text.Bottom > height-marginY) {
				safeFailed = true
				findings = append(findings, issue("TEXT_OUT_OF_SAFE_AREA", "output", fmt.Sprintf("%.2fs 文本进入安全区外: %s", scene.Time, text.Text), "标题/解释/事实卡保持在 5% 安全区内"))
			}
		}
		for i := 0; i < len(scene.Texts); i++ {
			for j := i + 1; j < len(scene.Texts); j++ {
				a, b := scene.Texts[i], scene.Texts[j]
				if a.FontSize < 18 || b.FontSize < 18 {
					continue
				}
				if strings.TrimSpace(a.Text) == strings.TrimSpace(b.Text) {
					continue
				}
				ratio := textOverlapRatio(a, b)
				if ratio > 0.35 {
					overlapFailed = true
					findings = append(findings, issue("TEXT_OVERLAP", "output", fmt.Sprintf("%.2fs 文本重叠: %s / %s (%.0f%%)", scene.Time, a.Text, b.Text, ratio*100), "调整文字层级或位置"))
				} else if ratio > 0.20 {
					overlapWarned = true
					warnings = append(warnings, fmt.Sprintf("%.2fs 文本轻微重叠: %s / %s (%.0f%%)", scene.Time, a.Text, b.Text, ratio*100))
				}
			}
		}
		if strings.TrimSpace(scene.Title) != "" {
			hasTitle = true
			found := false
			for _, text := range scene.Texts {
				if strings.Contains(text.Text, scene.Title) {
					found = true
					break
				}
			}
			if !found {
				titleMissing = true
				findings = append(findings, issue("TEXT_TITLE_MISSING", "output", fmt.Sprintf("%.2fs 未找到场景标题: %s", scene.Time, scene.Title), "检查数据契约与文字绘制"))
			}
		}
	}
	status := func(failed bool, detail string) visualCheck {
		if failed {
			return visualCheck{ID: "", Status: "failed", Level: "error", Detail: detail}
		}
		return visualCheck{ID: "", Status: "passed", Level: "error", Detail: detail}
	}
	bounds := status(boundsFailed, fmt.Sprintf("检查 %d 条文字", textCount))
	bounds.ID = "text_bounds"
	safe := status(safeFailed, "标题/大字号文字 5% 安全区")
	safe.ID = "safe_area"
	overlap := visualCheck{ID: "text_overlap", Status: "passed", Level: "warn", Detail: "18px 以上文字无重叠"}
	if overlapFailed {
		overlap.Status = "failed"
		overlap.Level = "error"
		overlap.Detail = "检测到严重文字重叠"
	} else if overlapWarned {
		overlap.Status = "warning"
		overlap.Detail = "检测到轻微文字重叠"
	}
	title := visualCheck{ID: "title_hit", Status: "skipped", Level: "warn", Detail: "Style Pack 未提供场景标题元数据"}
	if hasTitle {
		title.Status = "passed"
		title.Detail = "场景标题全部命中"
		if titleMissing {
			title.Status = "failed"
			title.Level = "error"
			title.Detail = "存在未命中的场景标题"
		}
	}
	return []visualCheck{bounds, safe, overlap, title}, findings, warnings
}

func textOverlapRatio(a, b textAuditText) float64 {
	left := math.Max(a.Left, b.Left)
	right := math.Min(a.Right, b.Right)
	top := math.Max(a.Top, b.Top)
	bottom := math.Min(a.Bottom, b.Bottom)
	if right <= left || bottom <= top {
		return 0
	}
	intersection := (right - left) * (bottom - top)
	areaA := math.Max(1, (a.Right-a.Left)*(a.Bottom-a.Top))
	areaB := math.Max(1, (b.Right-b.Left)*(b.Bottom-b.Top))
	return intersection / math.Min(areaA, areaB)
}
func bandingGateForProject(root, projectID string) bool {
	if strings.TrimSpace(projectID) == "" {
		return true
	}
	item, err := loadProject(root, projectID)
	if err != nil {
		return true
	}
	sourceRoot, err := sourceRootForProject(root, item)
	if err != nil {
		return true
	}
	readGate := func(dir string) (bool, bool) {
		pack, readErr := readCodeStylePack(dir)
		if readErr != nil || pack.Quality.BandingGate == nil {
			return true, false
		}
		return *pack.Quality.BandingGate, true
	}
	if gate, found := readGate(sourceRoot); found {
		return gate
	}
	pack, ok := findAssetPack(root, item.StylePack, item.StyleVersion)
	if !ok {
		return true
	}
	resource, ok := findAssetResource(pack, "template")
	if !ok {
		return true
	}
	templateRoot, err := packRelativeFile(pack, resource.Path)
	if err != nil {
		return true
	}
	if gate, found := readGate(templateRoot); found {
		return gate
	}
	return true
}
