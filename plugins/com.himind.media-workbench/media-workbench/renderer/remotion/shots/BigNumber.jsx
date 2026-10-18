import React from 'react';
import {AbsoluteFill, useCurrentFrame, useVideoConfig} from 'remotion';
import {Background} from '../foundation/background.jsx';
import {FONT_FAMILY, NUM_FONT, canvasSize, makePx, resolvePalette, safeArea} from '../foundation/theme.js';
import {countUpProgress, entrance} from '../foundation/motion.js';
import {deltaTone, formatDelta, formatValue, withUnit} from '../foundation/format.js';
import {estimateTextWidth, fitFontSize} from '../foundation/text.js';

/**
 * shot.big-number —— 数字冲击。
 *
 * 用于钩子、结论、里程碑。只做一件事：让一个数字成为画面主体。
 * 数据契约见 catalog/contracts/big-number.schema.json。
 */
export const BigNumber = (props) => {
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
  const value = Number(data.value ?? props.value ?? 0);
  const unit = data.unit ?? props.unit ?? '';
  const label = data.label ?? props.title ?? '';
  const note = data.note ?? props.subtitle ?? '';
  const delta = formatDelta(data.delta ?? props.delta ?? '');
  const source = props.source ?? data.source ?? '';
  const decimals = data.decimals;

  const progress = countUpProgress(frame, durationInFrames);
  const shown = value * progress;
  // 用最终文本算字号，避免滚动过程中字号跳变
  // 单位单独一行：中文单位（公里/亿元）占宽很大，挤在同一行会迫使数字缩小。
  const numberText = formatValue(value, {decimals});
  const baseSize = px(640);
  const fontSize = fitFontSize(numberText, contentW, baseSize, px(240));

  const labelIn = entrance({frame, fps, delay: Math.round(fps * 0.1), preset: 'fade-up'});
  const numberIn = entrance({frame, fps, delay: Math.round(fps * 0.15), preset: 'spring-pop'});
  const noteIn = entrance({frame, fps, delay: Math.round(fps * 0.7), preset: 'fade-up'});

  const tone = deltaTone(delta);
  const deltaColor = tone === 'down' ? palette.err : palette.ok;

  return (
    <AbsoluteFill style={{backgroundColor: palette.bg0, color: palette.text, fontFamily: FONT_FAMILY, overflow: 'hidden'}}>
      <Background palette={palette} px={px} />

      <div
        style={{
          position: 'absolute',
          inset: 0,
          paddingLeft: safe.left,
          paddingRight: safe.right,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          gap: px(36),
        }}
      >
        {label ? (
          <div
            style={{
              ...labelIn,
              fontSize: px(60),
              color: palette.muted,
              letterSpacing: '0.06em',
              textAlign: 'center',
            }}
          >
            {label}
          </div>
        ) : null}

        <div style={{...numberIn, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: px(6)}}>
          <span
            style={{
              fontFamily: NUM_FONT,
              fontSize,
              fontWeight: 800,
              lineHeight: 1.0,
              color: palette.accent,
              fontVariantNumeric: 'tabular-nums',
              letterSpacing: '-0.02em',
              textShadow: `0 ${px(10)}px ${px(50)}px ${palette.accent}55`,
            }}
          >
            {formatValue(shown, {decimals})}
          </span>
          {unit ? (
            <span style={{fontSize: px(190), color: palette.muted, fontWeight: 600, lineHeight: 1.1}}>{unit}</span>
          ) : null}
        </div>

        {delta ? (
          <div
            style={{
              ...entrance({frame, fps, delay: Math.round(fps * 0.55), preset: 'fade-up'}),
              display: 'inline-flex',
              alignItems: 'center',
              gap: px(10),
              padding: `${px(10)}px ${px(28)}px`,
              borderRadius: px(999),
              background: `${deltaColor}22`,
              border: `${px(2)}px solid ${deltaColor}55`,
              color: deltaColor,
              fontSize: px(56),
              fontWeight: 700,
              fontFamily: NUM_FONT,
            }}
          >
            {delta}
          </div>
        ) : null}

        {note ? (
          <div style={{...noteIn, fontSize: px(48), color: palette.muted, textAlign: 'center', maxWidth: contentW * 0.86}}>
            {note}
          </div>
        ) : null}
      </div>

      {source ? (
        <div style={{position: 'absolute', left: safe.left, right: safe.right, bottom: safe.bottom, textAlign: 'center', fontSize: px(30), color: palette.watermark}}>
          {source}
        </div>
      ) : null}
    </AbsoluteFill>
  );
};
