// foundation/layout1d.js —— 一维非重叠布局。
//
// 场景：时间线上的事件标签、年份刻度、里程碑。
// 目标：在 [min, max] 内给 anchors 分配位置，保证相邻间距 >= minGap，
// 且尽量贴近原始锚点。纯函数，无迭代不收敛问题。

/**
 * @param {number[]} anchors 原始锚点（任意顺序）
 * @param {{minGap:number, min:number, max:number}} options
 * @returns {number[]} 与 anchors 同序的位置
 */
export function spread1D(anchors, options = {}) {
  const {minGap = 40, min = -Infinity, max = Infinity} = options;
  const n = anchors.length;
  if (n === 0) return [];
  if (n === 1) return [Math.max(min, Math.min(max, anchors[0]))];

  const order = anchors.map((value, index) => ({value, index})).sort((a, b) => a.value - b.value);
  const placed = new Array(n).fill(0);

  // 前向：保证递增且满足间距
  let cursor = Math.max(min, order[0].value);
  for (let i = 0; i < n; i++) {
    const target = Math.max(order[i].value, cursor);
    placed[order[i].index] = target;
    cursor = target + minGap;
  }

  // 后向：如果越过 max，整体回压并保证仍满足间距
  let overflow = placed[order[n - 1].index] - max;
  if (overflow > 0) {
    let limit = max;
    for (let i = n - 1; i >= 0; i--) {
      const index = order[i].index;
      placed[index] = Math.min(placed[index], limit);
      limit = placed[index] - minGap;
    }
    // 若回压后仍低于 min，前向再抬一次
    if (placed[order[0].index] < min) {
      let floor = min;
      for (let i = 0; i < n; i++) {
        const index = order[i].index;
        placed[index] = Math.max(placed[index], floor);
        floor = placed[index] + minGap;
      }
    }
  }

  return placed;
}

/** 把数值从 [inMin,inMax] 线性映射到 [outMin,outMax]，并 clamp。 */
export function mapRange(value, inMin, inMax, outMin, outMax) {
  const span = inMax - inMin;
  if (span === 0) return outMin;
  const t = Math.max(0, Math.min(1, (value - inMin) / span));
  return outMin + (outMax - outMin) * t;
}
