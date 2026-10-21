import React, {useEffect, useMemo, useRef} from 'react';
import {createRoot} from 'react-dom/client';
import {Player} from '@remotion/player';
import {COMPOSITIONS} from './index.jsx';

const clamp = (value, min, max) => Math.max(min, Math.min(max, Number(value) || 0));

function resolveSpec() {
  if (typeof window !== 'undefined' && window.__PREVIEW_SPEC__) {
    return window.__PREVIEW_SPEC__;
  }
  try {
    const raw = new URLSearchParams(window.location.search).get('p');
    return raw ? JSON.parse(decodeURIComponent(raw)) : null;
  } catch (err) {
    return null;
  }
}

function PlayerApp() {
  const spec = useMemo(resolveSpec, []);
  const playerRef = useRef(null);
  useEffect(() => {
    window.__HIMIND_PLAYER__ = {
      pause: () => playerRef.current?.pause(),
      seekTo: (frame) => playerRef.current?.seekTo(frame),
      getCurrentFrame: () => playerRef.current?.getCurrentFrame?.() ?? 0,
    };
    return () => { delete window.__HIMIND_PLAYER__; };
  }, []);
  const canvas = (spec && spec.canvas) || {};
  const fps = clamp(canvas.fps || 30, 1, 60);
  const width = clamp(canvas.width || 1080, 240, 4096);
  const height = clamp(canvas.height || 1920, 240, 4096);
  const duration = Math.max(1, Math.round(clamp((spec && spec.duration_seconds) || 12, 1, 300) * fps));
  // 查表而不是硬编码三元表达式：未知 composition 必须**可见地失败**，
  // 而不是静默渲染成另一个模板（参考实现的坑）。
  const requested = (spec && spec.composition) || 'LeaderboardTech';
  const component = COMPOSITIONS[requested];
  if (!component) {
    return (
      <div style={{width: '100%', height: '100%', background: '#0a0f1f', color: '#f5f1e8', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 16, fontFamily: 'Microsoft YaHei, sans-serif', padding: 32, textAlign: 'center'}}>
        <div style={{fontSize: 22, fontWeight: 700}}>未知的 composition：{requested}</div>
        <div style={{fontSize: 15, opacity: 0.75, lineHeight: 1.6}}>
          渲染器里没有注册这个组件。请检查模板的 composition 字段是否与
          renderer/remotion/index.jsx 中的注册名一致。
        </div>
        <div style={{fontSize: 13, opacity: 0.55}}>已注册：{Object.keys(COMPOSITIONS).join('、')}</div>
      </div>
    );
  }
  return (
    <div style={{width: '100%', height: '100%', background: '#0a0f1f', display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden'}}>
      <Player
        ref={playerRef}
        component={component}
        inputProps={spec || {}}
        durationInFrames={duration}
        compositionWidth={width}
        compositionHeight={height}
        fps={fps}
        style={{width: '100%', height: '100%'}}
        controls
        loop
        autoPlay
        doubleClickToFullscreen
        showVolumeControls
        showPlaybackRateControl
        acknowledgeRemotionLicense
        theme={{accent: '#0b7d85'}}
      />
    </div>
  );
}

createRoot(document.getElementById('root')).render(<PlayerApp />);
