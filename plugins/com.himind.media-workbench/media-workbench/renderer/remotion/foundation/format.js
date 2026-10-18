// foundation/format.js —— 数字/单位/百分比的中文表达。
//
// 中文短视频里"2,972 公里"和"1.23 亿"是两种完全不同的语感，
// 不要指望调用方每次自己拼。

/** 千分位。 */
export function groupDigits(value, decimals = 0) {
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  return n.toLocaleString('en-US', {minimumFractionDigits: decimals, maximumFractionDigits: decimals});
}

/** 紧凑中文：12345 → 1.23万；1.2e8 → 1.20亿。 */
export function compactCN(value, {decimals = 2} = {}) {
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  const abs = Math.abs(n);
  if (abs >= 1e8) return `${(n / 1e8).toFixed(decimals).replace(/\.?0+$/, '')}亿`;
  if (abs >= 1e4) return `${(n / 1e4).toFixed(decimals).replace(/\.?0+$/, '')}万`;
  return groupDigits(n);
}

/** 主数值展示：整数走千分位，小数保留指定位数。 */
export function formatValue(value, {decimals} = {}) {
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  const d = decimals ?? (Number.isInteger(n) ? 0 : 1);
  return groupDigits(n, d);
}

/** 变化量：接受 12.4 / "12.4%" / "+12.4" 三种输入，统一成带符号百分比。 */
export function formatDelta(raw) {
  if (raw === null || raw === undefined || raw === '') return '';
  const text = String(raw).trim();
  const sign = /^-/.test(text) ? '-' : '+';
  const number = Number(text.replace(/[^0-9.\-]/g, ''));
  if (!Number.isFinite(number)) return text;
  const magnitude = Math.abs(number);
  return `${sign}${magnitude.toFixed(magnitude % 1 === 0 ? 0 : 1)}%`;
}

export function deltaTone(raw) {
  if (raw === null || raw === undefined || raw === '') return 'neutral';
  const text = String(raw);
  if (/^-/.test(text.trim())) return 'down';
  const n = Number(text.replace(/[^0-9.\-]/g, ''));
  if (Number.isFinite(n) && n < 0) return 'down';
  return 'up';
}

/** 单位与数值之间的排版间距（中文习惯不加空格）。 */
export function withUnit(valueText, unit) {
  return unit ? `${valueText} ${unit}` : valueText;
}
