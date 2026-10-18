const W = 1080, H = 1920, FPS = 30, DURATION = 10;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const DEFAULT = {
  title: '一条路线，穿过山海',
  subtitle: 'CINEMATIC ROUTE / ANHUI',
  route_name: '皖南高铁走廊',
  distance: '约 320 km',
  nodes: [
    {name: '合肥', x: 0.18, y: 0.64, year: '2015'},
    {name: '芜湖', x: 0.36, y: 0.56, year: '2015'},
    {name: '宣城', x: 0.55, y: 0.48, year: '2020'},
    {name: '黄山', x: 0.72, y: 0.60, year: '2024'},
    {name: '杭州', x: 0.86, y: 0.42, year: '2024'},
  ],
  source: 'SOURCE / SCHEMATIC ROUTE',
};
let DATA = DEFAULT;
async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) {
      const value = await response.json();
      const nodes = Array.isArray(value.nodes) ? value.nodes.slice(0, 7).filter((x) => x && x.name != null).map((x) => ({name:String(x.name), x:Number(x.x || 0), y:Number(x.y || 0), year:String(x.year || '')})) : [];
      if (nodes.length >= 2) DATA = {title:String(value.title || DEFAULT.title), subtitle:String(value.subtitle || DEFAULT.subtitle), route_name:String(value.route_name || DEFAULT.route_name), distance:String(value.distance || DEFAULT.distance), nodes, source:String(value.source || DEFAULT.source)};
    }
  } catch { DATA = DEFAULT; }
}
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const ease = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
const hash = (n) => { const x = Math.sin(n * 127.1 + 311.7) * 43758.5453; return x - Math.floor(x); };
const C = {bg:'#07181b',bg2:'#0c2b2e',ink:'#eefbf8',muted:'rgba(238,251,248,.50)',dim:'rgba(238,251,248,.22)',cyan:'#54e0d0',gold:'#e4b363',red:'#ff7a6b',line:'rgba(84,224,208,.14)'};
function bg(t) {
  const g = ctx.createLinearGradient(0, 0, W, H); g.addColorStop(0, '#061518'); g.addColorStop(0.55, C.bg2); g.addColorStop(1, '#061518'); ctx.fillStyle = g; ctx.fillRect(0, 0, W, H);
  ctx.save(); ctx.strokeStyle = C.line; ctx.lineWidth = 1;
  for (let x = 0; x < W; x += 54) { ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, H); ctx.stroke(); }
  for (let y = 0; y < H; y += 54) { ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(W, y); ctx.stroke(); }
  ctx.restore();
  ctx.save();
  for (let k = 0; k < 12; k++) {
    ctx.strokeStyle = `rgba(84,224,208,${(0.05 + k * 0.006).toFixed(3)})`; ctx.lineWidth = 1.2; ctx.beginPath();
    for (let i = 0; i <= 80; i++) { const x = i / 80 * W; const y = 800 + Math.sin(i * 0.18 + k * 0.9) * 90 + Math.sin(i * 0.05 + k) * 140 + k * 22; if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y); }
    ctx.stroke();
  }
  ctx.restore();
  const glow = ctx.createRadialGradient(W*0.65, 1000, 0, W*0.65, 1000, 700); glow.addColorStop(0, 'rgba(84,224,208,.12)'); glow.addColorStop(1, 'rgba(84,224,208,0)'); ctx.fillStyle = glow; ctx.fillRect(0, 0, W, H);
}
function header(t) {
  const p = ease(t / 1.2); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.ink; ctx.font = '700 68px "Microsoft YaHei", sans-serif'; ctx.fillText(DATA.title, 80, 220);
  ctx.fillStyle = C.cyan; ctx.font = '600 22px "Courier New",monospace'; ctx.fillText(DATA.subtitle, 84, 266);
  ctx.strokeStyle = 'rgba(84,224,208,.5)'; ctx.lineWidth = 3; ctx.beginPath(); ctx.moveTo(84, 300); ctx.lineTo(84 + 420 * p, 300); ctx.stroke(); ctx.restore();
}
function point(node) { return {x: 70 + clamp(node.x, 0, 1) * (W - 140), y: 600 + clamp(node.y, 0, 1) * 900}; }
function routePath(points, p) {
  if (points.length < 2 || p <= 0) return;
  ctx.save(); ctx.beginPath(); ctx.rect(0, 0, W * p, H); ctx.clip();
  for (let pass = 0; pass < 2; pass++) {
    ctx.beginPath(); ctx.moveTo(points[0].x, points[0].y);
    for (let i = 1; i < points.length; i++) { const prev = points[i-1], cur = points[i]; const mx = (prev.x + cur.x) / 2, my = (prev.y + cur.y) / 2 - 50; ctx.quadraticCurveTo(mx, my, cur.x, cur.y); }
    ctx.strokeStyle = pass === 0 ? 'rgba(84,224,208,.20)' : C.cyan; ctx.lineWidth = pass === 0 ? 22 : 5; ctx.lineCap = 'round'; ctx.shadowColor = C.cyan; ctx.shadowBlur = pass === 0 ? 30 : 18; ctx.stroke();
  }
  ctx.shadowBlur = 0; ctx.restore();
}
function nodes(t, points) {
  DATA.nodes.forEach((node, i) => {
    const p = ease((t - (1.9 + i * 0.8)) / 0.55); if (p <= 0) return;
    const pt = points[i]; const last = i === points.length - 1;
    ctx.save(); ctx.globalAlpha = p;
    ctx.fillStyle = last ? C.gold : C.cyan; ctx.shadowColor = last ? C.gold : C.cyan; ctx.shadowBlur = 22;
    ctx.beginPath(); ctx.arc(pt.x, pt.y, 12 + 4 * p, 0, Math.PI * 2); ctx.fill(); ctx.shadowBlur = 0;
    ctx.strokeStyle = last ? C.gold : C.cyan; ctx.lineWidth = 2; ctx.beginPath(); ctx.arc(pt.x, pt.y, 24 + 8 * p, 0, Math.PI * 2); ctx.stroke();
    ctx.fillStyle = C.ink; ctx.font = '700 28px "Microsoft YaHei", sans-serif'; ctx.fillText(node.name, pt.x + 34, pt.y + 8);
    ctx.fillStyle = C.muted; ctx.font = '500 17px "Courier New",monospace'; ctx.fillText(node.year, pt.x + 34, pt.y + 36);
    ctx.restore();
  });
}
function summary(t) {
  const p = ease((t - 7.4) / 0.7); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = 'rgba(4,20,22,.72)'; ctx.fillRect(80, 1600, 920, 150);
  ctx.strokeStyle = C.cyan; ctx.lineWidth = 2; ctx.strokeRect(80, 1600, 920, 150);
  ctx.fillStyle = C.cyan; ctx.font = '600 18px "Courier New",monospace'; ctx.fillText('ROUTE / DISTANCE', 110, 1640);
  ctx.fillStyle = C.ink; ctx.font = '700 42px "Microsoft YaHei", sans-serif'; ctx.fillText(DATA.route_name, 110, 1700);
  ctx.textAlign = 'right'; ctx.fillStyle = C.gold; ctx.font = '700 36px "Courier New",monospace'; ctx.fillText(DATA.distance, 970, 1700); ctx.restore();
}
function footer(t) {
  const p = ease((t - 8.4) / 0.5); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.muted; ctx.font = '500 17px "Courier New",monospace'; ctx.fillText(DATA.source, 84, 1815);
  ctx.fillStyle = C.cyan; ctx.fillRect(84, 1780, 180, 4);
  ctx.textAlign = 'right'; ctx.fillText('SCHEMATIC / NOT TO SCALE', W - 84, 1815); ctx.restore();
}
function renderAt(t) {
  const time = clamp(t, 0, DURATION); ctx.clearRect(0, 0, W, H); bg(time); header(time);
  const points = DATA.nodes.map(point); const p = ease((time - 1.6) / 5.6); routePath(points, p); nodes(time, points); summary(time); footer(time);
}
window.renderAt = renderAt; window.__DURATION__ = DURATION; window.__FPS__ = FPS;
loadData().then(() => document.fonts.ready.then(() => { renderAt(0); window.__READY__ = true; }));
