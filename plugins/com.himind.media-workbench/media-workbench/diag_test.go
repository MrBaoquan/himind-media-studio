package main

import "testing"

// 本文件记录"修复能力对示例数据集的实测效果"。
// 它同时是回归测试：如果哪天 repair 退化，尖刺数会回升，这里会失败。

// spikeStatsOf 返回 (总尖刺数, 无歧义尖刺数 = 返回点距出发点 <250m)。
func spikeStatsOf(coords [][2]float64) (int, int) {
	total, unambiguous := 0, 0
	for i := 1; i < len(coords)-1; i++ {
		if turnDegrees(coords[i-1], coords[i], coords[i+1]) > 150 {
			total++
			if haversineKm(coords[i-1], coords[i+1]) < 0.25 {
				unambiguous++
			}
		}
	}
	return total, unambiguous
}

func TestRepairRemovesDeadEndSpikesFromDataset(t *testing.T) {
	dataset, err := loadTransitDataset()
	if err != nil {
		t.Fatalf("dataset: %v", err)
	}
	policy := defaultRoutePolicy()

	totalIn, totalOut := 0, 0
	unambiguousIn, unambiguousOut := 0, 0
	for _, line := range dataset.Lines {
		in := line.asRouteInput()
		if len(in.Coords) < 3 {
			continue
		}
		bi, bia := spikeStatsOf(in.Coords)
		repaired := repairRoute(in, policy)
		ai, aia := spikeStatsOf(repaired.Coords)
		totalIn += bi
		totalOut += ai
		unambiguousIn += bia
		unambiguousOut += aia
		if bia > 0 && aia != 0 {
			t.Errorf("%s: repair left %d unambiguous dead-end spikes (had %d)", line.ID, aia, bia)
		}
	}

	t.Logf("dead-end spikes: unambiguous %d -> %d, total %d -> %d", unambiguousIn, unambiguousOut, totalIn, totalOut)

	// 数据集本身必须已经不含无歧义死胡同 —— 它是用本扩展的能力生成的。
	if unambiguousIn != 0 {
		t.Errorf("shipped dataset still contains %d unambiguous dead-end spikes; run HIMIND_REGEN_GEO=1 to regenerate", unambiguousIn)
	}
}
