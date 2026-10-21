package main

import (
	"regexp"
	"strconv"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// intent_plan.go — 把自由文本创作请求路由成可执行的生产路径。
//
// 这里不生成内容，只做确定性分流：快路径（数据/模板/排行榜/路线）
// 与故事路径（CreativePackage/分镜/角色/旁白）。AI 客户端先调用它，
// 再按返回的能力序列推进，避免每个入口各写一套隐式判断。

type intentMissing struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Why     string `json:"why"`
	Example string `json:"example"`
}

type intentStep struct {
	Order       int    `json:"order"`
	Capability  string `json:"capability"`
	Purpose     string `json:"purpose"`
	WaitForUser bool   `json:"wait_for_user,omitempty"`
}

type intentPlanResult struct {
	Route                    string          `json:"route"`
	RouteLabel               string          `json:"route_label"`
	Confidence               float64         `json:"confidence"`
	Reason                   string          `json:"reason"`
	Signals                  map[string]any  `json:"signals"`
	Missing                  []intentMissing `json:"missing"`
	Questions                []string        `json:"questions"`
	RecommendedCapabilities  []intentStep    `json:"recommended_capabilities"`
	RequiresCreativePackage  bool            `json:"requires_creative_package"`
	RequiresCreativeApproval bool            `json:"requires_creative_approval"`
	RequiresNetwork          bool            `json:"requires_network"`
	EstimatedUserTouches     int             `json:"estimated_user_touches"`
	NextAction               string          `json:"next_action"`
	StopConditions           []string        `json:"stop_conditions"`
}

var durationPattern = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(秒|s|分钟|min)`)

func intentPlan(in requestInput) (any, *himindjsonrpc.Error) {
	request := strings.TrimSpace(in.Request)
	if request == "" {
		request = strings.TrimSpace(in.Prompt)
	}
	if request == "" {
		request = strings.TrimSpace(in.Description)
	}
	if request == "" {
		request = strings.TrimSpace(in.CreativeBrief)
	}
	if request == "" {
		return nil, himindjsonrpc.InvalidParams("request is required")
	}
	text := strings.ToLower(request)

	fastHits := keywordHits(text, []string{
		"排行榜", "排名", "榜单", "数据", "图表", "柱状", "折线", "饼图", "地图", "路线",
		"高铁", "地铁", "统计", "指数", "价格", "销量", "人口", "收入", "占比", "增长率",
		"gdp", "一版", "快速", "模板", "口播", "卡片", "对比", "数字",
	})
	storyHits := keywordHits(text, []string{
		"故事", "剧情", "叙事", "科普", "讲解", "原理", "手绘", "分镜", "角色", "人物",
		"情节", "剧本", "旁白", "配音", "字幕", "纪录片", "电影", "短片", "镜头", "场景",
	})
	if strings.Contains(text, "动画") || strings.Contains(text, "科普") {
		storyHits = append(storyHits, "动画叙事")
	}

	storyScore := len(storyHits)
	fastScore := len(fastHits)
	// “科普/故事/分镜/旁白”这类强叙事信号优先；单纯的“动画”不足以压过数据类请求。
	strongStory := containsAny(text, []string{"故事", "剧情", "分镜", "旁白", "手绘", "角色", "剧本"})
	promoHits := keywordHits(text, []string{"宣传片", "产品宣传", "产品视频", "品牌片", "广告片", "产品介绍视频", "promo"})
	route := "fast"
	reason := "请求以数据、模板或路线为主，适合用确定性模板路径先出可审阅版本。"
	if len(promoHits) > 0 {
		route = "promo"
		reason = "请求是产品宣传片，适合走真实产品调研 → 风格样帧 → CreativePackage → promo-video 模板 → 审核渲染的专用路径。"
	} else if strongStory || storyScore >= fastScore+2 || (storyScore > fastScore && !containsAny(text, []string{"排名", "排行榜", "数据"})) {
		route = "story"
		reason = "请求包含叙事、分镜或讲解结构，适合先形成 CreativePackage 并通过人工确认再进入生产。"
	}

	signals := map[string]any{
		"fast_score":               fastScore,
		"story_score":              storyScore,
		"fast_hits":                fastHits,
		"story_hits":               storyHits,
		"promo_score":              len(promoHits),
		"promo_hits":               promoHits,
		"has_numbers":              regexp.MustCompile(`\d`).MatchString(request),
		"data_like":                containsAny(text, []string{"gdp", "排名", "排行", "榜单", "数据", "统计", "指数", "占比", "增长"}),
		"platform":                 strings.TrimSpace(in.Platform),
		"style_hint":               strings.TrimSpace(in.StyleHint),
		"narration":                wantsNarration(text, in.WantsNarration),
		"aspect_ratio":             inferredAspect(text, in.Platform),
		"duration":                 inferredDuration(request, in.TargetDurationSeconds),
		"duration_default_seconds": defaultDuration(route),
	}

	missing := intentMissingInputs(text, in, route)
	questions := make([]string, 0, len(missing))
	for _, item := range missing {
		questions = append(questions, item.Example)
	}

	plan := intentPlanResult{
		Route:                    route,
		Confidence:               intentConfidence(storyScore, fastScore, strongStory),
		Reason:                   reason,
		Signals:                  signals,
		Missing:                  missing,
		Questions:                questions,
		RequiresCreativePackage:  route == "story" || route == "promo",
		RequiresCreativeApproval: route == "story" || route == "promo",
		RequiresNetwork:          route == "story" || route == "promo" || containsAny(text, []string{"http://", "https://", "api", "联网", "搜索", "抓取"}),
		EstimatedUserTouches:     boolToInt(route == "story" || route == "promo", 2, 1),
		StopConditions: []string{
			"预览未完成时不得宣称已出片",
			"渲染任务必须轮询到 completed，并给出真实 artifact",
			"CreativePackage 未经用户确认不得进入生产",
		},
	}
	if route == "promo" {
		plan.RouteLabel = "宣传片路径 · 真实产品素材"
		plan.RecommendedCapabilities = promoCapabilitySteps()
		plan.NextAction = "先补齐产品名/网址与风格方向；用 3-4 个风格样帧确认视觉方向，创建 CreativePackage，审批后 materialize 到 promo-video 模板，预览、质检、渲染并轮询 artifact。"
	} else if route == "story" {
		plan.RouteLabel = "故事路径 · CreativePackage"
		plan.RecommendedCapabilities = storyCapabilitySteps()
		plan.NextAction = "先创建或读取项目，调用 media.workbench.creative.create 形成 CreativePackage；补齐 missing 后请用户确认，再 materialize 并逐镜头生产。"
	} else {
		plan.RouteLabel = "快路径 · 数据/模板"
		plan.RecommendedCapabilities = fastCapabilitySteps()
		plan.NextAction = "先创建或读取项目；有数据则 ingest/bind，随后 plan.from-recipe → recipe.compile → preview.start，确认预览后再渲染。"
	}
	return plan, nil
}

func keywordHits(text string, words []string) []string {
	hits := make([]string, 0, len(words))
	for _, word := range words {
		if strings.Contains(text, word) {
			hits = append(hits, word)
		}
	}
	return hits
}

func containsAny(text string, words []string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

func wantsNarration(text string, explicit *bool) bool {
	if explicit != nil {
		return *explicit
	}
	return containsAny(text, []string{"旁白", "配音", "口播", "解说", "tts", "语音", "字幕"})
}

func inferredAspect(text, platform string) string {
	joined := text + " " + strings.ToLower(platform)
	if containsAny(joined, []string{"16:9", "横屏", "b站", "bilibili", "youtube"}) {
		return "16:9"
	}
	if containsAny(joined, []string{"9:16", "竖屏", "抖音", "快手", "视频号", "小红书", "shorts"}) {
		return "9:16"
	}
	return ""
}

func inferredDuration(request string, explicit float64) float64 {
	if explicit > 0 {
		return explicit
	}
	if match := durationPattern.FindStringSubmatch(request); len(match) == 3 {
		value, err := strconv.ParseFloat(match[1], 64)
		if err == nil {
			if match[2] == "分钟" || strings.EqualFold(match[2], "min") {
				value *= 60
			}
			if value > 0 {
				return value
			}
		}
	}
	return 0
}

func defaultDuration(route string) float64 {
	if route == "story" {
		return 60
	}
	if route == "promo" {
		return 45
	}
	return 30
}

func intentMissingInputs(text string, in requestInput, route string) []intentMissing {
	missing := make([]intentMissing, 0, 5)
	if !durationPattern.MatchString(text) && in.TargetDurationSeconds <= 0 {
		missing = append(missing, intentMissing{ID: "duration", Label: "成片时长", Why: "时长决定镜头数与渲染成本。", Example: "成片希望多长？例如 30 秒或 60 秒。"})
	}
	if inferredAspect(text, in.Platform) == "" {
		missing = append(missing, intentMissing{ID: "aspect_ratio", Label: "画幅", Why: "竖屏与横屏会改变构图和字幕安全区。", Example: "发布到哪个平台，或要 9:16 竖屏还是 16:9 横屏？"})
	}
	if strings.TrimSpace(in.StyleHint) == "" && !containsAny(text, []string{"手绘", "信息图", "3d", "电影感", "赛博", "极简", "插画", "水墨", "漫画", "科幻", "写实", "扁平", "国潮", "复古", "等距", "isometric"}) {
		missing = append(missing, intentMissing{ID: "style", Label: "视觉风格", Why: "风格决定模板、色彩与镜头语法。", Example: "希望偏手绘、信息图、3D 还是电影感？"})
	}
	if route == "promo" {
		if !containsAny(text, []string{"http://", "https://", "github", "gitlab"}) && strings.TrimSpace(in.CreativeBrief) == "" && strings.TrimSpace(in.SourceProjectID) == "" {
			missing = append(missing, intentMissing{ID: "product_reference", Label: "产品素材", Why: "产品宣传片必须基于真实产品名、网址或仓库，不能凭空编造。", Example: "请给我产品名/官网/仓库路径；如果是本地项目，也可以给目录。"})
		}
	}
	if (route == "story" || route == "promo") && !wantsNarration(text, in.WantsNarration) {
		missing = append(missing, intentMissing{ID: "narration", Label: "旁白方案", Why: "故事路径需要决定是否使用 TTS 以及音色。", Example: "需要旁白吗？希望男声、女声还是童声？"})
	}
	dataLike := containsAny(text, []string{"gdp", "排名", "排行", "榜单", "数据", "统计", "指数", "占比", "增长", "价格", "销量", "人口"})
	hasSource := in.HasDataSource != nil && *in.HasDataSource
	hasSource = hasSource || containsAny(text, []string{"csv", "json", "表格", "数据源", "数据集", "excel", "xlsx", "统计局", "世界银行", "imf", "api"})
	if dataLike && !hasSource && route != "promo" {
		missing = append(missing, intentMissing{ID: "data_source", Label: "数据来源", Why: "排行榜/路线类内容需要可追溯的数据，否则无法校验数字。", Example: "请提供 CSV/JSON 路径或数据来源链接；没有的话我先用示例数据占位。"})
	}
	return missing
}

func intentConfidence(storyScore, fastScore int, strongStory bool) float64 {
	margin := storyScore - fastScore
	if margin < 0 {
		margin = -margin
	}
	if strongStory || margin >= 3 {
		return 0.86
	}
	if margin >= 1 {
		return 0.72
	}
	return 0.6
}

func fastCapabilitySteps() []intentStep {
	return []intentStep{
		{1, "media.workbench.project.get", "读取现有项目；没有则 project.create", false},
		{2, "media.workbench.data.ingest", "读取 CSV/JSON 并规范化数据", false},
		{3, "media.workbench.data.bind", "把数据绑定到模板契约", false},
		{4, "media.workbench.plan.from-recipe", "从模板配方反推可审阅计划", false},
		{5, "media.workbench.recipe.validate", "校验配方与数据契约", false},
		{6, "media.workbench.recipe.compile", "编译成分镜与时间线", false},
		{7, "media.workbench.preview.start", "启动真实预览", false},
		{8, "media.workbench.preview.status", "轮询预览直到完成", false},
		{9, "media.workbench.preview.review", "生成审阅页并检查", false},
		{10, "media.workbench.quality.check", "执行质量门禁", false},
		{11, "media.workbench.render.start", "发起真实渲染", false},
		{12, "media.workbench.render.status", "轮询渲染直到 completed", false},
		{13, "media.workbench.artifact.export", "导出并返回真实 artifact", false},
	}
}

func promoCapabilitySteps() []intentStep {
	return []intentStep{
		{1, "media.workbench.creative.create", "建立产品宣传片 CreativePackage（事实/脚本/分镜/制作规格）", false},
		{2, "media.workbench.creative.update", "按风格样帧与用户反馈迭代", false},
		{3, "media.workbench.creative.validate", "校验创意包完整性与事实来源", false},
		{4, "media.workbench.creative.approve", "用户确认风格/脚本/分镜后才生产", true},
		{5, "media.workbench.creative.materialize", "落成 promo-video 项目", false},
		{6, "media.workbench.plan.from-recipe", "反推宣传片计划", false},
		{7, "media.workbench.recipe.compile", "编译 promo-video 配方", false},
		{8, "media.workbench.preview.start", "启动真实预览", false},
		{9, "media.workbench.preview.status", "轮询预览完成", false},
		{10, "media.workbench.quality.check", "执行质量门禁", false},
		{11, "media.workbench.render.start", "审批并提交渲染", true},
		{12, "media.workbench.render.status", "轮询到 completed", false},
		{13, "media.workbench.artifact.export", "返回真实 MP4", false},
	}
}

func storyCapabilitySteps() []intentStep {
	return []intentStep{
		{1, "media.workbench.project.get", "读取现有项目；没有则 project.create", false},
		{2, "media.workbench.creative.create", "建立 CreativePackage（brief/research/script/storyboard/production）", false},
		{3, "media.workbench.creative.update", "按用户反馈迭代脚本与分镜", false},
		{4, "media.workbench.creative.validate", "校验创意包完整性", false},
		{5, "media.workbench.creative.approve", "用户确认创意包后才允许生产", true},
		{6, "media.workbench.creative.materialize", "把创意包落成项目计划", false},
		{7, "media.workbench.shot.providers", "查询可用镜头生产器（程序/AI/素材）", false},
		{8, "media.workbench.shot.assign", "逐镜头选择生成方式", false},
		{9, "media.workbench.shot.produce", "生产镜头素材", false},
		{10, "media.workbench.shot.status", "轮询镜头生产状态", false},
		{11, "media.workbench.audio.speech", "合成旁白", false},
		{12, "media.workbench.subtitle.generate", "生成并校对字幕", false},
		{13, "media.workbench.timeline.plan", "编排多轨时间线", false},
		{14, "media.workbench.timeline.save", "保存时间线修订", false},
		{15, "media.workbench.preview.start", "启动真实预览", false},
		{16, "media.workbench.preview.status", "轮询预览直到完成", false},
		{17, "media.workbench.quality.check", "执行质量门禁", false},
		{18, "media.workbench.render.start", "发起真实渲染", false},
		{19, "media.workbench.render.status", "轮询渲染直到 completed", false},
		{20, "media.workbench.artifact.export", "导出并返回真实 artifact", false},
	}
}
func boolToInt(value bool, whenTrue, whenFalse int) int {
	if value {
		return whenTrue
	}
	return whenFalse
}
