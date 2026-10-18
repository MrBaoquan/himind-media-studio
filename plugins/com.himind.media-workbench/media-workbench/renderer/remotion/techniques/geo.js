// techniques/geo.js — 地理投影技法（drawer/pipeline 共用）
//
// 纯函数，无随机、无时间依赖：同一输入必然得到同一输出，
// 这是 Remotion 帧级确定性（以及黄金样例 SSIM 回归）的前提。

/** 经纬度 → 墨卡托平面坐标。 */
export const mercator = (lon, lat) => [
  lon,
  (180 / Math.PI) * Math.log(Math.tan(Math.PI / 4 + (lat * Math.PI) / 180 / 2)),
];

/**
 * 为一组坐标建立"自适应铺满"的投影。
 *
 * 关键：经度方向按 cos(中心纬度) 缩放，否则东西向与南北向的相同度数
 * 会代表不同实际距离，地图会被横向拉伸。
 */
export function createProjection(coordsList, { width, height, padding = 0 }) {
  let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
  let sumLat = 0, count = 0;

  for (const coords of coordsList) {
    for (const [lon, lat] of coords) {
      const [mx, my] = mercator(lon, lat);
      if (mx < minX) minX = mx;
      if (mx > maxX) maxX = mx;
      if (my < minY) minY = my;
      if (my > maxY) maxY = my;
      sumLat += lat;
      count += 1;
    }
  }
  if (!count) {
    return {toXY: () => [0, 0], empty: true};
  }

  const centerLat = sumLat / count;
  const lonScale = Math.cos((centerLat * Math.PI) / 180);
  const geoW = (maxX - minX) * lonScale || 1;
  const geoH = maxY - minY || 1;

  const availW = Math.max(1, width - padding * 2);
  const availH = Math.max(1, height - padding * 2);
  const scale = Math.min(availW / geoW, availH / geoH);
  const offsetX = padding + (availW - geoW * scale) / 2;
  const offsetY = padding + (availH - geoH * scale) / 2;

  const toXY = (lon, lat) => {
    const [mx, my] = mercator(lon, lat);
    return [
      offsetX + (mx - minX) * lonScale * scale,
      offsetY + (maxY - my) * scale,
    ];
  };

  return {toXY, empty: false, scale, centerLat};
}

/** 把一串经纬度投影成 SVG path 的 d 属性。 */
export function toPathD(coords, toXY, precision = 1) {
  if (!coords || !coords.length) return '';
  return coords
    .map(([lon, lat], i) => {
      const [x, y] = toXY(lon, lat);
      return `${i === 0 ? 'M' : 'L'}${x.toFixed(precision)},${y.toFixed(precision)}`;
    })
    .join(' ');
}

// ── 折线弧长工具（不依赖 @remotion/paths，避免运行时多装一个包） ──

/** 折线（SVG 坐标）总长度。 */
export function polylineLength(points) {
  let total = 0;
  for (let i = 1; i < points.length; i++) {
    total += Math.hypot(points[i][0] - points[i - 1][0], points[i][1] - points[i - 1][1]);
  }
  return total;
}

/** 折线上按弧长比例 t（0..1）取点。 */
export function pointAtLength(points, t) {
  if (!points.length) return [0, 0];
  if (points.length === 1) return points[0];
  const clamped = Math.max(0, Math.min(1, t));
  const target = polylineLength(points) * clamped;
  let walked = 0;
  for (let i = 1; i < points.length; i++) {
    const dx = points[i][0] - points[i - 1][0];
    const dy = points[i][1] - points[i - 1][1];
    const seg = Math.hypot(dx, dy);
    if (walked + seg >= target || i === points.length - 1) {
      const ratio = seg > 0 ? (target - walked) / seg : 0;
      return [points[i - 1][0] + dx * ratio, points[i - 1][1] + dy * ratio];
    }
    walked += seg;
  }
  return points[points.length - 1];
}

/** 折线上按弧长比例 t 处的切向角（度）。 */
export function tangentAngle(points, t) {
  if (points.length < 2) return 0;
  const clamped = Math.max(0, Math.min(1, t));
  const target = polylineLength(points) * clamped;
  let walked = 0;
  for (let i = 1; i < points.length; i++) {
    const dx = points[i][0] - points[i - 1][0];
    const dy = points[i][1] - points[i - 1][1];
    const seg = Math.hypot(dx, dy);
    if (walked + seg >= target || i === points.length - 1) {
      return (Math.atan2(dy, dx) * 180) / Math.PI;
    }
    walked += seg;
  }
  return 0;
}
