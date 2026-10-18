// techniques/schedule.js — 时间轴调度技法（solver）
//
// 把"按开通年份分组、逐条生长"这件事做成可参数化的纯函数。
//
// 与参考实现的差别：**时长自适应**。参考实现用固定的
// MIN_DRAW_FRAMES=10 / MAX_DRAW_FRAMES=24，换个时长就崩；
// 这里把可用帧数按"年份组数"与"每条线路的复杂度"分配，
// 因此 6 秒和 60 秒都能正确铺满。

/** 由路径点数估算复杂度权重（点越多画得越久）。 */
function complexity(coords, minPoints, maxPoints) {
  const n = coords ? coords.length : 0;
  const t = Math.max(0, Math.min(1, (n - minPoints) / Math.max(1, maxPoints - minPoints)));
  return 0.4 + 0.6 * t;
}

/**
 * 构建绘制时间表。
 *
 * @param lines   线路数组，需含 {id, openYear, coords}
 * @param options.startFrame  第一组开始的帧
 * @param options.endFrame    最后一组结束的帧（含）
 * @returns {{schedule: Array, groups: Array, startFrame:number, endFrame:number}}
 */
export function buildTimeline(lines, options = {}) {
  const {
    startFrame = 30,
    endFrame = 480,
    groupGapFrames = 8,
    minPoints = 6,
    maxPoints = 60,
    tailHoldFrames = 30,
  } = options;

  const drawable = (lines || []).filter((l) => l && l.coords && l.coords.length >= 2);
  if (!drawable.length) {
    return {schedule: [], groups: [], startFrame, endFrame, totalFrames: endFrame - startFrame};
  }

  // 按年份分组（同一年开通的线路并排生长）
  const byYear = new Map();
  for (const line of drawable) {
    const year = Number(line.openYear) || 0;
    if (!byYear.has(year)) byYear.set(year, []);
    byYear.get(year).push(line);
  }
  const years = [...byYear.keys()].sort((a, b) => a - b);

  const usable = Math.max(1, endFrame - tailHoldFrames - startFrame);
  const gaps = Math.max(0, years.length - 1) * groupGapFrames;
  const drawBudget = Math.max(1, usable - gaps);

  // 每组权重 = 组内最大复杂度
  const weights = years.map((year) => {
    const group = byYear.get(year);
    return Math.max(...group.map((l) => complexity(l.coords, minPoints, maxPoints)));
  });
  const weightSum = weights.reduce((a, b) => a + b, 0) || 1;

  const schedule = [];
  const groups = [];
  let cursor = startFrame;

  years.forEach((year, index) => {
    const group = byYear.get(year);
    const groupDraw = Math.max(6, Math.round((weights[index] / weightSum) * drawBudget));
    for (const line of group) {
      const ratio = complexity(line.coords, minPoints, maxPoints);
      schedule.push({
        lineId: line.id,
        openYear: year,
        startFrame: cursor,
        drawFrames: groupDraw,
        endFrame: cursor + groupDraw,
        // 同组内按复杂度做轻微错峰，避免所有线同时起笔显得机械
        staggerFrame: Math.round((1 - ratio) * groupDraw * 0.25),
      });
    }
    groups.push({year, startFrame: cursor, endFrame: cursor + groupDraw, lineIds: group.map((l) => l.id)});
    cursor += groupDraw + groupGapFrames;
  });

  return {schedule, groups, startFrame, endFrame: cursor - groupGapFrames, totalFrames: cursor - groupGapFrames - startFrame};
}

/** 某条线路在当前帧的绘制进度（0..1）。 */
export function lineProgress(entry, frame) {
  if (!entry) return 0;
  const start = entry.startFrame + (entry.staggerFrame || 0);
  if (frame <= start) return 0;
  if (frame >= start + entry.drawFrames) return 1;
  return (frame - start) / entry.drawFrames;
}

/**
 * 派生指标：当前年份、累计里程、已完成线路。
 * 这些是"数据可视化"区别于"装饰动画"的关键 —— 画面上的数字必须来自数据。
 */
export function deriveMetrics(timeline, frame, lines, options = {}) {
  const {startFrame = 30, yearAt = () => null} = options;
  const byId = new Map((lines || []).map((l) => [l.id, l]));
  let mileage = 0;
  let currentYear = null;
  const completed = [];

  for (const entry of timeline.schedule) {
    const progress = lineProgress(entry, frame);
    const line = byId.get(entry.lineId);
    const lengthKm = line ? Number(line.lengthKm) || 0 : 0;
    mileage += lengthKm * progress;
    if (progress >= 1) completed.push(entry.lineId);
    if (progress > 0 && currentYear === null) currentYear = entry.openYear;
    if (progress > 0) currentYear = Math.max(currentYear ?? entry.openYear, entry.openYear);
  }

  return {
    mileage: Math.round(mileage),
    completedLineIds: completed,
    completedCount: completed.length,
    currentYear: frame < startFrame ? null : currentYear,
    year: yearAt(frame),
  };
}

/** 年份里程碑帧位（用于进度条刻度）。 */
export function yearMilestones(timeline) {
  return (timeline.groups || []).map((g) => ({year: g.year, frame: g.startFrame}));
}
