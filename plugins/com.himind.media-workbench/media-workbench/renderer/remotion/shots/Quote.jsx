import React from 'react';
import {AbsoluteFill, useCurrentFrame, useVideoConfig} from 'remotion';
import {Background} from '../foundation/background.jsx';
import {FONT_FAMILY, canvasSize, makePx, resolvePalette, safeArea} from '../foundation/theme.js';
import {entrance, stagger} from '../foundation/motion.js';
import {wrapText} from '../foundation/text.js';

/**
 * shot.quote —— 引语。
 *
 * 不需要数据，纯排版；但决定了"账号质感"。
 * 数据契约见 catalog/contracts/quote.schema.json。
 */
export const Quote = (props) => {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const {width} = canvasSize(props);
  const px = makePx({width});
  const palette = resolvePalette(props);
  const safe = safeArea(props, px);
  const contentW = width - safe.left - safe.right;

  const data = props.data || {};
  const text = String(data.text ?? props.title ?? '');
  const author = data.author ?? '';
  const role = data.role ?? '';
  const source = data.source ?? props.source ?? '';

  const baseSize = px(text.length > 40 ? 132 : 168);
  const lines = wrapText(text, contentW * 0.92, baseSize, 5);

  const glyphIn = entrance({frame, fps, delay: 4, preset: 'fade-up', distance: px(24)});
  const authorIn = entrance({frame, fps, delay: Math.round(fps * 0.8), preset: 'fade-up'});

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
          justifyContent: 'center',
        }}
      >
        <div style={{...glyphIn, fontSize: px(300), lineHeight: 1, color: palette.accent, opacity: 0.3, height: px(210)}}>
          “
        </div>

        <div style={{fontSize: baseSize, fontWeight: 700, lineHeight: 1.5, color: palette.text}}>
          {lines.map((line, index) => {
            const style = entrance({frame, fps, delay: stagger(index, {base: Math.round(fps * 0.18), step: 5}), preset: 'fade-up', distance: px(28)});
            return (
              <div key={index} style={style}>
                {line}
              </div>
            );
          })}
        </div>

        {author ? (
          <div style={{...authorIn, marginTop: px(72), display: 'flex', alignItems: 'center', gap: px(20)}}>
            <div style={{width: px(64), height: px(3), background: palette.accent, borderRadius: px(2)}} />
            <div style={{fontSize: px(56), fontWeight: 600}}>{author}</div>
            {role ? <div style={{fontSize: px(42), color: palette.muted}}>{role}</div> : null}
          </div>
        ) : null}

        {source ? (
          <div style={{marginTop: px(24), fontSize: px(30), color: palette.watermark}}>{source}</div>
        ) : null}
      </div>
    </AbsoluteFill>
  );
};
