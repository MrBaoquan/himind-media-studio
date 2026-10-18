// foundation/text.js —— 确定性的文本度量与自适应。
//
// 不依赖 DOM 测量：Remotion 多线程渲染时，DOM 测量需要 delayRender 等待字体，
// 且不同机器字体回退会让结果漂移（破坏"同配方同输出"）。
// 这里用 CJK 感知的宽度估算，跨机器确定。

const CJK = /[\u3000-\u303f\u3400-\u4dbf\u4e00-\u9fff\uf900-\ufaff\uff00-\uffef]/;

/** 估算单行文本宽度（px）。 */
export function estimateTextWidth(text, fontSize, ratios = {}) {
  const {cjk = 1.0, latin = 0.54, digit = 0.58, space = 0.28, punct = 0.45} = ratios;
  let width = 0;
  for (const ch of String(text ?? '')) {
    if (ch === ' ') width += fontSize * space;
    else if (CJK.test(ch)) width += fontSize * cjk;
    else if (/[0-9]/.test(ch)) width += fontSize * digit;
    else if (/[.,:;!?'"()[\]{}\-_/\\|]/.test(ch)) width += fontSize * punct;
    else width += fontSize * latin;
  }
  return width;
}

/** 在 maxWidth 内把字号从 base 缩到 min，返回可用字号。 */
export function fitFontSize(text, maxWidth, baseSize, minSize = 24) {
  if (!text) return baseSize;
  const width = estimateTextWidth(text, baseSize);
  if (width <= maxWidth) return baseSize;
  const scaled = Math.floor((baseSize * maxWidth) / width);
  return Math.max(minSize, scaled);
}

/** 按宽度折行，最多 maxLines 行；超出时最后一行加省略号。 */
export function wrapText(text, maxWidth, fontSize, maxLines = 4) {
  const chars = [...String(text ?? '')];
  const lines = [];
  let current = '';
  for (const ch of chars) {
    const candidate = current + ch;
    if (estimateTextWidth(candidate, fontSize) > maxWidth && current) {
      lines.push(current);
      current = ch;
      if (lines.length === maxLines) break;
    } else {
      current = candidate;
    }
  }
  if (lines.length < maxLines && current) lines.push(current);
  if (lines.length === maxLines) {
    const consumed = lines.join('').length;
    if (consumed < chars.length) {
      lines[maxLines - 1] = lines[maxLines - 1].replace(/.$/, '…');
    }
  }
  return lines;
}

/** 数字字符宽度更整齐，用于让计数器不抖动。 */
export function tabularStyle(fontFamily) {
  return {fontFamily, fontVariantNumeric: 'tabular-nums', letterSpacing: '-0.01em'};
}
