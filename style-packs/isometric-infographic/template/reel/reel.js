const W = 1080, H = 1920, FPS = 30, DURATION = 10;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const DEFAULT = {
  title: '系统结构',
  subtitle: 'ISOMETRIC INFOGRAPHIC',
  nodes: [
    {label: '需求', value: '01'}, {label: '计划', value: '02'}, {label: '审查', value: '03'}, {label: '成片', value: '04'},
  ],
  conclusion: '结构清楚，执行才会稳定。',
  source: 'HIMIND / ISOMETRIC SYSTEM',
};
let DATA = DEFAULT;
async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) {
      const value = await response.json();
      const nodes = Array.isArray(value.nodes) ? value.nodes.slice(0, 4) : [];
      while (nodes.length < 4) { const f = DEFAULT.nodes[nodes.length]; const item = nodes[nodes.length] || {}; nodes.push({label:String(item.label || f.label), value:String(item.value || f.value)}); }
      DATA = {title:String(value.title || DEFAULT.title), subtitle:String(value.subtitle || DEFAULT.subtitle), nodes, conclusion:String(value.conclusion || DEFAULT.conclusion), source:String(value.source || DEFAULT.source)};
    }
  } catch { DATA = DEFAULT; }
}
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const ease = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
const C = {bg:'#eef0ea',ink:'#26312f',muted:'#78827d',grid:'rgba(38,49,47,.08)',blue:'#3b82c4',orange:'#d98324',green:'#2d8a68',red:'#c54a3d',paper:'#fffdf7'};
const COLORS = [C.blue, C.orange, C.green, C.red];
const iso = (x, y, z = 0) => ({x: W/2 + (x - y) * 0.86, y: 500 + (x + y) * 0.48 - z});
function bg() {
  ctx.fillStyle = C.bg; ctx.fillRect(0, 0, W, H);
  ctx.save(); ctx.strokeStyle = C.grid; ctx.lineWidth = 1;
  for (let x = 0; x < W; x += 36) { ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, H); ctx.stroke(); }
  for (let y = 0; y < H; y += 36) { ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(W, y); ctx.stroke(); }
  ctx.restore();
  const glow = ctx.createRadialGradient(W/2, 1000, 40, W/2, 1000, 700); glow.addColorStop(0, 'rgba(59,130,196,.10)'); glow.addColorStop(1, 'rgba(59,130,196,0)'); ctx.fillStyle = glow; ctx.fillRect(0, 0, W, H);
}
function title(t) {
  const p = ease(t / 1.2); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.textAlign = 'center'; ctx.fillStyle = C.ink; ctx.font = '700 76px "Microsoft YaHei", sans-serif'; ctx.fillText(DATA.title, W/2, 260);
  ctx.fillStyle = C.muted; ctx.font = '600 22px "Courier New",monospace'; ctx.fillText(DATA.subtitle, W/2, 310);
  ctx.strokeStyle = C.orange; ctx.lineWidth = 5; ctx.beginPath(); ctx.moveTo(W/2 - 150, 344); ctx.lineTo(W/2 - 150 + 300 * p, 344); ctx.stroke(); ctx.restore();
}
function block(index, x, y, t) {
  const start = 1.3 + index * 0.9; const p = ease((t - start) / 0.8); if (p <= 0) return null;
  const h = 120 * p, top = iso(x, y, h), left = iso(x, y, 0), right = iso(x + 200, y, 0), far = iso(x + 200, y + 160, 0), back = iso(x, y + 160, 0);
  const color = COLORS[index];
  ctx.save(); ctx.globalAlpha = clamp(p * 1.2, 0, 1);
  ctx.fillStyle = 'rgba(38,49,47,.12)'; ctx.beginPath(); ctx.ellipse(iso(x + 100, y + 80, 0).x, iso(x + 100, y + 80, 0).y + 20, 130, 44, 0, 0, Math.PI * 2); ctx.fill();
  ctx.fillStyle = color; ctx.beginPath(); ctx.moveTo(left.x, left.y); ctx.lineTo(back.x, back.y); ctx.lineTo(back.x, back.y - h); ctx.lineTo(left.x, left.y - h); ctx.closePath(); ctx.fill();
  ctx.fillStyle = 'rgba(0,0,0,.18)'; ctx.beginPath(); ctx.moveTo(back.x, back.y); ctx.lineTo(far.x, far.y); ctx.lineTo(far.x, far.y - h); ctx.lineTo(back.x, back.y - h); ctx.closePath(); ctx.fill();
  ctx.fillStyle = '#fffdf7'; ctx.beginPath(); ctx.moveTo(left.x, left.y - h); ctx.lineTo(back.x, back.y - h); ctx.lineTo(far.x, far.y - h); ctx.lineTo(right.x, right.y - h); ctx.closePath(); ctx.fill();
  ctx.strokeStyle = color; ctx.lineWidth = 3; ctx.stroke();
  ctx.fillStyle = C.ink; ctx.font = '700 30px "Microsoft YaHei", sans-serif'; ctx.textAlign = 'center'; ctx.fillText(DATA.nodes[index].label, (left.x + far.x)/2, (left.y + far.y)/2 - h + 10);
  ctx.fillStyle = color; ctx.font = '700 22px "Courier New",monospace'; ctx.fillText(DATA.nodes[index].value, (left.x + far.x)/2, (left.y + far.y)/2 - h + 44);
  ctx.restore();
  return {x: (left.x + far.x)/2, y: (left.y + far.y)/2 - h, color};
}
function connectors(t) {
  const positions = [[-170,-120],[80,-120],[280,30],[20,150]];
  for (let i = 0; i < 3; i++) {
    const p = ease((t - (2.2 + i * 0.9)) / 0.6); if (p <= 0) continue;
    const a = iso(positions[i][0] + 100, positions[i][1] + 80, 110); const b = iso(positions[i+1][0] + 100, positions[i+1][1] + 80, 110);
    ctx.save(); ctx.globalAlpha = p; ctx.strokeStyle = C.ink; ctx.lineWidth = 3; ctx.setLineDash([10, 8]); ctx.beginPath(); ctx.moveTo(a.x, a.y); ctx.lineTo(a.x + (b.x - a.x) * p, a.y + (b.y - a.y) * p); ctx.stroke(); ctx.setLineDash([]);
    if (p > 0.8) { const angle = Math.atan2(b.y - a.y, b.x - a.x); ctx.fillStyle = C.ink; ctx.beginPath(); ctx.moveTo(b.x, b.y); ctx.lineTo(b.x - 18 * Math.cos(angle - .4), b.y - 18 * Math.sin(angle - .4)); ctx.lineTo(b.x - 18 * Math.cos(angle + .4), b.y - 18 * Math.sin(angle + .4)); ctx.closePath(); ctx.fill(); }
    ctx.restore();
  }
}
function conclusion(t) {
  const p = ease((t - 7.8) / 0.7); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.textAlign = 'center'; ctx.fillStyle = C.ink; ctx.font = '700 46px "Microsoft YaHei", sans-serif'; ctx.fillText(DATA.conclusion, W/2, 1690);
  ctx.fillStyle = C.muted; ctx.font = '500 18px "Courier New",monospace'; ctx.fillText(DATA.source, W/2, 1810); ctx.restore();
}
function renderAt(t) {
  const time = clamp(t, 0, DURATION); ctx.clearRect(0, 0, W, H); bg(); title(time);
  const pos = [[-170,-120],[80,-120],[280,30],[20,150]];
  pos.forEach((p, i) => block(i, p[0], p[1], time)); connectors(time); conclusion(time);
}
window.renderAt = renderAt; window.__DURATION__ = DURATION; window.__FPS__ = FPS;
loadData().then(() => document.fonts.ready.then(() => { renderAt(0); window.__READY__ = true; }));
