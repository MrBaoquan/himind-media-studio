const W = 1080, H = 1920, FPS = 30, DURATION = 10;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const DEFAULT = {
  title: '四格故事',
  panels: [
    {label: '01 / 起点', caption: '一个想法', bubble: '开始！'},
    {label: '02 / 转折', caption: '遇到阻力', bubble: '等一下…'},
    {label: '03 / 行动', caption: '做出选择', bubble: '就是现在！'},
    {label: '04 / 结果', caption: '得到答案', bubble: '原来如此。'},
  ],
  conclusion: '每一格，都是一次推进。',
  source: 'HIMIND COMICS',
};
let DATA = DEFAULT;
async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) {
      const value = await response.json();
      const panels = Array.isArray(value.panels) ? value.panels.slice(0, 4) : [];
      while (panels.length < 4) {
        const fallback = DEFAULT.panels[panels.length];
        const item = panels[panels.length] || {};
        panels.push({label: String(item.label || fallback.label), caption: String(item.caption || fallback.caption), bubble: String(item.bubble || fallback.bubble)});
      }
      DATA = {title: String(value.title || DEFAULT.title), panels, conclusion: String(value.conclusion || DEFAULT.conclusion), source: String(value.source || DEFAULT.source)};
    }
  } catch { DATA = DEFAULT; }
}
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const ease = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
const back = (v) => { const x = clamp(v, 0, 1) - 1; return 1 + x * x * (2.5 * x + 1.5); };
const hash = (n) => { const x = Math.sin(n * 127.1 + 311.7) * 43758.5453; return x - Math.floor(x); };
const C = {paper:'#f6f1e6',ink:'#111111',red:'#e63946',yellow:'#ffd23f',blue:'#2f6fed',muted:'#726a5c'};
function halftone(x, y, w, h, color, gap = 14) {
  ctx.save(); ctx.fillStyle = color;
  for (let yy = y; yy < y + h; yy += gap) for (let xx = x; xx < x + w; xx += gap) {
    const r = 1.2 + hash(xx * 0.7 + yy * 1.3) * 2.2; ctx.beginPath(); ctx.arc(xx, yy, r, 0, Math.PI * 2); ctx.fill();
  }
  ctx.restore();
}
function paper() {
  ctx.fillStyle = C.paper; ctx.fillRect(0, 0, W, H);
  ctx.save(); ctx.strokeStyle = 'rgba(17,17,17,.06)'; ctx.lineWidth = 1;
  for (let x = 0; x < W; x += 28) { ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, H); ctx.stroke(); }
  for (let y = 0; y < H; y += 28) { ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(W, y); ctx.stroke(); }
  ctx.restore();
}
function header(t) {
  const p = ease(t / 1.1); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p;
  ctx.fillStyle = C.ink; ctx.font = '900 82px Impact, "Arial Black", sans-serif'; ctx.textAlign = 'center';
  ctx.fillText(DATA.title, W/2, 220 + 20 * (1 - p));
  ctx.fillStyle = C.red; ctx.fillRect(310, 250, 460, 8);
  ctx.font = '700 22px "Courier New",monospace'; ctx.fillStyle = C.muted; ctx.fillText('HIMIND / COMIC PANEL', W/2, 292);
  ctx.restore();
}
function burst(x, y, r, color, seed) {
  ctx.save(); ctx.translate(x, y); ctx.rotate(-0.12); ctx.fillStyle = color; ctx.beginPath();
  for (let i = 0; i < 20; i++) { const a = i / 20 * Math.PI * 2, rr = r * (0.78 + hash(seed + i) * 0.28); ctx.lineTo(Math.cos(a) * rr, Math.sin(a) * rr); }
  ctx.closePath(); ctx.fill(); ctx.strokeStyle = C.ink; ctx.lineWidth = 5; ctx.stroke(); ctx.restore();
}
function panel(index, x, y, w, h, t) {
  const start = 1.15 + index * 0.85; const p = back((t - start) / 0.65); if (p <= 0) return;
  const data = DATA.panels[index];
  ctx.save(); ctx.globalAlpha = clamp(p, 0, 1); ctx.translate(x + w/2, y + h/2); ctx.scale(p, p); ctx.translate(-w/2, -h/2);
  ctx.fillStyle = '#fffdf7'; ctx.fillRect(0, 0, w, h);
  halftone(18, 18, w - 36, h - 36, 'rgba(17,17,17,.08)', 15);
  ctx.fillStyle = index === 3 ? 'rgba(255,210,63,.28)' : index === 2 ? 'rgba(47,111,237,.16)' : 'rgba(230,57,70,.10)';
  ctx.fillRect(18, 18, w - 36, h - 36);
  ctx.strokeStyle = C.ink; ctx.lineWidth = 8; ctx.strokeRect(0, 0, w, h);
  ctx.fillStyle = C.ink; ctx.font = '900 25px Impact, "Arial Black", sans-serif'; ctx.fillText(data.label, 34, 56);
  ctx.fillStyle = C.ink; ctx.font = '900 54px "Microsoft YaHei", sans-serif'; ctx.fillText(data.caption, 34, h - 54);
  // Speech bubble.
  const bx = 44, by = 90, bw = Math.min(w - 88, 300), bh = 110;
  ctx.save(); ctx.fillStyle = '#ffffff'; ctx.strokeStyle = C.ink; ctx.lineWidth = 5;
  ctx.beginPath(); ctx.roundRect(bx, by, bw, bh, 26); ctx.fill(); ctx.stroke();
  ctx.beginPath(); ctx.moveTo(bx + 40, by + bh); ctx.lineTo(bx + 24, by + bh + 34); ctx.lineTo(bx + 78, by + bh); ctx.closePath(); ctx.fill(); ctx.stroke();
  ctx.fillStyle = C.ink; ctx.font = '700 28px "Microsoft YaHei", sans-serif'; ctx.fillText(data.bubble, bx + 24, by + 68); ctx.restore();
  // Icon.
  const cx = w - 100, cy = h - 130;
  if (index === 0) { ctx.fillStyle = C.red; ctx.beginPath(); ctx.moveTo(cx, cy - 52); ctx.lineTo(cx + 20, cy - 8); ctx.lineTo(cx + 54, cy - 4); ctx.lineTo(cx + 28, cy + 22); ctx.lineTo(cx + 36, cy + 58); ctx.lineTo(cx, cy + 36); ctx.lineTo(cx - 36, cy + 58); ctx.lineTo(cx - 28, cy + 22); ctx.lineTo(cx - 54, cy - 4); ctx.lineTo(cx - 20, cy - 8); ctx.closePath(); ctx.fill(); ctx.strokeStyle = C.ink; ctx.lineWidth = 4; ctx.stroke(); }
  if (index === 1) { ctx.strokeStyle = C.ink; ctx.lineWidth = 6; ctx.beginPath(); ctx.ellipse(cx, cy, 54, 32, 0, 0, Math.PI * 2); ctx.stroke(); ctx.fillStyle = C.blue; ctx.beginPath(); ctx.arc(cx, cy, 18, 0, Math.PI * 2); ctx.fill(); }
  if (index === 2) { ctx.strokeStyle = C.ink; ctx.lineWidth = 8; ctx.beginPath(); ctx.moveTo(cx - 50, cy + 30); ctx.lineTo(cx + 30, cy - 30); ctx.lineTo(cx + 18, cy - 40); ctx.moveTo(cx + 30, cy - 30); ctx.lineTo(cx + 40, cy - 8); ctx.stroke(); }
  if (index === 3) { burst(cx, cy, 56, C.yellow, 700 + index); }
  ctx.restore();
}
function actionLines(t) {
  const p = ease((t - 6.0) / 0.7); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.strokeStyle = C.ink; ctx.lineWidth = 5;
  for (let i = 0; i < 18; i++) { const a = i / 18 * Math.PI * 2, r1 = 300 + hash(i) * 130, r2 = r1 + 120; ctx.beginPath(); ctx.moveTo(W/2 + Math.cos(a) * r1, 980 + Math.sin(a) * r1 * 0.6); ctx.lineTo(W/2 + Math.cos(a) * r2, 980 + Math.sin(a) * r2 * 0.6); ctx.stroke(); }
  ctx.restore();
  burst(780, 1090, 110, C.yellow, 901);
  ctx.fillStyle = C.red; ctx.font = '900 54px Impact, "Arial Black", sans-serif'; ctx.textAlign = 'center'; ctx.fillText('POW!', 780, 1110); ctx.textAlign = 'left';
}
function conclusion(t) {
  const p = ease((t - 7.5) / 0.7); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.ink; ctx.fillRect(90, 1640, 900, 120);
  ctx.fillStyle = C.yellow; ctx.fillRect(110, 1660, 860, 8);
  ctx.fillStyle = C.paper; ctx.font = '900 44px "Microsoft YaHei", sans-serif'; ctx.textAlign = 'center'; ctx.fillText(DATA.conclusion, W/2, 1718);
  ctx.fillStyle = C.paper; ctx.font = '600 18px "Courier New",monospace'; ctx.fillText(DATA.source, W/2, 1820); ctx.restore();
}
function renderAt(t) {
  const time = clamp(t, 0, DURATION); ctx.clearRect(0, 0, W, H); paper(); header(time);
  const w = 400, h = 520;
  panel(0, 100, 380, w, h, time); panel(1, 580, 380, w, h, time); panel(2, 100, 960, w, h, time); panel(3, 580, 960, w, h, time);
  actionLines(time); conclusion(time);
}
window.renderAt = renderAt; window.__DURATION__ = DURATION; window.__FPS__ = FPS;
loadData().then(() => document.fonts.ready.then(() => { renderAt(0); window.__READY__ = true; }));
