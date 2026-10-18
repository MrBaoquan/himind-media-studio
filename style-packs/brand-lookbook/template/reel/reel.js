const W = 1080, H = 1920, FPS = 30, DURATION = 10;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const DEFAULT = {
  brand: 'HIMIND',
  tagline: 'CREATE WITH CLARITY.',
  palette: [
    {name: 'INK', hex: '#1f2421'},
    {name: 'CLAY', hex: '#b65f3f'},
    {name: 'SAGE', hex: '#899b83'},
  ],
  principles: ['先定义问题', '再沉淀能力', '最后交给系统'],
  source: 'BRAND LOOKBOOK / 2026',
};
let DATA = DEFAULT;
async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) {
      const value = await response.json();
      const palette = Array.isArray(value.palette) ? value.palette.slice(0, 3).map((x) => ({name:String(x.name || ''), hex:String(x.hex || '#000000')})) : [];
      const principles = Array.isArray(value.principles) ? value.principles.slice(0, 3).map(String) : [];
      DATA = {brand:String(value.brand || DEFAULT.brand), tagline:String(value.tagline || DEFAULT.tagline), palette:palette.length ? palette : DEFAULT.palette, principles:principles.length ? principles : DEFAULT.principles, source:String(value.source || DEFAULT.source)};
    }
  } catch { DATA = DEFAULT; }
}
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const ease = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
const C = {paper:'#f1ede5',ink:'#1f2421',muted:'#77756c',line:'rgba(31,36,33,.20)',accent:'#b65f3f',sage:'#899b83'};
function paper() {
  ctx.fillStyle = C.paper; ctx.fillRect(0, 0, W, H);
  ctx.save(); ctx.strokeStyle = C.line; ctx.lineWidth = 1;
  for (let x = 70; x < W; x += 80) { ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, H); ctx.stroke(); }
  for (let y = 70; y < H; y += 80) { ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(W, y); ctx.stroke(); }
  ctx.restore();
}
function marks() {
  ctx.save(); ctx.strokeStyle = C.ink; ctx.lineWidth = 2;
  [[50,50],[W-50,50],[50,H-50],[W-50,H-50]].forEach(([x,y]) => { ctx.beginPath(); ctx.moveTo(x-14,y); ctx.lineTo(x+14,y); ctx.moveTo(x,y-14); ctx.lineTo(x,y+14); ctx.stroke(); });
  ctx.restore();
}
function header(t) {
  const p = ease(t / 1.0); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.ink; ctx.font = '600 18px "Courier New",monospace'; ctx.fillText('BRAND LOOKBOOK / 2026', 80, 110);
  ctx.textAlign = 'right'; ctx.fillStyle = C.accent; ctx.fillText('IDENTITY SYSTEM', W - 80, 110); ctx.textAlign = 'left';
  ctx.strokeStyle = C.ink; ctx.lineWidth = 2; ctx.beginPath(); ctx.moveTo(80, 136); ctx.lineTo(W - 80, 136); ctx.stroke(); ctx.restore();
}
function brand(t) {
  const p = ease(t / 1.3); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.ink; ctx.font = '700 150px Georgia, "Times New Roman", serif'; ctx.fillText(DATA.brand, 80, 380);
  ctx.fillStyle = C.accent; ctx.font = '600 26px "Courier New",monospace'; ctx.fillText(DATA.tagline, 86, 438);
  ctx.strokeStyle = C.accent; ctx.lineWidth = 5; ctx.beginPath(); ctx.moveTo(86, 470); ctx.lineTo(86 + 420 * p, 470); ctx.stroke(); ctx.restore();
}
function swatches(t) {
  DATA.palette.forEach((color, i) => {
    const p = ease((t - (2.0 + i * 0.65)) / 0.7); if (p <= 0) return;
    const x = 80 + i * 310; const y = 650;
    ctx.save(); ctx.globalAlpha = p; ctx.translate(0, 30 * (1 - p));
    ctx.fillStyle = color.hex; ctx.fillRect(x, y, 260, 260);
    ctx.strokeStyle = C.ink; ctx.lineWidth = 2; ctx.strokeRect(x, y, 260, 260);
    ctx.fillStyle = C.ink; ctx.font = '700 22px "Courier New",monospace'; ctx.fillText(color.name, x, y + 304);
    ctx.fillStyle = C.muted; ctx.font = '500 18px "Courier New",monospace'; ctx.fillText(color.hex.toUpperCase(), x, y + 336);
    ctx.restore();
  });
}
function typography(t) {
  const p = ease((t - 4.4) / 0.8); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.ink; ctx.font = '700 220px Georgia, "Times New Roman", serif'; ctx.fillText('Aa', 80, 1120);
  ctx.fillStyle = C.muted; ctx.font = '500 22px "Courier New",monospace'; ctx.fillText('DISPLAY / GEORGIA', 400, 980); ctx.fillText('BODY / INTER / SANS', 400, 1024);
  ctx.strokeStyle = C.line; ctx.lineWidth = 2; ctx.beginPath(); ctx.moveTo(400, 1060); ctx.lineTo(1000, 1060); ctx.stroke();
  ctx.fillStyle = C.ink; ctx.font = '400 30px Georgia, serif'; ctx.fillText('Clarity is a design system.', 400, 1120);
  ctx.restore();
}
function principles(t) {
  DATA.principles.forEach((text, i) => {
    const p = ease((t - (5.6 + i * 0.6)) / 0.7); if (p <= 0) return;
    const y = 1370 + i * 110;
    ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.accent; ctx.font = '700 22px "Courier New",monospace'; ctx.fillText(`0${i+1}`, 82, y);
    ctx.strokeStyle = C.line; ctx.lineWidth = 2; ctx.beginPath(); ctx.moveTo(140, y - 10); ctx.lineTo(1000, y - 10); ctx.stroke();
    ctx.fillStyle = C.ink; ctx.font = '700 42px "Microsoft YaHei", sans-serif'; ctx.fillText(text, 160, y + 6); ctx.restore();
  });
}
function footer(t) {
  const p = ease((t - 8.4) / 0.6); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.muted; ctx.font = '500 18px "Courier New",monospace'; ctx.fillText(DATA.source, 80, 1810);
  ctx.textAlign = 'right'; ctx.fillStyle = C.accent; ctx.fillText('HIMIND / BRAND', W - 80, 1810); ctx.restore();
}
function renderAt(t) {
  const time = clamp(t, 0, DURATION); ctx.clearRect(0, 0, W, H); paper(); marks(); header(time); brand(time); swatches(time); typography(time); principles(time); footer(time);
}
window.renderAt = renderAt; window.__DURATION__ = DURATION; window.__FPS__ = FPS;
loadData().then(() => document.fonts.ready.then(() => { renderAt(0); window.__READY__ = true; }));
