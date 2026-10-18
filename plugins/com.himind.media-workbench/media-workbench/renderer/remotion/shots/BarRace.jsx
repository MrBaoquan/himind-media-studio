import React, {useMemo} from 'react';
import {AbsoluteFill, useCurrentFrame, useVideoConfig} from 'remotion';
import {Background} from '../foundation/background.jsx';
import {FONT_FAMILY, NUM_FONT, canvasSize, makePx, resolvePalette, safeArea} from '../foundation/theme.js';
import {entrance} from '../foundation/motion.js';
import {formatValue} from '../foundation/format.js';
import {buildRaceSeries, raceStateAt} from '../techniques/race.js';

/**
 * shot.bar-race —— 时序排名竞速。
 *
 * 整数年稳定快照 + 帧间平滑插值，避免名次抖动。
 * 数据契约见 catalog/contracts/bar-race.schema.json。
 */
export const BarRace = (props) => {
  const frame = useCurrentFrame();
  const {fps, durationInFrames: compositionDuration} = useVideoConfig();
  const durationInFrames = Number(props.durationFrames) || compositionDuration;
  const {width, height} = canvasSize(props);
  const px = makePx({width});
  const palette = resolvePalette(props);
  const safe = safeArea(props, px);
  const contentW = width - safe.left - safe.right;

  const data = props.data || {};
  const items = Array.isArray(data.items) ? data.items : [];
  const start = Number(data.years?.start ?? 2008);
  const end = Number(data.years?.end ?? 2025);
  const unit = data.unit ?? '';
  const title = props.title ?? data.title ?? '';

  const series = useMemo(() => buildRaceSeries(items, {start, end}), [items, start, end]);

  const animStart = Math.round(durationInFrames * 0.06);
  const animEnd = Math.round(durationInFrames * 0.94);
  const progress = Math.max(0, Math.min(1, (frame - animStart) / Math.max(1, animEnd - animStart)));
  const yearFloat = start + (end - start) * progress;
  const year = Math.floor(yearFloat);

  const rows = useMemo(() => raceStateAt(series, yearFloat, items), [series, yearFloat, items]);
  const maxValue = Math.max(1, ...rows.map((row) => row.value));

  const maxVisible = Math.min(10, Math.max(3, items.length));
  const rowsTop = safe.top + px(560);
  // 行高自适应：条目少时行更高，避免画面下半部大片留白。
  const availableRows = Math.max(1, height - rowsTop - safe.bottom - px(150));
  const rowH = Math.min(px(300), Math.max(px(130), Math.floor(availableRows / maxVisible)));
  const rowsHeight = rowH * maxVisible;
  const badgeSize = px(56);

  return (
    <AbsoluteFill style={{backgroundColor: palette.bg0, color: palette.text, fontFamily: FONT_FAMILY, overflow: 'hidden'}}>
      <Background palette={palette} px={px} />

      <div style={{position: 'absolute', top: safe.top, left: safe.left, width: contentW, paddingTop: px(20)}}>
        <div style={{fontSize: px(56), color: palette.muted, letterSpacing: '0.06em'}}>{title}</div>
        <div style={{display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', marginTop: px(10)}}>
          <span style={{fontFamily: NUM_FONT, fontSize: px(200), fontWeight: 800, lineHeight: 1, color: palette.accent, fontVariantNumeric: 'tabular-nums'}}>
            {year}
          </span>
          <span style={{fontSize: px(36), color: palette.watermark, paddingBottom: px(16)}}>{start} — {end}</span>
        </div>
      </div>

      <div style={{position: 'absolute', top: rowsTop, left: safe.left, width: contentW, height: rowsHeight, overflow: 'hidden'}}>
        {rows.map((row) => {
          const y = row.rank * rowH;
          if (row.rank > maxVisible + 0.5) return null;
          const opacity = Math.max(0, Math.min(1, (maxVisible + 0.5 - row.rank) / 1.2)) * entrance({frame, fps, delay: 0, preset: 'fade-up', distance: 0}).opacity;
          const barWidth = Math.max(0, Math.min(1, row.value / maxValue));
          const isTop3 = row.rank < 3;
          return (
            <div key={row.id} style={{position: 'absolute', top: y, left: 0, width: '100%', height: rowH, display: 'flex', alignItems: 'center', gap: px(16), opacity}}>
              <span
                style={{
                  width: badgeSize, height: badgeSize, borderRadius: px(10),
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  fontFamily: NUM_FONT, fontSize: px(30), fontWeight: 800,
                  background: isTop3 ? palette.accent : palette.panel,
                  color: isTop3 ? palette.bg0 : palette.muted,
                }}
              >
                {Math.floor(row.rank) + 1}
              </span>
              <span style={{width: px(230), fontSize: px(40), fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis'}}>
                {row.name}
              </span>
              <div style={{flex: 1, height: px(44), borderRadius: px(999), background: palette.line, overflow: 'hidden'}}>
                <div style={{height: '100%', width: `${barWidth * 100}%`, background: row.color || palette.accent, borderRadius: px(999), boxShadow: `0 0 ${px(18)}px ${(row.color || palette.accent)}55`}} />
              </div>
              <span style={{width: px(250), textAlign: 'right', fontFamily: NUM_FONT, fontSize: px(40), fontVariantNumeric: 'tabular-nums'}}>
                {formatValue(row.value)} <span style={{fontSize: px(28), color: palette.muted}}>{unit}</span>
              </span>
            </div>
          );
        })}
      </div>

      <div style={{position: 'absolute', left: safe.left, right: safe.right, bottom: safe.bottom, display: 'flex', alignItems: 'center', gap: px(16)}}>
        <span style={{fontFamily: NUM_FONT, fontSize: px(28), color: palette.watermark}}>{start}</span>
        <div style={{flex: 1, height: px(10), borderRadius: px(999), background: palette.line, overflow: 'hidden'}}>
          <div style={{height: '100%', width: `${progress * 100}%`, background: `linear-gradient(90deg, ${palette.accent}, ${palette.accent2})`}} />
        </div>
        <span style={{fontFamily: NUM_FONT, fontSize: px(28), color: palette.watermark}}>{end}</span>
      </div>
    </AbsoluteFill>
  );
};
