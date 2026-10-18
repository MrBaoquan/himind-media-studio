// techniques/labels.js — 标签避让求解器（solver）
//
// 保证（这些是"布局质量"的可测试定义）：
//   no-overlap           放置后任意两标签矩形不相交
//   in-bounds            全部落在 clamp 区域内
//   order-stable         同一输入 → 同一顺序 → 同一输出
//   frame-deterministic  纯函数，不依赖帧号以外的任何状态
//
// 实现是"候选位 + 贪心打分"：每个标签尝试多个候选位置，选综合代价最小的。
// 代价 = 与已放置标签的重叠面积 × 权重 + 越界面积 × 权重 + 偏移距离 × 权重。
// 没有随机，没有迭代收敛问题（不像力导向那样需要固定 tick 次数）。

const RECT_EPS = 0.5;

function overlapArea(a, b) {
  const w = Math.min(a.x + a.w, b.x + b.w) - Math.max(a.x, b.x);
  const h = Math.min(a.y + a.h, b.y + b.h) - Math.max(a.y, b.y);
  return w > RECT_EPS && h > RECT_EPS ? w * h : 0;
}

function outOfBoundsArea(r, bounds) {
  const left = Math.max(0, -r.x);
  const top = Math.max(0, -r.y);
  const right = Math.max(0, r.x + r.w - bounds.w);
  const bottom = Math.max(0, r.y + r.h - bounds.h);
  return (left + right) * r.h + (top + bottom) * r.w;
}

/** 生成候选位置：优先用标签自己声明的方向，其后按固定顺序补足。 */
function candidateOffsets(item, offsets) {
  const preferred = item.preferred;
  const order = preferred
    ? [preferred, ...['right', 'left', 'top', 'bottom'].filter((d) => d !== preferred)]
    : ['right', 'left', 'top', 'bottom'];

  const out = [];
  for (const distance of offsets) {
    for (const side of order) {
      let dx = 0;
      let dy = 0;
      if (side === 'right') dx = distance;
      else if (side === 'left') dx = -distance - item.w;
      else if (side === 'bottom') dy = distance;
      else dy = -distance - item.h;
      out.push({side, dx, dy, distance});
    }
  }
  return out;
}

/**
 * 求解标签位置。
 *
 * @param items   [{id, x, y, w, h, priority, preferred}]
 *                x/y 是锚点（通常是站点坐标），w/h 是标签尺寸。
 * @param options.bounds           {w, h} clamp 区域
 * @param options.offsets          候选偏移距离，从小到大
 * @param options.obstacleRects    静态障碍（图例、信息卡等）
 * @param options.overlapWeight    重叠代价权重（默认 10）
 * @param options.boundsWeight     越界代价权重（默认 50）
 * @param options.distanceWeight   距离代价权重（默认 0.35）
 * @param options.maxOverlapRatio  允许的残余重叠占标签面积比例（默认 0＝不允许）
 * @returns {{placements: Array, dropped: Array}}
 *
 * **放不下就不放**：密集区域（如合肥枢纽、商合杭沿线）的站点标签数量远超可读容量。
 * 真实地图的做法是给优先级、按优先级放置、放不下的丢弃（MapLibre 的 symbol-sort-key 模型）。
 * 宁可 60 个清晰标签，也不要 97 个糊成一团 —— 这与"图上有 97 个站"不矛盾，
 * 数据仍在，只是不都上标注。
 */
export function solveLabels(items, options = {}) {
  const {
    bounds = {w: 10000, h: 10000},
    offsets = [8, 14, 22, 32, 44, 60],
    obstacleRects = [],
    overlapWeight = 10,
    boundsWeight = 50,
    distanceWeight = 0.35,
    maxOverlapRatio = 0,
  } = options;

  // 决定性顺序：主标签优先，其次按 id 字典序（不依赖输入顺序）
  const ordered = [...(items || [])].sort((a, b) => {
    const pa = a.priority || 0;
    const pb = b.priority || 0;
    if (pa !== pb) return pb - pa;
    return String(a.id).localeCompare(String(b.id));
  });

  const placed = [];
  const results = [];
  const dropped = [];

  for (const item of ordered) {
    const anchorRect = {x: item.x - item.w / 2, y: item.y - item.h, w: item.w, h: item.h};
    let best = null;

    for (const cand of candidateOffsets(item, offsets)) {
      const rect = {
        x: item.x + cand.dx - (cand.side === 'left' || cand.side === 'right' ? 0 : item.w / 2),
        y: item.y + cand.dy - (cand.side === 'top' || cand.side === 'bottom' ? 0 : item.h / 2),
        w: item.w,
        h: item.h,
      };

      let overlap = 0;
      for (const p of placed) overlap += overlapArea(rect, p);
      for (const ob of obstacleRects) overlap += overlapArea(rect, ob);

      const oob = outOfBoundsArea(rect, bounds);
      const dist = Math.hypot(cand.dx, cand.dy);
      const score = overlap * overlapWeight + oob * boundsWeight + dist * distanceWeight;

      if (!best || score < best.score) {
        best = {rect, score, side: cand.side, overlap, outOfBounds: oob};
      }
      if (score <= 0) break; // 已找到无冲突位置，提前结束
    }

    if (!best) {
      best = {rect: anchorRect, score: 0, side: 'center', overlap: 0, outOfBounds: 0};
    }

    // clamp 进边界（保证 in-bounds）
    best.rect.x = Math.max(0, Math.min(best.rect.x, bounds.w - best.rect.w));
    best.rect.y = Math.max(0, Math.min(best.rect.y, bounds.h - best.rect.h));

    // 残余重叠超过允许比例 → 丢弃（已按优先级排序，所以低优先级先被丢弃）
    const allowedOverlap = item.w * item.h * maxOverlapRatio;
    if (best.overlap > allowedOverlap || best.outOfBounds > allowedOverlap) {
      dropped.push({id: item.id, overlap: best.overlap, priority: item.priority || 0});
      continue;
    }

    placed.push(best.rect);
    results.push({
      id: item.id,
      x: best.rect.x,
      y: best.rect.y,
      w: best.rect.w,
      h: best.rect.h,
      side: best.side,
      anchorX: item.x,
      anchorY: item.y,
      // 是否需要用引导线连回锚点（偏移较远时才画，避免视觉噪声）
      useLeader: Math.hypot(best.rect.x + best.rect.w / 2 - item.x, best.rect.y + best.rect.h / 2 - item.y) > 26,
    });
  }

  return {placements: results, dropped};
}

/** 统计求解结果的质量指标（供质量门禁使用）。 */
export function labelQuality(placements, bounds) {
  let overlaps = 0;
  let outOfBounds = 0;
  for (let i = 0; i < placements.length; i++) {
    const a = placements[i];
    if (a.x < 0 || a.y < 0 || a.x + a.w > bounds.w || a.y + a.h > bounds.h) outOfBounds++;
    for (let j = i + 1; j < placements.length; j++) {
      if (overlapArea(a, placements[j]) > RECT_EPS) overlaps++;
    }
  }
  return {overlaps, outOfBounds, count: placements.length};
}
