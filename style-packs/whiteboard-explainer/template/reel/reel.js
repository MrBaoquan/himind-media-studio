import rough from './vendor/rough.esm.js';

const W = 1080;
const H = 1920;
const FPS = 30;
const DURATION = 10;

const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const rc = rough.canvas(canvas);

const DEFAULT_DATA = {
  title: '从想法到成片',
  subtitle: '一条短视频的四步生产链',
  steps: [
    {label: '需求', title: '一句话需求', detail: 'AI 识别目标、数据与约束'},
    {label: '计划', title: '分镜与数据契约', detail: '镜头、时长、素材先确定'},
    {label: '审查', title: '预览与反馈', detail: '在创作台看见真实画面'},
    {label: '成片', title: '渲染与交付', detail: '质量门禁通过后输出 MP4'},
  ],
  conclusion: '沉淀能力，而不是重复劳动',
  source: 'HiMind 自媒体创作台',
};

let DATA = DEFAULT_DATA;

function normalizeData(value) {
  if (!value || typeof value !== 'object') return DEFAULT_DATA;
  const steps = Array.isArray(value.steps) ? value.steps.slice(0, 4) : [];
  while (steps.length < 4) {
    const fallback = DEFAULT_DATA.steps[steps.length];
    const item = steps[steps.length] || {};
    steps.push({
      label: String(item.label || fallback.label),
      title: String(item.title || fallback.title),
      detail: String(item.detail || fallback.detail),
    });
  }
  return {
    title: String(value.title || DEFAULT_DATA.title),
    subtitle: String(value.subtitle || DEFAULT_DATA.subtitle),
    steps,
    conclusion: String(value.conclusion || DEFAULT_DATA.conclusion),
    source: String(value.source || DEFAULT_DATA.source),
  };
}

async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) DATA = normalizeData(await response.json());
  } catch {
    DATA = DEFAULT_DATA;
  }
}

const C = {
  board: '#f7f5ee',
  paper: '#fffdf7',
  ink: '#26313b',
  soft: '#72808a',
  faint: 'rgba(38,49,59,0.11)',
  blue: '#2b74b5',
  orange: '#d47c26',
  green: '#2d8a68',
  red: '#c54a3d',
  yellow: '#f0c94b',
};

const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const ease = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
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

function setFont(size, weight = 400, family = 'body') {
  const stack = family === 'hand'
    ? '"KaiTi", "STKaiti", "Segoe Print", "Comic Sans MS", "Noto Sans SC", sans-serif'
    : family === 'mono'
      ? '"Cascadia Mono", "JetBrains Mono", Consolas, monospace'
      : '"Microsoft YaHei", "Noto Sans SC", "PingFang SC", sans-serif';
  ctx.font = `${weight} ${size}px ${stack}`;
}

function text(value, x, y, options = {}) {
  const {
    size = 32, weight = 400, family = 'body', color = C.ink,
    align = 'left', baseline = 'alphabetic', alpha = 1,
  } = options;
  ctx.save();
  ctx.globalAlpha *= alpha;
  ctx.fillStyle = color;
  ctx.textAlign = align;
  ctx.textBaseline = baseline;
  setFont(size, weight, family);
  ctx.fillText(value, x, y);
  ctx.restore();
}

function drawBoard(t) {
  ctx.fillStyle = C.board;
  ctx.fillRect(0, 0, W, H);

  ctx.save();
  ctx.strokeStyle = 'rgba(38,49,59,0.032)';
  ctx.lineWidth = 1;
  for (let x = 40; x < W; x += 40) {
    ctx.beginPath();
    ctx.moveTo(x, 0);
    ctx.lineTo(x, H);
    ctx.stroke();
  }
  for (let y = 40; y < H; y += 40) {
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(W, y);
    ctx.stroke();
  }
  for (let i = 0; i < 180; i += 1) {
    const x = hash(i + 1) * W;
    const y = hash(i + 101) * H;
    const a = 0.018 + hash(i + 201) * 0.025;
    ctx.fillStyle = `rgba(38,49,59,${a.toFixed(3)})`;
    ctx.fillRect(x, y, 2, 2);
  }
  ctx.restore();

  rc.rectangle(48, 48, W - 96, H - 96, {
    stroke: 'rgba(38,49,59,0.48)', strokeWidth: 3, roughness: 1.25, bowing: 1.1, seed: 11,
  });
  rc.line(92, 138, 986, 138, {
    stroke: 'rgba(38,49,59,0.28)', strokeWidth: 2, roughness: 1.1, seed: 12,
  });
  text('HIMIND / WHITEBOARD', 92, 112, {size: 22, weight: 600, family: 'mono', color: C.ink});
  text('EXPLAINER 01', 988, 112, {size: 22, weight: 600, family: 'mono', color: C.red, align: 'right'});

  const markerX = 92 + ((t * 105) % 560);
  rc.line(markerX, 154, markerX + 44, 154, {
    stroke: 'rgba(43,116,181,0.34)', strokeWidth: 5, roughness: 1.8, seed: Math.floor(t * 30) + 13,
  });
}

function drawTitle(t) {
  const p = ease(t / 1.25);
  if (p <= 0) return;

  ctx.save();
  ctx.beginPath();
  ctx.rect(90, 220, 900 * p, 270);
  ctx.clip();
  text(DATA.title, 112, 350, {size: 86, weight: 700, family: 'hand', color: C.ink});
  text(DATA.subtitle, 116, 424, {size: 32, weight: 500, family: 'body', color: C.soft});
  ctx.restore();

  const lineP = ease((t - 0.75) / 0.8);
  if (lineP > 0) {
    rc.line(116, 458, 116 + 610 * lineP, 458, {
      stroke: C.blue, strokeWidth: 7, roughness: 2.4, bowing: 2.2, seed: 21,
    });
  }

  const tagP = ease((t - 1.0) / 0.55);
  if (tagP > 0) {
    ctx.save();
    ctx.globalAlpha = tagP;
    ctx.translate(0, 12 * (1 - tagP));
    ctx.fillStyle = 'rgba(240,201,75,0.32)';
    ctx.fillRect(816, 300, 148, 52);
    text('可复用', 890, 338, {size: 29, weight: 700, family: 'hand', color: C.ink, align: 'center'});
    ctx.restore();
  }
}

const STEP_COLORS = [C.blue, C.orange, C.green, C.red];

function drawChatIcon(cx, cy, color, seed) {
  rc.rectangle(cx - 34, cy - 28, 68, 48, {
    stroke: color, strokeWidth: 3.4, roughness: 1.5, seed,
  });
  rc.line(cx - 12, cy + 20, cx - 2, cy + 35, {stroke: color, strokeWidth: 3.2, roughness: 1.4, seed: seed + 1});
  rc.line(cx - 2, cy + 35, cx + 10, cy + 20, {stroke: color, strokeWidth: 3.2, roughness: 1.4, seed: seed + 2});
  rc.line(cx - 16, cy - 10, cx + 17, cy - 10, {stroke: color, strokeWidth: 2.4, roughness: 1.1, seed: seed + 3});
  rc.line(cx - 16, cy + 2, cx + 6, cy + 2, {stroke: color, strokeWidth: 2.4, roughness: 1.1, seed: seed + 4});
}

function drawBoardIcon(cx, cy, color, seed) {
  rc.rectangle(cx - 39, cy - 30, 78, 60, {stroke: color, strokeWidth: 3.2, roughness: 1.5, seed});
  rc.rectangle(cx - 29, cy - 20, 22, 40, {stroke: color, strokeWidth: 2.4, roughness: 1.2, seed: seed + 1});
  rc.rectangle(cx - 3, cy - 20, 22, 17, {stroke: color, strokeWidth: 2.4, roughness: 1.2, seed: seed + 2});
  rc.rectangle(cx - 3, cy + 3, 22, 17, {stroke: color, strokeWidth: 2.4, roughness: 1.2, seed: seed + 3});
}

function drawEyeIcon(cx, cy, color, seed) {
  rc.ellipse(cx, cy, 82, 46, {stroke: color, strokeWidth: 3.3, roughness: 1.5, seed});
  rc.circle(cx, cy, 23, {stroke: color, strokeWidth: 3.2, roughness: 1.3, seed: seed + 1});
  rc.circle(cx - 4, cy - 4, 7, {stroke: color, strokeWidth: 2.2, roughness: 1.0, seed: seed + 2});
  rc.line(cx - 60, cy + 34, cx - 40, cy + 14, {stroke: color, strokeWidth: 2.6, roughness: 1.1, seed: seed + 3});
}

function drawPlayIcon(cx, cy, color, seed) {
  rc.circle(cx, cy, 70, {stroke: color, strokeWidth: 3.2, roughness: 1.4, seed});
  rc.polygon([[cx - 13, cy - 23], [cx - 13, cy + 23], [cx + 26, cy]], {
    stroke: color, strokeWidth: 3.2, roughness: 1.25, seed: seed + 1,
  });
}

function drawStepCard(index, y, t) {
  const appear = 1.75 + index * 0.82;
  const p = ease((t - appear) / 0.82);
  if (p <= 0) return;

  const x = 140;
  const w = 800;
  const h = 196;
  const cx = x + w / 2;
  const cy = y + h / 2;
  const rot = index % 2 === 0 ? -0.0045 : 0.0045;
  const color = STEP_COLORS[index];
  const step = DATA.steps[index];

  ctx.save();
  ctx.globalAlpha = clamp(p * 1.35, 0, 1);
  ctx.translate(0, 18 * (1 - p));
  ctx.translate(cx, cy);
  ctx.rotate(rot);
  ctx.translate(-cx, -cy);

  ctx.save();
  ctx.shadowColor = 'rgba(38,49,59,0.10)';
  ctx.shadowBlur = 18;
  ctx.shadowOffsetY = 9;
  roundRect(x, y, w, h, 18);
  ctx.fillStyle = C.paper;
  ctx.fill();
  ctx.restore();

  ctx.save();
  ctx.beginPath();
  ctx.rect(x, y, w * p, h);
  ctx.clip();
  rc.rectangle(x, y, w, h, {
    stroke: 'rgba(38,49,59,0.62)', strokeWidth: 3.2, roughness: 1.45, bowing: 1.2, seed: 100 + index * 20,
  });
  rc.line(x + 132, y + 22, x + 132, y + h - 22, {
    stroke: color, strokeWidth: 2.4, roughness: 1.4, seed: 101 + index * 20,
  });
  rc.circle(x + 80, cy, 92, {
    stroke: color, strokeWidth: 3.2, roughness: 1.5, seed: 102 + index * 20,
  });

  if (index === 0) drawChatIcon(x + 80, cy, color, 120);
  if (index === 1) drawBoardIcon(x + 80, cy, color, 140);
  if (index === 2) drawEyeIcon(x + 80, cy, color, 160);
  if (index === 3) drawPlayIcon(x + 80, cy, color, 180);

  text(String(index + 1).padStart(2, '0'), x + 28, y + 36, {
    size: 20, weight: 700, family: 'mono', color, alpha: 0.72,
  });
  text(`STEP ${String(index + 1).padStart(2, '0')} / ${step.label}`, x + 164, y + 46, {
    size: 20, weight: 700, family: 'mono', color,
  });
  text(step.title, x + 164, y + 100, {size: 43, weight: 700, family: 'hand', color: C.ink});
  text(step.detail, x + 164, y + 145, {size: 25, weight: 400, color: C.soft});
  ctx.restore();
  ctx.restore();
}

function drawConnectors(t) {
  const startY = 755;
  const gap = 250;
  for (let i = 0; i < 3; i += 1) {
    const p = ease((t - (2.45 + i * 0.82)) / 0.5);
    if (p <= 0) continue;
    const x = 220;
    const y1 = startY + i * gap + 6;
    const y2 = startY + (i + 1) * gap - 8;
    ctx.save();
    ctx.beginPath();
    ctx.rect(x - 20, y1, 40, (y2 - y1) * p);
    ctx.clip();
    rc.line(x, y1, x, y2, {
      stroke: 'rgba(38,49,59,0.48)', strokeWidth: 3.2, roughness: 1.5, seed: 300 + i,
    });
    rc.line(x - 11, y2 - 14, x, y2, {stroke: 'rgba(38,49,59,0.48)', strokeWidth: 3, roughness: 1.1, seed: 310 + i});
    rc.line(x + 11, y2 - 14, x, y2, {stroke: 'rgba(38,49,59,0.48)', strokeWidth: 3, roughness: 1.1, seed: 320 + i});
    ctx.restore();
  }
}

function drawPipelineBracket(t) {
  const p = ease((t - 5.3) / 0.9);
  if (p <= 0) return;
  ctx.save();
  ctx.globalAlpha = clamp(p * 1.25, 0, 1);
  ctx.beginPath();
  ctx.rect(58, 560, 80, 1050 * p);
  ctx.clip();
  rc.line(100, 590, 100, 1580, {
    stroke: C.blue, strokeWidth: 4.6, roughness: 1.9, bowing: 1.5, seed: 401,
  });
  rc.line(100, 590, 126, 590, {stroke: C.blue, strokeWidth: 4.2, roughness: 1.4, seed: 402});
  rc.line(100, 1580, 126, 1580, {stroke: C.blue, strokeWidth: 4.2, roughness: 1.4, seed: 403});
  ctx.restore();
  text('ONE PIPELINE', 78, 1070, {
    size: 18, weight: 700, family: 'mono', color: C.blue, align: 'center', alpha: clamp(p * 1.2, 0, 1),
  });
}

function drawReviewRing(t) {
  const p = ease((t - 7.0) / 0.75);
  if (p <= 0) return;
  ctx.save();
  ctx.globalAlpha = clamp(p * 1.4, 0, 1);
  ctx.beginPath();
  ctx.rect(110, 1020, 860 * p, 300);
  ctx.clip();
  rc.ellipse(542, 1140, 790, 230, {
    stroke: C.yellow, strokeWidth: 9, roughness: 2.4, bowing: 1.8, seed: 501,
  });
  ctx.restore();

  text('反馈闭环', 854, 1000, {
    size: 28, weight: 700, family: 'hand', color: C.red, align: 'right', alpha: clamp((t - 7.45) / 0.35, 0, 1),
  });
  ctx.save();
  ctx.globalAlpha = clamp((t - 7.45) / 0.35, 0, 1);
  rc.line(824, 1014, 788, 1050, {stroke: C.red, strokeWidth: 3.2, roughness: 1.5, seed: 502});
  rc.line(788, 1050, 810, 1046, {stroke: C.red, strokeWidth: 3.2, roughness: 1.2, seed: 503});
  ctx.restore();
}

function drawConclusion(t) {
  const p = ease((t - 8.15) / 0.8);
  if (p <= 0) return;
  ctx.save();
  ctx.globalAlpha = clamp(p * 1.25, 0, 1);
  ctx.translate(0, 16 * (1 - p));
  text(DATA.conclusion, 540, 1718, {
    size: 54, weight: 700, family: 'hand', color: C.ink, align: 'center',
  });
  ctx.restore();

  const lineP = ease((t - 8.5) / 0.6);
  if (lineP > 0) {
    rc.line(540 - 280 * lineP, 1756, 540 + 280 * lineP, 1756, {
      stroke: C.red, strokeWidth: 5.5, roughness: 2.2, bowing: 1.6, seed: 601,
    });
  }

  const checkP = ease((t - 8.7) / 0.45);
  if (checkP > 0) {
    ctx.save();
    ctx.globalAlpha = checkP;
    rc.line(224, 1688, 250, 1716, {stroke: C.green, strokeWidth: 6, roughness: 1.3, seed: 602});
    rc.line(250, 1716, 305, 1648, {stroke: C.green, strokeWidth: 6, roughness: 1.3, seed: 603});
    ctx.restore();
  }

  text(DATA.source, 540, 1812, {
    size: 19, weight: 400, family: 'mono', color: C.soft, align: 'center',
    alpha: clamp((t - 8.9) / 0.5, 0, 1),
  });
}

function drawMarkerTip(t) {
  const index = clamp(Math.floor((t - 1.6) / 0.82), 0, 3);
  const local = (t - 1.6) - index * 0.82;
  if (local < 0 || local > 0.9) return;
  const y = 590 + index * 250 + 196 * ease(local / 0.65);
  const x = 140 + 800 * ease(local / 0.65);
  ctx.save();
  ctx.globalAlpha = clamp(1 - Math.abs(local - 0.45) / 0.45, 0, 1);
  ctx.fillStyle = STEP_COLORS[index];
  ctx.beginPath();
  ctx.arc(clamp(x, 160, 920), clamp(y, 610, 1510), 6, 0, Math.PI * 2);
  ctx.fill();
  ctx.restore();
}

function renderAt(t) {
  const time = clamp(t, 0, DURATION);
  ctx.clearRect(0, 0, W, H);
  drawBoard(time);
  drawTitle(time);
  for (let i = 0; i < 4; i += 1) drawStepCard(i, 590 + i * 250, time);
  drawConnectors(time);
  drawPipelineBracket(time);
  drawReviewRing(time);
  drawConclusion(time);
  drawMarkerTip(time);
}

window.renderAt = renderAt;
window.__DURATION__ = DURATION;
window.__FPS__ = FPS;

loadData().then(() => {
  document.fonts.ready.then(() => {
    renderAt(0);
    window.__READY__ = true;
  });
});
