package main

import (
	"math"
	"testing"
)

// route_test.go — 技法资产 pipeline.geo.route 的契约测试。
//
// 这些断言就是"合格的路径生长视频"的可测试定义。没有它们，
// "布局质量"只是手感；有了它们，质量才可复现、可累积。

func approx(t *testing.T, label string, got, want, tolerance float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Fatalf("%s: got %.6f want %.6f (±%.6f)", label, got, want, tolerance)
	}
}

// ── 几何原语 ──

func TestPointToSegmentUsesSegmentNotVertex(t *testing.T) {
	// 东西向长段：从 (0,0) 到 (0.5,0)。在纬度 0 处 0.5 度经度 ≈ 55.66 km。
	a := [2]float64{0, 0}
	b := [2]float64{0.5, 0}

	// 端点上的点距离应为 0
	approx(t, "on-endpoint", pointToSegmentKm(a, a, b), 0, 0.001)

	// 段中点，垂直偏移 0.01 度经度 ≈ 1.113 km
	// 这正是"顶点比较"会算错的情形：中点不是顶点，旧实现会给出 ~27.8 km。
	mid := [2]float64{0.25, 0}
	approx(t, "perp-at-midpoint", pointToSegmentKm([2]float64{0.25, 0.01}, a, b), 1.113, 0.02)

	// 中点正上方 0 距离
	approx(t, "exactly-on-midpoint", pointToSegmentKm(mid, a, b), 0, 0.001)

	// 超出端点的点应回退到端点距离
	approx(t, "beyond-endpoint", pointToSegmentKm([2]float64{0.6, 0}, a, b), 11.132, 0.05)

	// 对照：旧实现（只比顶点）在同一输入上会给出约 27.83 km，
	// 说明这个修正确实消除了假阳性。
	vertexOnly := math.Min(haversineKm([2]float64{0.25, 0.01}, a), haversineKm([2]float64{0.25, 0.01}, b))
	if vertexOnly < 20 {
		t.Fatalf("expected vertex-only distance to be large (demonstrating the old bug), got %.3f", vertexOnly)
	}
}

func TestProgressAlongIsMonotonicOnStraightLine(t *testing.T) {
	a := [2]float64{0, 0}
	b := [2]float64{1, 0}
	previous := math.Inf(-1)
	for i := 0; i <= 10; i++ {
		progress := progressAlong([2]float64{float64(i) / 10, 0}, a, b)
		if progress < previous-1e-9 {
			t.Fatalf("progress regressed at i=%d: %.6f < %.6f", i, progress, previous)
		}
		previous = progress
	}
	approx(t, "progress-at-b", progressAlong(b, a, b), 1, 1e-9)
}

func TestDedupeConsecutiveRemovesZeroLengthSegments(t *testing.T) {
	in := [][2]float64{{0, 0}, {0, 0}, {0.001, 0}, {0.001, 0}, {0.002, 0}}
	out := dedupeConsecutive(in, 1e-7)
	if len(out) != 3 {
		t.Fatalf("expected 3 points, got %d: %v", len(out), out)
	}
}

// ── 反走回消除 ──

func TestRemoveBacktrackingDropsReturningSegment(t *testing.T) {
	a := [2]float64{0, 0}
	b := [2]float64{1, 0}
	// 图搜索在无向图上的典型产物：走出去 0.9 又退回 0.6/0.5，再前进到终点。
	coords := [][2]float64{
		{0, 0}, {0.3, 0}, {0.6, 0}, {0.9, 0}, {0.6, 0}, {0.5, 0}, {0.8, 0}, {1, 0},
	}
	out := removeBacktracking(coords, a, b, 0.03, 1)

	if len(out) != 5 {
		t.Fatalf("expected the returning segment to be dropped (5 points), got %d: %v", len(out), out)
	}
	// 输出必须进度单调
	previous := math.Inf(-1)
	for _, c := range out {
		progress := progressAlong(c, a, b)
		if progress < previous-1e-9 {
			t.Fatalf("output is not monotonic: %.4f after %.4f", progress, previous)
		}
		previous = progress
	}
	// 端点必须保留
	if out[0] != a || out[len(out)-1] != b {
		t.Fatalf("endpoints must be preserved, got %v .. %v", out[0], out[len(out)-1])
	}
}

func TestRemoveBacktrackingIsIdempotent(t *testing.T) {
	a := [2]float64{0, 0}
	b := [2]float64{1, 0}
	coords := [][2]float64{
		{0, 0}, {0.3, 0}, {0.6, 0}, {0.9, 0}, {0.6, 0}, {0.5, 0}, {0.8, 0}, {1, 0},
	}
	once := removeBacktracking(coords, a, b, 0.03, 1)
	twice := removeBacktracking(once, a, b, 0.03, 1)
	if len(once) != len(twice) {
		t.Fatalf("not idempotent: %d then %d points", len(once), len(twice))
	}
	for i := range once {
		if once[i] != twice[i] {
			t.Fatalf("point %d changed on second pass: %v -> %v", i, once[i], twice[i])
		}
	}
}

func TestRemoveBacktrackingKeepsShortSegmentsUntouched(t *testing.T) {
	// 极短段（< MinSegmentKm）不处理，避免在小尺度上误删
	a := [2]float64{0, 0}
	b := [2]float64{0.001, 0}
	coords := [][2]float64{{0, 0}, {0.0008, 0}, {0.0002, 0}, {0.001, 0}}
	out := removeBacktracking(coords, a, b, 0.03, 1)
	if len(out) != len(coords) {
		t.Fatalf("short segment should be untouched, got %d vs %d", len(out), len(coords))
	}
}

// ── 去环 ──

func TestRemoveLoopsCutsClosedLoop(t *testing.T) {
	// 绕一圈回到原点：索引跨度 6 > minSpan 5，距离 0 < threshold
	coords := [][2]float64{
		{0, 0}, {0.01, 0}, {0.02, 0.01}, {0.01, 0.02}, {0, 0.02}, {0, 0.01}, {0, 0}, {0.02, 0}, {0.04, 0},
	}
	out, removed := removeLoopsO2(coords, 0.3, 5, 10, 2.0, 8)
	if removed == 0 {
		t.Fatal("expected at least one loop to be removed")
	}
	if len(out) >= len(coords) {
		t.Fatalf("expected fewer points after loop removal, got %d vs %d", len(out), len(coords))
	}
}

func TestRemoveLoopsLeavesCleanPolylineAlone(t *testing.T) {
	coords := [][2]float64{{0, 0}, {0.1, 0}, {0.2, 0}, {0.3, 0}, {0.4, 0}, {0.5, 0}, {0.6, 0}, {0.7, 0}}
	out, removed := removeLoopsO2(coords, 0.3, 5, 10, 2.0, 8)
	if removed != 0 || len(out) != len(coords) {
		t.Fatalf("clean polyline should be untouched, removed=%d len=%d", removed, len(out))
	}
}

// ── 站点锚定 ──

func TestRepairAnchorsEndpointsToStations(t *testing.T) {
	stationA := waypointInput{Name: "甲站", LonLat: [2]float64{117.0, 33.0}}
	stationB := waypointInput{Name: "乙站", LonLat: [2]float64{117.5, 33.5}}
	// 原始几何故意不精确经过站点
	coords := [][2]float64{
		{117.0, 33.0}, {117.05, 33.02}, {117.1, 33.1}, {117.2, 33.2},
		{117.3, 33.3}, {117.4, 33.4}, {117.5, 33.5},
	}
	policy := defaultRoutePolicy()
	policy.PrecisionDigits = 9
	result := repairRoute(routeInput{ID: "t", Coords: coords, Waypoints: []waypointInput{stationA, stationB}}, policy)

	head := haversineKm(result.Coords[0], stationA.LonLat)
	tail := haversineKm(result.Coords[len(result.Coords)-1], stationB.LonLat)
	if head > 1e-6 || tail > 1e-6 {
		t.Fatalf("endpoints not anchored: head=%.9f km tail=%.9f km", head, tail)
	}
}

// ── 修复幂等性（几何不动点） ──

func TestRepairIsGeometryIdempotent(t *testing.T) {
	dataset, err := loadTransitDataset()
	if err != nil {
		t.Skipf("example dataset unavailable: %v", err)
	}
	policy := defaultRoutePolicy()
	for _, line := range dataset.Lines {
		if len(line.RepairedCoords) < 2 || len(line.Stations) < 2 {
			continue
		}
		first := repairRoute(line.asRouteInput(), policy)
		second := repairRoute(routeInput{ID: line.ID, Coords: first.Coords, Waypoints: line.asRouteInput().Waypoints}, policy)
		if len(first.Coords) != len(second.Coords) {
			t.Fatalf("%s: repair not idempotent, %d -> %d points", line.ID, len(first.Coords), len(second.Coords))
		}
		for i := range first.Coords {
			if first.Coords[i] != second.Coords[i] {
				t.Fatalf("%s: point %d changed on second repair: %v -> %v", line.ID, i, first.Coords[i], second.Coords[i])
			}
		}
	}
}

// ── 真实数据上的不变量（headline 测试） ──

func TestExampleDatasetPassesQualityInvariants(t *testing.T) {
	dataset, err := loadTransitDataset()
	if err != nil {
		t.Fatalf("example dataset must ship with the plugin: %v", err)
	}
	if len(dataset.Lines) == 0 {
		t.Fatal("example dataset is empty")
	}

	policy := defaultRoutePolicy()
	checked, total := 0, 0
	for _, line := range dataset.Lines {
		total++
		if len(line.RepairedCoords) < 2 {
			continue // 未提供修复坐标的线路不参与几何断言
		}
		checked++
		result := validateRoute(line.asRouteInput(), policy)
		if !result.OK {
			t.Errorf("%s (%s) failed invariants:", line.ID, line.Name)
			for _, f := range result.Findings {
				t.Errorf("  [%s] %s %s", f.Severity, f.Code, f.Message)
			}
			continue
		}
		offset, _ := result.Summary["max_station_offset_km"].(float64)
		ratio, _ := result.Summary["max_segment_ratio"].(float64)
		spikes, _ := result.Summary["unambiguous_spikes"].(int)
		revisits, _ := result.Summary["revisited_pairs"].(int)
		if offset > 0.5 {
			t.Errorf("%s: max station offset %.3f km exceeds 0.5 km", line.ID, offset)
		}
		if ratio > 1.5 {
			t.Errorf("%s: max segment ratio %.3f exceeds 1.5", line.ID, ratio)
		}
		if spikes != 0 {
			t.Errorf("%s: %d visible out-and-back spikes remain", line.ID, spikes)
		}
		if revisits != 0 {
			t.Errorf("%s: %d revisited point pairs (loop) remain", line.ID, revisits)
		}
	}
	t.Logf("validated %d/%d lines from the example dataset", checked, total)
	if checked < 10 {
		t.Fatalf("expected at least 10 lines with repaired geometry, got %d", checked)
	}
}

func TestValidateReportsStationDriftAsFinding(t *testing.T) {
	stationA := waypointInput{Name: "甲站", LonLat: [2]float64{117.0, 33.0}}
	stationB := waypointInput{Name: "乙站", LonLat: [2]float64{117.5, 33.5}}
	coords := [][2]float64{{117.0, 33.0}, {117.2, 33.2}, {117.5, 33.5}}
	// 把第二个站点挪走 3 km，应当被报出来
	drifted := waypointInput{Name: "乙站", LonLat: [2]float64{117.62, 33.62}}

	result := validateRoute(routeInput{ID: "t", Coords: coords, Waypoints: []waypointInput{stationA, drifted}}, defaultRoutePolicy())
	if result.OK {
		t.Fatal("expected validation to fail for a drifted station")
	}
	found := false
	for _, f := range result.Findings {
		if f.Code == "STATION_OFF_ROUTE" && f.Severity == "error" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected STATION_OFF_ROUTE error, got %+v", result.Findings)
	}
	_ = stationB
}

func TestValidateRejectsTooFewPoints(t *testing.T) {
	result := validateRoute(routeInput{ID: "t", Coords: [][2]float64{{117, 33}}}, defaultRoutePolicy())
	if result.OK {
		t.Fatal("single-point route must not validate")
	}
	if result.Findings[0].Code != "TOO_FEW_POINTS" {
		t.Fatalf("expected TOO_FEW_POINTS, got %s", result.Findings[0].Code)
	}
}

// ── 折返尖刺（走出再原路返回） ──

func TestValidateDetectsOutAndBackSpike(t *testing.T) {
	a := waypointInput{Name: "甲站", LonLat: [2]float64{117.0, 33.0}}
	b := waypointInput{Name: "乙站", LonLat: [2]float64{117.4, 33.0}}
	// 在一条笔直的线上，中段"飞出去 5.5km 再原路返回"（Dijkstra 死胡同的典型形态）
	coords := [][2]float64{
		{117.0, 33.0}, {117.1, 33.0}, {117.15, 33.0},
		{117.15, 33.05}, {117.15, 33.0},
		{117.3, 33.0}, {117.4, 33.0},
	}
	result := validateRoute(routeInput{ID: "spike", Coords: coords, Waypoints: []waypointInput{a, b}}, defaultRoutePolicy())
	if result.OK {
		t.Fatal("expected an out-and-back spike to be reported as an error")
	}
	found := false
	for _, f := range result.Findings {
		if f.Code == "SPIKE_VISIBLE" && f.Severity == "error" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected SPIKE_VISIBLE, got %+v", result.Findings)
	}
}

func TestRepairRemovesOutAndBackSpike(t *testing.T) {
	a := waypointInput{Name: "甲站", LonLat: [2]float64{117.0, 33.0}}
	b := waypointInput{Name: "乙站", LonLat: [2]float64{117.4, 33.0}}
	coords := [][2]float64{
		{117.0, 33.0}, {117.1, 33.0}, {117.15, 33.0},
		{117.15, 33.05}, {117.15, 33.0},
		{117.3, 33.0}, {117.4, 33.0},
	}
	policy := defaultRoutePolicy()
	repaired := repairRoute(routeInput{ID: "spike", Coords: coords, Waypoints: []waypointInput{a, b}}, policy)
	if len(repaired.Coords) >= len(coords) {
		t.Fatalf("expected the spike to be removed, got %d points (was %d)", len(repaired.Coords), len(coords))
	}
	for _, pt := range repaired.Coords {
		if pt[1] > 33.0001 {
			t.Fatalf("spike apex survived repair: %v", pt)
		}
	}
	after := validateRoute(routeInput{ID: "spike", Coords: repaired.Coords, Waypoints: []waypointInput{a, b}}, policy)
	if !after.OK {
		t.Fatalf("repaired route must pass invariants, got %+v", after.Findings)
	}
}
