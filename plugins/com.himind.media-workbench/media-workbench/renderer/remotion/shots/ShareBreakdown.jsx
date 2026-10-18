import React, {useMemo} from 'react';
import {AbsoluteFill, useCurrentFrame, useVideoConfig} from 'remotion';
import {Background} from '../foundation/background.jsx';
import {FONT_FAMILY, NUM_FONT, canvasSize, makePx, resolvePalette, safeArea} from '../foundation/theme.js';
import {countUpProgress, easeOutCubic, entrance, progressAt, stagger} from '../foundation/motion.js';
import {formatValue} from '../foundation/format.js';
import {fitFontSize} from '../foundation/text.js';

/**
 * shot.share-breakdown —— 占比 / 构成。
 *
 * 用于「钱花在哪」「时间怎么分配」「收入由什么组成」这类问题。
 * 一个环形图回答整体，一条 100% 堆叠条回答结构，图例回答具体数值。
 * 数据契约见 catalog/contracts/composition.schema.json。
 */

function polar(cx, cy, radius, degrees) {
  const rad = ((degrees - 90) * Math.PI) / 180;
  return [cx + radius * Math.cos(rad), cy + radius * Math.sin(rad)];
}

function arcPath(cx, cy, radius, startDegrees, endDegrees) {
  const start = Math.max(0, Math.min(359.99, startDegrees));
  const end = Math.max(start, Math.min(359.99, endDegrees));
  const [x1, y1] = polar(cx, cy, radius, start);
  const [x2, y2] = polar(cx, cy, radius, end);
  const largeArc = end - start > 180 ? 1 : 0;
  return `M ${x1.toFixed(3)} ${y1.toFixed(3)} A ${radius} ${radius} 0 ${largeArc} 1 ${x2.toFixed(3)} ${y2.toFixed(3)}`;
}

const Notice = ({palette, px, title, detail}) => (
  <AbsoluteFill style={{backgroundColor: palette.bg0, color: palette.text, fontFamily: FONT_FAMILY, alignItems: 'center', justifyContent: 'center', gap: px(18), padding: px(120)}}>
    <div style={{fontSize: px(56), fontWeight: 700, textAlign: 'center'}}>{title}</div>
    {detail ? <div style={{fontSize: px(36), color: palette.muted, textAlign: 'center', lineHeight: 1.5}}>{detail}</div> : null}
  </AbsoluteFill>
);

export const ShareBreakdown = (props) => {
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
  const unit = data.unit ?? props.unit ?? '';
  const totalLabel = data.total_label ?? '总计';
  const source = props.source ?? data.source ?? '';
  const valueDecimals = data.decimals;

  const categories = useMemo(() => {
    const raw = Array.isArray(data.categories) ? data.categories : [];
    return raw
      .map((item, index) => {
        const value = Number(item && item.value);
        return {
          label: String((item && (item.label ?? item.name)) ?? `分类 ${index + 1}`),
          value: Number.isFinite(value) && value > 0 ? value : 0,
          color: item && typeof item.color === 'string' ? item.color : '',
        };
      })
      .filter((item) => item.value > 0)
      .slice(0, 8);
  }, [data]);

  const total = useMemo(() => categories.reduce((sum, item) => sum + item.value, 0), [categories]);
  const series = Array.isArray(palette.series) && palette.series.length ? palette.series : [palette.accent, palette.accent2, palette.ok];

  const donutSize = Math.min(px(1350), contentW * 0.72);
  const stroke = Math.max(px(52), Math.round(donutSize * 0.12));
  const radius = donutSize / 2 - stroke / 2 - px(24);
  const donutTop = safe.top + px(360);
  const donutCenter = donutSize / 2;
  const barTop = donutTop + donutSize + px(112);
  const legendTop = barTop + px(196);
  const columns = categories.length > 2 ? 2 : 1;
  const rowH = px(128);
  const rows = Math.ceil(categories.length / columns);
  const legendBottom = legendTop + rows * rowH;
  const insightTop = Math.min(height - safe.bottom - px(210), legendBottom + px(42));

  const progress = countUpProgress(frame, durationInFrames, {startRatio: 0.08, endRatio: 0.6});
  const barProgress = easeOutCubic(progressAt(frame, Math.round(durationInFrames * 0.1), Math.round(durationInFrames * 0.48)));

  const segments = useMemo(() => {
    const gapDeg = categories.length > 1 ? 1.35 : 0;
    let cursor = 0;
    return categories.map((item, index) => {
      const share = total > 0 ? item.value / total : 0;
      const start = cursor + gapDeg / 2;
      const end = cursor + share * 360 - gapDeg / 2;
      cursor += share * 360;
      return {...item, index, share, start, end, color: item.color || series[index % series.length]};
    });
  }, [categories, total, series]);

  const top = segments.length ? [...segments].sort((a, b) => b.value - a.value)[0] : null;
  const insight = data.insight || (top && total > 0
    ? `${top.label} 占比 ${(top.share * 100).toFixed(top.share >= 0.1 ? 0 : 1)}%，是最大的构成项`
    : '');
  const titleSize = fitFontSize(title, contentW, px(78), px(48));
  const centerSize = fitFontSize(formatValue(total, {decimals: valueDecimals}), donutSize * 0.55, px(150), px(72));
  const insightSize = fitFontSize(insight, contentW * 0.9, px(36), px(24));

  if (!categories.length || total <= 0) {
    return (
      <AbsoluteFill style={{backgroundColor: palette.bg0}}>
        <Background palette={palette} px={px} />
        <Notice
          palette={palette}
          px={px}
          title="这个构成图还没有分类数据"
          detail='在 recipe.data.categories 里放入 [{ "label": "交通", "value": 2860 }]'
        />
      </AbsoluteFill>
    );
  }

  return (
    <AbsoluteFill style={{backgroundColor: palette.bg0, color: palette.text, fontFamily: FONT_FAMILY, overflow: 'hidden'}}>
      <Background palette={palette} px={px} />

      <div style={{position: 'absolute', top: safe.top, left: safe.left, width: contentW, textAlign: 'center'}}>
        <div style={{...entrance({frame, fps, delay: Math.round(fps * 0.06), preset: 'fade-up'}), fontSize: titleSize, fontWeight: 800, lineHeight: 1.24}}>
          {title || '构成占比'}
        </div>
        {subtitle ? (
          <div style={{...entrance({frame, fps, delay: Math.round(fps * 0.12), preset: 'fade-up'}), marginTop: px(14), fontSize: px(38), color: palette.muted, letterSpacing: '0.03em'}}>
            {subtitle}
          </div>
        ) : null}
      </div>

      <svg
        width={donutSize}
        height={donutSize}
        viewBox={`0 0 ${donutSize} ${donutSize}`}
        style={{position: 'absolute', top: donutTop, left: (width - donutSize) / 2, overflow: 'visible'}}
      >
        <circle cx={donutCenter} cy={donutCenter} r={radius} fill="none" stroke={palette.line} strokeWidth={stroke} />
        {segments.map((segment) => {
          const media = segment.end - segment.start;
          if (media <= 0) return null;
          const startFrame = Math.round(durationInFrames * (0.08 + segment.index * 0.035));
          const segmentProgress = easeOutCubic(progressAt(frame, startFrame, Math.round(durationInFrames * 0.36)));
          if (segmentProgress <= 0.001) return null;
          const end = segment.start + media * segmentProgress;
          return (
            <path
              key={`${segment.label}-${segment.index}`}
              d={arcPath(donutCenter, donutCenter, radius, segment.start, end)}
              fill="none"
              stroke={segment.color}
              strokeWidth={stroke}
              strokeLinecap="butt"
            />
          );
        })}
      </svg>

      <div
        style={{
          position: 'absolute',
          top: donutTop,
          left: (width - donutSize) / 2,
          width: donutSize,
          height: donutSize,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          textAlign: 'center',
          pointerEvents: 'none',
        }}
      >
        <div style={{fontSize: px(34), color: palette.muted, letterSpacing: '0.1em'}}>{totalLabel}</div>
        <div
          style={{
            fontFamily: NUM_FONT,
            fontSize: centerSize,
            fontWeight: 800,
            lineHeight: 1.05,
            marginTop: px(6),
            color: palette.text,
            fontVariantNumeric: 'tabular-nums',
            letterSpacing: '-0.025em',
          }}
        >
          {formatValue(total * progress, {decimals: valueDecimals})}
        </div>
        {unit ? <div style={{fontSize: px(40), color: palette.muted, marginTop: px(8)}}>{unit}</div> : null}
        <div style={{fontSize: px(28), color: palette.watermark, marginTop: px(12)}}>{categories.length} 类构成</div>
      </div>

      <div style={{position: 'absolute', top: barTop, left: safe.left, width: contentW}}>
        <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: px(14)}}>
          <span style={{fontSize: px(32), color: palette.muted, letterSpacing: '0.06em'}}>构成占比</span>
          <span style={{fontFamily: NUM_FONT, fontSize: px(30), color: palette.watermark}}>100%</span>
        </div>
        <div style={{height: px(72), borderRadius: px(999), background: palette.line, overflow: 'hidden'}}>
          <div style={{height: '100%', width: `${barProgress * 100}%`, display: 'flex', borderRadius: px(999), overflow: 'hidden'}}>
            {segments.map((segment) => (
              <div
                key={`bar-${segment.index}`}
                style={{width: `${segment.share * 100}%`, height: '100%', background: segment.color, borderRight: segment.index === segments.length - 1 ? 'none' : `${px(3)}px solid ${palette.bg0}`}}
              />
            ))}
          </div>
        </div>
      </div>

      <div
        style={{
          position: 'absolute',
          top: legendTop,
          left: safe.left,
          width: contentW,
          display: 'grid',
          gridTemplateColumns: columns === 2 ? '1fr 1fr' : '1fr',
          columnGap: px(64),
          rowGap: px(10),
        }}
      >
        {segments.map((segment, index) => {
          const inStyle = entrance({frame, fps, delay: stagger(index, {base: Math.round(fps * 0.25), step: 5}), preset: 'fade-up', distance: px(26)});
          const percent = segment.share * 100 * progress;
          return (
            <div
              key={`legend-${segment.index}`}
              style={{
                ...inStyle,
                height: rowH - px(10),
                display: 'flex',
                alignItems: 'center',
                gap: px(18),
                minWidth: 0,
                borderBottom: `${px(2)}px solid ${palette.line}`,
              }}
            >
              <span style={{width: px(24), height: px(24), borderRadius: px(8), background: segment.color, flexShrink: 0, boxShadow: `0 0 ${px(16)}px ${segment.color}55`}} />
              <div style={{flex: 1, minWidth: 0}}>
                <div style={{fontSize: px(42), fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis'}}>{segment.label}</div>
                <div style={{marginTop: px(4), fontFamily: NUM_FONT, fontSize: px(28), color: palette.muted, fontVariantNumeric: 'tabular-nums'}}>
                  {formatValue(segment.value, {decimals: valueDecimals})}{unit ? ` ${unit}` : ''}
                </div>
              </div>
              <span style={{fontFamily: NUM_FONT, fontSize: px(52), fontWeight: 800, color: segment.color, fontVariantNumeric: 'tabular-nums'}}>
                {percent.toFixed(percent >= 10 ? 0 : 1)}%
              </span>
            </div>
          );
        })}
      </div>

      {insight ? (
        <div
          style={{
            position: 'absolute',
            top: insightTop,
            left: safe.left,
            width: contentW,
            display: 'flex',
            justifyContent: 'center',
          }}
        >
          <div
            style={{
              ...entrance({frame, fps, delay: Math.round(durationInFrames * 0.62), preset: 'fade-up', distance: px(30)}),
              display: 'inline-flex',
              alignItems: 'center',
              gap: px(14),
              padding: `${px(14)}px ${px(32)}px`,
              borderRadius: px(999),
              background: palette.panel,
              border: `${px(2)}px solid ${palette.line}`,
              color: palette.text,
              fontSize: insightSize,
              fontWeight: 600,
            }}
          >
            <span style={{width: px(12), height: px(12), borderRadius: px(999), background: palette.accent}} />
            {insight}
          </div>
        </div>
      ) : null}

      {source ? (
        <div style={{position: 'absolute', left: safe.left, right: safe.right, bottom: safe.bottom, textAlign: 'center', fontSize: px(30), color: palette.watermark}}>
          {source}
        </div>
      ) : null}
    </AbsoluteFill>
  );
};


