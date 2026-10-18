const W = 1080, H = 1920, FPS = 30, DURATION = 10;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const DEFAULT = {
  title: '系统蓝图',
  subtitle: 'TECHNICAL BLUEPRINT / REV A',
  modules: [
    {label: '01 / INPUT', value: '需求', detail: '目标、约束与数据'},
    {label: '02 / PLAN', value: '计划', detail: '分镜、时长与契约'},
    {label: '03 / REVIEW', value: '审查', detail: '预览、反馈与修订'},
    {label: '04 / OUTPUT', value: '成片', detail: '质量门禁与交付'},
  ],
  source: 'HIMIND ENGINEERING',
};
let DATA = DEFAULT;
async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) {
      const value = await response.json();
      const modules = Array.isArray(value.modules) ? value.modules.slice(0, 4) : [];
      while (modules.length < 4) {
        const fallback = DEFAULT.modules[modules.length]; const item = modules[modules.length] || {};
        modules.push({label: String(item.label || fallback.label), value: String(item.value || fallback.value), detail: String(item.detail || fallback.detail)});
      }
      DATA = {title: String(value.title || DEFAULT.title), subtitle: String(value.subtitle || DEFAULT.subtitle), modules, source: String(value.source || DEFAULT.source)};
    }
  } catch { DATA = DEFAULT; }
}
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const ease = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
const hash = (n) => { const x = Math.sin(n * 127.1 + 311.7) * 43758.5453; return x - Math.floor(x); };
const C = {bg:'#0a2a4f',bg2:'#071e3b',line:'rgba(194,232,255,.74)',dim:'rgba(194,232,255,.28)',white:'#e6f6ff',cyan:'#53d8ff',yellow:'#ffd166',red:'#ff6b6b'};
function grid() {
  ctx.fillStyle = C.bg; ctx.fillRect(0, 0, W, H);
  const g = ctx.createLinearGradient(0, 0, W, H); g.addColorStop(0, 'rgba(83,216,255,.06)'); g.addColorStop(1, 'rgba(6,20,40,.4)'); ctx.fillStyle = g; ctx.fillRect(0, 0, W, H);
  ctx.save(); ctx.lineWidth = 1;
  for (let x = 0; x < W; x += 20) { ctx.strokeStyle = x % 100 === 0 ? C.dim : 'rgba(194,232,255,.10)'; ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, H); ctx.stroke(); }
  for (let y = 0; y < H; y += 20) { ctx.strokeStyle = y % 100 === 0 ? C.dim : 'rgba(194,232,255,.10)'; ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(W, y); ctx.stroke(); }
  ctx.restore();
}
function chrome(t) {
  ctx.save(); ctx.strokeStyle = C.line; ctx.lineWidth = 2; ctx.strokeRect(48, 48, W - 96, H - 96);
  ctx.strokeStyle = C.dim; ctx.strokeRect(70, 70, W - 140, H - 140);
  ctx.fillStyle = C.cyan; ctx.font = '700 22px "Courier New",monospace'; ctx.fillText('PROJECT / HIMIND', 92, 112);
  ctx.fillStyle = C.white; ctx.font = '400 16px "Courier New",monospace'; ctx.fillText('DRAWING NO. HM-2026-001', 92, 140);
  ctx.textAlign = 'right'; ctx.fillStyle = C.yellow; ctx.fillText('REV A', W - 92, 112); ctx.fillStyle = C.dim; ctx.fillText('SCALE 1:100', W - 92, 140); ctx.textAlign = 'left';
  ctx.strokeStyle = C.dim; ctx.beginPath(); ctx.moveTo(92, 166); ctx.lineTo(W - 92, 166); ctx.stroke();
  ctx.restore();
}
function title(t) {
  const p = ease(t / 1.2); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.white; ctx.font = '700 76px "Courier New",monospace'; ctx.fillText(DATA.title, 92, 300);
  ctx.fillStyle = C.cyan; ctx.font = '500 24px "Courier New",monospace'; ctx.fillText(DATA.subtitle, 96, 350);
  ctx.strokeStyle = C.yellow; ctx.lineWidth = 4; ctx.beginPath(); ctx.moveTo(96, 382); ctx.lineTo(96 + 620 * p, 382); ctx.stroke();
  ctx.restore();
}
function crosshair(x, y, r, color) {
  ctx.save(); ctx.strokeStyle = color; ctx.lineWidth = 2; ctx.beginPath(); ctx.arc(x, y, r, 0, Math.PI * 2); ctx.stroke();
  ctx.beginPath(); ctx.moveTo(x - r - 16, y); ctx.lineTo(x + r + 16, y); ctx.moveTo(x, y - r - 16); ctx.lineTo(x, y + r + 16); ctx.stroke(); ctx.restore();
}
function moduleCard(index, x, y, w, h, t) {
  const start = 1.5 + index * 1.05; const p = ease((t - start) / 0.8); if (p <= 0) return;
  const data = DATA.modules[index];
  ctx.save(); ctx.globalAlpha = clamp(p * 1.2, 0, 1);
  ctx.beginPath(); ctx.rect(x, y, w * p, h); ctx.clip();
  ctx.fillStyle = 'rgba(7,30,59,.55)'; ctx.fillRect(x, y, w, h);
  ctx.strokeStyle = C.line; ctx.lineWidth = 3; ctx.strokeRect(x, y, w, h);
  ctx.strokeStyle = C.dim; ctx.lineWidth = 1; ctx.strokeRect(x + 14, y + 14, w - 28, h - 28);
  ctx.fillStyle = C.cyan; ctx.font = '700 18px "Courier New",monospace'; ctx.fillText(data.label, x + 28, y + 44);
  ctx.fillStyle = C.white; ctx.font = '700 50px "Microsoft YaHei", sans-serif'; ctx.fillText(data.value, x + 28, y + 120);
  ctx.fillStyle = 'rgba(230,246,255,.70)'; ctx.font = '400 23px "Microsoft YaHei", sans-serif'; ctx.fillText(data.detail, x + 28, y + 165);
  crosshair(x + w - 64, y + h - 64, 25, C.yellow);
  // Dimension line.
  ctx.strokeStyle = C.dim; ctx.lineWidth = 1; ctx.beginPath(); ctx.moveTo(x + 28, y + h - 26); ctx.lineTo(x + w - 28, y + h - 26); ctx.stroke();
  ctx.fillStyle = C.dim; ctx.font = '400 13px "Courier New",monospace'; ctx.fillText(`${w} × ${h}`, x + 30, y + h - 32);
  ctx.restore();
}
function annotations(t) {
  const p = ease((t - 6.2) / 0.8); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.strokeStyle = C.yellow; ctx.lineWidth = 2; ctx.setLineDash([10, 8]);
  ctx.beginPath(); ctx.moveTo(870, 560); ctx.lineTo(990, 560); ctx.lineTo(990, 700); ctx.stroke(); ctx.setLineDash([]);
  ctx.fillStyle = C.yellow; ctx.font = '700 16px "Courier New",monospace'; ctx.fillText('CHECK', 914, 548); ctx.fillText('INTERFACE', 914, 736);
  ctx.restore();
}
function titleBlock(t) {
  const p = ease((t - 7.8) / 0.7); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = 'rgba(7,30,59,.85)'; ctx.fillRect(90, 1600, 900, 210); ctx.strokeStyle = C.line; ctx.lineWidth = 3; ctx.strokeRect(90, 1600, 900, 210);
  ctx.beginPath(); ctx.moveTo(90, 1660); ctx.lineTo(990, 1660); ctx.moveTo(90, 1730); ctx.lineTo(990, 1730); ctx.moveTo(620, 1600); ctx.lineTo(620, 1810); ctx.stroke();
  ctx.fillStyle = C.cyan; ctx.font = '700 18px "Courier New",monospace'; ctx.fillText('TITLE', 110, 1634); ctx.fillStyle = C.white; ctx.font = '700 28px "Microsoft YaHei", sans-serif'; ctx.fillText(DATA.title, 110, 1692);
  ctx.fillStyle = C.dim; ctx.font = '400 16px "Courier New",monospace'; ctx.fillText('SOURCE', 110, 1760); ctx.fillStyle = C.white; ctx.font = '400 20px "Courier New",monospace'; ctx.fillText(DATA.source, 110, 1790);
  ctx.fillStyle = C.cyan; ctx.font = '700 16px "Courier New",monospace'; ctx.fillText('STATUS', 650, 1640); ctx.fillStyle = C.yellow; ctx.font = '700 26px "Courier New",monospace'; ctx.fillText('APPROVED', 650, 1682);
  ctx.fillStyle = C.dim; ctx.font = '400 15px "Courier New",monospace'; ctx.fillText('DATE 2026-10-18  /  SHEET 01 OF 01', 650, 1740);
  ctx.restore();
}
function renderAt(t) {
  const time = clamp(t, 0, DURATION); ctx.clearRect(0, 0, W, H); grid(); chrome(time); title(time);
  moduleCard(0, 92, 450, 420, 300, time); moduleCard(1, 568, 450, 420, 300, time); moduleCard(2, 92, 800, 420, 300, time); moduleCard(3, 568, 800, 420, 300, time);
  annotations(time); titleBlock(time);
}
window.renderAt = renderAt; window.__DURATION__ = DURATION; window.__FPS__ = FPS;
loadData().then(() => document.fonts.ready.then(() => { renderAt(0); window.__READY__ = true; }));
