const W = 1080;
const H = 1920;
const FPS = 30;
const DURATION = 10;
const canvas = document.getElementById('stage');
let DATA = null;
async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) DATA = await response.json();
  } catch {
    DATA = null;
  }
}
const ctx = canvas.getContext('2d');

const C = {
  bg: '#080c1a',
  bg2: '#101a33',
  panel: 'rgba(255,255,255,0.045)',
  line: 'rgba(255,255,255,0.10)',
  text: '#f7fbff',
  muted: 'rgba(220,235,255,0.58)',
  dim: 'rgba(220,235,255,0.32)',
  blue: '#4fc3f7',
  warm: '#ff8e53',
  green: '#3ddc97',
  yellow: '#ffca28',
};

const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const easeOut = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
const easeInOut = (v) => {
  const x = clamp(v, 0, 1);
  return x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2;
};
const hash = (n) => {
  const x = Math.sin(n * 127.1 + 311.7) * 43758.5453;
  return x - Math.floor(x);
};

function roundRect(x, y, w, h, r) {
  const radius = Math.min(r, w / 2, h / 2);
  ctx.beginPath();
  ctx.moveTo(x + radius, y);
  ctx.arcTo(x + w, y, x + w, y + h, radius);
  ctx.arcTo(x + w, y + h, x, y + h, radius);
  ctx.arcTo(x, y + h, x, y, radius);
  ctx.arcTo(x, y, x + w, y, radius);
  ctx.closePath();
}

function drawBackground(t) {
  const g = ctx.createLinearGradient(0, 0, W, H);
  g.addColorStop(0, '#060a16');
  g.addColorStop(0.55, C.bg);
  g.addColorStop(1, '#0c1730');
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, W, H);

  // Slow aurora glows.
  const glow = ctx.createRadialGradient(W * 0.15, H * 0.18, 0, W * 0.15, H * 0.18, W * 0.9);
  glow.addColorStop(0, 'rgba(79,195,247,0.16)');
  glow.addColorStop(1, 'rgba(79,195,247,0)');
  ctx.fillStyle = glow;
  ctx.fillRect(0, 0, W, H);

  const warm = ctx.createRadialGradient(W * 0.95, H * 0.78, 0, W * 0.95, H * 0.78, W * 0.8);
  warm.addColorStop(0, 'rgba(255,142,83,0.10)');
  warm.addColorStop(1, 'rgba(255,142,83,0)');
  ctx.fillStyle = warm;
  ctx.fillRect(0, 0, W, H);

  // Technical grid.
  ctx.save();
  ctx.globalAlpha = 0.12;
  ctx.strokeStyle = C.line;
  ctx.lineWidth = 1;
  const offset = (t * 18) % 54;
  for (let x = -54; x < W + 54; x += 54) {
    ctx.beginPath();
    ctx.moveTo(x + offset, 0);
    ctx.lineTo(x + offset, H);
    ctx.stroke();
  }
  for (let y = -54; y < H + 54; y += 54) {
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(W, y);
    ctx.stroke();
  }
  ctx.restore();

  // Film grain / seeded particles.
  ctx.save();
  for (let i = 0; i < 44; i += 1) {
    const px = hash(i + 3) * W;
    const py = (hash(i + 11) * H + t * (8 + hash(i) * 18)) % H;
    const a = 0.08 + hash(i + 7) * 0.16;
    ctx.fillStyle = `rgba(190,225,255,${a.toFixed(3)})`;
    ctx.fillRect(px, py, 2, 2);
  }
  ctx.restore();
}

function drawChrome(t) {
  ctx.save();
  ctx.fillStyle = C.text;
  ctx.font = '600 26px "JetBrains Mono", "Cascadia Mono", Consolas, monospace';
  ctx.letterSpacing = '2px';
  ctx.fillText('HIMIND / CODE PROJECT', 96, 112);
  ctx.textAlign = 'right';
  ctx.fillText('SCHEMATIC', W - 96, 112);
  ctx.textAlign = 'left';

  ctx.strokeStyle = 'rgba(79,195,247,0.28)';
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.moveTo(96, 142);
  ctx.lineTo(338, 142);
  ctx.stroke();

  ctx.font = '400 18px "JetBrains Mono", "Cascadia Mono", Consolas, monospace';
  ctx.fillStyle = C.dim;
  ctx.fillText(`${t.toFixed(2)}s / ${DURATION.toFixed(0)}s`, 96, H - 88);
  ctx.textAlign = 'right';
  ctx.fillText('1080 × 1920 · 30 FPS', W - 96, H - 88);
  ctx.restore();
}

function drawTitle(t) {
  const p = easeOut(t / 1.8);
  const rise = 48 * (1 - p);
  const cx = W / 2;

  ctx.save();
  ctx.globalAlpha = p;
  ctx.textAlign = 'center';

  ctx.fillStyle = C.muted;
  ctx.font = '600 24px "JetBrains Mono", "Cascadia Mono", Consolas, monospace';
  ctx.letterSpacing = '6px';
  ctx.fillText('REUSABLE  CAPABILITY  ENGINE', cx, 620 + rise);

  ctx.fillStyle = C.text;
  ctx.font = '800 104px "Microsoft YaHei", "PingFang SC", sans-serif';
  ctx.fillText(DATA?.title || 'AI 能力沉淀', cx, 770 + rise);

  ctx.fillStyle = C.blue;
  ctx.font = '700 42px "Microsoft YaHei", "PingFang SC", sans-serif';
  ctx.fillText(DATA?.subtitle || '不是一次性视频', cx, 850 + rise);

  // Underline grows from center.
  const lineW = 420 * easeOut((t - 0.35) / 1.4);
  const grad = ctx.createLinearGradient(cx - lineW / 2, 0, cx + lineW / 2, 0);
  grad.addColorStop(0, 'rgba(79,195,247,0)');
  grad.addColorStop(0.5, C.blue);
  grad.addColorStop(1, 'rgba(79,195,247,0)');
  ctx.fillStyle = grad;
  ctx.fillRect(cx - lineW / 2, 900, lineW, 3);
  ctx.restore();
}

function drawCapabilityCards(t) {
  const dataItems = Array.isArray(DATA?.items) ? DATA.items.slice(0, 3) : null;
  const cards = dataItems
    ? dataItems.map((item, index) => ({
        tag: `DATA ${index + 1}`,
        title: String(item.name ?? item.label ?? '数据项'),
        body: `${item.value ?? ''} ${DATA.unit ?? ''}`.trim() || '数据驱动',
        color: [C.blue, C.green, C.warm][index % 3],
      }))
    : [
        {tag: 'SHOT', title: '镜头', body: '一个镜头一个文件', color: C.blue},
        {tag: 'TECHNIQUE', title: '技法', body: '布局算法纯函数', color: C.green},
        {tag: 'CONTRACT', title: '契约', body: '数据先于画面', color: C.warm},
      ];
  const start = 2.1;
  const cardW = 264;
  const cardH = 330;
  const gap = 28;
  const total = cards.length * cardW + (cards.length - 1) * gap;
  const baseX = (W - total) / 2;

  cards.forEach((card, i) => {
    const p = easeOut((t - start - i * 0.16) / 0.7);
    if (p <= 0) return;
    const x = baseX + i * (cardW + gap);
    const y = 1030 - 70 * (1 - p);
    ctx.save();
    ctx.globalAlpha = p;
    ctx.fillStyle = C.panel;
    roundRect(x, y, cardW, cardH, 26);
    ctx.fill();
    ctx.strokeStyle = 'rgba(255,255,255,0.12)';
    ctx.lineWidth = 1.5;
    ctx.stroke();

    // Color strip.
    ctx.fillStyle = card.color;
    roundRect(x + 28, y + 28, 54, 8, 4);
    ctx.fill();

    ctx.fillStyle = card.color;
    ctx.font = '700 20px "JetBrains Mono", "Cascadia Mono", Consolas, monospace';
    ctx.letterSpacing = '3px';
    ctx.fillText(card.tag, x + 28, y + 78);

    ctx.fillStyle = C.text;
    ctx.font = '800 52px "Microsoft YaHei", "PingFang SC", sans-serif';
    ctx.fillText(card.title, x + 28, y + 156);

    ctx.fillStyle = C.muted;
    ctx.font = '500 24px "Microsoft YaHei", "PingFang SC", sans-serif';
    ctx.fillText(card.body, x + 28, y + 220);

    // Tiny animated bar.
    const bar = 130 * easeInOut((t - start - 0.5 - i * 0.15) / 1.4);
    ctx.fillStyle = 'rgba(255,255,255,0.12)';
    roundRect(x + 28, y + 270, 130, 6, 3);
    ctx.fill();
    ctx.fillStyle = card.color;
    roundRect(x + 28, y + 270, Math.max(0, bar), 6, 3);
    ctx.fill();
    ctx.restore();
  });
}

function drawFlow(t) {
  const start = 5.0;
  const p = easeOut((t - start) / 0.8);
  if (p <= 0) return;
  const steps = ['需求', '计划', '预览', '反馈'];
  const y = 1510;
  const x0 = 118;
  const x1 = W - 118;
  const stepW = (x1 - x0) / (steps.length - 1);

  ctx.save();
  ctx.globalAlpha = p;

  ctx.strokeStyle = 'rgba(255,255,255,0.18)';
  ctx.lineWidth = 2;
  ctx.setLineDash([10, 12]);
  ctx.lineDashOffset = -t * 42;
  ctx.beginPath();
  ctx.moveTo(x0, y);
  ctx.lineTo(x1, y);
  ctx.stroke();
  ctx.setLineDash([]);

  const dotX = x0 + (x1 - x0) * ((t - start) / 3.4 % 1);
  ctx.fillStyle = C.blue;
  ctx.shadowColor = C.blue;
  ctx.shadowBlur = 24;
  ctx.beginPath();
  ctx.arc(dotX, y, 7, 0, Math.PI * 2);
  ctx.fill();
  ctx.shadowBlur = 0;

  steps.forEach((label, i) => {
    const x = x0 + stepW * i;
    const active = t > start + i * 0.45;
    ctx.fillStyle = active ? C.text : C.dim;
    ctx.beginPath();
    ctx.arc(x, y, 15, 0, Math.PI * 2);
    ctx.fill();
    ctx.fillStyle = active ? C.bg : 'rgba(255,255,255,0.18)';
    ctx.beginPath();
    ctx.arc(x, y, 6, 0, Math.PI * 2);
    ctx.fill();
    ctx.textAlign = 'center';
    ctx.fillStyle = active ? C.text : C.dim;
    ctx.font = '600 27px "Microsoft YaHei", "PingFang SC", sans-serif';
    ctx.fillText(label, x, y + 66);
  });
  ctx.restore();
}

function drawFinale(t) {
  const start = 7.9;
  const p = easeOut((t - start) / 0.8);
  if (p <= 0) return;
  const cx = W / 2;
  ctx.save();
  ctx.globalAlpha = p;
  ctx.textAlign = 'center';

  ctx.fillStyle = C.green;
  ctx.font = '700 24px "JetBrains Mono", "Cascadia Mono", Consolas, monospace';
  ctx.letterSpacing = '5px';
  ctx.fillText('CAPABILITY LIBRARY / READY', cx, 760);

  ctx.fillStyle = C.text;
  ctx.font = '800 72px "Microsoft YaHei", "PingFang SC", sans-serif';
  ctx.fillText('越用越多', cx, 870);

  ctx.fillStyle = C.muted;
  ctx.font = '500 30px "Microsoft YaHei", "PingFang SC", sans-serif';
  ctx.fillText('需求 → 计划 → 审查 → 沉淀', cx, 930);

  // Framing corners.
  const s = 70;
  const m = 92;
  ctx.strokeStyle = 'rgba(79,195,247,0.55)';
  ctx.lineWidth = 3;
  for (const [x, y, dx, dy] of [[m, 205, 1, 1], [W - m, 205, -1, 1], [m, H - 205, 1, -1], [W - m, H - 205, -1, -1]]) {
    ctx.beginPath();
    ctx.moveTo(x + s * dx, y);
    ctx.lineTo(x, y);
    ctx.lineTo(x, y + s * dy);
    ctx.stroke();
  }
  ctx.restore();
}

function renderAt(t) {
  const time = clamp(t, 0, DURATION);
  ctx.clearRect(0, 0, W, H);
  drawBackground(time);
  drawChrome(time);
  drawTitle(time);
  drawCapabilityCards(time);
  drawFlow(time);
  drawFinale(time);
}

window.renderAt = renderAt;
window.__DURATION__ = DURATION;
window.__FPS__ = FPS;

document.fonts.ready.then(async () => {
  await loadData();
  renderAt(0);
  window.__READY__ = true;
});
