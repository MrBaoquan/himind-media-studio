import React, {useMemo} from 'react';
import {AbsoluteFill, useCurrentFrame, useVideoConfig} from 'remotion';
import {Background} from '../foundation/background.jsx';
import {FONT_FAMILY, canvasSize, makePx, resolvePalette, safeArea} from '../foundation/theme.js';
import {easeOutCubic, entrance, progressAt, stagger} from '../foundation/motion.js';
import {fitFontSize} from '../foundation/text.js';

/**
 * shot.flow —— 流程 / 链路 / 步骤。
 *
 * 与 timeline 的分工：
 *   timeline 回答"什么时候发生"（带年份/时间顺序）
 *   flow     回答"怎么做/怎么走"（步骤、环节、链路）
 * 数据契约见 catalog/contracts/flow.schema.json。
 */
export const Flow = (props) => {
  const frame = useCurrentFrame();
  const {fps, durationInFrames: compositionDuration} = useVideoConfig();
  const durationInFrames = Number(props.durationFrames) || compositionDuration;
  const {width, height} = canvasSize(props);
  const px = makePx({width});
  const palette = resolvePalette(props);
  const safe = safeArea(props, px);
  const contentW = width - safe.left - safe.right;

  const data = props.data || {};
  const title = props.title ?? data.title ?? '';
  const subtitle = props.subtitle ?? data.subtitle ?? '';
  const source = props.source ?? data.source ?? '';
  const conclusion = data.conclusion ?? '';
  const steps = useMemo(() => {
    const raw = Array.isArray(data.steps) ? data.steps : [];
    return raw
      .map((item, index) => ({
        label: String((item && (item.label ?? item.title)) ?? `步骤 ${index + 1}`),
        description: String((item && (item.description ?? item.note)) ?? ''),
      }))
      .slice(0, 6);
  }, [data]);
  const series = Array.isArray(palette.series) && palette.series.length ? palette.series : [palette.accent, palette.accent2];

  const headerHeight = px(subtitle ? 300 : 240);
  const contentTop = safe.top + headerHeight;
  const bottomReserve = px(conclusion ? 190 : 70);
  const contentBottom = height - safe.bottom - bottomReserve;
  const count = steps.length;
  const gap = px(22);
  const available = Math.max(px(240), contentBottom - contentTop);
  const stepH = count > 0 ? Math.min(px(360), Math.max(px(150), (available - gap * Math.max(0, count - 1)) / count)) : px(240);
  const used = stepH * count + gap * Math.max(0, count - 1);
  const startY = contentTop + Math.max(0, (available - used) / 2);
  const badgeX = safe.left + px(66);
  const cardLeft = safe.left + px(190);
  const cardW = contentW - px(190);
  const titleSize = fitFontSize(title, contentW, px(78), px(48));
  const lineProgress = easeOutCubic(progressAt(frame, Math.round(durationInFrames * 0.08), Math.round(durationInFrames * 0.42)));

  if (!count) {
    return (
      <AbsoluteFill style={{backgroundColor: palette.bg0, fontFamily: FONT_FAMILY}}>
        <Background palette={palette} px={px} />
        <div style={{position: 'absolute', inset: 0, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: px(16), color: palette.text, padding: px(120), textAlign: 'center'}}>
          <div style={{fontSize: px(56), fontWeight: 700}}>这个流程图还没有步骤</div>
          <div style={{fontSize: px(36), color: palette.muted, lineHeight: 1.5}}>
            在 recipe.data.steps 里放入 {`[{ "label": "需求", "description": "确认选题" }]`}
          </div>
        </div>
      </AbsoluteFill>
    );
  }

  return (
    <AbsoluteFill style={{backgroundColor: palette.bg0, color: palette.text, fontFamily: FONT_FAMILY, overflow: 'hidden'}}>
      <Background palette={palette} px={px} />

      <div style={{position: 'absolute', top: safe.top, left: safe.left, width: contentW, textAlign: 'center'}}>
        <div style={{...entrance({frame, fps, delay: Math.round(fps * 0.05), preset: 'fade-up'}), fontSize: titleSize, fontWeight: 800, lineHeight: 1.24}}>
          {title || '流程'}
        </div>
        {subtitle ? (
          <div style={{...entrance({frame, fps, delay: Math.round(fps * 0.12), preset: 'fade-up'}), marginTop: px(14), fontSize: px(38), color: palette.muted, letterSpacing: '0.03em'}}>
            {subtitle}
          </div>
        ) : null}
      </div>

      {/* 主轴：从第一个节点生长到最后一个节点。 */}
      <div style={{position: 'absolute', left: badgeX - px(2), top: startY + stepH / 2, width: px(4), height: Math.max(0, (count - 1) * (stepH + gap)), background: palette.line, borderRadius: px(999)}} />
      <div style={{position: 'absolute', left: badgeX - px(2), top: startY + stepH / 2, width: px(4), height: Math.max(0, (count - 1) * (stepH + gap)) * lineProgress, background: `linear-gradient(180deg, ${palette.accent}, ${palette.accent2})`, borderRadius: px(999), boxShadow: `0 0 ${px(20)}px ${palette.accent}55`}} />

      {steps.map((step, index) => {
        const indexIn = entrance({frame, fps, delay: stagger(index, {base: Math.round(fps * 0.16), step: 7}), preset: 'slide-right', distance: px(50)});
        const badgeIn = entrance({frame, fps, delay: stagger(index, {base: Math.round(fps * 0.12), step: 7}), preset: 'spring-pop'});
        const y = startY + index * (stepH + gap);
        const color = series[index % series.length];
        return (
          <div key={`${step.label}-${index}`} style={{position: 'absolute', left: 0, top: y, width: '100%', height: stepH}}>
            <div style={{...badgeIn, position: 'absolute', left: badgeX - px(48), top: stepH / 2 - px(48), width: px(96), height: px(96), borderRadius: px(999), background: color, color: palette.bg0, display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Helvetica Neue', Arial, sans-serif", fontSize: px(38), fontWeight: 800, boxShadow: `0 0 ${px(24)}px ${color}55`}}>
              {String(index + 1).padStart(2, '0')}
            </div>
            <div style={{...indexIn, position: 'absolute', left: cardLeft, top: 0, width: cardW, height: stepH, background: palette.panel, border: `${px(2)}px solid ${palette.line}`, borderRadius: px(30), padding: `${px(28)}px ${px(36)}px`, display: 'flex', flexDirection: 'column', justifyContent: 'center', gap: px(8), overflow: 'hidden'}}>
              <div style={{fontSize: px(50), fontWeight: 700, lineHeight: 1.2}}>{step.label}</div>
              {step.description ? <div style={{fontSize: px(32), color: palette.muted, lineHeight: 1.38}}>{step.description}</div> : null}
            </div>
          </div>
        );
      })}

      {conclusion ? (
        <div style={{position: 'absolute', left: safe.left, right: safe.right, bottom: safe.bottom + px(28), display: 'flex', justifyContent: 'center'}}>
          <div style={{...entrance({frame, fps, delay: Math.round(durationInFrames * 0.66), preset: 'fade-up', distance: px(24)}), display: 'inline-flex', alignItems: 'center', gap: px(12), padding: `${px(12)}px ${px(30)}px`, borderRadius: px(999), background: palette.accent, color: palette.bg0, fontSize: px(34), fontWeight: 800}}>
            {conclusion}
          </div>
        </div>
      ) : null}

      {source ? (
        <div style={{position: 'absolute', left: safe.left, right: safe.right, bottom: safe.bottom, textAlign: 'center', fontSize: px(28), color: palette.watermark}}>
          {source}
        </div>
      ) : null}
    </AbsoluteFill>
  );
};

