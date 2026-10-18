const W = 1080, H = 1920, FPS = 30, DURATION = 10;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const DEFAULT = {
  title: '把此刻唱完',
  artist: 'HIMIND / LIVE SESSION',
  lyrics: [
    {text: '灯光落下之前', start: 1.2, end: 2.8},
    {text: '我们把故事写成旋律', start: 2.8, end: 4.5},
    {text: '每一句都在靠近', start: 4.5, end: 6.2},
    {text: '把此刻留在画面里', start: 6.2, end: 7.9},
    {text: '然后继续向前', start: 7.9, end: 9.4},
  ],
  source: 'HIMIND KARAOKE',
};
let DATA = DEFAULT;
async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) {
      const value = await response.json();
      const lyrics = Array.isArray(value.lyrics) ? value.lyrics.slice(0, 8).filter((x) => x && x.text).map((x) => ({text:String(x.text), start:Number(x.start || 0), end:Number(x.end || 0)})) : [];
      DATA = {title:String(value.title || DEFAULT.title), artist:String(value.artist || DEFAULT.artist), lyrics:lyrics.length ? lyrics : DEFAULT.lyrics, source:String(value.source || DEFAULT.source)};
    }
  } catch { DATA = DEFAULT; }
}
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const ease = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
const hash = (n) => { const x = Math.sin(n * 127.1 + 311.7) * 43758.5453; return x - Math.floor(x); };
const C = {bg:'#07050d',bg2:'#150b24',ink:'#f8f3ff',dim:'rgba(248,243,255,.24)',muted:'rgba(248,243,255,.45)',pink:'#ff5fa2',violet:'#9b6cff',cyan:'#5ee7ff'};
function bg(t) {
  const g = ctx.createLinearGradient(0, 0, 0, H); g.addColorStop(0, '#07050d'); g.addColorStop(0.55, '#150b24'); g.addColorStop(1, '#050307'); ctx.fillStyle = g; ctx.fillRect(0, 0, W, H);
  ctx.save(); const cone = ctx.createLinearGradient(W/2, 0, W/2, 1000); cone.addColorStop(0, 'rgba(155,108,255,.38)'); cone.addColorStop(0.5, 'rgba(255,95,162,.12)'); cone.addColorStop(1, 'rgba(255,95,162,0)');
  ctx.fillStyle = cone; ctx.beginPath(); ctx.moveTo(W/2 - 80, 0); ctx.lineTo(W/2 + 80, 0); ctx.lineTo(W/2 + 420, 1120); ctx.lineTo(W/2 - 420, 1120); ctx.closePath(); ctx.fill(); ctx.restore();
  ctx.save(); for (let i = 0; i < 60; i++) { const x = hash(i + 1) * W, y = (hash(i + 101) * H + t * (18 + hash(i + 201) * 24)) % H, a = 0.05 + hash(i + 301) * 0.18; ctx.fillStyle = `rgba(255,255,255,${a.toFixed(3)})`; ctx.beginPath(); ctx.arc(x, y, 1 + hash(i + 401) * 3, 0, Math.PI * 2); ctx.fill(); } ctx.restore();
}
function header(t) {
  const p = ease(t / 1.0); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha = p; ctx.textAlign = 'center'; ctx.fillStyle = C.ink; ctx.font = '700 70px "Microsoft YaHei", sans-serif'; ctx.fillText(DATA.title, W/2, 230);
  ctx.fillStyle = C.pink; ctx.font = '600 22px "Courier New",monospace'; ctx.fillText(DATA.artist, W/2, 282);
  ctx.strokeStyle = 'rgba(255,95,162,.55)'; ctx.lineWidth = 3; ctx.beginPath(); ctx.moveTo(W/2 - 180, 314); ctx.lineTo(W/2 + 180, 314); ctx.stroke(); ctx.restore();
}
function currentIndex(t) {
  for (let i = 0; i < DATA.lyrics.length; i++) if (t >= DATA.lyrics[i].start && t <= DATA.lyrics[i].end) return i;
  let last = -1; for (let i = 0; i < DATA.lyrics.length; i++) if (t >= DATA.lyrics[i].start) last = i;
  return last;
}
function lyricLine(line, y, size, alpha, progress, active) {
  ctx.save(); ctx.globalAlpha = alpha; ctx.textAlign = 'center'; ctx.font = `700 ${size}px "Microsoft YaHei", sans-serif`;
  const width = ctx.measureText(line.text).width;
  ctx.fillStyle = active ? C.dim : 'rgba(248,243,255,.16)'; ctx.fillText(line.text, W/2, y);
  if (active && progress > 0) { ctx.save(); ctx.beginPath(); ctx.rect(W/2 - width/2, y - size, width * clamp(progress, 0, 1), size * 1.35); ctx.clip(); ctx.fillStyle = C.ink; ctx.fillText(line.text, W/2, y); ctx.restore(); }
  ctx.restore();
}
function lyrics(t) {
  const index = currentIndex(t); if (index < 0) return;
  const current = DATA.lyrics[index];
  const progress = current.end > current.start ? (t - current.start) / (current.end - current.start) : 0;
  for (let i = Math.max(0, index - 2); i < Math.min(DATA.lyrics.length, index + 3); i++) {
    const line = DATA.lyrics[i]; const distance = Math.abs(i - index);
    const y = 900 + (i - index) * 120;
    const size = distance === 0 ? 58 : distance === 1 ? 38 : 30;
    const alpha = distance === 0 ? 1 : distance === 1 ? 0.48 : 0.20;
    lyricLine(line, y, size, alpha, i === index ? progress : 0, i === index);
  }
  // Karaoke dot / cursor.
  const currentWidth = (() => { ctx.save(); ctx.font = '700 58px "Microsoft YaHei", sans-serif'; const w = ctx.measureText(current.text).width; ctx.restore(); return w; })();
  ctx.save(); ctx.fillStyle = C.pink; ctx.beginPath(); ctx.arc(W/2 - currentWidth/2 + currentWidth * clamp(progress,0,1), 928, 5, 0, Math.PI * 2); ctx.fill(); ctx.restore();
}
function waveform(t) {
  const base = 1480; ctx.save(); ctx.globalAlpha = 0.85;
  for (let i = 0; i < 72; i++) {
    const x = 90 + i * 12.5; const env = Math.sin((i / 71) * Math.PI); const h = 14 + Math.abs(Math.sin(t * 4 + i * 0.55)) * 90 * env + hash(i + Math.floor(t * 20)) * 26;
    const g = ctx.createLinearGradient(0, base - h, 0, base + h); g.addColorStop(0, C.cyan); g.addColorStop(0.5, C.pink); g.addColorStop(1, C.violet);
    ctx.fillStyle = g; ctx.fillRect(x, base - h / 2, 7, h);
  }
  ctx.restore();
}
function progress(t) {
  const p = clamp(t / DURATION, 0, 1); ctx.save();
  ctx.fillStyle = 'rgba(248,243,255,.12)'; ctx.fillRect(90, 1600, W - 180, 8);
  const g = ctx.createLinearGradient(90, 0, W - 90, 0); g.addColorStop(0, C.cyan); g.addColorStop(0.5, C.pink); g.addColorStop(1, C.violet);
  ctx.fillStyle = g; ctx.fillRect(90, 1600, (W - 180) * p, 8);
  ctx.fillStyle = C.muted; ctx.font = '500 17px "Courier New",monospace'; ctx.fillText(`00:${String(Math.floor(t)).padStart(2,'0')} / 00:10`, 90, 1640);
  ctx.textAlign = 'right'; ctx.fillText('HIMIND / KARAOKE', W - 90, 1640);
  ctx.textAlign = 'left'; ctx.fillStyle = C.pink; ctx.fillRect(90, 1780, 220, 5);
  ctx.fillStyle = C.ink; ctx.font = '600 18px "Courier New",monospace'; ctx.fillText(DATA.source, 90, 1820); ctx.restore();
}
function renderAt(t) {
  const time = clamp(t, 0, DURATION); ctx.clearRect(0, 0, W, H); bg(time); header(time); lyrics(time); waveform(time); progress(time);
}
window.renderAt = renderAt; window.__DURATION__ = DURATION; window.__FPS__ = FPS;
loadData().then(() => document.fonts.ready.then(() => { renderAt(0); window.__READY__ = true; }));
