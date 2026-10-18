const W = 1080, H = 1920, FPS = 30, DURATION = 38;
const INTRO = 3.6;
const EVENT_LEN = 6.2;
const OUTRO_START = INTRO + 5 * EVENT_LEN;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const MAP = {x0: 60, x1: 1020, y0: 300, y1: 1400, cx: 540, cy: 850};
const FONT_CN = '"Microsoft YaHei", "PingFang SC", "Noto Sans SC", sans-serif';
const FONT_MONO = '"DIN Alternate", "Bahnschrift", "Courier New", monospace';
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const lerp = (a, b, t) => a + (b - a) * t;
const ease = (v) => { v = clamp(v, 0, 1); return 1 - Math.pow(1 - v, 3); };
const easeInOut = (v) => { v = clamp(v, 0, 1); return v < 0.5 ? 4 * v * v * v : 1 - Math.pow(-2 * v + 2, 3) / 2; };
const easeOutExpo = (v) => { v = clamp(v, 0, 1); return v === 1 ? 1 : 1 - Math.pow(2, -10 * v); };

let DATA = {
  title: '合肥轨道交通',
  subtitle: '对 12/26 情有独钟',
  as_of: '2026-10',
  source: '示意线网 · 基于公开资料整理 · 仅展示部分线路与站点',
  disclaimer: '示意图，非完整官方线网，不用于出行',
  lines: [],
  stations: [],
  events: [],
  stats: {lines: 0, stations: 0, km: 0},
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
    const stationDistances = new Map();
    for (const id of line.stations) {
      const station = byId.get(id);
      if (station) stationDistances.set(id, distanceToStation(line, station));
    }
    for (const id of line.stations) {
      const station = byId.get(id);
      if (station && !station.lines.includes(line.id)) station.lines.push(line.id);
    }
    return {...line, index, total, stationDistances};
  });
  const terminals = new Set();
  for (const line of lines) {
    if (line.stations.length) {
      terminals.add(line.stations[0]);
      terminals.add(line.stations[line.stations.length - 1]);
    }
  }
  for (const station of stations) {
    station.interchange = station.lines.length > 1;
    station.priority = station.interchange ? 100 : terminals.has(station.id) ? 60 : 20;
  }
  const state = {stations, byId, lines, terminals};
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
  const candidates = [
    {dx: 18, dy: 0, align: 'left'}, {dx: -18, dy: 0, align: 'right'},
    {dx: 0, dy: -26, align: 'center'}, {dx: 0, dy: 28, align: 'center'},
    {dx: 14, dy: -20, align: 'left'}, {dx: 14, dy: 20, align: 'left'},
    {dx: -14, dy: -20, align: 'right'}, {dx: -14, dy: 20, align: 'right'},
    {dx: 34, dy: 0, align: 'left'}, {dx: -34, dy: 0, align: 'right'},
  ];
  for (const station of ordered) {
    const size = station.interchange ? 27 : station.priority >= 60 ? 24 : 21;
    const weight = station.interchange ? 700 : 500;
    ctx.font = `${weight} ${size}px ${FONT_CN}`;
    const textW = Math.ceil(ctx.measureText(station.name).width);
    const textH = Math.ceil(size * 1.35);
    let chosen = null;
    for (const candidate of candidates) {
      const textX = station.x + candidate.dx;
      const textY = station.y + candidate.dy;
      const x = candidate.align === 'left' ? textX : candidate.align === 'right' ? textX - textW : textX - textW / 2;
      const y = textY - textH / 2;
      const box = {x, y, w: textW, h: textH, textX, textY, align: candidate.align, size, weight, leader: false};
      if (box.x < MAP.x0 + 8 || box.x + box.w > MAP.x1 - 8 || box.y < MAP.y0 + 8 || box.y + box.h > MAP.y1 - 8) continue;
      if (placed.some((b) => boxesOverlap(box, b, 8))) continue;
      let hitsLine = false;
      for (const line of state.lines) {
        for (let i = 1; i < line.path.length; i += 1) {
          const [x1, y1] = line.path[i - 1];
          const [x2, y2] = line.path[i];
          if (segmentIntersectsRect(x1, y1, x2, y2, {x: box.x - 4, y: box.y - 4, w: box.w + 8, h: box.h + 8})) {
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
        if (other.x > box.x - 12 && other.x < box.x + box.w + 12 && other.y > box.y - 12 && other.y < box.y + box.h + 12) {
          hitsStation = true;
          break;
        }
      }
      if (hitsStation) continue;
      chosen = box;
      break;
    }
    if (chosen) {
      chosen.leader = Math.hypot(chosen.textX - station.x, chosen.textY - station.y) > 40;
      placed.push(chosen);
      placements.set(station.id, chosen);
    }
  }
  return placements;
}

function drawBackground() {
  ctx.fillStyle = '#F4F3EE';
  ctx.fillRect(0, 0, W, H);
  ctx.save();
  ctx.fillStyle = 'rgba(20,20,20,0.035)';
  for (let y = 40; y < H; y += 40) {
    for (let x = 40; x < W; x += 40) {
      ctx.beginPath();
      ctx.arc(x, y, 1.1, 0, Math.PI * 2);
      ctx.fill();
    }
  }
  ctx.restore();
}

function drawRiver() {
  ctx.save();
  ctx.lineCap = 'round';
  ctx.strokeStyle = 'rgba(180,214,230,0.28)';
  ctx.lineWidth = 74;
  ctx.beginPath();
  ctx.moveTo(60, 560);
  ctx.bezierCurveTo(280, 630, 690, 760, 1020, 980);
  ctx.stroke();
  ctx.strokeStyle = 'rgba(155,200,222,0.34)';
  ctx.lineWidth = 26;
  ctx.beginPath();
  ctx.moveTo(60, 560);
  ctx.bezierCurveTo(280, 630, 690, 760, 1020, 980);
  ctx.stroke();
  ctx.fillStyle = 'rgba(110,155,178,0.65)';
  ctx.font = `500 17px ${FONT_CN}`;
  ctx.save();
  ctx.translate(92, 620);
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

function lineBBox(line) {
  let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
  for (const [x, y] of line.path) {
    minX = Math.min(minX, x); maxX = Math.max(maxX, x);
    minY = Math.min(minY, y); maxY = Math.max(maxY, y);
  }
  return {minX, maxX, minY, maxY, cx: (minX + maxX) / 2, cy: (minY + maxY) / 2, w: maxX - minX, h: maxY - minY};
}

function fullCamera() {
  return {zoom: 1, cx: MAP.cx, cy: MAP.cy};
}

function eventCamera(line) {
  const b = lineBBox(line);
  const pad = 220;
  const w = Math.max(240, b.w + pad);
  const h = Math.max(240, b.h + pad);
  const zoom = clamp(Math.min((MAP.x1 - MAP.x0) / w, (MAP.y1 - MAP.y0) / h), 1.02, 1.42);
  return {zoom, cx: b.cx, cy: b.cy};
}

function activeIndexAt(t) {
  if (t < INTRO) return -1;
  if (t >= OUTRO_START) return 5;
  return clamp(Math.floor((t - INTRO) / EVENT_LEN), 0, 4);
}

function localEventT(t, index) {
  return clamp((t - (INTRO + index * EVENT_LEN)) / EVENT_LEN, 0, 1);
}

function cameraAt(t, activeIndex) {
  const full = fullCamera();
  if (activeIndex < 0 || activeIndex >= 5) return full;
  const line = STATE.lines[activeIndex];
  const target = eventCamera(line);
  const local = localEventT(t, activeIndex);
  const push = ease(local / 0.24);
  const pull = ease((local - 0.82) / 0.18);
  const mix = push * (1 - pull);
  return {
    zoom: lerp(full.zoom, target.zoom, mix),
    cx: lerp(full.cx, target.cx, mix),
    cy: lerp(full.cy, target.cy, mix),
  };
}

function drawLines(t, activeIndex) {
  if (activeIndex < 0) {
    ctx.save();
    ctx.globalAlpha = 0.06;
    for (const line of STATE.lines) strokePath(line.path, line.color, 10);
    ctx.restore();
    return;
  }
  for (const line of STATE.lines) {
    const i = line.index;
    if (i > activeIndex && activeIndex < 5) continue;
    if (i < activeIndex || activeIndex === 5) {
      ctx.save();
      ctx.globalAlpha = activeIndex === 5 ? 1 : 0.34;
      strokePath(line.path, '#FFFFFF', 20);
      strokePath(line.path, line.color, 11);
      ctx.restore();
    }
  }
  if (activeIndex >= 0 && activeIndex < 5) {
    const line = STATE.lines[activeIndex];
    const local = localEventT(t, activeIndex);
    const progress = easeInOut(local / 0.46);
    const points = partialPoints(line.path, progress);
    if (points.length >= 2) {
      ctx.save();
      ctx.shadowColor = line.color + '66';
      ctx.shadowBlur = 22;
      strokePath(points, '#FFFFFF', 24);
      strokePath(points, line.color, 13);
      ctx.restore();
    }
  }
}

function drawStations(t, activeIndex) {
  for (const station of STATE.stations) {
    const owner = STATE.lines.find((line) => line.stations.includes(station.id));
    if (!owner) continue;
    const i = owner.index;
    let alpha = 0;
    let scale = 1;
    if (activeIndex === 5) {
      alpha = 1;
    } else if (i < activeIndex) {
      alpha = 0.42;
    } else if (i === activeIndex) {
      const local = localEventT(t, activeIndex);
      const d = owner.stationDistances.get(station.id) ?? 0;
      const revealAt = 0.10 + (owner.total > 0 ? d / owner.total : 0) * 0.40;
      const k = ease((local - revealAt) / 0.10);
      alpha = k;
      scale = 0.6 + 0.4 * k;
    }
    if (alpha <= 0) continue;
    const lineColor = owner.color;
    ctx.save();
    ctx.globalAlpha = alpha;
    if (station.interchange) {
      ctx.beginPath();
      ctx.arc(station.x, station.y, 13 * scale, 0, Math.PI * 2);
      ctx.fillStyle = '#FFFFFF';
      ctx.fill();
      ctx.strokeStyle = '#202020';
      ctx.lineWidth = 3.6;
      ctx.stroke();
      ctx.beginPath();
      ctx.arc(station.x, station.y, 5 * scale, 0, Math.PI * 2);
      ctx.fillStyle = lineColor;
      ctx.fill();
    } else {
      ctx.beginPath();
      ctx.arc(station.x, station.y, 7 * scale, 0, Math.PI * 2);
      ctx.fillStyle = '#FFFFFF';
      ctx.fill();
      ctx.strokeStyle = lineColor;
      ctx.lineWidth = 2.8;
      ctx.stroke();
    }
    if (i === activeIndex && activeIndex < 5) {
      const local = localEventT(t, activeIndex);
      const pulse = Math.max(0, Math.sin(local * Math.PI * 8));
      if (station.interchange && local > 0.55) {
        ctx.beginPath();
        ctx.arc(station.x, station.y, 19 + pulse * 7, 0, Math.PI * 2);
        ctx.strokeStyle = lineColor + '55';
        ctx.lineWidth = 5;
        ctx.stroke();
      }
    }
    ctx.restore();
  }
}

function drawLabels(t, activeIndex) {
  for (const station of STATE.stations) {
    const placement = STATE.placements.get(station.id);
    if (!placement) continue;
    const owner = STATE.lines.find((line) => line.stations.includes(station.id));
    if (!owner) continue;
    let alpha = 0;
    if (activeIndex === 5) {
      alpha = station.interchange || station.priority >= 60 ? 0.92 : 0.55;
    } else if (owner.index === activeIndex) {
      const local = localEventT(t, activeIndex);
      const d = owner.stationDistances.get(station.id) ?? 0;
      const revealAt = 0.16 + (owner.total > 0 ? d / owner.total : 0) * 0.40;
      alpha = ease((local - revealAt) / 0.12);
    } else if (owner.index < activeIndex && (station.interchange || station.priority >= 60)) {
      alpha = 0.30;
    }
    if (alpha <= 0) continue;
    ctx.save();
    ctx.globalAlpha = alpha;
    if (placement.leader) {
      ctx.strokeStyle = 'rgba(40,40,40,0.32)';
      ctx.lineWidth = 1.4;
      ctx.beginPath();
      ctx.moveTo(station.x, station.y);
      ctx.lineTo(placement.textX, placement.textY);
      ctx.stroke();
    }
    ctx.fillStyle = station.interchange ? '#181818' : '#2D2D2D';
    ctx.font = `${placement.weight} ${placement.size}px ${FONT_CN}`;
    ctx.textAlign = placement.align;
    ctx.textBaseline = 'middle';
    ctx.fillText(station.name, placement.textX, placement.textY);
    ctx.restore();
  }
}

function drawTrain(t, activeIndex) {
  if (activeIndex < 0 || activeIndex >= 5) return;
  const line = STATE.lines[activeIndex];
  const local = localEventT(t, activeIndex);
  if (local < 0.50) return;
  const u = ((local - 0.50) / 0.50) % 1;
  const [x, y] = pointAlongPath(line.path, u * line.total);
  ctx.save();
  ctx.globalAlpha = 0.95;
  ctx.shadowColor = line.color;
  ctx.shadowBlur = 18;
  ctx.beginPath();
  ctx.arc(x, y, 7, 0, Math.PI * 2);
  ctx.fillStyle = '#FFFFFF';
  ctx.fill();
  ctx.strokeStyle = line.color;
  ctx.lineWidth = 3.5;
  ctx.stroke();
  ctx.restore();
}

function drawIntro(t) {
  const a = Math.min(ease(t / 0.7), ease((INTRO - t) / 0.6));
  if (a <= 0) return;
  ctx.save();
  ctx.globalAlpha = a;
  ctx.fillStyle = 'rgba(230,0,18,0.07)';
  ctx.font = `900 230px ${FONT_MONO}`;
  ctx.textAlign = 'right';
  ctx.fillText('12/26', 1040, 430);
  ctx.textAlign = 'left';
  ctx.fillStyle = '#171717';
  ctx.font = `800 66px ${FONT_CN}`;
  ctx.fillText('合肥轨道交通', 70, 190);
  ctx.fillStyle = '#D71920';
  ctx.font = `700 38px ${FONT_CN}`;
  ctx.fillText('对 12/26 情有独钟', 72, 258);
  ctx.fillStyle = '#777777';
  ctx.font = `500 20px ${FONT_CN}`;
  ctx.fillText('一条时间轴，看见一座城市的轨道生长', 74, 312);
  ctx.strokeStyle = '#D71920';
  ctx.lineWidth = 3;
  ctx.beginPath();
  ctx.moveTo(74, 335);
  ctx.lineTo(74 + 260 * ease((t - 0.5) / 1.2), 335);
  ctx.stroke();
  ctx.restore();
}

function roundRect(x, y, w, h, r) {
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.arcTo(x + w, y, x + w, y + h, r);
  ctx.arcTo(x + w, y + h, x, y + h, r);
  ctx.arcTo(x, y + h, x, y, r);
  ctx.arcTo(x, y, x + w, y, r);
  ctx.closePath();
}

function drawPanel(t, activeIndex) {
  if (activeIndex < 0 || activeIndex >= 5) return;
  const event = DATA.events[activeIndex];
  if (!event) return;
  const line = STATE.lines.find((l) => l.id === event.lineId) || STATE.lines[activeIndex];
  const local = localEventT(t, activeIndex);
  const enter = ease(local / 0.18);
  const exit = ease((local - 0.88) / 0.12);
  const alpha = enter * (1 - exit);
  const offsetY = (1 - enter) * 26 - exit * 26;
  if (alpha <= 0) return;
  const x = 54, y = 1440 + offsetY, w = 972, h = 330;
  ctx.save();
  ctx.globalAlpha = alpha;
  ctx.shadowColor = 'rgba(0,0,0,0.12)';
  ctx.shadowBlur = 28;
  ctx.shadowOffsetY = 12;
  ctx.fillStyle = 'rgba(255,255,255,0.95)';
  roundRect(x, y, w, h, 26);
  ctx.fill();
  ctx.shadowColor = 'transparent';
  ctx.strokeStyle = 'rgba(20,20,20,0.08)';
  ctx.lineWidth = 1.5;
  roundRect(x, y, w, h, 26);
  ctx.stroke();

  ctx.fillStyle = '#8C8C8C';
  ctx.font = `600 14px ${FONT_MONO}`;
  ctx.fillText('HEFEI METRO · OPENING TIMELINE', x + 38, y + 40);

  const parts = String(event.date).split('.');
  const year = parts[0] || String(event.year);
  const md = parts.slice(1).join('.');
  ctx.fillStyle = '#171717';
  ctx.font = `800 94px ${FONT_MONO}`;
  ctx.fillText(year, x + 36, y + 150);
  const yearW = ctx.measureText(year).width;
  ctx.fillStyle = '#D71920';
  ctx.font = `800 46px ${FONT_MONO}`;
  ctx.fillText('.' + md, x + 42 + yearW, y + 150);

  const chipY = y + 186;
  ctx.fillStyle = line.color;
  roundRect(x + 38, chipY, 106, 38, 19);
  ctx.fill();
  ctx.fillStyle = '#FFFFFF';
  ctx.font = `700 21px ${FONT_CN}`;
  ctx.textAlign = 'center';
  ctx.fillText(event.lineName, x + 91, chipY + 26);
  ctx.textAlign = 'left';

  ctx.fillStyle = '#262626';
  ctx.font = `700 27px ${FONT_CN}`;
  ctx.fillText(event.segment, x + 162, chipY + 28);
  ctx.fillStyle = '#8A8A8A';
  ctx.font = `500 15px ${FONT_CN}`;
  ctx.fillText('开通运营 · ' + (event.note || '新线加入'), x + 162, chipY + 56);

  const prevKm = activeIndex > 0 ? DATA.events[activeIndex - 1].km : 0;
  const km = lerp(prevKm, event.km, easeOutExpo(local / 0.62));
  ctx.textAlign = 'right';
  ctx.fillStyle = '#8C8C8C';
  ctx.font = `600 14px ${FONT_CN}`;
  ctx.fillText('累计里程', x + w - 42, y + 46);
  ctx.fillStyle = '#171717';
  ctx.font = `800 74px ${FONT_MONO}`;
  ctx.fillText(km.toFixed(1), x + w - 92, y + 150);
  ctx.fillStyle = '#777777';
  ctx.font = `600 22px ${FONT_MONO}`;
  ctx.fillText('km', x + w - 42, y + 150);
  ctx.textAlign = 'left';

  ctx.strokeStyle = 'rgba(20,20,20,0.09)';
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(x + 38, y + 246);
  ctx.lineTo(x + w - 38, y + 246);
  ctx.stroke();

  ctx.fillStyle = '#D71920';
  roundRect(x + 38, y + 266, 84, 30, 15);
  ctx.fill();
  ctx.fillStyle = '#FFFFFF';
  ctx.font = `700 16px ${FONT_MONO}`;
  ctx.textAlign = 'center';
  ctx.fillText('12/26', x + 80, y + 287);
  ctx.textAlign = 'left';
  ctx.fillStyle = '#8C8C8C';
  ctx.font = `500 15px ${FONT_CN}`;
  ctx.fillText('合肥轨道交通 · 历年新线开通', x + 140, y + 286);

  const years = DATA.events.map((e) => e.year);
  const railX0 = x + 560, railX1 = x + w - 42, railY = y + 281;
  ctx.strokeStyle = 'rgba(20,20,20,0.15)';
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.moveTo(railX0, railY);
  ctx.lineTo(railX1, railY);
  ctx.stroke();
  years.forEach((yr, i) => {
    const px = lerp(railX0, railX1, years.length === 1 ? 0 : i / (years.length - 1));
    const active = i === activeIndex;
    ctx.beginPath();
    ctx.arc(px, railY, active ? 7 : 4, 0, Math.PI * 2);
    ctx.fillStyle = active ? line.color : '#B8B8B8';
    ctx.fill();
    if (active) {
      ctx.beginPath();
      ctx.arc(px, railY, 12, 0, Math.PI * 2);
      ctx.strokeStyle = line.color + '66';
      ctx.lineWidth = 3;
      ctx.stroke();
    }
    ctx.fillStyle = active ? '#333333' : '#A0A0A0';
    ctx.font = `${active ? 700 : 500} 12px ${FONT_MONO}`;
    ctx.textAlign = 'center';
    ctx.fillText(String(yr), px, railY - 12);
    ctx.textAlign = 'left';
  });
  ctx.restore();
}

function drawHeader(t, activeIndex) {
  if (activeIndex < 0) return;
  const alpha = activeIndex === 5 ? 1 : 0.9;
  ctx.save();
  ctx.globalAlpha = alpha;
  ctx.fillStyle = '#1B1B1B';
  ctx.font = `800 34px ${FONT_CN}`;
  ctx.fillText(DATA.title, 66, 108);
  ctx.fillStyle = '#D71920';
  ctx.font = `700 20px ${FONT_CN}`;
  ctx.fillText(DATA.subtitle, 68, 142);
  ctx.textAlign = 'right';
  ctx.fillStyle = '#8C8C8C';
  ctx.font = `600 14px ${FONT_MONO}`;
  const count = activeIndex === 5 ? 5 : Math.min(5, activeIndex + 1);
  ctx.fillText(`${String(count).padStart(2, '0')} / 05 LINES`, 1014, 112);
  ctx.fillStyle = '#B0B0B0';
  ctx.font = `500 13px ${FONT_CN}`;
  ctx.fillText('2016 — 2022', 1014, 140);
  ctx.textAlign = 'left';
  ctx.restore();
}

function drawOutro(t) {
  const local = clamp((t - OUTRO_START) / (DURATION - OUTRO_START), 0, 1);
  const a = ease(local / 0.25);
  if (a <= 0) return;
  ctx.save();
  ctx.globalAlpha = a;
  ctx.fillStyle = 'rgba(255,255,255,0.88)';
  roundRect(54, 1425, 972, 300, 26);
  ctx.fill();
  ctx.strokeStyle = 'rgba(20,20,20,0.08)';
  ctx.lineWidth = 1.5;
  roundRect(54, 1425, 972, 300, 26);
  ctx.stroke();
  ctx.fillStyle = '#171717';
  ctx.font = `800 50px ${FONT_CN}`;
  ctx.fillText('同一天，醒来的城市', 92, 1510);
  ctx.fillStyle = '#6F6F6F';
  ctx.font = `500 20px ${FONT_CN}`;
  ctx.fillText('12 月 26 日，合肥轨道交通的开通记忆', 94, 1552);
  const stats = [
    ['05', '条线路'],
    ['171.2', '公里'],
    ['20', '座车站'],
  ];
  stats.forEach(([num, label], i) => {
    const x = 96 + i * 300;
    ctx.fillStyle = '#D71920';
    ctx.font = `800 54px ${FONT_MONO}`;
    ctx.fillText(num, x, 1658);
    ctx.fillStyle = '#7A7A7A';
    ctx.font = `500 18px ${FONT_CN}`;
    ctx.fillText(label, x + 4, 1688);
  });
  ctx.restore();
}

function drawSource() {
  ctx.save();
  ctx.fillStyle = '#9A9A9A';
  ctx.font = `500 14px ${FONT_CN}`;
  ctx.fillText(`${DATA.source} · 数据截至 ${DATA.as_of}`, 70, 1836);
  ctx.fillStyle = '#B5B5B5';
  ctx.font = `400 12px ${FONT_CN}`;
  ctx.fillText(DATA.disclaimer, 70, 1864);
  ctx.textAlign = 'right';
  ctx.fillStyle = '#B7B7B7';
  ctx.font = `600 12px ${FONT_MONO}`;
  ctx.fillText('HIMIND / TRANSIT TIMELINE', W - 70, 1864);
  ctx.restore();
}

function renderAt(t) {
  const time = clamp(t, 0, DURATION);
  const activeIndex = activeIndexAt(time);
  const cam = cameraAt(time, activeIndex);
  drawBackground();
  ctx.save();
  ctx.translate(cam.cx, cam.cy);
  ctx.scale(cam.zoom, cam.zoom);
  ctx.translate(-cam.cx, -cam.cy);
  drawRiver();
  drawLines(time, activeIndex);
  drawStations(time, activeIndex);
  drawLabels(time, activeIndex);
  drawTrain(time, activeIndex);
  ctx.restore();
  drawHeader(time, activeIndex);
  drawIntro(time);
  drawPanel(time, activeIndex);
  drawOutro(time);
  drawSource();
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
    const size = station.interchange ? 27 : station.priority >= 60 ? 24 : 21;
    next.set(station.id, {
      x, y, w, h,
      textX: x,
      textY: y + h / 2,
      align: 'left',
      size,
      weight: station.interchange ? 700 : 500,
      leader: Array.isArray(item.leader) && item.leader.length >= 2,
    });
  }
  if (next.size) state.placements = next;
}

async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) DATA = {...DATA, ...(await response.json())};
  } catch {
    // Keep minimal fallback.
  }
  STATE = buildState(DATA);
  try {
    const labelsResponse = await fetch('/input/labels.json', {cache: 'no-store'});
    if (labelsResponse.ok) applyPrecomputedPlacements(STATE, await labelsResponse.json());
  } catch {
    // Fall back to deterministic local placement.
  }
  window.__DEBUG__ = {
    events: DATA.events.length,
    lines: STATE.lines.length,
    stations: STATE.stations.length,
    placements: STATE.placements.size,
  };
  window.renderAt = renderAt;
  window.__DURATION__ = DURATION;
  window.__FPS__ = FPS;
  await document.fonts.ready;
  renderAt(0);
  window.__READY__ = true;
}

loadData();
