const W = 1080, H = 1920, FPS = 30, DURATION = 10;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const DEFAULT = {
  title: '数据在生长',
  subtitle: 'TOP 5 / 2026',
  unit: '单位：指数',
  items: [
    {name: '华东', value: 92},
    {name: '华南', value: 81},
    {name: '华北', value: 74},
    {name: '西南', value: 63},
    {name: '东北', value: 52},
  ],
  source: 'SOURCE / HIMIND SAMPLE DATA',
};
let DATA = DEFAULT;
async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) {
      const value = await response.json();
      const items = Array.isArray(value.items) ? value.items.slice(0, 6).filter((x) => x && x.name != null).map((x) => ({name:String(x.name), value:Number(x.value || 0)})) : [];
      if (items.length) DATA = {title:String(value.title || DEFAULT.title), subtitle:String(value.subtitle || DEFAULT.subtitle), unit:String(value.unit || DEFAULT.unit), items, source:String(value.source || DEFAULT.source)};
    }
  } catch { DATA = DEFAULT; }
}
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const ease = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
const C = {bg:'#06141c',panel:'rgba(255,255,255,.035)',line:'rgba(180,240,255,.12)',text:'#effcff',muted:'rgba(220,246,255,.56)',dim:'rgba(220,246,255,.28)',cyan:'#4de2ff',green:'#4ef2b0',orange:'#ff9f43',yellow:'#ffd166'};
function bg(t) {
  const g = ctx.createLinearGradient(0, 0, W, H); g.addColorStop(0, '#041018'); g.addColorStop(0.55, C.bg); g.addColorStop(1, '#08222a'); ctx.fillStyle = g; ctx.fillRect(0, 0, W, H);
  ctx.save(); ctx.strokeStyle = C.line; ctx.lineWidth = 1;
  for (let x = 0; x < W; x += 54) { ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, H); ctx.stroke(); }
  for (let y = 0; y < H; y += 54) { ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(W, y); ctx.stroke(); }
  ctx.restore();
  const glow = ctx.createRadialGradient(W*0.75, 500, 0, W*0.75, 500, 700); glow.addColorStop(0, 'rgba(77,226,255,.12)'); glow.addColorStop(1, 'rgba(77,226,255,0)'); ctx.fillStyle = glow; ctx.fillRect(0, 0, W, H);
}
function header(t) {
  const p = ease(t / 1.1); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.text; ctx.font = '800 76px "Microsoft YaHei", sans-serif'; ctx.fillText(DATA.title, 100, 230);
  ctx.fillStyle = C.cyan; ctx.font = '600 24px "Courier New",monospace'; ctx.fillText(DATA.subtitle, 104, 278);
  ctx.fillStyle = C.muted; ctx.font = '400 20px "Microsoft YaHei", sans-serif'; ctx.fillText(DATA.unit, 104, 316);
  ctx.strokeStyle = 'rgba(77,226,255,.4)'; ctx.lineWidth = 3; ctx.beginPath(); ctx.moveTo(104, 348); ctx.lineTo(104 + 420 * p, 348); ctx.stroke(); ctx.restore();
}
function chart(t) {
  const left = 250, top = 470, width = 730, rowGap = 190, maxValue = Math.max(...DATA.items.map((x) => x.value), 1);
  ctx.save();
  ctx.strokeStyle = C.line; ctx.lineWidth = 2;
  for (let i = 0; i <= 4; i++) { const x = left + width * i / 4; ctx.beginPath(); ctx.moveTo(x, top - 30); ctx.lineTo(x, top + rowGap * DATA.items.length - 60); ctx.stroke(); }
  DATA.items.forEach((item, index) => {
    const start = 1.45 + index * 0.65; const p = ease((t - start) / 0.9); if (p <= 0) return;
    const y = top + index * rowGap; const barWidth = width * (item.value / maxValue) * p; const color = index === 0 ? C.orange : index === 1 ? C.green : C.cyan;
    ctx.fillStyle = 'rgba(255,255,255,.035)'; ctx.fillRect(left, y - 30, width, 70);
    const g = ctx.createLinearGradient(left, 0, left + barWidth, 0); g.addColorStop(0, color); g.addColorStop(1, index === 0 ? '#ffd166' : 'rgba(77,226,255,.42)');
    ctx.fillStyle = g; ctx.fillRect(left, y - 30, barWidth, 70);
    ctx.fillStyle = C.text; ctx.font = '700 34px "Microsoft YaHei", sans-serif'; ctx.textAlign = 'right'; ctx.fillText(item.name, left - 28, y + 12); ctx.textAlign = 'left';
    const count = Math.round(item.value * p);
    ctx.fillStyle = C.text; ctx.font = '800 38px "DIN Alternate", "Arial Narrow", sans-serif'; ctx.fillText(String(count), left + barWidth + 22, y + 12);
    ctx.fillStyle = index === 0 ? C.orange : C.dim; ctx.font = '600 18px "Courier New",monospace'; ctx.fillText(`#${index + 1}`, left + 14, y - 42);
  });
  ctx.restore();
}
function highlight(t) {
  const p = ease((t - 5.2) / 0.8); if (p <= 0) return;
  const top = DATA.items[0]; if (!top) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.yellow; ctx.font = '700 26px "Courier New",monospace'; ctx.fillText('TOP SIGNAL', 104, 1640);
  ctx.fillStyle = C.text; ctx.font = '700 52px "Microsoft YaHei", sans-serif'; ctx.fillText(`${top.name} / ${top.value}`, 104, 1700);
  ctx.strokeStyle = C.orange; ctx.lineWidth = 3; ctx.beginPath(); ctx.moveTo(104, 1726); ctx.lineTo(500, 1726); ctx.stroke(); ctx.restore();
}
function footer(t) {
  const p = ease((t - 7.6) / 0.7); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = C.dim; ctx.font = '500 17px "Courier New",monospace'; ctx.fillText(DATA.source, 104, 1820);
  ctx.textAlign = 'right'; ctx.fillText('HIMIND / DATA CHART', W - 104, 1820); ctx.restore();
}
function renderAt(t) {
  const time = clamp(t, 0, DURATION); ctx.clearRect(0, 0, W, H); bg(time); header(time); chart(time); highlight(time); footer(time);
}
window.renderAt = renderAt; window.__DURATION__ = DURATION; window.__FPS__ = FPS;
loadData().then(() => document.fonts.ready.then(() => { renderAt(0); window.__READY__ = true; }));
