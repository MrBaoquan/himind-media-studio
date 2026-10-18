// foundation/theme.js —— 所有镜头共用的主题解析。
//
// 风格是声明资产：渲染器只认 DTCG 令牌，不认具体配色方案。
// 新增风格 = 改 styles.json，不改这里。

export const FONT_FAMILY = "'Microsoft YaHei', 'PingFang SC', 'Noto Sans CJK SC', sans-serif";
export const NUM_FONT = "'Helvetica Neue', Arial, sans-serif";

const SERIES_FALLBACK = ['#4fc3f7', '#ff8e53', '#3ddc97', '#ffca28', '#ab47bc', '#26c6da', '#ec407a', '#90a4ae'];

export const PALETTES = {
  'tech-blue': {
    bg0: '#080c1a', bg1: '#0f1629', bg2: '#131b33',
    panel: 'rgba(255,255,255,0.04)', line: 'rgba(255,255,255,0.08)',
    text: '#ffffff', muted: 'rgba(255,255,255,0.55)',
    accent: '#4fc3f7', accent2: '#ff8e53', ok: '#3ddc97', err: '#ff6b6b', watermark: 'rgba(255,255,255,0.18)',
    series: SERIES_FALLBACK,
  },
  'culture-ink': {
    bg0: '#0b0f14', bg1: '#131a22', bg2: '#1b2430',
    panel: 'rgba(255,255,255,0.05)', line: 'rgba(255,255,255,0.10)',
    text: '#f5f1e8', muted: 'rgba(245,241,232,0.55)',
    accent: '#d8b26a', accent2: '#8fb8a8', ok: '#8fb8a8', err: '#e08a7a', watermark: 'rgba(245,241,232,0.20)',
    series: ['#d8b26a', '#8fb8a8', '#c9775f', '#6f8fa6', '#b9a17b', '#7f8c7a', '#a56a5a', '#8f98a3'],
  },
};

const tokenValue = (tokens, key, fallback) => {
  if (!tokens) return fallback;
  const entry = tokens[key];
  const value = entry && typeof entry === 'object' ? entry.$value : entry;
  return value === undefined || value === null || value === '' ? fallback : value;
};

/** 从 recipe.style_tokens 解析调色板；没有令牌时按 style_id 回退。 */
export function resolvePalette(props) {
  const tokens = props && props.style_tokens;
  if (tokens && tokens['color.bg']) {
    const series = Array.from({length: 8}, (_, index) => tokenValue(tokens, `color.series.${index + 1}`, SERIES_FALLBACK[index]));
    return {
      bg0: tokenValue(tokens, 'color.bg', '#080c1a'),
      bg1: tokenValue(tokens, 'color.bgSecondary', '#0f1629'),
      bg2: tokenValue(tokens, 'color.bgSecondary', '#131b33'),
      panel: tokenValue(tokens, 'color.panel', 'rgba(255,255,255,0.04)'),
      line: tokenValue(tokens, 'color.line', 'rgba(255,255,255,0.08)'),
      text: tokenValue(tokens, 'color.text', '#ffffff'),
      muted: tokenValue(tokens, 'color.muted', 'rgba(255,255,255,0.55)'),
      accent: tokenValue(tokens, 'color.accent', '#4fc3f7'),
      accent2: tokenValue(tokens, 'color.accentWarm', '#ff8e53'),
      ok: '#3ddc97',
      err: '#ff6b6b',
      watermark: 'rgba(255,255,255,0.18)',
      series,
    };
  }
  return PALETTES[props && props.style_id] || PALETTES['tech-blue'];
}

/** 画布缩放：设计基准 2160 宽。 */
export function makePx(canvas) {
  const width = Number(canvas && canvas.width) || 2160;
  const scale = width / 2160;
  return (value) => Math.max(1, Math.round(value * scale));
}

export function canvasSize(props) {
  const canvas = (props && props.canvas) || {};
  return {
    width: Number(canvas.width) || 2160,
    height: Number(canvas.height) || 3840,
    fps: Number(canvas.fps) || 30,
  };
}

export function safeArea(props, px) {
  const safe = (props && props.safe_area) || {};
  return {
    top: Number(safe.top) || px(170),
    right: Number(safe.right) || px(200),
    left: Number(safe.left) || px(110),
    bottom: Number(safe.bottom) || px(220),
  };
}


/** 给 #rrggbb 颜色加透明度；解析失败时原样返回。 */
export function withAlpha(color, alpha) {
  if (typeof color !== 'string') return color;
  const match = color.trim().match(/^#([0-9a-fA-F]{6})$/);
  if (!match) return color;
  const value = parseInt(match[1], 16);
  const r = (value >> 16) & 255;
  const g = (value >> 8) & 255;
  const b = value & 255;
  return `rgba(${r},${g},${b},${Math.max(0, Math.min(1, alpha))})`;
}

