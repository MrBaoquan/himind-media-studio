// techniques/race.js —— 时序排名竞速（bar chart race）。
//
// 关键设计：**整数年稳定快照**。
// 直接用浮点年份算排名会因为微小数值差异导致名次抖动；
// 这里只在整数年计算排名，帧间用平滑插值过渡。

/** 构建整数年快照：每年每个条目的名次（0-based）与数值。 */
export function buildRaceSeries(items, options = {}) {
  const {start, end} = options;
  const years = [];
  for (let year = start; year <= end; year++) years.push(year);

  const ranksByYear = {};
  const valuesByYear = {};

  for (const year of years) {
    const rows = items
      .map((item) => ({
        id: item.id,
        value: Number((item.valueByYear || {})[year] ?? (item.valueByYear || {})[String(year)] ?? 0),
      }))
      .sort((a, b) => b.value - a.value);

    const ranks = {};
    const values = {};
    rows.forEach((row, index) => {
      ranks[row.id] = index;
      values[row.id] = row.value;
    });
    ranksByYear[year] = ranks;
    valuesByYear[year] = values;
  }

  return {years, ranksByYear, valuesByYear};
}

const smoothstep = (t) => t * t * (3 - 2 * t);

/**
 * 某一帧的竞速状态。
 * rank 是插值后的浮点名次（用于纵向滑动），value 是线性插值。
 */
export function raceStateAt(series, yearFloat, items) {
  const years = series.years;
  if (!years.length) return [];
  const start = years[0];
  const end = years[years.length - 1];
  const clamped = Math.max(start, Math.min(end, yearFloat));
  const y0 = Math.floor(clamped);
  const y1 = Math.min(end, y0 + 1);
  const t = clamped - y0;
  const eased = smoothstep(t);

  return items.map((item) => {
    const r0 = series.ranksByYear[y0]?.[item.id] ?? items.length;
    const r1 = series.ranksByYear[y1]?.[item.id] ?? r0;
    const v0 = series.valuesByYear[y0]?.[item.id] ?? 0;
    const v1 = series.valuesByYear[y1]?.[item.id] ?? v0;
    return {
      id: item.id,
      name: item.name ?? item.id,
      color: item.color,
      rank: r0 + (r1 - r0) * eased,
      value: v0 + (v1 - v0) * t,
      visibleRank: Math.min(r0, r1),
    };
  });
}
