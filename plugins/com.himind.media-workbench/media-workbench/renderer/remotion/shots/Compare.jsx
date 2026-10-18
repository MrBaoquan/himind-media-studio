import React from 'react';
import {AbsoluteFill, useCurrentFrame, useVideoConfig} from 'remotion';
import {Background} from '../foundation/background.jsx';
import {FONT_FAMILY, NUM_FONT, canvasSize, makePx, resolvePalette, safeArea} from '../foundation/theme.js';
import {countUpProgress, entrance} from '../foundation/motion.js';
import {formatValue} from '../foundation/format.js';
import {fitFontSize} from '../foundation/text.js';

/**
 * shot.compare —— 对比。
 *
 * 用于同比、对标、前后、A/B。竖屏里上下两栏比左右两栏更易读。
 * 数据契约见 catalog/contracts/compare.schema.json。
 */
export const Compare = (props) => {
  const frame = useCurrentFrame();
  const {fps, durationInFrames: compositionDuration} = useVideoConfig();
  // 作为独立成片时用整片时长；被 Storyboard 包进 <Sequence> 时必须用显式传入的镜头时长
  // （useVideoConfig().durationInFrames 在 Sequence 内仍返回整片时长）。
  const durationInFrames = Number(props.durationFrames) || compositionDuration;
  const {width} = canvasSize(props);
  const px = makePx({width});
  const palette = resolvePalette(props);
  const safe = safeArea(props, px);
  const contentW = width - safe.left - safe.right;

  const data = props.data || {};
  const title = props.title ?? data.title ?? '';
  const subtitle = props.subtitle ?? data.subtitle ?? '';
  const unit = data.unit ?? props.unit ?? '';
  const left = data.left || {};
  const right = data.right || {};
  const verdict = data.verdict ?? '';
  const source = props.source ?? data.source ?? '';

  const leftValue = Number(left.value) || 0;
  const rightValue = Number(right.value) || 0;
  const maxValue = Math.max(1, Math.abs(leftValue), Math.abs(rightValue));

  const progress = countUpProgress(frame, durationInFrames, {startRatio: 0.12, endRatio: 0.5});
  const leftShown = leftValue * progress;
  const rightShown = rightValue * progress;

  const fontSize = fitFontSize(formatValue(Math.max(Math.abs(leftValue), Math.abs(rightValue))), contentW * 0.8, px(240), px(120));

  const panel = (side, item, shown, sideColor, delay, preset) => {
    const inStyle = entrance({frame, fps, delay, preset, distance: px(90)});
    const barPct = Math.min(100, (Math.abs(Number(item.value) || 0) / maxValue) * 100) * progress;
    return (
      <div style={{...inStyle, flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center', gap: px(18), padding: `${px(28)}px ${px(8)}px`}}>
        <div style={{fontSize: px(56), color: palette.muted, letterSpacing: '0.04em'}}>{item.label ?? side}</div>
        <div style={{display: 'flex', alignItems: 'baseline', gap: px(12)}}>
          <span style={{fontFamily: NUM_FONT, fontSize, fontWeight: 800, color: sideColor, fontVariantNumeric: 'tabular-nums', lineHeight: 1}}>
            {formatValue(shown, {decimals: item.decimals})}
          </span>
          {unit ? <span style={{fontSize: px(52), color: palette.muted}}>{unit}</span> : null}
        </div>
        <div style={{height: px(28), borderRadius: px(999), background: palette.line, overflow: 'hidden'}}>
          <div style={{height: '100%', width: `${barPct}%`, background: sideColor, borderRadius: px(999), boxShadow: `0 0 ${px(24)}px ${sideColor}66`}} />
        </div>
        {item.note ? <div style={{fontSize: px(40), color: palette.muted}}>{item.note}</div> : null}
      </div>
    );
  };

  return (
    <AbsoluteFill style={{backgroundColor: palette.bg0, color: palette.text, fontFamily: FONT_FAMILY, overflow: 'hidden'}}>
      <Background palette={palette} px={px} />

      <div
        style={{
          position: 'absolute',
          inset: 0,
          paddingLeft: safe.left,
          paddingRight: safe.right,
          paddingTop: safe.top,
          paddingBottom: safe.bottom,
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <div style={{...entrance({frame, fps, delay: Math.round(fps * 0.08), preset: 'fade-up'}), textAlign: 'center', flexShrink: 0}}>
          {title ? <div style={{fontSize: px(76), fontWeight: 800, lineHeight: 1.25}}>{title}</div> : null}
          {subtitle ? <div style={{marginTop: px(12), fontSize: px(38), color: palette.muted}}>{subtitle}</div> : null}
        </div>

        <div style={{flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center', position: 'relative'}}>
          {panel('A', left, leftShown, palette.accent, Math.round(fps * 0.2), 'slide-left')}
          <div style={{height: px(2), background: palette.line, margin: `${px(6)}px 0`}} />
          {panel('B', right, rightShown, palette.accent2, Math.round(fps * 0.32), 'slide-right')}
        </div>

        {verdict ? (
          <div
            style={{
              ...entrance({frame, fps, delay: Math.round(fps * 0.85), preset: 'fade-up'}),
              alignSelf: 'center',
              background: palette.accent,
              color: palette.bg0,
              fontSize: px(52),
              fontWeight: 800,
              padding: `${px(14)}px ${px(36)}px`,
              borderRadius: px(999),
              letterSpacing: '0.02em',
            }}
          >
            {verdict}
          </div>
        ) : null}

        {source ? (
          <div style={{marginTop: px(28), textAlign: 'center', fontSize: px(30), color: palette.watermark}}>{source}</div>
        ) : null}
      </div>
    </AbsoluteFill>
  );
};
