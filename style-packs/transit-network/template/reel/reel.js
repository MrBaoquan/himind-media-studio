const W = 1080, H = 1920, FPS = 30, DURATION = 30;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const MAP = {x0: 60, x1: 1020, y0: 330, y1: 1480, cx: 540, cy: 900};
const FONT_CN = '"Microsoft YaHei", "PingFang SC", "Noto Sans SC", sans-serif';
const FONT_MONO = '"Courier New", monospace';
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const lerp = (a, b, t) => a + (b - a) * t;
const ease = (v) => { v = clamp(v, 0, 1); return 1 - Math.pow(1 - v, 3); };
const easeInOut = (v) => { v = clamp(v, 0, 1); return v < 0.5 ? 4 * v * v * v : 1 - Math.pow(-2 * v + 2, 3) / 2; };

let DATA = {
  title: '合肥地铁',
  subtitle: '轨道上的城市生长',
  as_of: '2025-12',
  source: '示意线网 · 仅展示部分线路与站点',
  disclaimer: '示意图，非完整线网，不用于出行',
  lines: [],
  stations: [],
};
let STATE = null;

function pathLength(points) {
  let total = 0;
  for (let i = 1; i < points.length; i += 1) {
    total += Math.hypot(points[i][0] - points[i - 1][0], points[i][1] - points[i - 1][1]);
  }
  return total;
}

function distanceToStation(line, station) {
  let best = Infinity;
  let bestAt = 0;
  let acc = 0;
  for (let i = 1; i < line.path.length; i += 1) {
    const [x1, y1] = line.path[i - 1];
    const [x2, y2] = line.path[i];
    const dx = x2 - x1, dy = y2 - y1;
    const len = Math.hypot(dx, dy);
    if (len <= 0) continue;
    const t = clamp(((station.x - x1) * dx + (station.y - y1) * dy) / (len * len), 0, 1);
    const px = x1 + dx * t, py = y1 + dy * t;
    const d = Math.hypot(station.x - px, station.y - py);
    if (d < best) {
      best = d;
      bestAt = acc + len * t;
    }
    acc += len;
  }
  return bestAt;
}

function buildState(data) {
  const stations = data.stations.map((s) => ({...s, lines: []}));
  const byId = new Map(stations.map((s) => [s.id, s]));
  const lines = data.lines.map((line, index) => {
    const total = pathLength(line.path);
    const start = 0.8 + index * 1.3;
    const stationDistances = new Map();
    for (const id of line.stations) {
      const station = byId.get(id);
      if (station) stationDistances.set(id, distanceToStation(line, station));
    }
    for (const id of line.stations) {
      const station = byId.get(id);
      if (station && !station.lines.includes(line.id)) station.lines.push(line.id);
    }
    return {...line, total, start, stationDistances};
  });
  const stationReveal = new Map();
  for (const station of stations) {
    let reveal = Infinity;
    for (const line of lines) {
      if (!line.stationDistances.has(station.id)) continue;
      const d = line.stationDistances.get(station.id);
      reveal = Math.min(reveal, line.start + (line.total > 0 ? d / line.total : 1) * 1.4);
    }
    stationReveal.set(station.id, Number.isFinite(reveal) ? reveal : 3.2);
    station.interchange = station.lines.length > 1;
  }
  const terminals = new Set();
  for (const line of lines) {
    if (line.stations.length) {
      terminals.add(line.stations[0]);
      terminals.add(line.stations[line.stations.length - 1]);
    }
  }
  for (const station of stations) {
    station.priority = station.interchange ? 100 : terminals.has(station.id) ? 60 : 20;
  }
  const state = {stations, byId, lines, stationReveal, terminals};
  state.placements = placeLabels(state);
  return state;
}

function segmentIntersectsRect(x1, y1, x2, y2, box) {
  const minX = box.x, maxX = box.x + box.w, minY = box.y, maxY = box.y + box.h;
  const inside = (x, y) => x >= minX && x <= maxX && y >= minY && y <= maxY;
  if (inside(x1, y1) || inside(x2, y2)) return true;
  const edges = [
    [minX, minY, maxX, minY], [maxX, minY, maxX, maxY],
    [maxX, maxY, minX, maxY], [minX, maxY, minX, minY],
  ];
  for (const [ax, ay, bx, by] of edges) {
    const d1 = (x2 - x1) * (ay - y1) - (y2 - y1) * (ax - x1);
    const d2 = (x2 - x1) * (by - y1) - (y2 - y1) * (bx - x1);
    const d3 = (bx - ax) * (y1 - ay) - (by - ay) * (x1 - ax);
    const d4 = (bx - ax) * (y2 - ay) - (by - ay) * (x2 - ax);
    if (((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) && ((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0))) return true;
  }
  return false;
}

function boxesOverlap(a, b, pad = 6) {
  return !(a.x + a.w + pad < b.x || b.x + b.w + pad < a.x || a.y + a.h + pad < b.y || b.y + b.h + pad < a.y);
}

function placeLabels(state) {
  const placements = new Map();
  const placed = [];
  const ordered = [...state.stations].sort((a, b) => b.priority - a.priority || a.y - b.y || a.x - b.x);
  const baseCandidates = [
    {dx: 16, dy: 0, align: 'left'},
    {dx: -16, dy: 0, align: 'right'},
    {dx: 0, dy: -24, align: 'center'},
    {dx: 0, dy: 26, align: 'center'},
    {dx: 12, dy: -18, align: 'left'},
    {dx: 12, dy: 18, align: 'left'},
    {dx: -12, dy: -18, align: 'right'},
    {dx: -12, dy: 18, align: 'right'},
    {dx: 30, dy: 0, align: 'left'},
    {dx: -30, dy: 0, align: 'right'},
    {dx: 0, dy: -42, align: 'center'},
    {dx: 0, dy: 44, align: 'center'},
  ];
  for (const station of ordered) {
    const size = station.interchange ? 26 : station.priority >= 60 ? 24 : 22;
    const weight = station.interchange ? 700 : 500;
    ctx.font = `${weight} ${size}px ${FONT_CN}`;
    const textW = Math.ceil(ctx.measureText(station.name).width);
    const textH = Math.ceil(size * 1.35);
    let chosen = null;
    for (const candidate of baseCandidates) {
      const textX = station.x + candidate.dx;
      const textY = station.y + candidate.dy;
      const x = candidate.align === 'left' ? textX : candidate.align === 'right' ? textX - textW : textX - textW / 2;
      const y = textY - textH / 2;
      const box = {x, y, w: textW, h: textH, textX, textY, align: candidate.align, size, weight, leader: false};
      if (box.x < MAP.x0 + 8 || box.x + box.w > MAP.x1 - 8 || box.y < MAP.y0 + 8 || box.y + box.h > MAP.y1 - 8) continue;
      if (placed.some((b) => boxesOverlap(box, b, 6))) continue;
      let hitsLine = false;
      for (const line of state.lines) {
        for (let i = 1; i < line.path.length; i += 1) {
          const [x1, y1] = line.path[i - 1];
          const [x2, y2] = line.path[i];
          if (segmentIntersectsRect(x1, y1, x2, y2, {x: box.x - 3, y: box.y - 3, w: box.w + 6, h: box.h + 6})) {
            hitsLine = true;
            break;
          }
        }
        if (hitsLine) break;
      }
      if (hitsLine) continue;
      let hitsStation = false;
      for (const other of state.stations) {
        if (other.id === station.id) continue;
        if (other.x > box.x - 10 && other.x < box.x + box.w + 10 && other.y > box.y - 10 && other.y < box.y + box.h + 10) {
          hitsStation = true;
          break;
        }
      }
      if (hitsStation) continue;
      chosen = box;
      break;
    }
    if (!chosen) {
      const fallbackCandidates = [
        {dx: 0, dy: -64, align: 'center'},
        {dx: 0, dy: 64, align: 'center'},
        {dx: -56, dy: 0, align: 'right'},
        {dx: 56, dy: 0, align: 'left'},
        {dx: -50, dy: -58, align: 'right'},
        {dx: 50, dy: -58, align: 'left'},
        {dx: -50, dy: 58, align: 'right'},
        {dx: 50, dy: 58, align: 'left'},
      ];
      for (const candidate of fallbackCandidates) {
        const textX = station.x + candidate.dx;
        const textY = station.y + candidate.dy;
        const x = candidate.align === 'left' ? textX : candidate.align === 'right' ? textX - textW : textX - textW / 2;
        const y = textY - textH / 2;
        const box = {x, y, w: textW, h: textH, textX, textY, align: candidate.align, size, weight, leader: true};
        if (box.x < MAP.x0 + 8 || box.x + box.w > MAP.x1 - 8 || box.y < MAP.y0 + 8 || box.y + box.h > MAP.y1 - 8) continue;
        if (placed.some((b) => boxesOverlap(box, b, 6))) continue;
        let hitsStation = false;
        for (const other of state.stations) {
          if (other.id === station.id) continue;
          if (other.x > box.x - 10 && other.x < box.x + box.w + 10 && other.y > box.y - 10 && other.y < box.y + box.h + 10) {
            hitsStation = true;
            break;
          }
        }
        if (hitsStation) continue;
        chosen = box;
        break;
      }
    }
    if (chosen) {
      chosen.leader = chosen.leader || Math.hypot(chosen.textX - station.x, chosen.textY - station.y) > 38;
      placed.push(chosen);
      placements.set(station.id, chosen);
    }
  }
  return placements;
}

function drawBackground() {
  ctx.fillStyle = '#FCFCFA';
  ctx.fillRect(0, 0, W, H);
}

function drawRiver() {
  ctx.save();
  ctx.lineCap = 'round';
  ctx.strokeStyle = '#E8F5FF';
  ctx.lineWidth = 66;
  ctx.beginPath();
  ctx.moveTo(64, 560);
  ctx.bezierCurveTo(280, 620, 690, 760, 1016, 980);
  ctx.stroke();
  ctx.strokeStyle = '#D6EEFF';
  ctx.lineWidth = 28;
  ctx.beginPath();
  ctx.moveTo(64, 560);
  ctx.bezierCurveTo(280, 620, 690, 760, 1016, 980);
  ctx.stroke();
  ctx.fillStyle = '#9FCBE5';
  ctx.font = `500 18px ${FONT_CN}`;
  ctx.save();
  ctx.translate(94, 622);
  ctx.rotate(-0.12);
  ctx.fillText('南淝河', 0, 0);
  ctx.restore();
  ctx.restore();
}

function partialPoints(points, progress) {
  if (progress >= 1) return points;
  if (progress <= 0) return [];
  const total = pathLength(points);
  let target = total * progress;
  const out = [points[0]];
  for (let i = 1; i < points.length; i += 1) {
    const [x1, y1] = points[i - 1];
    const [x2, y2] = points[i];
    const len = Math.hypot(x2 - x1, y2 - y1);
    if (target >= len) {
      out.push(points[i]);
      target -= len;
    } else {
      const t = len > 0 ? target / len : 0;
      out.push([lerp(x1, x2, t), lerp(y1, y2, t)]);
      break;
    }
  }
  return out;
}

function strokePath(points, color, width) {
  if (points.length < 2) return;
  ctx.strokeStyle = color;
  ctx.lineWidth = width;
  ctx.lineCap = 'round';
  ctx.lineJoin = 'round';
  ctx.beginPath();
  ctx.moveTo(points[0][0], points[0][1]);
  for (let i = 1; i < points.length; i += 1) ctx.lineTo(points[i][0], points[i][1]);
  ctx.stroke();
}

function drawLines(t) {
  const partials = STATE.lines.map((line) => ({
    line,
    points: partialPoints(line.path, ease((t - line.start) / 1.4)),
  })).filter((item) => item.points.length >= 2);
  for (const item of partials) strokePath(item.points, '#FFFFFF', 22);
  for (const item of partials) strokePath(item.points, item.line.color, 12);
}

function drawStations(t) {
  for (const station of STATE.stations) {
    const reveal = STATE.stationReveal.get(station.id) ?? 3.2;
    const alpha = ease((t - reveal) / 0.35);
    if (alpha <= 0) continue;
    const lineColor = STATE.lines.find((line) => line.stations.includes(station.id))?.color || '#333333';
    ctx.save();
    ctx.globalAlpha = alpha;
    if (station.interchange) {
      const pulseWindow = clamp((t - 10.5) / 0.8, 0, 1) * clamp((18.5 - t) / 1.0, 0, 1);
      const pulse = 1 + 0.18 * Math.sin((t - 10.5) * Math.PI * 2 * 0.75) * pulseWindow;
      if (pulseWindow > 0) {
        ctx.beginPath();
        ctx.arc(station.x, station.y, 17 * pulse, 0, Math.PI * 2);
        ctx.strokeStyle = 'rgba(20,20,20,0.24)';
        ctx.lineWidth = 8;
        ctx.stroke();
      }
      ctx.beginPath();
      ctx.arc(station.x, station.y, 13, 0, Math.PI * 2);
      ctx.fillStyle = '#FFFFFF';
      ctx.fill();
      ctx.strokeStyle = '#1B1B1B';
      ctx.lineWidth = 4;
      ctx.stroke();
      ctx.beginPath();
      ctx.arc(station.x, station.y, 5, 0, Math.PI * 2);
      ctx.fillStyle = lineColor;
      ctx.fill();
    } else {
      ctx.beginPath();
      ctx.arc(station.x, station.y, 7, 0, Math.PI * 2);
      ctx.fillStyle = '#FFFFFF';
      ctx.fill();
      ctx.strokeStyle = lineColor;
      ctx.lineWidth = 3;
      ctx.stroke();
    }
    ctx.restore();
  }
}

function pointAlongPath(points, distance) {
  let remaining = Math.max(0, distance);
  for (let i = 1; i < points.length; i += 1) {
    const [x1, y1] = points[i - 1];
    const [x2, y2] = points[i];
    const len = Math.hypot(x2 - x1, y2 - y1);
    if (remaining <= len) {
      const t = len > 0 ? remaining / len : 0;
      return [lerp(x1, x2, t), lerp(y1, y2, t)];
    }
    remaining -= len;
  }
  return points[points.length - 1];
}

function drawTrains(t) {
  STATE.lines.forEach((line, index) => {
    const doneAt = line.start + 1.4;
    if (t <= doneAt) return;
    const phase = (index * 0.23) % 1;
    const u = ((t - doneAt) / 4.5 + phase) % 1;
    const [x, y] = pointAlongPath(line.path, u * line.total);
    const [px, py] = pointAlongPath(line.path, ((u + 0.97) % 1) * line.total);
    ctx.save();
    ctx.globalAlpha = 0.34;
    ctx.strokeStyle = line.color;
    ctx.lineWidth = 7;
    ctx.lineCap = 'round';
    ctx.beginPath();
    ctx.moveTo(px, py);
    ctx.lineTo(x, y);
    ctx.stroke();
    ctx.globalAlpha = 0.95;
    ctx.beginPath();
    ctx.arc(x, y, 6.5, 0, Math.PI * 2);
    ctx.fillStyle = '#FFFFFF';
    ctx.fill();
    ctx.strokeStyle = line.color;
    ctx.lineWidth = 3;
    ctx.stroke();
    ctx.restore();
  });
}

function drawLabels(t) {
  for (const station of STATE.stations) {
    const placement = STATE.placements.get(station.id);
    if (!placement) continue;
    const reveal = (STATE.stationReveal.get(station.id) ?? 3.2) + 0.2;
    const alpha = ease((t - reveal) / 0.5);
    if (alpha <= 0) continue;
    ctx.save();
    ctx.globalAlpha = alpha;
    if (placement.leader) {
      const points = Array.isArray(placement.leaderPoints) && placement.leaderPoints.length >= 2
        ? placement.leaderPoints
        : [[station.x, station.y], [placement.x + (station.x < placement.x + placement.w / 2 ? 0 : placement.w), placement.textY]];
      ctx.strokeStyle = 'rgba(60,60,60,0.28)';
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(points[0][0], points[0][1]);
      for (let i = 1; i < points.length; i += 1) ctx.lineTo(points[i][0], points[i][1]);
      ctx.stroke();
    }
    ctx.font = `${placement.weight} ${placement.size}px ${FONT_CN}`;
    ctx.textAlign = placement.align;
    ctx.textBaseline = 'middle';
    ctx.lineJoin = 'round';
    ctx.lineWidth = 5;
    ctx.strokeStyle = 'rgba(255,255,255,0.96)';
    ctx.strokeText(station.name, placement.textX, placement.textY);
    ctx.fillStyle = station.interchange ? '#111111' : '#2A2A2A';
    ctx.fillText(station.name, placement.textX, placement.textY);
    ctx.restore();
  }
}

function drawTitle(t) {
  ctx.save();
  ctx.globalAlpha = ease(t / 0.8);
  ctx.fillStyle = '#141414';
  ctx.font = `700 64px ${FONT_CN}`;
  ctx.fillText(DATA.title, 70, 150);
  ctx.fillStyle = '#555555';
  ctx.font = `500 26px ${FONT_CN}`;
  ctx.fillText(DATA.subtitle, 72, 205);
  ctx.fillStyle = '#9A9A9A';
  ctx.font = `500 15px ${FONT_MONO}`;
  ctx.fillText('HEFEI METRO · SCHEMATIC', 72, 245);
  const colors = DATA.lines.map((line) => line.color);
  colors.forEach((color, i) => {
    ctx.fillStyle = color;
    ctx.fillRect(72 + i * 48, 270, 40, 5);
  });
  ctx.restore();
}

function drawCaption(t) {
  let lines = null;
  let alpha = 0;
  if (t >= 10.5 && t < 16) {
    lines = ['换乘节点', '让线路真正连成网络'];
    alpha = Math.min(ease((t - 10.5) / 0.5), ease((16 - t) / 0.6));
  } else if (t >= 16 && t < 22) {
    lines = ['2016 — 2025', '合肥地铁线网生长'];
    alpha = Math.min(ease((t - 16) / 0.5), ease((22 - t) / 0.6));
  } else if (t >= 22) {
    lines = ['轨道上的合肥', '一张图，看见城市生长'];
    alpha = ease((t - 22) / 0.6);
  }
  if (!lines || alpha <= 0) return;
  ctx.save();
  ctx.globalAlpha = alpha;
  ctx.fillStyle = '#1D1D1D';
  ctx.font = `700 30px ${FONT_CN}`;
  ctx.fillText(lines[0], 70, 326);
  ctx.fillStyle = '#777777';
  ctx.font = `500 20px ${FONT_CN}`;
  ctx.fillText(lines[1], 72, 358);
  if (lines[0] === '轨道上的合肥') {
    ctx.strokeStyle = '#E60012';
    ctx.lineWidth = 3;
    ctx.beginPath();
    ctx.moveTo(72, 374);
    ctx.lineTo(72 + 210 * ease((t - 22.5) / 1.2), 374);
    ctx.stroke();
  }
  ctx.restore();
}

function drawLegend(t) {
  const alpha = ease((t - 8.6) / 0.8);
  if (alpha <= 0) return;
  ctx.save();
  ctx.globalAlpha = alpha;
  ctx.fillStyle = '#8A8A8A';
  ctx.font = `500 16px ${FONT_CN}`;
  ctx.fillText('线路', 70, 1548);
  const active = t >= 14 && t < 18.5 ? Math.min(DATA.lines.length - 1, Math.floor((t - 14) / 0.85)) : -1;
  DATA.lines.forEach((line, i) => {
    const y = 1582 + i * 34;
    if (i === active) {
      ctx.fillStyle = 'rgba(20,20,20,0.05)';
      ctx.fillRect(64, y - 20, 430, 30);
    }
    ctx.fillStyle = line.color;
    ctx.fillRect(70, y - 6, 46, 10);
    ctx.fillStyle = '#222222';
    ctx.font = `${i === active ? 700 : 500} 21px ${FONT_CN}`;
    ctx.fillText(line.name, 130, y);
    ctx.fillStyle = '#9A9A9A';
    ctx.font = `500 15px ${FONT_MONO}`;
    ctx.fillText(String(line.opened), 250, y);
  });
  ctx.restore();
}

function drawYearAxis(t) {
  const alpha = ease((t - 12.5) / 0.8);
  if (alpha <= 0) return;
  const x0 = 70, x1 = 1010, y = 1780;
  ctx.save();
  ctx.globalAlpha = alpha;
  ctx.strokeStyle = '#D8D8D8';
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.moveTo(x0, y);
  ctx.lineTo(x1, y);
  ctx.stroke();
  const years = [2016, 2018, 2020, 2022, 2025];
  ctx.fillStyle = '#9A9A9A';
  ctx.font = `500 14px ${FONT_MONO}`;
  ctx.textAlign = 'center';
  years.forEach((year) => {
    const x = x0 + (year - 2016) / 9 * (x1 - x0);
    ctx.fillRect(x - 1, y - 5, 2, 10);
    ctx.fillText(String(year), x, y + 24);
  });
  const progress = ease((t - 13.5) / 6);
  const x = lerp(x0, x1, progress);
  ctx.strokeStyle = 'rgba(20,20,20,0.50)';
  ctx.lineWidth = 8;
  ctx.beginPath();
  ctx.moveTo(x0, y);
  ctx.lineTo(x, y);
  ctx.stroke();
  ctx.beginPath();
  ctx.arc(x, y, 7, 0, Math.PI * 2);
  ctx.fillStyle = '#222222';
  ctx.fill();
  ctx.fillStyle = '#333333';
  ctx.font = `700 16px ${FONT_MONO}`;
  ctx.textAlign = 'left';
  ctx.fillText(String(2016 + Math.round(progress * 9)), 70, y - 18);
  ctx.restore();
}

function drawSource() {
  ctx.save();
  ctx.fillStyle = '#8A8A8A';
  ctx.font = `500 15px ${FONT_CN}`;
  ctx.fillText(`${DATA.source} · 数据截至 ${DATA.as_of}`, 70, 1840);
  ctx.fillStyle = '#B0B0B0';
  ctx.font = `400 13px ${FONT_CN}`;
  ctx.fillText(DATA.disclaimer, 70, 1868);
  ctx.textAlign = 'right';
  ctx.fillStyle = '#B7B7B7';
  ctx.font = `500 13px ${FONT_MONO}`;
  ctx.fillText('HIMIND / TRANSIT', W - 70, 1868);
  ctx.restore();
}

function cameraAt(t) {
  const zoom = 1 + 0.03 * Math.sin((t / 30) * Math.PI) + 0.025 * ease((t - 10) / 4) * (1 - ease((t - 22) / 5));
  const cx = 540 + 22 * Math.sin((t / 30) * Math.PI * 2);
  return {zoom, cx, cy: 900};
}

function renderAt(t) {
  const time = clamp(t, 0, DURATION);
  drawBackground();
  const cam = cameraAt(time);
  ctx.save();
  ctx.translate(cam.cx, cam.cy);
  ctx.scale(cam.zoom, cam.zoom);
  ctx.translate(-cam.cx, -cam.cy);
  drawRiver();
  drawLines(time);
  drawStations(time);
  drawTrains(time);
  drawLabels(time);
  ctx.restore();
  drawTitle(time);
  drawCaption(time);
  drawLegend(time);
  drawYearAxis(time);
  drawSource();
}

async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) DATA = await response.json();
  } catch {
    // Keep the minimal fallback; the renderer still produces a titled frame.
  }
function applyPrecomputedPlacements(state, payload) {
  const items = Array.isArray(payload?.placements) ? payload.placements : [];
  if (!items.length) return;
  const next = new Map();
  for (const item of items) {
    if (!item || item.visible === false) continue;
    const station = state.byId.get(item.id);
    if (!station || !Array.isArray(item.box) || item.box.length !== 4) continue;
    const [x, y, w, h] = item.box.map(Number);
    if (![x, y, w, h].every(Number.isFinite)) continue;
    const size = station.interchange ? 26 : station.priority >= 60 ? 24 : 22;
    next.set(station.id, {
      x, y, w, h,
      textX: x,
      textY: y + h / 2,
      align: 'left',
      size,
      weight: station.interchange ? 700 : 500,
      leader: Array.isArray(item.leader) && item.leader.length >= 2,
      leaderPoints: Array.isArray(item.leader) ? item.leader : null,
    });
  }
  if (next.size) state.placements = next;
}

  STATE = buildState(DATA);
  try {
    const labelsResponse = await fetch('/input/labels.json', {cache: 'no-store'});
    if (labelsResponse.ok) applyPrecomputedPlacements(STATE, await labelsResponse.json());
  } catch {
    // No precomputed placements: keep the deterministic 8-direction fallback.
  }
  window.__DEBUG__ = {
    placements: [...STATE.placements.entries()].map(([id, p]) => ({id, name: STATE.byId.get(id)?.name, ...p})),
    hidden: STATE.stations.filter((s) => !STATE.placements.has(s.id)).map((s) => s.name),
    interchanges: STATE.stations.filter((s) => s.interchange).map((s) => s.name),
    lines: STATE.lines.map((l) => ({id: l.id, total: Math.round(l.total)})),
  };
  window.renderAt = renderAt;
  window.__DURATION__ = DURATION;
  window.__FPS__ = FPS;
  await document.fonts.ready;
  renderAt(0);
  window.__READY__ = true;
}

loadData();


