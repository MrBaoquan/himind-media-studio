package main

import "math"

// geo.go — 地理几何原语。
//
// 这一层是"技法资产"的地基：投影、距离、降采样都必须是**纯函数**，
// 因为下游的帧级渲染要求同一输入产生同一输出（Remotion 的帧号决定论）。

const (
	// kmPerDegLat 是纬度 1 度的近似公里数（WGS84 中纬度均值）。
	kmPerDegLat = 111.32
	earthRadius = 6371.0088
)

// lonLatKm 把经纬度点投影到以 ref 为原点的局部平面（单位 km）。
//
// 关键：经度方向按 cos(lat) 缩放。忽略这一步会让东西向与南北向的
// "1 度"代表不同的实际距离（在北纬 31 度差约 15%），于是同一个容差
// 在东西向段和南北向段上表现得不一样。
func lonLatKm(p, ref [2]float64) [2]float64 {
	kmPerDegLon := kmPerDegLat * math.Cos(ref[1]*math.Pi/180)
	return [2]float64{(p[0] - ref[0]) * kmPerDegLon, (p[1] - ref[1]) * kmPerDegLat}
}

// haversineKm 返回两点的大圆距离（公里）。
func haversineKm(a, b [2]float64) float64 {
	lat1 := a[1] * math.Pi / 180
	lat2 := b[1] * math.Pi / 180
	dLat := lat2 - lat1
	dLon := (b[0] - a[0]) * math.Pi / 180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadius * math.Asin(math.Min(1, math.Sqrt(h)))
}

// pathLengthKm 返回折线总长（公里）。
func pathLengthKm(coords [][2]float64) float64 {
	total := 0.0
	for i := 1; i < len(coords); i++ {
		total += haversineKm(coords[i-1], coords[i])
	}
	return total
}

// midpoint 返回两点中点。
func midpoint(a, b [2]float64) [2]float64 {
	return [2]float64{(a[0] + b[0]) / 2, (a[1] + b[1]) / 2}
}

// pointToSegmentKm 返回点到线段的最短距离（公里）。
//
// 这是 validate 的核心度量。此前 video-flow 的校验器只比较折线**顶点**，
// 于是：站点落在两个相距 24km 的顶点之间时会被算成 12km（假阳性），
// 而只要站点恰好等于某个顶点就永远是 0（假阴性）。这里改成真正的
// 点到线段距离，并在段中点建立局部平面以保证左右对称。
func pointToSegmentKm(p, a, b [2]float64) float64 {
	ref := midpoint(a, b)
	pp := lonLatKm(p, ref)
	aa := lonLatKm(a, ref)
	bb := lonLatKm(b, ref)
	dx := bb[0] - aa[0]
	dy := bb[1] - aa[1]
	norm2 := dx*dx + dy*dy
	if norm2 == 0 {
		return math.Hypot(pp[0]-aa[0], pp[1]-aa[1])
	}
	t := ((pp[0]-aa[0])*dx + (pp[1]-aa[1])*dy) / norm2
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(pp[0]-(aa[0]+t*dx), pp[1]-(aa[1]+t*dy))
}

// distanceToPolylineKm 返回点到整条折线的最短距离（公里）。
func distanceToPolylineKm(p [2]float64, coords [][2]float64) float64 {
	if len(coords) == 0 {
		return math.Inf(1)
	}
	if len(coords) == 1 {
		return haversineKm(p, coords[0])
	}
	best := math.Inf(1)
	for i := 1; i < len(coords); i++ {
		if d := pointToSegmentKm(p, coords[i-1], coords[i]); d < best {
			best = d
		}
	}
	return best
}

// nearestVertexIndex 返回折线上距 p 最近的顶点下标与距离。
func nearestVertexIndex(coords [][2]float64, p [2]float64) (int, float64) {
	best, bestIndex := math.Inf(1), 0
	for i, c := range coords {
		if d := haversineKm(c, p); d < best {
			best, bestIndex = d, i
		}
	}
	return bestIndex, best
}

// progressAlong 返回 p 在 a→b 轴上的归一化投影进度（0..1，可越界）。
//
// 反走回消除就建立在这个量上：单调递增 = 一直在往前走。
func progressAlong(p, a, b [2]float64) float64 {
	ref := midpoint(a, b)
	pp := lonLatKm(p, ref)
	aa := lonLatKm(a, ref)
	bb := lonLatKm(b, ref)
	dx := bb[0] - aa[0]
	dy := bb[1] - aa[1]
	norm2 := dx*dx + dy*dy
	if norm2 == 0 {
		return 0
	}
	return ((pp[0]-aa[0])*dx + (pp[1]-aa[1])*dy) / norm2
}

// dedupeConsecutive 删除相邻的重复点（阈值单位为度）。
func dedupeConsecutive(coords [][2]float64, epsilon float64) [][2]float64 {
	if len(coords) == 0 {
		return coords
	}
	out := [][2]float64{coords[0]}
	for i := 1; i < len(coords); i++ {
		prev := out[len(out)-1]
		if math.Abs(coords[i][0]-prev[0]) <= epsilon && math.Abs(coords[i][1]-prev[1]) <= epsilon {
			continue
		}
		out = append(out, coords[i])
	}
	return out
}

// roundCoords 按小数位四舍五入，控制产物体积。
func roundCoords(coords [][2]float64, digits int) [][2]float64 {
	if digits < 0 || digits > 12 {
		digits = 6
	}
	factor := math.Pow10(digits)
	out := make([][2]float64, 0, len(coords))
	for _, c := range coords {
		out = append(out, [2]float64{
			math.Round(c[0]*factor) / factor,
			math.Round(c[1]*factor) / factor,
		})
	}
	return out
}

// turnDegrees 返回在 cur 处的转向角度：0 = 直线前进，180 = 完全折返。
//
// 这是判断"折返尖刺"的正确度量。用弦投影进度判断反走回会产生假阳性 ——
// 一条弯曲的线路（如皖南山区的绩溪北→歙县北）会有点在弦上投影回退，
// 但列车其实一直在向前开。转向角只看局部几何，不受整段弯曲影响。
func turnDegrees(prev, cur, next [2]float64) float64 {
	a := lonLatKm(prev, cur)
	b := lonLatKm(next, cur)
	na := math.Hypot(a[0], a[1])
	nb := math.Hypot(b[0], b[1])
	if na < 1e-9 || nb < 1e-9 {
		return 0
	}
	cosine := (a[0]*b[0] + a[1]*b[1]) / (na * nb)
	return 180 - math.Acos(math.Max(-1, math.Min(1, cosine)))*180/math.Pi
}
