const W = 1080, H = 1920, FPS = 30, DURATION = 10;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const DEFAULT = {
  kicker: 'ISSUE 01 / CREATIVE PRACTICE',
  title: '少一点噪声，\n多一点判断。',
  quote: '真正可复用的不是模板，\n而是被验证过的选择。',
  author: 'HIMIND EDITORIAL',
  stats: [
    {label: '复用能力', value: '05'},
    {label: '质量标准', value: '46'},
    {label: '交付模式', value: '02'},
  ],
  source: 'HIMIND / EDITORIAL MINIMAL',
};
let DATA = DEFAULT;
async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) {
      const value = await response.json();
      const stats = Array.isArray(value.stats) ? value.stats.slice(0, 3).map((x) => ({label:String(x.label || ''), value:String(x.value || '')})) : [];
      DATA = {
        kicker: String(value.kicker || DEFAULT.kicker), title: String(value.title || DEFAULT.title), quote: String(value.quote || DEFAULT.quote),
        author: String(value.author || DEFAULT.author), stats: stats.length ? stats : DEFAULT.stats, source: String(value.source || DEFAULT.source),
      };
    }
  } catch { DATA = DEFAULT; }
}
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const ease = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
const C = {paper:'#f4efe6',ink:'#1c1916',muted:'#6f675d',faint:'rgba(28,25,22,.18)',accent:'#b94a2f',gold:'#c59a55'};
function paper() {
  ctx.fillStyle = C.paper; ctx.fillRect(0, 0, W, H);
  ctx.save();
  for (let i = 0; i < 260; i++) { const x = Math.sin(i * 12.9898) * 43758.5453; const y = Math.sin(i * 78.233) * 43758.5453; const px = (x - Math.floor(x)) * W; const py = (y - Math.floor(y)) * H; ctx.fillStyle = `rgba(28,25,22,${(0.012 + (i % 7) * 0.004).toFixed(3)})`; ctx.fillRect(px, py, 2, 2); }
  ctx.restore();
}
function header(t) {
  const p = ease(t / 1.0); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.ink; ctx.font = '600 19px "Courier New",monospace'; ctx.fillText(DATA.kicker, 90, 100);
  ctx.textAlign = 'right'; ctx.fillStyle = C.accent; ctx.fillText('HIMIND / 2026', W - 90, 100); ctx.textAlign = 'left';
  ctx.strokeStyle = C.faint; ctx.lineWidth = 2; ctx.beginPath(); ctx.moveTo(90, 126); ctx.lineTo(W - 90, 126); ctx.stroke(); ctx.restore();
}
function title(t) {
  const p = ease(t / 1.5); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.ink; ctx.font = '700 82px Georgia, "Times New Roman", serif';
  const lines = DATA.title.split('\n'); lines.forEach((line, i) => ctx.fillText(line, 90, 360 + i * 104));
  ctx.fillStyle = C.accent; ctx.fillRect(90, 520 + (lines.length - 1) * 104, 180 * p, 5);
  ctx.restore();
}
function quote(t) {
  const p = ease((t - 2.0) / 1.0); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.translate(0, 26 * (1 - p));
  ctx.fillStyle = C.ink; ctx.font = 'italic 48px Georgia, "Times New Roman", serif';
  const lines = DATA.quote.split('\n'); lines.forEach((line, i) => ctx.fillText(line, 90, 820 + i * 70));
  ctx.fillStyle = C.muted; ctx.font = '500 20px "Courier New",monospace'; ctx.fillText(DATA.author, 90, 1040);
  ctx.strokeStyle = C.gold; ctx.lineWidth = 2; ctx.beginPath(); ctx.moveTo(90, 1070); ctx.lineTo(440, 1070); ctx.stroke(); ctx.restore();
}
function stats(t) {
  const start = 5.3;
  DATA.stats.forEach((stat, i) => {
    const p = ease((t - (start + i * 0.65)) / 0.8); if (p <= 0) return;
    const x = 90 + i * 310; const y = 1280;
    ctx.save(); ctx.globalAlpha = p; ctx.translate(0, 20 * (1 - p));
    ctx.fillStyle = C.faint; ctx.fillRect(x, y, 260, 2);
    ctx.fillStyle = C.accent; ctx.font = '700 80px Georgia, serif'; ctx.fillText(stat.value, x, y + 100);
    ctx.fillStyle = C.muted; ctx.font = '500 18px "Microsoft YaHei", sans-serif'; ctx.fillText(stat.label, x, y + 142);
    ctx.restore();
  });
}
function footer(t) {
  const p = ease((t - 8.3) / 0.6); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.strokeStyle = C.faint; ctx.lineWidth = 2; ctx.beginPath(); ctx.moveTo(90, 1780); ctx.lineTo(W - 90, 1780); ctx.stroke();
  ctx.fillStyle = C.muted; ctx.font = '500 17px "Courier New",monospace'; ctx.fillText(DATA.source, 90, 1820);
  ctx.textAlign = 'right'; ctx.fillStyle = C.accent; ctx.fillText('01 / 01', W - 90, 1820); ctx.restore();
}
function renderAt(t) {
  const time = clamp(t, 0, DURATION); ctx.clearRect(0, 0, W, H); paper(); header(time); title(time); quote(time); stats(time); footer(time);
}
window.renderAt = renderAt; window.__DURATION__ = DURATION; window.__FPS__ = FPS;
loadData().then(() => document.fonts.ready.then(() => { renderAt(0); window.__READY__ = true; }));
