import React, {useMemo} from 'react';
import {AbsoluteFill, useCurrentFrame, useVideoConfig} from 'remotion';
import {Background} from '../foundation/background.jsx';
import {FONT_FAMILY, NUM_FONT, canvasSize, makePx, resolvePalette, safeArea, withAlpha} from '../foundation/theme.js';
import {entrance} from '../foundation/motion.js';
import {createProjection} from '../techniques/geo.js';
import {labelQuality, solveLabels} from '../techniques/labels.js';
import {formatValue} from '../foundation/format.js';

/**
 * shot.choropleth —— 分级填色地图。
 *
 * 复用投影技法（techniques/geo.js）与标签避让（techniques/labels.js）。
 * 数据契约见 catalog/contracts/choropleth.schema.json。
 */

function featureRings(feature) {
  if (!feature || !feature.geometry) return [];
  const geometry = feature.geometry;
  const polygons = geometry.type === 'Polygon' ? [geometry.coordinates] : geometry.coordinates;
  const rings = [];
  for (const polygon of polygons) {
    for (const ring of polygon) {
      if (Array.isArray(ring) && ring.length >= 3) rings.push(ring);
    }
  }
  return rings;
}

export const Choropleth = (props) => {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const {width, height} = canvasSize(props);
  const px = makePx({width});
  const palette = resolvePalette(props);
  const safe = safeArea(props, px);
  const contentW = width - safe.left - safe.right;

  const data = props.data || {};
  const geo = data.geo || {};
  const features = Array.isArray(geo.features) ? geo.features : [];
  const values = data.values || {};
  const unit = data.unit ?? '';
  const title = props.title ?? data.title ?? '';
  const source = props.source ?? data.source ?? '';

  const mapTop = safe.top + px(500);
  const mapH = Math.min(px(2000), height - mapTop - safe.bottom - px(420));
  const mapW = Math.min(contentW, px(1800));
  const mapLeft = safe.left + (contentW - mapW) / 2;

  const numeric = useMemo(
    () => Object.values(values).map(Number).filter((v) => Number.isFinite(v) && v > 0),
    [values],
  );
  const minValue = numeric.length ? Math.min(...numeric) : 0;
  const maxValue = numeric.length ? Math.max(...numeric) : 1;

  const projection = useMemo(() => {
    const coordsList = features.flatMap((feature) => featureRings(feature));
    if (!coordsList.length) return null;
    return createProjection(coordsList, {width: mapW, height: mapH, padding: px(40)});
  }, [features, mapW, mapH, px]);

  const regions = useMemo(() => {
    if (!projection) return [];
    return features.map((feature, index) => {
      const name = String(feature.properties?.name ?? `区域${index + 1}`);
      const value = Number(values[name] ?? values[name.replace(/市$/, '')] ?? 0);
      const t = maxValue > minValue ? Math.max(0, Math.min(1, (value - minValue) / (maxValue - minValue))) : 0;
      const rings = featureRings(feature);
      const d = rings
        .map((ring) => {
          const points = ring.map(([lon, lat], i) => {
            const [x, y] = projection.toXY(lon, lat);
            return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`;
          });
          return points.join(' ') + ' Z';
        })
        .join(' ');
      const centroid = feature.properties?.centroid ?? feature.properties?.center;
      const [cx, cy] = centroid && projection ? projection.toXY(centroid[0], centroid[1]) : [0, 0];
      return {name, value, t, d, cx, cy, index};
    });
  }, [features, values, projection, minValue, maxValue]);

  const labels = useMemo(() => {
    if (!regions.length) return [];
    const items = regions
      .filter((region) => Number.isFinite(region.cx) && Number.isFinite(region.cy))
      .map((region) => ({
        id: region.name,
        x: region.cx,
        y: region.cy,
        w: Math.max(px(120), region.name.length * px(26) + px(16)),
        h: px(40),
        priority: region.t * 10,
      }));
    return solveLabels(items, {
      bounds: {w: mapW, h: mapH},
      offsets: [px(4), px(10), px(18), px(28)],
    }).placements;
  }, [regions, mapW, mapH, px]);

  function alphaFor(t) {
    return 0.12 + 0.72 * t;
  }

  return (
    <AbsoluteFill style={{backgroundColor: palette.bg0, color: palette.text, fontFamily: FONT_FAMILY, overflow: 'hidden'}}>
      <Background palette={palette} px={px} />

      <div style={{position: 'absolute', top: safe.top, left: safe.left, width: contentW, textAlign: 'center', paddingTop: px(20)}}>
        <div style={{fontSize: px(76), fontWeight: 800, lineHeight: 1.25}}>{title}</div>
        {data.subtitle ? <div style={{marginTop: px(12), fontSize: px(38), color: palette.muted}}>{data.subtitle}</div> : null}
      </div>

      <div style={{position: 'absolute', top: mapTop, left: mapLeft, width: mapW, height: mapH}}>
        <svg width={mapW} height={mapH} viewBox={`0 0 ${mapW} ${mapH}`}>
          {regions.map((region) => {
            const inStyle = entrance({frame, fps, delay: region.index * 2, preset: 'fade-up', distance: 0});
            return (
              <g key={region.name} style={{opacity: inStyle.opacity}}>
                <path
                  d={region.d}
                  fill={withAlpha(palette.accent, alphaFor(region.t))}
                  stroke={withAlpha(palette.accent, 0.55)}
                  strokeWidth={px(2)}
                  strokeLinejoin="round"
                />
              </g>
            );
          })}
        </svg>

        <div style={{position: 'absolute', inset: 0, pointerEvents: 'none'}}>
          {labels.map((placement) => {
            const region = regions.find((r) => r.name === placement.id);
            if (!region) return null;
            const style = entrance({frame, fps, delay: Math.round(fps * 0.5) + region.index * 2, preset: 'fade-up', distance: 0});
            return (
              <div
                key={placement.id}
                style={{
                  ...style,
                  position: 'absolute',
                  left: placement.x,
                  top: placement.y,
                  width: placement.w,
                  height: placement.h,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: px(28),
                  fontWeight: region.t > 0.6 ? 700 : 500,
                  color: region.t > 0.6 ? '#ffffff' : 'rgba(255,255,255,0.86)',
                  textShadow: `0 ${px(2)}px ${px(8)}px rgba(0,0,0,0.95)`,
                  whiteSpace: 'nowrap',
                }}
              >
                {region.name.replace(/市$/, '')}
              </div>
            );
          })}
        </div>
      </div>

      <div style={{position: 'absolute', left: mapLeft, right: width - mapLeft - mapW, bottom: safe.bottom + px(120)}}>
        <div style={{display: 'flex', alignItems: 'center', gap: px(16)}}>
          <span style={{fontFamily: NUM_FONT, fontSize: px(26), color: palette.muted, width: px(120), textAlign: 'right'}}>{formatValue(minValue)}</span>
          <div style={{flex: 1, height: px(16), borderRadius: px(999), background: `linear-gradient(90deg, ${withAlpha(palette.accent, 0.12)}, ${withAlpha(palette.accent, 0.84)})`}} />
          <span style={{fontFamily: NUM_FONT, fontSize: px(26), color: palette.muted, width: px(120)}}>{formatValue(maxValue)}</span>
          <span style={{fontSize: px(26), color: palette.muted}}>{unit}</span>
        </div>
      </div>

      {source ? (
        <div style={{position: 'absolute', left: safe.left, right: safe.right, bottom: safe.bottom, textAlign: 'center', fontSize: px(28), color: palette.watermark}}>
          {source}
        </div>
      ) : null}
    </AbsoluteFill>
  );
};
