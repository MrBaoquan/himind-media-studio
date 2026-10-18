const W = 1080, H = 1920, FPS = 30, DURATION = 10;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const DEFAULT = {
  title: '午夜信号',
  subtitle: 'SIGNAL FROM 1987',
  lines: ['磁带开始转动', '城市在扫描线里闪烁', '记忆延迟了 0.3 秒'],
  footer: 'HIMIND / VHS ARCHIVE',
};
let DATA = DEFAULT;
async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) {
      const value = await response.json();
      DATA = {
        title: String(value.title || DEFAULT.title),
        subtitle: String(value.subtitle || DEFAULT.subtitle),
        lines: Array.isArray(value.lines) ? value.lines.slice(0, 3).map(String) : DEFAULT.lines,
        footer: String(value.footer || DEFAULT.footer),
      };
    }
  } catch { DATA = DEFAULT; }
}
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const ease = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
const hash = (n) => { const x = Math.sin(n * 127.1 + 311.7) * 43758.5453; return x - Math.floor(x); };
const C = {ink:'#f6e9ff',pink:'#ff3d9a',cyan:'#35f2ff',purple:'#7b2cbf',yellow:'#ffd166',dark:'#090312'};
function bg(t) {
  const g = ctx.createLinearGradient(0, 0, 0, H);
  g.addColorStop(0, '#090312'); g.addColorStop(0.55, '#1a0730'); g.addColorStop(1, '#030208');
  ctx.fillStyle = g; ctx.fillRect(0, 0, W, H);
  const sunY = 690 + Math.sin(t * 0.7) * 5;
  const glow = ctx.createRadialGradient(W/2, sunY, 10, W/2, sunY, 420);
  glow.addColorStop(0, 'rgba(255,61,154,.55)'); glow.addColorStop(0.55, 'rgba(255,61,154,.16)'); glow.addColorStop(1, 'rgba(255,61,154,0)');
  ctx.fillStyle = glow; ctx.fillRect(0, 0, W, H);
  ctx.save(); ctx.beginPath(); ctx.arc(W/2, sunY, 230, 0, Math.PI * 2); ctx.clip();
  const sun = ctx.createLinearGradient(0, sunY - 230, 0, sunY + 230);
  sun.addColorStop(0, '#ffe66d'); sun.addColorStop(0.45, '#ff6b9d'); sun.addColorStop(1, '#7b2cbf');
  ctx.fillStyle = sun; ctx.fillRect(W/2 - 230, sunY - 230, 460, 460);
  ctx.fillStyle = '#12051f';
  for (let i = 0; i < 8; i++) ctx.fillRect(W/2 - 240, sunY - 40 + i * 34, 480, 10 + i * 2);
  ctx.restore();
  ctx.save(); ctx.strokeStyle = 'rgba(53,242,255,.22)'; ctx.lineWidth = 2;
  const horizon = 980;
  for (let i = -10; i <= 10; i++) { ctx.beginPath(); ctx.moveTo(W/2 + i * 24, horizon); ctx.lineTo(W/2 + i * 150, H + 80); ctx.stroke(); }
  for (let i = 0; i < 14; i++) { const y = horizon + Math.pow(i / 13, 1.7) * (H - horizon + 60); ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(W, y); ctx.stroke(); }
  ctx.restore();
}
function scanlines() { ctx.save(); ctx.fillStyle = 'rgba(0,0,0,.18)'; for (let y = 0; y < H; y += 6) ctx.fillRect(0, y, W, 2); ctx.restore(); }
function noise(t) {
  ctx.save();
  for (let i = 0; i < 220; i++) {
    const x = hash(i + 1 + Math.floor(t * 30)) * W, y = hash(i + 101 + Math.floor(t * 30)) * H;
    ctx.fillStyle = `rgba(255,255,255,${(0.03 + hash(i + 201) * 0.10).toFixed(3)})`;
    ctx.fillRect(x, y, 2 + hash(i + 301) * 4, 1);
  }
  ctx.restore();
}
function chrome(t) {
  const seconds = Math.floor(t), frame = Math.floor((t - seconds) * 30);
  ctx.save(); ctx.fillStyle = C.cyan; ctx.font = '600 24px "Courier New",monospace'; ctx.fillText('PLAY  ▶', 70, 96);
  ctx.fillStyle = C.ink; ctx.fillText(`00:00:${String(seconds).padStart(2,'0')}:${String(frame).padStart(2,'0')}`, 70, 132);
  ctx.textAlign = 'right'; ctx.fillStyle = C.pink; ctx.fillText('CH 03 / 1987', W - 70, 96); ctx.textAlign = 'left';
  ctx.strokeStyle = 'rgba(246,233,255,.28)'; ctx.lineWidth = 2; ctx.beginPath(); ctx.moveTo(70, 156); ctx.lineTo(1010, 156); ctx.stroke();
  ctx.restore();
}
function title(t) {
  const p = ease(t / 1.4); if (p <= 0) return;
  const jitter = t > 6 && t < 7 ? (hash(Math.floor(t * 20)) - .5) * 12 : 0;
  ctx.save(); ctx.globalAlpha = p; ctx.textAlign = 'center'; ctx.font = 'italic 900 92px "Arial Black", Impact, sans-serif';
  ctx.fillStyle = C.cyan; ctx.fillText(DATA.title, W/2 + 5 + jitter, 350);
  ctx.fillStyle = C.pink; ctx.fillText(DATA.title, W/2 - 5 - jitter, 350);
  ctx.fillStyle = C.ink; ctx.fillText(DATA.title, W/2, 350);
  ctx.font = '700 27px "Courier New",monospace'; ctx.fillStyle = C.yellow; ctx.fillText(DATA.subtitle, W/2, 410);
  ctx.restore();
}
function lines(t) {
  DATA.lines.forEach((line, i) => {
    const p = ease((t - (2.0 + i * 1.05)) / 0.7); if (p <= 0) return;
    const y = 1180 + i * 150, jitter = (hash(i + 20 + Math.floor(t * 12)) - .5) * (t > 6 && t < 7 ? 10 : 2);
    ctx.save(); ctx.globalAlpha = p; ctx.translate(jitter, 0);
    ctx.fillStyle = 'rgba(9,3,18,.74)'; ctx.fillRect(120, y - 52, 840, 94);
    ctx.strokeStyle = i === 0 ? C.pink : C.cyan; ctx.lineWidth = 6; ctx.beginPath(); ctx.moveTo(120, y - 52); ctx.lineTo(120, y + 42); ctx.stroke();
    ctx.fillStyle = C.ink; ctx.font = '700 42px "Microsoft YaHei", sans-serif'; ctx.fillText(line, 164, y + 18);
    ctx.fillStyle = C.yellow; ctx.font = '700 18px "Courier New",monospace'; ctx.fillText(`0${i + 1} / TRACK`, 810, y + 16);
    ctx.restore();
  });
}
function tracking(t) {
  if (t < 5.8 || t > 7.4) return;
  const p = ease((t - 5.8) / 0.4) * (1 - ease((t - 6.9) / 0.5)), bandY = 760 + (t - 5.8) * 280;
  ctx.save(); ctx.globalAlpha = p;
  for (let i = 0; i < 18; i++) { const y = bandY + i * 12, shift = (hash(i + Math.floor(t * 30)) - .5) * 90; ctx.fillStyle = i % 3 === 0 ? 'rgba(53,242,255,.18)' : 'rgba(255,61,154,.14)'; ctx.fillRect(shift, y, W, 5); }
  ctx.fillStyle = 'rgba(255,255,255,.75)'; ctx.font = '700 22px "Courier New",monospace'; ctx.fillText('TRACKING ERROR', 70, bandY + 20);
  ctx.restore();
}
function footer(t) {
  const p = ease((t - 8.4) / 0.6); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.textAlign = 'center';
  ctx.fillStyle = C.ink; ctx.font = '600 24px "Courier New",monospace'; ctx.fillText(DATA.footer, W/2, 1780);
  ctx.fillStyle = C.pink; ctx.fillRect(W/2 - 110, 1802, 220, 4);
  ctx.restore();
}
function renderAt(t) {
  const time = clamp(t, 0, DURATION);
  ctx.clearRect(0, 0, W, H);
  bg(time); chrome(time); title(time); lines(time); tracking(time); footer(time); scanlines(); noise(time);
}
window.renderAt = renderAt; window.__DURATION__ = DURATION; window.__FPS__ = FPS;
loadData().then(() => document.fonts.ready.then(() => { renderAt(0); window.__READY__ = true; }));
