package main

import (
	"fmt"
	"math"
	"sort"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// route.go — 轨迹修复与校验（技法资产：pipeline.geo.route）
//
// 这是"路径生长"类视频的质量核心。它解决三件事：
//   1) 反走回消除：图搜索（Dijkstra）在无向图上会产出 `…→X→Y→X→…` 的支线。
//      渲染时指示器沿 d 属性顶点顺序推进，于是会倒退几帧再瞬移，描边也会
//      把同一段画第二遍。
//   2) 去环：绕一圈回到原地的闭合环，会让动画节奏被拉长。
//   3) 站点锚定：让折线精确穿过站点坐标，否则站点圆点会漂在路线外。
//
// 三者合起来的效果有量化证据（在 video-flow 的提交物上复算）：
// 站点偏差 95/95 = 0.00 km，最差站间段比率 1.50。

type waypointInput struct {
	Name   string     `json:"name"`
	LonLat [2]float64 `json:"lonLat"`
}

type routeInput struct {
	ID        string          `json:"id"`
	Coords    [][2]float64    `json:"coords"`
	Waypoints []waypointInput `json:"waypoints"`
}

type routePolicy struct {
	// 反走回：允许的归一化进度回退量。0.03 表示允许 3% 的曲线回弯。
	BacktrackTolerance float64 `json:"backtrack_tolerance"`
	// 小于该长度的段不做反走回处理（太短没有意义）。
	MinSegmentKm float64 `json:"min_segment_km"`
	// 去环：两点距离小于该值视为"同一位置"。
	LoopThresholdKm float64 `json:"loop_threshold_km"`
	// 去环：形成环所需的最小索引跨度。
	LoopMinSpan int `json:"loop_min_span"`
	// 去环：最大迭代轮数。
	LoopMaxIterations int `json:"loop_max_iterations"`
	// 去环：环长超过直线的该倍数即剪除。
	LoopCutRatio float64 `json:"loop_cut_ratio"`
	// 去环：索引跨度超过该值即剪除（即使环长不算长）。
	LoopCutSpan int `json:"loop_cut_span"`
	// 强力清理：段比率超过该值时启动"掰直"。
	AggressiveRatio float64 `json:"aggressive_ratio"`
	// 强力清理：垂距上限 = max(floor, 直线 * ratio)。
	PerpFloorKm float64 `json:"perp_floor_km"`
	PerpRatio   float64 `json:"perp_ratio"`
	// 强力清理：短段（< ShortSegmentKm）用更紧的阈值。
	PerpFloorShortKm float64 `json:"perp_floor_short_km"`
	PerpRatioShort   float64 `json:"perp_ratio_short"`
	ShortSegmentKm   float64 `json:"short_segment_km"`
	// 产出坐标小数位。
	PrecisionDigits int `json:"precision_digits"`
	// 超过该值的相邻点间距视为"大缺口"，仅报告、不静默替换。
	MaxGapKm float64 `json:"max_gap_km"`

	// ── validate 专用阈值 ──
	MaxStationOffsetWarnKm float64 `json:"max_station_offset_warn_km"`
	MaxStationOffsetErrKm  float64 `json:"max_station_offset_err_km"`
	MaxSegmentRatioWarn    float64 `json:"max_segment_ratio_warn"`
	MaxSegmentRatioErr     float64 `json:"max_segment_ratio_err"`
	MaxAdjacentGapKm       float64 `json:"max_adjacent_gap_km"`
	AnchorToleranceKm      float64 `json:"anchor_tolerance_km"`
	// 折返尖刺：转向角超过该值即视为折返。
	SpikeMaxTurnDeg float64 `json:"spike_max_turn_deg"`
	// 折返尖刺：若"返回点"距"出发点"小于该值，则是无歧义的原路返回（死胡同）。
	SpikeReturnToleranceKm float64 `json:"spike_return_tolerance_km"`
}

func defaultRoutePolicy() routePolicy {
	return routePolicy{
		BacktrackTolerance: 0.03,
		MinSegmentKm:       1,
		LoopThresholdKm:    0.3,
		// 2 而不是 5：死胡同尖刺常常只跨 2 个索引（走出→返回），
		// 真正的护栏是 LoopThresholdKm 与 LoopCutRatio，不是跨度。
		LoopMinSpan:       2,
		LoopMaxIterations: 10,
		LoopCutRatio:      2.0,
		LoopCutSpan:       8,
		AggressiveRatio:   1.5,
		PerpFloorKm:       6,
		PerpRatio:         0.15,
		PerpFloorShortKm:  3,
		PerpRatioShort:    0.2,
		ShortSegmentKm:    30,
		PrecisionDigits:   6,
		MaxGapKm:          30,

		MaxStationOffsetWarnKm: 0.5,
		MaxStationOffsetErrKm:  5,
		MaxSegmentRatioWarn:    1.5,
		MaxSegmentRatioErr:     2.0,
		MaxAdjacentGapKm:       30,
		AnchorToleranceKm:      0.05,

		SpikeMaxTurnDeg:        150,
		SpikeReturnToleranceKm: 0.25,
	}
}

// withDefaults 用调用方提供的值覆盖默认值（仅覆盖非零值）。
func (p routePolicy) withDefaults(base routePolicy) routePolicy {
	merge := func(dst *float64, src float64) {
		if src != 0 {
			*dst = src
		}
	}
	merge(&p.BacktrackTolerance, base.BacktrackTolerance)
	merge(&p.MinSegmentKm, base.MinSegmentKm)
	merge(&p.LoopThresholdKm, base.LoopThresholdKm)
	if base.LoopMinSpan != 0 {
		p.LoopMinSpan = base.LoopMinSpan
	}
	if base.LoopMaxIterations != 0 {
		p.LoopMaxIterations = base.LoopMaxIterations
	}
	merge(&p.LoopCutRatio, base.LoopCutRatio)
	if base.LoopCutSpan != 0 {
		p.LoopCutSpan = base.LoopCutSpan
	}
	merge(&p.AggressiveRatio, base.AggressiveRatio)
	merge(&p.PerpFloorKm, base.PerpFloorKm)
	merge(&p.PerpRatio, base.PerpRatio)
	merge(&p.PerpFloorShortKm, base.PerpFloorShortKm)
	merge(&p.PerpRatioShort, base.PerpRatioShort)
	merge(&p.ShortSegmentKm, base.ShortSegmentKm)
	if base.PrecisionDigits != 0 {
		p.PrecisionDigits = base.PrecisionDigits
	}
	merge(&p.MaxGapKm, base.MaxGapKm)
	merge(&p.MaxStationOffsetWarnKm, base.MaxStationOffsetWarnKm)
	merge(&p.MaxStationOffsetErrKm, base.MaxStationOffsetErrKm)
	merge(&p.MaxSegmentRatioWarn, base.MaxSegmentRatioWarn)
	merge(&p.MaxSegmentRatioErr, base.MaxSegmentRatioErr)
	merge(&p.MaxAdjacentGapKm, base.MaxAdjacentGapKm)
	merge(&p.AnchorToleranceKm, base.AnchorToleranceKm)
	merge(&p.SpikeMaxTurnDeg, base.SpikeMaxTurnDeg)
	merge(&p.SpikeReturnToleranceKm, base.SpikeReturnToleranceKm)
	return p
}

func resolveRoutePolicy(raw *routePolicy) routePolicy {
	base := defaultRoutePolicy()
	if raw == nil {
		return base
	}
	return raw.withDefaults(base)
}

// ── 核心算法（全部为纯函数） ──

// removeBacktracking 沿 a→b 轴做单调进度过滤。
//
// 注意 tolerance 的单位是**归一化进度**而不是公里：0.03 表示"允许回退整段
// 直线的 3%"。这是刻意保留的语义（与既有产物一致），但调用方需要知道
// 短段上的绝对容差比长段小。
func removeBacktracking(coords [][2]float64, a, b [2]float64, tolerance, minSegmentKm float64) [][2]float64 {
	if len(coords) <= 3 {
		return coords
	}
	if haversineKm(a, b) < minSegmentKm {
		return coords
	}
	out := make([][2]float64, 0, len(coords))
	out = append(out, coords[0])
	maxProgress := progressAlong(coords[0], a, b)
	for i := 1; i < len(coords)-1; i++ {
		p := progressAlong(coords[i], a, b)
		if p >= maxProgress-tolerance {
			out = append(out, coords[i])
			if p > maxProgress {
				maxProgress = p
			}
		}
	}
	out = append(out, coords[len(coords)-1])
	return out
}

// removeLoopsO2 剪掉"绕一圈回到原地"的闭合环。
//
// 复杂度 O(n²)，外层最多 maxIterations 轮，每轮剪掉第一处符合条件的环后重来。
// 对几千点级别的轨迹足够；再大应当先降采样。
func removeLoopsO2(coords [][2]float64, thresholdKm float64, minSpan, maxIterations int, cutRatio float64, cutSpan int) ([][2]float64, int) {
	if len(coords) <= minSpan {
		return coords, 0
	}
	result := append([][2]float64(nil), coords...)
	removed := 0
	for iter := 0; iter < maxIterations; iter++ {
		found := false
		for i := 0; i < len(result)-minSpan; i++ {
			for j := i + minSpan; j < len(result); j++ {
				if haversineKm(result[i], result[j]) >= thresholdKm {
					continue
				}
				loopLen := pathLengthKm(result[i : j+1])
				directLen := haversineKm(result[i], result[j])
				if loopLen <= directLen*cutRatio && (j-i) <= cutSpan {
					continue
				}
				merged := make([][2]float64, 0, len(result)-(j-i))
				merged = append(merged, result[:i+1]...)
				merged = append(merged, result[j:]...)
				result = merged
				removed++
				found = true
				break
			}
			if found {
				break
			}
		}
		if !found {
			break
		}
	}
	return result, removed
}

// aggressiveCleanup 把"绕远路"压回主线：只保留离 a→b 直线足够近、且仍在
// 向前走的点。
//
// 代价必须说清：这是**把真实弯道掰直**。高铁在山区确实会有超过 15km 的
// 垂直偏移，超过阈值就会被删。只在段比率仍然过高时才启用。
func aggressiveCleanup(coords [][2]float64, a, b [2]float64, p routePolicy) [][2]float64 {
	if len(coords) <= 3 {
		return coords
	}
	direct := haversineKm(a, b)
	if direct < 3 {
		return coords
	}
	maxPerp := math.Max(p.PerpFloorKm, direct*p.PerpRatio)
	if direct < p.ShortSegmentKm {
		maxPerp = math.Max(p.PerpFloorShortKm, direct*p.PerpRatioShort)
	}
	out := make([][2]float64, 0, len(coords))
	out = append(out, coords[0])
	maxProgress := progressAlong(coords[0], a, b)
	for i := 1; i < len(coords)-1; i++ {
		progress := progressAlong(coords[i], a, b)
		perp := pointToSegmentKm(coords[i], a, b)
		if progress >= maxProgress-0.01 && perp < maxPerp {
			out = append(out, coords[i])
			if progress > maxProgress {
				maxProgress = progress
			}
		}
	}
	out = append(out, coords[len(coords)-1])
	return out
}

// projectWaypoints 把每个站点投影到轨迹上，并**强制索引单调递增**。
//
// 单调化是必需的：站点吸附到最近的顶点时可能出现 idx 回退（后一个站
// 匹配到更靠前的顶点），那会导致切段时出现负长度区间。
func projectWaypoints(coords [][2]float64, waypoints []waypointInput) []int {
	indices := make([]int, len(waypoints))
	ordered := true
	last := -1
	for i, w := range waypoints {
		idx, _ := nearestVertexIndex(coords, w.LonLat)
		indices[i] = idx
		if idx < last {
			ordered = false
		}
		last = idx
	}
	if ordered {
		return indices
	}
	searchStart := 0
	for i, w := range waypoints {
		bestIndex, best := searchStart, math.Inf(1)
		for j := searchStart; j < len(coords); j++ {
			if d := haversineKm(coords[j], w.LonLat); d < best {
				best, bestIndex = d, j
			}
		}
		indices[i] = bestIndex
		searchStart = bestIndex
	}
	return indices
}

type segmentReport struct {
	Name         string  `json:"name,omitempty"`
	FromStation  string  `json:"from_station,omitempty"`
	ToStation    string  `json:"to_station,omitempty"`
	PointsBefore int     `json:"points_before"`
	PointsAfter  int     `json:"points_after"`
	Dropped      int     `json:"dropped"`
	LoopsRemoved int     `json:"loops_removed"`
	Straightened bool    `json:"straightened"`
	DirectKm     float64 `json:"direct_km"`
	LengthBefore float64 `json:"length_before_km"`
	LengthAfter  float64 `json:"length_after_km"`
	RatioBefore  float64 `json:"ratio_before"`
	RatioAfter   float64 `json:"ratio_after"`
	Degraded     string  `json:"degraded,omitempty"`
}

type routeRepairResult struct {
	ID           string          `json:"id"`
	Coords       [][2]float64    `json:"coords"`
	Segments     []segmentReport `json:"segments"`
	LoopsRemoved int             `json:"loops_removed"`
	MaxGapKm     float64         `json:"max_gap_km"`
	LengthKm     float64         `json:"length_km"`
}

func ratioOf(length, direct float64) float64 {
	if direct <= 0 {
		return 1
	}
	return length / direct
}

// repairRoute 对单条轨迹做修复，返回新坐标与逐段报告。
//
// **顺序很重要**：全局去环必须在按站点切段**之前**。
// 死胡同尖刺的顶点常常恰好落在站点索引上（hefu 的站点索引是
// [0 2 13 21 27 39 42 52 67 76 79]，而两个尖刺顶点正好是 67 和 76），
// 逐段处理时 prev/cur/next 跨越了段边界，段内视角看不到整条往返路径。
//
// 设计原则：**不静默降级**。任何回退（缺站点、大缺口）都写进报告，
// 而不是悄悄换数据。video-flow 原实现会在缺口 >30km 时悄悄换成手绘
// 折线且不更新头部统计，这是质量不可追溯的直接来源。
func repairRoute(route routeInput, p routePolicy) routeRepairResult {
	result := routeRepairResult{ID: route.ID}
	if len(route.Coords) < 2 {
		result.Coords = route.Coords
		result.Segments = []segmentReport{{Degraded: "too_few_points"}}
		return result
	}

	maxGap := 0.0
	for i := 1; i < len(route.Coords); i++ {
		if d := haversineKm(route.Coords[i-1], route.Coords[i]); d > maxGap {
			maxGap = d
		}
	}
	result.MaxGapKm = maxGap

	// ── 第一步：全局去环/去尖刺 ──
	// 防误删的护栏在 removeLoopsO2 内部：必须"回到几乎同一位置"，
	// 且环长显著大于直线距离。不做段比率闸门 —— 死胡同尖刺几乎不抬高比率。
	globalClean, globalLoops := removeLoopsO2(
		route.Coords, p.LoopThresholdKm, p.LoopMinSpan, p.LoopMaxIterations, p.LoopCutRatio, p.LoopCutSpan)
	result.LoopsRemoved = globalLoops

	// ── 无站点：整条当作一段 ──
	if len(route.Waypoints) < 2 {
		a, b := globalClean[0], globalClean[len(globalClean)-1]
		repaired, report := repairSegment(globalClean, a, b, p)
		report.PointsBefore = len(globalClean)
		report.PointsAfter = len(repaired)
		report.Dropped = len(globalClean) - len(repaired)
		if maxGap > p.MaxGapKm {
			report.Degraded = "route_gap"
		}
		result.Coords = roundCoords(repaired, p.PrecisionDigits)
		result.Segments = []segmentReport{report}
		result.LengthKm = pathLengthKm(result.Coords)
		return result
	}

	// ── 第二步：在清洗后的轨迹上投影站点，逐段处理 ──
	indices := projectWaypoints(globalClean, route.Waypoints)
	merged := make([][2]float64, 0, len(globalClean))
	for i := 0; i < len(route.Waypoints)-1; i++ {
		from, to := route.Waypoints[i], route.Waypoints[i+1]
		startIdx, endIdx := indices[i], indices[i+1]
		report := segmentReport{FromStation: from.Name, ToStation: to.Name}

		if endIdx <= startIdx {
			// 无法切段：诚实报告，退化为直线段。
			merged = append(merged, from.LonLat, to.LonLat)
			report.PointsAfter = 2
			report.Straightened = true
			report.Degraded = "segment_not_found"
			report.DirectKm = haversineKm(from.LonLat, to.LonLat)
			report.LengthAfter = report.DirectKm
			report.RatioAfter = 1
			result.Segments = append(result.Segments, report)
			continue
		}

		segment := append([][2]float64(nil), globalClean[startIdx:endIdx+1]...)
		repaired, segReport := repairSegment(segment, from.LonLat, to.LonLat, p)
		segReport.FromStation = from.Name
		segReport.ToStation = to.Name

		if len(merged) > 0 && len(repaired) > 0 {
			repaired = repaired[1:] // 段首与上一段末重复
		}
		merged = append(merged, repaired...)
		result.Segments = append(result.Segments, segReport)
	}

	if maxGap > p.MaxGapKm {
		result.Segments = append(result.Segments, segmentReport{Name: "route", Degraded: "route_gap", DirectKm: maxGap})
	}

	final := dedupeConsecutive(merged, 1e-7)
	result.Coords = roundCoords(final, p.PrecisionDigits)
	result.LengthKm = pathLengthKm(result.Coords)
	return result
}

// repairSegment 处理单个站间段：去环 → 反走回 → 必要时强力清理 → 端点锚定。
func repairSegment(segment [][2]float64, a, b [2]float64, p routePolicy) ([][2]float64, segmentReport) {
	report := segmentReport{}
	direct := haversineKm(a, b)
	report.DirectKm = direct
	report.PointsBefore = len(segment)
	report.LengthBefore = pathLengthKm(segment)
	report.RatioBefore = ratioOf(report.LengthBefore, direct)

	current := segment
	// 去环/去尖刺**始终执行**，不用段比率做闸门：
	// 死胡同尖刺（走出 4km 再原路返回）常常几乎不抬高段比率，
	// 用比率闸门会把它们全部漏掉 —— 原实现正是这样漏掉了 12 处近 180° 折返。
	// 防误删的护栏在 removeLoopsO2 内部（必须"回到几乎同一位置"且环长显著大于直线）。
	cleaned, removed := removeLoopsO2(current, p.LoopThresholdKm, p.LoopMinSpan, p.LoopMaxIterations, p.LoopCutRatio, p.LoopCutSpan)
	current = cleaned
	report.LoopsRemoved = removed
	current = removeBacktracking(current, a, b, p.BacktrackTolerance, p.MinSegmentKm)

	if ratioOf(pathLengthKm(current), direct) > p.AggressiveRatio {
		straightened := aggressiveCleanup(current, a, b, p)
		if len(straightened) < len(current) {
			report.Straightened = true
		}
		current = straightened
	}

	// 站点锚定：首末点替换为站点坐标。
	if len(current) >= 2 {
		current = append([][2]float64(nil), current...)
		current[0] = a
		current[len(current)-1] = b
	} else {
		current = [][2]float64{a, b}
	}

	report.PointsAfter = len(current)
	report.Dropped = report.PointsBefore - report.PointsAfter
	report.LengthAfter = pathLengthKm(current)
	report.RatioAfter = ratioOf(report.LengthAfter, direct)
	return current, report
}

// ── 校验 ──

type routeFinding struct {
	Code        string         `json:"code"`
	Stage       string         `json:"stage"`
	Severity    string         `json:"severity"`
	Message     string         `json:"message"`
	Remediation string         `json:"remediation"`
	Retryable   bool           `json:"retryable"`
	At          map[string]any `json:"at,omitempty"`
}

type routeValidateResult struct {
	ID       string         `json:"id"`
	OK       bool           `json:"ok"`
	Findings []routeFinding `json:"findings"`
	Summary  map[string]any `json:"summary"`
}

func finding(code, stage, severity, message, remediation string, at map[string]any) routeFinding {
	return routeFinding{Code: code, Stage: stage, Severity: severity, Message: message, Remediation: remediation, Retryable: true, At: at}
}

// spikeStats 统计折返尖刺：total = 转向角超过阈值的顶点数；
// unambiguous = 其中"返回点"距"出发点"很近的无歧义原路返回（死胡同）。
func spikeStats(coords [][2]float64, maxTurnDeg, returnToleranceKm float64) (total, unambiguous int, worstTurn float64) {
	for i := 1; i < len(coords)-1; i++ {
		deg := turnDegrees(coords[i-1], coords[i], coords[i+1])
		if deg <= maxTurnDeg {
			continue
		}
		total++
		if deg > worstTurn {
			worstTurn = deg
		}
		if haversineKm(coords[i-1], coords[i+1]) < returnToleranceKm {
			unambiguous++
		}
	}
	return total, unambiguous, worstTurn
}

// validateRoute 检查一条轨迹的不变量。
//
// 这些不变量就是"合格的路径生长视频"的可测试定义：
//   - 站点必须落在轨迹上（点到**线段**距离，不是顶点距离）
//   - 相邻点间距不能出现大缺口
//   - 站间段比率不能离谱（否则动画节奏会被拉长）
//   - 不能有折返尖刺（走出再原路返回，画面上是一条飞出去的线段）
//   - 不能有闭环（轨迹回到先前位置，地图上出现"结"）
//   - 首末点必须锚定在首末站
//
// 折返用**转向角**判断，不用弦投影进度：一条弯曲的线路（如皖南山区）
// 会有点在 A→B 弦上投影回退，但列车其实一直在向前开 —— 用弦投影会大量误报。
func validateRoute(route routeInput, p routePolicy) routeValidateResult {
	result := routeValidateResult{ID: route.ID, Findings: []routeFinding{}}
	result.Summary = map[string]any{
		"points":                len(route.Coords),
		"waypoints":             len(route.Waypoints),
		"length_km":             0.0,
		"max_station_offset_km": 0.0,
		"max_segment_ratio":     0.0,
		"max_adjacent_gap_km":   0.0,
	}

	if len(route.Coords) < 2 {
		result.Findings = append(result.Findings,
			finding("TOO_FEW_POINTS", "geometry", "error", "轨迹点数不足 2，无法成线", "补充坐标或检查数据源", map[string]any{"points": len(route.Coords)}))
		result.Summary["findings"] = len(result.Findings)
		return result
	}

	result.Summary["length_km"] = math.Round(pathLengthKm(route.Coords)*1000) / 1000

	// ① 相邻点间距
	maxGap, maxGapAt := 0.0, 0
	for i := 1; i < len(route.Coords); i++ {
		if d := haversineKm(route.Coords[i-1], route.Coords[i]); d > maxGap {
			maxGap, maxGapAt = d, i
		}
	}
	result.Summary["max_adjacent_gap_km"] = math.Round(maxGap*1000) / 1000
	if maxGap > p.MaxAdjacentGapKm {
		result.Findings = append(result.Findings,
			finding("MAX_ADJACENT_GAP", "geometry", "warn",
				fmt.Sprintf("相邻点最大间距 %.1f km 超过 %.1f km，可能是数据缺口", maxGap, p.MaxAdjacentGapKm),
				"检查上游数据源覆盖范围，或降低简化容差后重新生成",
				map[string]any{"index": maxGapAt, "km": math.Round(maxGap*100) / 100}))
	}

	// ② 站点到轨迹（线段）的距离
	maxOffset := 0.0
	for _, w := range route.Waypoints {
		d := distanceToPolylineKm(w.LonLat, route.Coords)
		if d > maxOffset {
			maxOffset = d
		}
		if d > p.MaxStationOffsetErrKm {
			result.Findings = append(result.Findings,
				finding("STATION_OFF_ROUTE", "anchor", "error",
					fmt.Sprintf("站点 %s 距轨迹 %.2f km", w.Name, d),
					"重新运行 pipeline.geo.route.repair，或修正站点坐标",
					map[string]any{"station": w.Name, "km": math.Round(d*100) / 100}))
		} else if d > p.MaxStationOffsetWarnKm {
			result.Findings = append(result.Findings,
				finding("STATION_OFF_ROUTE", "anchor", "warn",
					fmt.Sprintf("站点 %s 距轨迹 %.2f km", w.Name, d),
					"重新运行 pipeline.geo.route.repair，或修正站点坐标",
					map[string]any{"station": w.Name, "km": math.Round(d*100) / 100}))
		}
	}
	result.Summary["max_station_offset_km"] = math.Round(maxOffset*1000) / 1000

	// ③/④ 逐段检查比率与反走回
	maxRatio := 0.0
	if len(route.Waypoints) >= 2 {
		indices := projectWaypoints(route.Coords, route.Waypoints)
		for i := 0; i < len(route.Waypoints)-1; i++ {
			from, to := route.Waypoints[i], route.Waypoints[i+1]
			startIdx, endIdx := indices[i], indices[i+1]
			if endIdx <= startIdx {
				result.Findings = append(result.Findings,
					finding("SEGMENT_NOT_FOUND", "geometry", "error",
						fmt.Sprintf("无法在轨迹上切出 %s→%s 段", from.Name, to.Name),
						"检查站点顺序是否与轨迹行进方向一致",
						map[string]any{"from": from.Name, "to": to.Name}))
				continue
			}
			segment := route.Coords[startIdx : endIdx+1]
			direct := haversineKm(from.LonLat, to.LonLat)
			ratio := ratioOf(pathLengthKm(segment), direct)
			if ratio > maxRatio {
				maxRatio = ratio
			}
			at := map[string]any{
				"from": from.Name, "to": to.Name,
				"ratio": math.Round(ratio*100) / 100,
			}
			if ratio > p.MaxSegmentRatioErr {
				result.Findings = append(result.Findings,
					finding("SEGMENT_RATIO_EXCESSIVE", "geometry", "error",
						fmt.Sprintf("%s→%s 段比率 %.2f 过高，动画节奏会被拉长", from.Name, to.Name, ratio),
						"运行 pipeline.geo.route.repair 做反走回消除与强力清理", at))
			} else if ratio > p.MaxSegmentRatioWarn {
				result.Findings = append(result.Findings,
					finding("SEGMENT_RATIO_HIGH", "geometry", "warn",
						fmt.Sprintf("%s→%s 段比率 %.2f 偏高", from.Name, to.Name, ratio),
						"检查该段是否为绕行，必要时运行 pipeline.geo.route.repair", at))
			}

		}
	}

	result.Summary["max_segment_ratio"] = math.Round(maxRatio*100) / 100
	result.Summary["length_km"] = math.Round(pathLengthKm(route.Coords)*1000) / 1000

	// ④ 复访（闭环）：轨迹绕一圈回到之前走过的地方 —— 地图上会出现一个"结"。
	revisits := 0
	for i := 0; i < len(route.Coords); i++ {
		for j := i + p.LoopMinSpan; j < len(route.Coords); j++ {
			if haversineKm(route.Coords[i], route.Coords[j]) < p.LoopThresholdKm {
				revisits++
			}
		}
	}
	result.Summary["revisited_pairs"] = revisits
	if revisits > 0 {
		result.Findings = append(result.Findings,
			finding("LOOP_DETECTED", "geometry", "error",
				fmt.Sprintf("轨迹回到先前位置 %d 次，地图上会出现闭合的结", revisits),
				"运行 pipeline.geo.route.repair（全局去环在切段之前执行）",
				map[string]any{"pairs": revisits}))
	}

	// ⑤ 折返尖刺（走出再原路返回 —— 画面上就是一条飞出去的线段）
	spikeTotal, spikeUnambiguous, worstTurn := spikeStats(route.Coords, p.SpikeMaxTurnDeg, p.SpikeReturnToleranceKm)
	result.Summary["spike_vertices"] = spikeTotal
	result.Summary["unambiguous_spikes"] = spikeUnambiguous
	result.Summary["worst_turn_deg"] = math.Round(worstTurn*10) / 10
	if spikeUnambiguous > 0 {
		result.Findings = append(result.Findings,
			finding("SPIKE_VISIBLE", "geometry", "error",
				fmt.Sprintf("检测到 %d 处原路返回尖刺（转向角最大 %.0f°，返回点距出发点 <%.0f m），画面会看到线段飞出去又折回", spikeUnambiguous, worstTurn, p.SpikeReturnToleranceKm*1000),
				"运行 pipeline.geo.route.repair（全局去环在切段之前执行）",
				map[string]any{"count": spikeUnambiguous, "worst_turn_deg": math.Round(worstTurn*10) / 10}))
	} else if spikeTotal > 0 {
		result.Findings = append(result.Findings,
			finding("SPIKE_SHARP_TURN", "geometry", "warn",
				fmt.Sprintf("检测到 %d 处急转折返（转向角最大 %.0f°），可能是真实山区走线，也可能是数据缺口造成的假象，建议人工复核", spikeTotal, worstTurn),
				"对照卫星影像确认该处是否为真实走线；若否，检查上游数据源在该段是否缺失",
				map[string]any{"count": spikeTotal, "worst_turn_deg": math.Round(worstTurn*10) / 10}))
	}

	// ⑥ 首末点锚定
	if len(route.Waypoints) >= 2 {
		head := haversineKm(route.Coords[0], route.Waypoints[0].LonLat)
		tail := haversineKm(route.Coords[len(route.Coords)-1], route.Waypoints[len(route.Waypoints)-1].LonLat)
		if head > p.AnchorToleranceKm || tail > p.AnchorToleranceKm {
			result.Findings = append(result.Findings,
				finding("ENDPOINT_NOT_ANCHORED", "anchor", "warn",
					fmt.Sprintf("首点距首站 %.3f km、末点距末站 %.3f km，未精确锚定", head, tail),
					"运行 pipeline.geo.route.repair（端点替换）",
					map[string]any{"head_km": math.Round(head*1000) / 1000, "tail_km": math.Round(tail*1000) / 1000}))
		}
	}

	result.OK = true
	for _, f := range result.Findings {
		if f.Severity == "error" {
			result.OK = false
			break
		}
	}
	result.Summary["findings"] = len(result.Findings)
	return result
}

// ── 能力入口 ──

func routeRepair(in requestInput) (any, *himindjsonrpc.Error) {
	if len(in.Routes) == 0 {
		return nil, himindjsonrpc.InvalidParams("routes is required and must not be empty")
	}
	policy := resolveRoutePolicy(in.Policy)
	results := make([]routeRepairResult, 0, len(in.Routes))
	totalBefore, totalAfter, degradations, loops := 0, 0, 0, 0
	for _, route := range in.Routes {
		if len(route.Coords) == 0 {
			return nil, himindjsonrpc.InvalidParams(fmt.Sprintf("route %q has no coords", route.ID))
		}
		item := repairRoute(route, policy)
		totalBefore += len(route.Coords)
		totalAfter += len(item.Coords)
		loops += item.LoopsRemoved
		for _, seg := range item.Segments {
			if seg.Degraded != "" {
				degradations++
			}
		}
		results = append(results, item)
	}
	return map[string]any{
		"routes": results,
		"summary": map[string]any{
			"routes":           len(results),
			"points_in":        totalBefore,
			"points_out":       totalAfter,
			"loops_removed":    loops,
			"degradations":     degradations,
			"deterministic":    true,
			"idempotent_after": "第二次调用应逐点不变（几何不动点）",
			"policy":           policy,
		},
		"next": "调用 media.workbench.route.validate 确认不变量全部通过",
	}, nil
}

func routeValidate(in requestInput) (any, *himindjsonrpc.Error) {
	if len(in.Routes) == 0 {
		return nil, himindjsonrpc.InvalidParams("routes is required and must not be empty")
	}
	policy := resolveRoutePolicy(in.Policy)
	results := make([]routeValidateResult, 0, len(in.Routes))
	ok := true
	errors, warns := 0, 0
	for _, route := range in.Routes {
		item := validateRoute(route, policy)
		if !item.OK {
			ok = false
		}
		for _, f := range item.Findings {
			switch f.Severity {
			case "error":
				errors++
			case "warn":
				warns++
			}
		}
		results = append(results, item)
	}
	return map[string]any{
		"ok":      ok,
		"routes":  results,
		"summary": map[string]any{"routes": len(results), "errors": errors, "warnings": warns},
		"policy":  policy,
	}, nil
}

// sortFindings 仅用于测试断言的稳定排序。
func sortFindings(f []routeFinding) {
	sort.SliceStable(f, func(i, j int) bool {
		if f[i].Severity != f[j].Severity {
			return f[i].Severity < f[j].Severity
		}
		return f[i].Code < f[j].Code
	})
}
