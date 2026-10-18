// foundation/motion.js —— 入场编排预设。
//
// 所有镜头共用同一套节奏语言，避免"每个模板一套动效"。

import {interpolate, spring} from 'remotion';

const clamp = {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'};

export function easeOutCubic(t) {
  const v = Math.max(0, Math.min(1, t));
  return 1 - Math.pow(1 - v, 3);
}

/** 0..1 进度，带延迟。 */
export function progressAt(frame, start, duration) {
  if (duration <= 0) return frame >= start ? 1 : 0;
  return interpolate(frame, [start, start + duration], [0, 1], clamp);
}

/**
 * 入场预设。返回可直接展开到 style 的 {opacity, transform}。
 *
 * preset:
 *   fade-up     淡入 + 上移（默认，最稳）
 *   spring-pop  弹簧缩放（用于数字、徽章）
 *   slide-left  从左滑入（用于对比左侧）
 *   slide-right 从右滑入（用于对比右侧）
 */
export function entrance({frame, fps, delay = 0, preset = 'fade-up', distance = 40}) {
  const local = frame - delay;
  if (local <= 0) return {opacity: 0, transform: 'translateY(0px) scale(1)'};

  if (preset === 'spring-pop') {
    const s = spring({frame: local, fps, config: {damping: 14, stiffness: 120, mass: 0.6}});
    return {opacity: Math.min(1, s * 1.4), transform: `scale(${0.86 + 0.14 * s})`};
  }
  if (preset === 'slide-left') {
    const p = easeOutCubic(local / 24);
    return {opacity: p, transform: `translateX(${(1 - p) * -distance}px)`};
  }
  if (preset === 'slide-right') {
    const p = easeOutCubic(local / 24);
    return {opacity: p, transform: `translateX(${(1 - p) * distance}px)`};
  }
  const p = easeOutCubic(local / 24);
  return {opacity: p, transform: `translateY(${(1 - p) * distance}px)`};
}

/** 交错延迟：第 index 个元素。 */
export function stagger(index, {base = 0, step = 6} = {}) {
  return base + index * step;
}

/** 数值滚动进度（0..1），自动留出入场时间。 */
export function countUpProgress(frame, durationInFrames, {startRatio = 0.08, endRatio = 0.42} = {}) {
  const start = Math.round(durationInFrames * startRatio);
  const end = Math.round(durationInFrames * endRatio);
  return easeOutCubic(progressAt(frame, start, Math.max(1, end - start)));
}
