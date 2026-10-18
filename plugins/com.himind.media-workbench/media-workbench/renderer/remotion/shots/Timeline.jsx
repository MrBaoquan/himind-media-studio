import React, {useMemo} from 'react';
import {AbsoluteFill, useCurrentFrame, useVideoConfig} from 'remotion';
import {Background} from '../foundation/background.jsx';
import {FONT_FAMILY, NUM_FONT, canvasSize, makePx, resolvePalette, safeArea} from '../foundation/theme.js';
import {entrance, stagger} from '../foundation/motion.js';
import {spread1D} from '../foundation/layout1d.js';

/**
 * shot.timeline —— 纵向事件时间线。
 *
 * 事件在时间轴上的真实位置会重叠，用 spread1D 做一维避让：
 * 位置尽量贴近真实年份，同时保证最小间距。
 * 数据契约见 catalog/contracts/timeline.schema.json。
 */
export const Timeline = (props) => {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const {width, height} = canvasSize(props);
  const px = makePx({width});
  const palette = resolvePalette(props);
  const safe = safeArea(props, px);
  const contentW = width - safe.left - safe.right;

  const data = props.data || {};
  const events = Array.isArray(data.events) ? [...data.events] : [];
  const title = props.title ?? data.title ?? '';
  const unit = data.unit ?? '年';

  const areaTop = safe.top + px(380);
  const areaBottom = height - safe.bottom - px(120);
  const spineX = safe.left + contentW / 2;
  const cardW = contentW * 0.44;

  const layout = useMemo(() => {
    if (!events.length) return [];
    const years = events.map((event) => Number(event.year) || 0);
    const minYear = Math.min(...years);
    const maxYear = Math.max(...years);
    const span = Math.max(1, maxYear - minYear);
    const anchors = years.map((year) => areaTop + ((year - minYear) / span) * (areaBottom - areaTop));
    const positions = spread1D(anchors, {minGap: px(210), min: areaTop, max: areaBottom});
    const order = events
      .map((event, index) => ({event, index, y: positions[index]}))
      .sort((a, b) => a.y - b.y);
    return order;
  }, [events, areaTop, areaBottom, px]);

  if (!layout.length) {
    return (
      <AbsoluteFill style={{backgroundColor: palette.bg0, color: palette.text, fontFamily: FONT_FAMILY, alignItems: 'center', justifyContent: 'center'}}>
        <div style={{fontSize: px(48), color: palette.muted}}>这个时间线还没有事件</div>
      </AbsoluteFill>
    );
  }

  return (
    <AbsoluteFill style={{backgroundColor: palette.bg0, color: palette.text, fontFamily: FONT_FAMILY, overflow: 'hidden'}}>
      <Background palette={palette} px={px} />

      <div style={{position: 'absolute', top: safe.top, left: safe.left, width: contentW, paddingTop: px(20), textAlign: 'center'}}>
        <div style={{fontSize: px(72), fontWeight: 800, lineHeight: 1.25}}>{title}</div>
      </div>

      <div style={{position: 'absolute', top: areaTop - px(40), left: spineX - px(1), width: px(2), height: areaBottom - areaTop + px(80), background: palette.line}} />
      <div
        style={{
          position: 'absolute',
          top: areaTop - px(40),
          left: spineX - px(2),
          width: px(4),
          height: (areaBottom - areaTop + px(80)) * Math.min(1, frame / Math.max(1, layout.length * 12)),
          background: palette.accent,
          borderRadius: px(2),
        }}
      />

      {layout.map(({event, y}, index) => {
        const leftSide = index % 2 === 0;
        const inStyle = entrance({frame, fps, delay: stagger(index, {base: Math.round(fps * 0.25), step: 6}), preset: leftSide ? 'slide-left' : 'slide-right', distance: px(60)});
        return (
          <div key={index} style={{position: 'absolute', top: y - px(56), left: 0, width: '100%', height: px(112)}}>
            <div
              style={{
                position: 'absolute',
                left: spineX - px(11),
                top: px(44),
                width: px(22),
                height: px(22),
                borderRadius: px(11),
                background: palette.accent,
                boxShadow: `0 0 ${px(18)}px ${palette.accent}88`,
              }}
            />
            <div
              style={{
                ...inStyle,
                position: 'absolute',
                top: 0,
                width: cardW,
                ...(leftSide ? {right: width - (spineX - px(28))} : {left: spineX + px(28)}),
                display: 'flex',
                flexDirection: 'column',
                gap: px(6),
                ...(leftSide ? {alignItems: 'flex-end', textAlign: 'right'} : {alignItems: 'flex-start', textAlign: 'left'}),
              }}
            >
              <div style={{fontFamily: NUM_FONT, fontSize: px(54), fontWeight: 800, color: palette.accent}}>
                {event.year}{unit}
              </div>
              <div style={{fontSize: px(50), fontWeight: 600, lineHeight: 1.3}}>{event.title}</div>
              {event.description ? (
                <div style={{fontSize: px(36), color: palette.muted, lineHeight: 1.4}}>{event.description}</div>
              ) : null}
            </div>
          </div>
        );
      })}
    </AbsoluteFill>
  );
};
