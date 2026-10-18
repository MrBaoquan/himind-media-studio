const W = 1080, H = 1920, FPS = 30, DURATION = 10;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const DEFAULT = {
  title: 'HIGH SCORE',
  items: [
    {label: 'CREATIVITY', score: 9000},
    {label: 'PRECISION', score: 7800},
    {label: 'SPEED', score: 8600},
  ],
  conclusion: 'GAME OVER? NEVER.',
  source: 'HIMIND ARCADE / PLAYER 1',
};
let DATA = DEFAULT;
async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) {
      const value = await response.json();
      const items = Array.isArray(value.items) ? value.items.slice(0, 3).map((x) => ({label:String(x.label || ''), score:Number(x.score || 0)})) : [];
      DATA = {title:String(value.title || DEFAULT.title), items:items.length ? items : DEFAULT.items, conclusion:String(value.conclusion || DEFAULT.conclusion), source:String(value.source || DEFAULT.source)};
    }
  } catch { DATA = DEFAULT; }
}
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const ease = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
const hash = (n) => { const x = Math.sin(n * 127.1 + 311.7) * 43758.5453; return x - Math.floor(x); };
const C = {bg:'#05040d',panel:'rgba(18,12,40,.82)',ink:'#f6f1ff',muted:'rgba(246,241,255,.48)',pink:'#ff3db8',cyan:'#3df5ff',yellow:'#ffe45e',green:'#54ff9f',line:'rgba(61,245,255,.18)'};
function bg(t) {
  const g = ctx.createLinearGradient(0, 0, 0, H); g.addColorStop(0, '#05040d'); g.addColorStop(0.55, '#110a2c'); g.addColorStop(1, '#05040d'); ctx.fillStyle = g; ctx.fillRect(0, 0, W, H);
  ctx.save(); ctx.strokeStyle = C.line; ctx.lineWidth = 1;
  for (let x = 0; x < W; x += 48) { ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, H); ctx.stroke(); }
  for (let y = 0; y < H; y += 48) { ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(W, y); ctx.stroke(); }
  ctx.restore();
  for (let i = 0; i < 90; i++) { const x = hash(i + 1) * W, y = (hash(i + 101) * H + t * 18) % H, a = 0.15 + hash(i + 201) * 0.65; ctx.fillStyle = i % 3 === 0 ? `rgba(255,61,184,${a})` : `rgba(61,245,255,${a})`; const s = i % 4 === 0 ? 4 : 2; ctx.fillRect(x, y, s, s); }
}
function glowText(text, x, y, color, size, align = 'left') {
  ctx.save(); ctx.textAlign = align; ctx.font = `700 ${size}px "Courier New",monospace`; ctx.shadowColor = color; ctx.shadowBlur = 22; ctx.fillStyle = color; ctx.fillText(text, x, y); ctx.shadowBlur = 0; ctx.restore();
}
function header(t) {
  const p = ease(t / 1.0); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p;
  glowText('PLAYER 1', 80, 100, C.cyan, 22);
  glowText('HIGH SCORE', W - 80, 100, C.pink, 22, 'right');
  glowText(DATA.title, W/2, 240, C.ink, 72, 'center');
  ctx.fillStyle = 'rgba(61,245,255,.5)'; ctx.fillRect(150, 280, 780, 3);
  ctx.restore();
}
function scoreRows(t) {
  DATA.items.forEach((item, i) => {
    const start = 1.5 + i * 1.1; const p = ease((t - start) / 0.9); if (p <= 0) return;
    const y = 520 + i * 260; const color = [C.yellow, C.pink, C.green][i];
    ctx.save(); ctx.globalAlpha = p;
    ctx.fillStyle = C.panel; ctx.fillRect(100, y - 74, 880, 150);
    ctx.strokeStyle = color; ctx.lineWidth = 3; ctx.strokeRect(100, y - 74, 880, 150);
    glowText(item.label, 132, y - 10, color, 28);
    const score = Math.round(item.score * p);
    glowText(String(score).padStart(6, '0'), W - 132, y + 38, C.ink, 62, 'right');
    ctx.fillStyle = 'rgba(246,241,255,.10)'; ctx.fillRect(132, y + 52, 780, 8);
    ctx.fillStyle = color; ctx.fillRect(132, y + 52, 780 * p, 8);
    ctx.restore();
  });
}
function combo(t) {
  const p = ease((t - 6.4) / 0.6); if (p <= 0) return;
  const pulse = 1 + Math.sin(t * 12) * 0.04;
  ctx.save(); ctx.globalAlpha = p; ctx.translate(W/2, 1390); ctx.scale(pulse, pulse);
  ctx.strokeStyle = C.yellow; ctx.lineWidth = 6; ctx.beginPath(); ctx.arc(0, 0, 155, 0, Math.PI * 2); ctx.stroke();
  glowText('COMBO x3', 0, 18, C.yellow, 58, 'center');
  ctx.restore();
  if (t > 7.0) { glowText(DATA.conclusion, W/2, 1630, C.pink, 44, 'center'); }
}
function footer(t) {
  const p = ease((t - 8.2) / 0.6); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; glowText(DATA.source, W/2, 1820, C.cyan, 18, 'center');
  for (let y = 0; y < H; y += 6) { ctx.fillStyle = 'rgba(0,0,0,.16)'; ctx.fillRect(0, y, W, 2); }
  ctx.restore();
}
function renderAt(t) {
  const time = clamp(t, 0, DURATION); ctx.clearRect(0, 0, W, H); bg(time); header(time); scoreRows(time); combo(time); footer(time);
}
window.renderAt = renderAt; window.__DURATION__ = DURATION; window.__FPS__ = FPS;
loadData().then(() => document.fonts.ready.then(() => { renderAt(0); window.__READY__ = true; }));
