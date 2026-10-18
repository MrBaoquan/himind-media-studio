import React, {useMemo} from 'react';
import {AbsoluteFill, interpolate, spring, useCurrentFrame, useVideoConfig} from 'remotion';
import {createProjection, pointAtLength, polylineLength, tangentAngle, toPathD} from '../techniques/geo.js';
import {buildTimeline, deriveMetrics, lineProgress, yearMilestones} from '../techniques/schedule.js';
import {labelQuality, solveLabels} from '../techniques/labels.js';
import {FONT_FAMILY, NUM_FONT, resolvePalette} from '../foundation/theme.js';

// compositions/RouteGrowth.jsx — 「路网生长式」地图
//
// 这一条镜头把四件技法组装起来：
//   画法  draw.map-base / draw.route-path / draw.station-marker
//   求解  solve.transit-timeline（调度）/ solve.label-avoid（标签避让）
// 数据来自 recipe.data.lines —— 几何由 media.workbench.route.repair 保证质量。

export const RouteGrowth = (props) => {
  const frame = useCurrentFrame();
  const {fps, durationInFrames: compositionDuration} = useVideoConfig();
  // 作为独立成片时用整片时长；被 Storyboard 包进 <Sequence> 时必须用显式传入的镜头时长
  // （useVideoConfig().durationInFrames 在 Sequence 内仍返回整片时长）。
  const durationInFrames = Number(props.durationFrames) || compositionDuration;

  const canvas = props.canvas || {};
  const width = Number(canvas.width) || 2160;
  const height = Number(canvas.height) || 3840;
  const scale = width / 2160;
  const px = (value) => Math.max(1, Math.round(value * scale));

  const palette = resolvePalette(props);
  const safe = props.safe_area || {};
  const safeTop = Number(safe.top) || px(170);
  const safeLeft = Number(safe.left) || px(110);
  const safeRight = Number(safe.right) || px(200);
  const safeBottom = Number(safe.bottom) || px(220);

  const data = props.data || {};
  const lines = useMemo(
    () => (Array.isArray(data.lines) ? data.lines : []).filter((l) => l && Array.isArray(l.coords) && l.coords.length >= 2),
    [data.lines],
  );

  const contentW = width - safeLeft - safeRight;

  // ── 布局分区 ──
  const headerTop = safeTop;
  const mapTop = safeTop + px(560);
  const mapH = Math.min(px(1900), height - mapTop - safeBottom - px(520));
  const mapW = Math.min(contentW, px(1760));
  const mapLeft = safeLeft + (contentW - mapW) / 2;
  const footerTop = height - safeBottom - px(120);

  // ── 技法：投影 ──
  const projection = useMemo(
    () => createProjection(lines.map((l) => l.coords), {width: mapW, height: mapH, padding: px(60)}),
    [lines, mapW, mapH],
  );

  const paths = useMemo(
    () =>
      lines.map((line) => {
        const xy = line.coords.map(([lon, lat]) => projection.toXY(lon, lat));
        return {
          id: line.id,
          color: line.color || palette.accent,
          d: toPathD(line.coords, projection.toXY),
          xy,
          length: polylineLength(xy),
          stations: (line.stations || []).map((s) => {
            const [x, y] = projection.toXY(s.lonLat[0], s.lonLat[1]);
            return {name: s.name, x, y, isMain: !!s.isMain, preferred: s.labelPos || undefined};
          }),
        };
      }),
    [lines, projection, palette.accent],
  );

  // ── 技法：调度 ──
  const timeline = useMemo(
    () =>
      buildTimeline(lines, {
        startFrame: px(40),
        endFrame: Math.max(px(60), durationInFrames - px(90)),
        groupGapFrames: Math.max(3, Math.round(durationInFrames * (8 / 540))),
      }),
    [lines, durationInFrames, px],
  );

  const metrics = useMemo(
    () => deriveMetrics(timeline, frame, lines, {startFrame: px(40)}),
    [timeline, frame, lines, px],
  );

  const milestones = useMemo(() => yearMilestones(timeline), [timeline]);

  // ── 技法：标签避让（只在插值到稳定帧上求解，保证帧确定性） ──
  const labelPlacements = useMemo(() => {
    const items = [];
    paths.forEach((path, lineIndex) => {
      const entry = timeline.schedule.find((s) => s.lineId === path.id);
      if (!entry) return;
      const progress = lineProgress(entry, frame);
      if (progress <= 0) return;
      path.stations.forEach((station, stationIndex) => {
        // 站点在路径上按比例依次出现
        const appearAt = (stationIndex + 1) / (path.stations.length + 1);
        if (progress < appearAt) return;
        const labelW = Math.max(px(150), station.name.length * px(34) + px(28));
        const labelH = px(48);
        items.push({
          id: `${path.id}-${stationIndex}-${station.name}`,
          x: station.x,
          y: station.y,
          w: labelW,
          h: labelH,
          priority: station.isMain ? 10 : 1,
          preferred: station.preferred,
          color: path.color,
          lineIndex,
        });
      });
    });
    const solved = solveLabels(items, {
      bounds: {w: mapW, h: mapH},
      offsets: [px(12), px(18), px(26), px(36), px(48), px(64), px(84), px(110)],
    });
    // 质量指标随求解结果一起带出：重叠/越界必须为 0，否则黄金样例会失败。
    const quality = {...labelQuality(solved.placements, {w: mapW, h: mapH}), dropped: solved.dropped.length};
    return {placements: solved.placements, quality};
  }, [paths, timeline, frame, mapW, mapH, px]);

  // ── 入场动画 ──
  const intro = spring({frame, fps, config: {damping: 18, stiffness: 90, mass: 0.7}});
  const titleY = (1 - intro) * px(40);

  const totalMileage = Number(data.totalMileage) || lines.reduce((acc, l) => acc + (Number(l.lengthKm) || 0), 0);
  const firstYear = milestones.length ? milestones[0].year : '';
  const lastYear = milestones.length ? milestones[milestones.length - 1].year : '';
  const barW = contentW;
  const barH = px(14);

  const barProgress = milestones.length > 1
    ? interpolate(frame, [milestones[0].frame, timeline.endFrame], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'})
    : 0;

  return (
    <AbsoluteFill style={{backgroundColor: palette.bg0, color: palette.text, fontFamily: FONT_FAMILY, overflow: 'hidden'}}>
      <div style={{position: 'absolute', inset: 0, background: `linear-gradient(180deg, ${palette.bg0} 0%, ${palette.bg1} 42%, ${palette.bg2} 74%, ${palette.bg0} 100%)`}} />
      <div style={{position: 'absolute', top: -px(240), left: '50%', transform: 'translateX(-50%)', width: px(1500), height: px(900), background: `radial-gradient(ellipse, ${palette.accent}22, transparent 70%)`, filter: `blur(${px(60)}px)`}} />
      <div style={{position: 'absolute', inset: 0, backgroundImage: `radial-gradient(circle at 1px 1px, ${palette.line} 1px, transparent 0)`, backgroundSize: `${px(54)}px ${px(54)}px`, opacity: 0.7}} />

      {/* ── 标题区 ── */}
      <div style={{position: 'absolute', top: headerTop, left: safeLeft, width: contentW, textAlign: 'center', transform: `translateY(${titleY}px)`, opacity: intro}}>
        {props.period ? (
          <div style={{display: 'inline-block', background: `linear-gradient(135deg, ${palette.accent}, ${palette.accent2})`, color: '#fff', fontSize: px(40), fontWeight: 700, padding: `${px(14)}px ${px(46)}px`, borderRadius: px(28), letterSpacing: '0.12em'}}>
            {props.period}
          </div>
        ) : null}
        <div style={{fontSize: px(112), fontWeight: 800, lineHeight: 1.16, marginTop: px(30), textShadow: `0 ${px(6)}px ${px(28)}px rgba(0,0,0,0.55)`}}>
          {String(props.title || `${data.region?.name || ''}路网生长`)
            .split(/\r?\n/)
            .map((line) => line.trim())
            .filter(Boolean)
            .map((line, index) => (
              <div key={index}>{line}</div>
            ))}
        </div>
        {props.subtitle ? (
          <div style={{fontSize: px(44), color: palette.muted, marginTop: px(18), letterSpacing: '0.04em'}}>{props.subtitle}</div>
        ) : null}
      </div>

      {/* ── 里程计数器 ── */}
      <div style={{position: 'absolute', top: headerTop + px(360), left: safeLeft, width: contentW, display: 'flex', justifyContent: 'center', alignItems: 'flex-end', gap: px(28), opacity: interpolate(frame, [px(20), px(50)], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'})}}>
        <span style={{fontSize: px(128), fontWeight: 800, color: palette.accent, fontFamily: NUM_FONT, lineHeight: 1, fontVariantNumeric: 'tabular-nums'}}>
          {metrics.mileage.toLocaleString()}
        </span>
        <span style={{fontSize: px(48), color: palette.muted, paddingBottom: px(14)}}>公里通车</span>
        <span style={{fontSize: px(48), color: palette.muted, paddingBottom: px(14), marginLeft: px(18)}}>· {metrics.completedCount} 条线路</span>
      </div>

      {/* ── 年份进度条 ── */}
      <div style={{position: 'absolute', top: headerTop + px(520), left: safeLeft, width: contentW, opacity: interpolate(frame, [px(30), px(60)], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'})}}>
        <div style={{position: 'relative', height: barH, background: palette.line, borderRadius: barH}}>
          <div style={{position: 'absolute', left: 0, top: 0, height: barH, width: barW * barProgress, background: `linear-gradient(90deg, ${palette.accent}, ${palette.accent2})`, borderRadius: barH}} />
          {milestones.map((m) => {
            const left = milestones.length > 1 ? ((m.frame - milestones[0].frame) / Math.max(1, timeline.endFrame - milestones[0].frame)) * barW : 0;
            const active = metrics.currentYear !== null && metrics.currentYear >= m.year;
            return <div key={m.year} style={{position: 'absolute', left, top: -px(6), width: px(4), height: barH + px(12), background: active ? palette.accent : palette.line, borderRadius: px(2)}} />;
          })}
        </div>
        <div style={{display: 'flex', justifyContent: 'space-between', marginTop: px(20), fontSize: px(40), color: palette.muted, fontFamily: NUM_FONT}}>
          <span>{firstYear}</span>
          <span style={{color: palette.accent, fontWeight: 700, fontSize: px(56)}}>{metrics.currentYear ?? firstYear}</span>
          <span>{lastYear}</span>
        </div>
      </div>

      {/* ── 地图区 ── */}
      <div style={{position: 'absolute', top: mapTop, left: mapLeft, width: mapW, height: mapH}}>
        <div style={{position: 'absolute', inset: 0, border: `${px(2)}px solid ${palette.line}`, borderRadius: px(28), background: 'rgba(0,0,0,0.18)'}} />
        <svg width={mapW} height={mapH} viewBox={`0 0 ${mapW} ${mapH}`} style={{position: 'absolute', inset: 0}}>
          <defs>
            {paths.map((p) => (
              <filter key={p.id} id={`glow-${p.id}`} x="-30%" y="-30%" width="160%" height="160%">
                <feGaussianBlur stdDeviation={px(6)} result="b" />
                <feMerge>
                  <feMergeNode in="b" />
                  <feMergeNode in="SourceGraphic" />
                </feMerge>
              </filter>
            ))}
          </defs>

          {paths.map((path) => {
            const entry = timeline.schedule.find((s) => s.lineId === path.id);
            if (!entry) return null;
            const progress = lineProgress(entry, frame);
            if (progress <= 0) return null;
            const dash = path.length;
            return (
              <g key={path.id}>
                <path
                  d={path.d}
                  fill="none"
                  stroke={path.color}
                  strokeWidth={px(14)}
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  opacity={0.28}
                  filter={`url(#glow-${path.id})`}
                  strokeDasharray={dash}
                  strokeDashoffset={dash * (1 - progress)}
                />
                <path
                  d={path.d}
                  fill="none"
                  stroke={path.color}
                  strokeWidth={px(7)}
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeDasharray={dash}
                  strokeDashoffset={dash * (1 - progress)}
                />
                {progress < 1 ? (
                  <g transform={`translate(${pointAtLength(path.xy, progress)[0]}, ${pointAtLength(path.xy, progress)[1]}) rotate(${tangentAngle(path.xy, progress)})`}>
                    <circle r={px(15)} fill={palette.text} opacity={0.95} />
                    <circle r={px(26)} fill={path.color} opacity={0.35} />
                  </g>
                ) : null}
              </g>
            );
          })}

          {/* 站点 */}
          {paths.map((path) => {
            const entry = timeline.schedule.find((s) => s.lineId === path.id);
            if (!entry) return null;
            const progress = lineProgress(entry, frame);
            return path.stations.map((station, index) => {
              const appearAt = (index + 1) / (path.stations.length + 1);
              if (progress < appearAt) return null;
              const pop = spring({frame: frame - entry.startFrame - index * 2, fps, config: {damping: 14, stiffness: 140, mass: 0.5}});
              const r = station.isMain ? px(13) : px(8);
              return (
                <g key={`${path.id}-${index}`} transform={`translate(${station.x}, ${station.y}) scale(${Math.max(0, pop)})`}>
                  <circle r={r + px(5)} fill="#000" opacity={0.35} />
                  <circle r={r} fill={palette.text} stroke={path.color} strokeWidth={px(4)} />
                </g>
              );
            });
          })}
        </svg>

        {/* 站点标签（HTML 层：避免 headless SVG 文字渲染问题） */}
        <div style={{position: 'absolute', inset: 0, pointerEvents: 'none'}}>
          {labelPlacements.placements.map((placement) => {
            const item = paths.flatMap((p) => p.stations.map((s, i) => ({...s, id: `${p.id}-${i}-${s.name}`, color: p.color}))).find((s) => s.id === placement.id);
            if (!item) return null;
            return (
              <div key={placement.id}>
                {placement.useLeader ? (
                  <svg style={{position: 'absolute', inset: 0, width: mapW, height: mapH}}>
                    <line x1={placement.anchorX} y1={placement.anchorY} x2={placement.x + placement.w / 2} y2={placement.y + placement.h / 2} stroke={item.color} strokeWidth={px(2)} strokeDasharray={`${px(6)},${px(5)}`} opacity={0.6} />
                  </svg>
                ) : null}
                <div style={{position: 'absolute', left: placement.x, top: placement.y, width: placement.w, height: placement.h, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: px(34), fontWeight: 600, color: palette.text, textShadow: `0 ${px(2)}px ${px(10)}px rgba(0,0,0,0.95), 0 0 ${px(3)}px rgba(0,0,0,1)`, whiteSpace: 'nowrap'}}>
                  {item.name}
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* ── 页脚 ── */}
      <div style={{position: 'absolute', top: footerTop, left: safeLeft, width: contentW, display: 'flex', justifyContent: 'space-between', alignItems: 'center', opacity: interpolate(frame, [px(60), px(100)], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'})}}>
        <span style={{fontSize: px(32), color: palette.watermark}}>{props.source || data.source || ''}</span>
        <span style={{fontSize: px(28), color: palette.watermark, fontFamily: NUM_FONT}}>{data.attribution || '© OpenStreetMap contributors'}</span>
      </div>
    </AbsoluteFill>
  );
};
