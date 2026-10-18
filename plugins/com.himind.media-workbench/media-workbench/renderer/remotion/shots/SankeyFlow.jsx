import React, {useMemo} from 'react';
import {AbsoluteFill, interpolate, useCurrentFrame, useVideoConfig} from 'remotion';
import {Background} from '../foundation/background.jsx';
import {FONT_FAMILY, NUM_FONT, canvasSize, makePx, resolvePalette, safeArea, withAlpha} from '../foundation/theme.js';
import {easeOutCubic, entrance, progressAt, stagger} from '../foundation/motion.js';
import {formatValue} from '../foundation/format.js';
import {fitFontSize} from '../foundation/text.js';
import {buildSankeyLayout} from '../techniques/sankey.js';

/**
 * shot.sankey-flow —— 桑基流向图。
 *
 * 适合回答“总量流向了哪些部分”，与 composition 的环形占比不同：
 * composition 强调比例，sankey 强调流向和分配关系。
 * 数据契约见 catalog/contracts/sankey.schema.json。
 */
export const SankeyFlow = (props) => {
  const frame = useCurrentFrame();
  const {fps, durationInFrames: compositionDuration} = useVideoConfig();
  const durationInFrames = Number(props.durationFrames) || compositionDuration;
  const {width, height} = canvasSize(props);
  const px = makePx({width});
  const palette = resolvePalette(props);
  const safe = safeArea(props, px);

  const data = props.data || {};
  const links = useMemo(() => (Array.isArray(data.links) ? data.links : []), [data]);
  const unit = data.unit ?? props.unit ?? '';
  const totalLabel = data.total_label ?? '总量';
  const title = props.title ?? data.title ?? '';
  const subtitle = props.subtitle ?? data.subtitle ?? '';
  const source = props.source ?? data.source ?? '';
  const series = Array.isArray(palette.series) && palette.series.length ? palette.series : [palette.accent, palette.accent2, palette.ok];

  const contentTop = safe.top + px(320);
  const flowHeight = Math.max(px(240), height - safe.bottom - contentTop - px(90));
  const layout = useMemo(() => buildSankeyLayout(links, {height: flowHeight, gap: px(18), minValue: 0}), [links, flowHeight]);
  const leftX = safe.left + px(250);
  const rightX = width - safe.right - px(430);
  const nodeW = px(30);
  const linkStartX = leftX + nodeW;
  const linkEndX = rightX;

  const targetColors = new Map((layout.nodes.targets || []).map((node, index) => [node.id, series[index % series.length]]));
  const titleSize = fitFontSize(title, width - safe.left - safe.right, px(76), px(48));

  if (!layout.total || !layout.links.length) {
    return (
      <AbsoluteFill style={{backgroundColor: palette.bg0, fontFamily: FONT_FAMILY}}>
        <Background palette={palette} px={px} />
        <div style={{position: 'absolute', inset: 0, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: px(16), color: palette.text, padding: px(120), textAlign: 'center'}}>
          <div style={{fontSize: px(56), fontWeight: 700}}>这个桑基图还没有流向数据</div>
          <div style={{fontSize: px(36), color: palette.muted, lineHeight: 1.5}}>
            在 recipe.data.links 里放入 {`[{ "source": "文旅消费", "target": "交通", "value": 2860 }]`}
          </div>
        </div>
      </AbsoluteFill>
    );
  }

  return (
    <AbsoluteFill style={{backgroundColor: palette.bg0, color: palette.text, fontFamily: FONT_FAMILY, overflow: 'hidden'}}>
      <Background palette={palette} px={px} />

      <div style={{position: 'absolute', top: safe.top, left: safe.left, width: width - safe.left - safe.right, textAlign: 'center'}}>
        <div style={{...entrance({frame, fps, delay: Math.round(fps * 0.05), preset: 'fade-up'}), fontSize: titleSize, fontWeight: 800, lineHeight: 1.22}}>
          {title || '流量分配'}
        </div>
        {subtitle ? (
          <div style={{...entrance({frame, fps, delay: Math.round(fps * 0.12), preset: 'fade-up'}), marginTop: px(14), fontSize: px(38), color: palette.muted, letterSpacing: '0.03em'}}>
            {subtitle}
          </div>
        ) : null}
        <div style={{...entrance({frame, fps, delay: Math.round(fps * 0.2), preset: 'fade-up'}), display: 'inline-flex', gap: px(12), marginTop: px(22), alignItems: 'baseline', color: palette.muted, fontSize: px(28), letterSpacing: '0.08em'}}>
          <span>{totalLabel}</span>
          <span style={{fontFamily: NUM_FONT, color: palette.accent, fontWeight: 800}}>{formatValue(layout.total)}</span>
          <span>{unit}</span>
        </div>
      </div>

      <svg width={width} height={flowHeight} style={{position: 'absolute', top: contentTop, left: 0, overflow: 'visible'}}>
        <defs>
          {layout.links.map((link) => (
            <clipPath key={`clip-${link.index}`} id={`sankey-link-${link.index}`}>
              <rect x={linkStartX} y={0} width={Math.max(0, (linkEndX - linkStartX) * linkProgress(frame, durationInFrames, link.index))} height={flowHeight} />
            </clipPath>
          ))}
        </defs>

        {layout.links.map((link) => {
          const color = targetColors.get(link.target) || palette.accent;
          const path = ribbonPath(linkStartX, linkEndX, link.sourceY, link.targetY, link.thickness);
          return (
            <path
              key={`link-${link.index}`}
              d={path}
              fill={withAlpha(color, 0.28)}
              stroke={withAlpha(color, 0.35)}
              strokeWidth={px(2)}
              clipPath={`url(#sankey-link-${link.index})`}
            />
          );
        })}

        {layout.nodes.sources.map((node, index) => (
          <NodeBar key={`source-${node.id}`} node={node} x={leftX} width={nodeW} color={palette.accent} frame={frame} durationInFrames={durationInFrames} index={index} />
        ))}
        {layout.nodes.targets.map((node, index) => (
          <NodeBar key={`target-${node.id}`} node={node} x={rightX} width={nodeW} color={targetColors.get(node.id) || palette.accent} frame={frame} durationInFrames={durationInFrames} index={index + layout.nodes.sources.length} />
        ))}
      </svg>

      <div style={{position: 'absolute', top: contentTop, left: 0, width: '100%', height: flowHeight, pointerEvents: 'none'}}>
        {layout.nodes.sources.map((node, index) => (
          <div key={`source-label-${node.id}`} style={{position: 'absolute', top: node.y, left: safe.left, width: px(220), transform: `translateY(${node.height / 2 - px(42)}px)`, textAlign: 'right', ...labelIn(frame, fps, index)}}>
            <div style={{fontSize: px(38), fontWeight: 700, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis'}}>{node.id}</div>
            <div style={{fontFamily: NUM_FONT, fontSize: px(28), color: palette.muted, fontVariantNumeric: 'tabular-nums'}}>{formatValue(node.value)}{unit ? ` ${unit}` : ''}</div>
          </div>
        ))}
        {layout.nodes.targets.map((node, index) => {
          const color = targetColors.get(node.id) || palette.accent;
          const percent = layout.total > 0 ? (node.value / layout.total) * 100 : 0;
          return (
            <div key={`target-label-${node.id}`} style={{position: 'absolute', top: node.y, left: rightX + nodeW + px(42), width: width - safe.right - rightX - nodeW - px(42), transform: `translateY(${Math.max(0, node.height / 2 - px(36))}px)`, ...labelIn(frame, fps, index + layout.nodes.sources.length)}}>
              <div style={{display: 'flex', alignItems: 'baseline', gap: px(12)}}>
                <span style={{fontSize: px(40), fontWeight: 700, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', flex: 1}}>{node.id}</span>
                <span style={{fontFamily: NUM_FONT, fontSize: px(32), fontWeight: 800, color, fontVariantNumeric: 'tabular-nums'}}>{percent.toFixed(percent >= 10 ? 0 : 1)}%</span>
              </div>
              <div style={{fontFamily: NUM_FONT, fontSize: px(26), color: palette.muted, fontVariantNumeric: 'tabular-nums'}}>{formatValue(node.value)}{unit ? ` ${unit}` : ''}</div>
            </div>
          );
        })}
      </div>

      {source ? (
        <div style={{position: 'absolute', left: safe.left, right: safe.right, bottom: safe.bottom, textAlign: 'center', fontSize: px(28), color: palette.watermark}}>
          {source}
        </div>
      ) : null}
    </AbsoluteFill>
  );
};

function NodeBar({node, x, width, color, frame, durationInFrames, index}) {
  const p = easeOutCubic(progressAt(frame, Math.round(durationInFrames * 0.08) + index * 3, Math.round(durationInFrames * 0.28)));
  const h = Math.max(0, node.height * p);
  return (
    <rect
      x={x}
      y={node.y + (node.height - h) / 2}
      width={width}
      height={h}
      rx={width / 2}
      fill={color}
      opacity={0.9}
    />
  );
}

function linkProgress(frame, durationInFrames, index) {
  return easeOutCubic(progressAt(frame, Math.round(durationInFrames * 0.1) + index * 4, Math.round(durationInFrames * 0.42)));
}

function labelIn(frame, fps, index) {
  return entrance({frame, fps, delay: stagger(index, {base: Math.round(fps * 0.3), step: 4}), preset: 'fade-up', distance: 20});
}

function ribbonPath(x1, x2, y1, y2, thickness) {
  const mid = (x1 + x2) / 2;
  return [
    `M ${x1} ${y1}`,
    `C ${mid} ${y1}, ${mid} ${y2}, ${x2} ${y2}`,
    `L ${x2} ${y2 + thickness}`,
    `C ${mid} ${y2 + thickness}, ${mid} ${y1 + thickness}, ${x1} ${y1 + thickness}`,
    'Z',
  ].join(' ');
}

