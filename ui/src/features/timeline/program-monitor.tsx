import {useEffect, useMemo, useRef, useState} from 'react';
import {Pause, Play, SkipBack, SkipForward} from 'lucide-react';
import type {Artifact, Timeline, TimelineClip} from './types';
import {useArtifactBlob} from './hooks';

function timecode(frame: number, fps: number) {
  const safeFps = Math.max(1, fps || 30);
  const total = Math.max(0, Math.round(frame));
  const ff = total % safeFps;
  const seconds = Math.floor(total / safeFps);
  return `${String(Math.floor(seconds / 3600)).padStart(2, '0')}:${String(Math.floor(seconds / 60) % 60).padStart(2, '0')}:${String(seconds % 60).padStart(2, '0')}:${String(ff).padStart(2, '0')}`;
}

export function activeClipAt(timeline: Timeline | null, frame: number) {
  if (!timeline) return null;
  const tracks = timeline.tracks.filter((track) => track.kind === 'video' && !track.muted).sort((left, right) => right.index - left.index);
  for (const track of tracks) {
    const clip = track.clips.find((item) => frame >= item.start_frame && frame < item.start_frame + item.duration_frames);
    if (clip) return {clip, track};
  }
  return null;
}

export function ProgramMonitor({timeline, artifacts, workspace, playhead, onPlayhead, playing, onPlaying, height = 220}: {
  timeline: Timeline | null; artifacts: Artifact[]; workspace: string; playhead: number;
  onPlayhead: (frame: number) => void; playing: boolean; onPlaying: (value: boolean) => void; height?: number;
}) {
  const fps = Math.max(1, timeline?.timebase.num ?? 30);
  const duration = Math.max(1, timeline?.duration_frames ?? 1, ...(timeline?.tracks.flatMap((track) => track.clips.map((item) => item.start_frame + item.duration_frames)) ?? [0]));
  const safePlayhead = Math.min(Math.max(0, playhead), Math.max(0, duration - 1));
  const active = useMemo(() => activeClipAt(timeline, safePlayhead), [safePlayhead, timeline]);
  const clip: TimelineClip | undefined = active?.clip;
  const artifact = artifacts.find((item) => item.artifact_id === clip?.media_id);
  const {url, error} = useArtifactBlob(workspace, clip?.media_id, Boolean(clip?.media_id));
  const videoRef = useRef<HTMLVideoElement>(null);
  const rafRef = useRef(0);
  const lastRef = useRef(0);
  const sourceFrame = clip ? Math.max(0, (clip.source_in_frame ?? 0) + Math.round((safePlayhead - clip.start_frame) * (clip.speed || 1))) : 0;
  const aspect = timeline ? `${timeline.canvas.width} / ${timeline.canvas.height}` : '9 / 16';
  useEffect(() => {
    const video = videoRef.current;
    if (!video || !clip || artifact?.kind?.startsWith('image/')) return;
    const target = Math.max(0, sourceFrame / fps);
    if (Math.abs(video.currentTime - target) > 0.045) {
      try { video.currentTime = target; } catch {}
    }
  }, [artifact?.kind, clip, fps, sourceFrame]);
  const playheadRef = useRef(playhead);
  const onPlayheadRef = useRef(onPlayhead);
  const onPlayingRef = useRef(onPlaying);
  useEffect(() => { playheadRef.current = playhead; }, [playhead]);
  useEffect(() => { onPlayheadRef.current = onPlayhead; }, [onPlayhead]);
  useEffect(() => { onPlayingRef.current = onPlaying; }, [onPlaying]);
  useEffect(() => {
    if (!playing) { cancelAnimationFrame(rafRef.current); return; }
    lastRef.current = performance.now();
    const tick = (now: number) => {
      const elapsed = Math.max(0, now - lastRef.current) / 1000;
      lastRef.current = now;
      const next = Math.min(duration - 1, Math.max(0, playheadRef.current + elapsed * fps));
      playheadRef.current = next;
      onPlayheadRef.current(next);
      if (next >= duration - 1) { onPlayingRef.current(false); return; }
      rafRef.current = requestAnimationFrame(tick);
    };
    rafRef.current = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(rafRef.current);
  }, [duration, fps, playing]);
  return <div className="program-monitor" style={{minHeight: 0}}>
    <div className="program-monitor-head">
      <div><span className="program-monitor-kicker">PROGRAM</span><strong>节目监视器</strong></div>
      <span className="program-monitor-time">{timecode(safePlayhead, fps)} <em>f{Math.round(safePlayhead)}</em></span>
    </div>
    <div className="program-monitor-stage" style={{aspectRatio: aspect, maxHeight: height}}>
      {url && artifact?.kind?.startsWith('image/') ? <img src={url} alt="节目画面" style={{width: '100%', height: '100%', objectFit: 'cover'}} /> : null}
      {url && !artifact?.kind?.startsWith('image/') ? <video ref={videoRef} src={url} muted playsInline preload="auto" style={{width: '100%', height: '100%', objectFit: 'cover'}} /> : null}
      {!url ? <div className="program-monitor-empty">{error ? `画面不可用：${error}` : clip ? '正在载入画面…' : timeline ? '当前播放头没有视频片段' : '还没有时间轴'}</div> : null}
      {clip ? <div className="program-monitor-clip">{clip.id}<span>{clip.track_id}</span></div> : null}
      <div className="program-monitor-safe" aria-hidden="true" />
    </div>
    <div className="program-monitor-controls">
      <button type="button" className="icon-btn" title="上一帧" aria-label="上一帧" onClick={() => onPlayhead(Math.max(0, safePlayhead - 1))}><SkipBack size={14} /></button>
      <button type="button" className="program-monitor-play" title={playing ? '暂停' : '播放'} aria-label={playing ? '暂停' : '播放'} onClick={() => onPlaying(!playing)}>{playing ? <Pause size={15} /> : <Play size={15} />}</button>
      <button type="button" className="icon-btn" title="下一帧" aria-label="下一帧" onClick={() => onPlayhead(Math.min(duration - 1, safePlayhead + 1))}><SkipForward size={14} /></button>
      <input className="program-monitor-scrub" type="range" min={0} max={duration - 1} step={0.1} value={safePlayhead} onChange={(event) => onPlayhead(Number(event.target.value))} aria-label="播放头位置" />
      <span className="program-monitor-duration">{timecode(duration, fps)}</span>
    </div>
  </div>;
}
