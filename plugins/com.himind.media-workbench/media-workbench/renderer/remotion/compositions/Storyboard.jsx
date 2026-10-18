import React, {useMemo} from 'react';
import {AbsoluteFill, Sequence, interpolate, useCurrentFrame, useVideoConfig} from 'remotion';
import {SHOT_REGISTRY} from '../shots/registry.js';
import {FONT_FAMILY, NUM_FONT, canvasSize, makePx, resolvePalette} from '../foundation/theme.js';

/**
 * 成片模板：把若干镜头串成一条片子。
 *
 * 外壳（创作台）不认识任何镜头；镜头在这里按 id 查表。
 * 新增镜头只改 shots/registry.js，不动这个文件。
 *
 * 数据契约见 catalog/contracts/storyboard.schema.json。
 */

const DEFAULT_SHOT_SECONDS = 4;

function planShots(storyboard, fps) {
  let cursor = 0;
  return (storyboard || []).map((shot, index) => {
    const seconds = Number(shot && shot.duration_seconds) || DEFAULT_SHOT_SECONDS;
    const len = Math.max(1, Math.round(seconds * fps));
    const start = cursor;
    cursor += len;
    return {shot, index, start, len};
  });
}

const Notice = ({palette, px, title, detail}) => (
  <AbsoluteFill style={{backgroundColor: palette.bg0, color: palette.text, fontFamily: FONT_FAMILY, alignItems: 'center', justifyContent: 'center', gap: px(18), padding: px(120)}}>
    <div style={{fontSize: px(56), fontWeight: 700, textAlign: 'center'}}>{title}</div>
    {detail ? <div style={{fontSize: px(36), color: palette.muted, textAlign: 'center', lineHeight: 1.5}}>{detail}</div> : null}
  </AbsoluteFill>
);

export const Storyboard = (props) => {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const {width} = canvasSize(props);
  const px = makePx({width});
  const palette = resolvePalette(props);

  const data = props.data || {};
  const storyboard = Array.isArray(data.storyboard) ? data.storyboard : [];

  // 交叉淡入淡出：相邻镜头在 transitionFrames 内重叠。
  const transitionFrames = Math.max(0, Math.min(
    Math.round(fps * 0.6),
    Number(data.transition_frames) || Math.round(fps * 0.3),
  ));

  const plan = useMemo(() => planShots(storyboard, fps), [storyboard, fps]);

  if (!plan.length) {
    return <Notice palette={palette} px={px} title="这条片子还没有镜头" detail={`在 recipe.data.storyboard 里放入镜头，例如 [{"shot":"big-number","duration_seconds":3}]`} />;
  }

  const unknown = plan.filter((item) => !SHOT_REGISTRY[item.shot?.shot]);
  if (unknown.length === plan.length) {
    return <Notice palette={palette} px={px} title="镜头类型都未注册" detail={`未知镜头：${unknown.map((i) => i.shot?.shot).join('、')}；已注册：${Object.keys(SHOT_REGISTRY).join('、')}`} />;
  }

  return (
    <AbsoluteFill style={{backgroundColor: '#000'}}>
      {plan.map(({shot, index, start, len}) => {
        const Component = SHOT_REGISTRY[shot.shot];
        const isFirst = index === 0;
        const isLast = index === plan.length - 1;
        const from = Math.max(0, start - (isFirst ? 0 : transitionFrames));
        const duration = len + (isFirst ? 0 : transitionFrames) + (isLast ? 0 : transitionFrames);

        const opacity = (() => {
          let value = 1;
          if (!isFirst && transitionFrames > 0) {
            value = Math.min(value, interpolate(frame, [start - transitionFrames, start], [0, 1], {
              extrapolateLeft: 'clamp',
              extrapolateRight: 'clamp',
            }));
          }
          if (!isLast && transitionFrames > 0) {
            value = Math.min(value, interpolate(frame, [start + len - transitionFrames, start + len], [1, 0], {
              extrapolateLeft: 'clamp',
              extrapolateRight: 'clamp',
            }));
          }
          return value;
        })();

        if (!Component) {
          if (frame < from || frame >= from + duration) return null;
          return (
            <Sequence key={index} from={from} durationInFrames={duration} layout="none">
              <Notice
                palette={palette}
                px={px}
                title={`未知镜头：${shot.shot}`}
                detail={`已注册：${Object.keys(SHOT_REGISTRY).join('、')}`}
              />
            </Sequence>
          );
        }
        if (opacity <= 0.001) return null;

        return (
          <Sequence key={index} from={from} durationInFrames={duration} layout="none">
            <AbsoluteFill style={{opacity}}>
              <Component
                {...props}
                data={shot.data ?? {}}
                durationFrames={len}
                title={shot.title ?? ''}
                subtitle={shot.subtitle ?? ''}
              />
            </AbsoluteFill>
          </Sequence>
        );
      })}

      {/* 镜头序号（右下角，极淡，用于排错；不需要时可删） */}
      <AbsoluteFill style={{pointerEvents: 'none'}}>
        <div style={{position: 'absolute', right: px(90), bottom: px(90), fontFamily: NUM_FONT, fontSize: px(28), color: 'rgba(255,255,255,0.18)'}}>
          {plan.findIndex((item) => frame >= item.start && frame < item.start + item.len) + 1} / {plan.length}
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
