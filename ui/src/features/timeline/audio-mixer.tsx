import {useCallback, useEffect, useRef, useState} from 'react';
import type {Timeline, TimelineClip, TimelineTrack} from './types';
import {useArtifactBlob} from './hooks';

function dbToGain(db: number) {
  return Math.pow(10, Math.max(-60, Math.min(12, db || 0)) / 20);
}

type RegisterAnalyser = (clipId: string, trackId: string, analyser: AnalyserNode | null) => void;

function AudioClipPlayer({workspace, clip, track, playhead, playing, fps, active, audible, ensureContext, registerAnalyser}: {
  workspace: string; clip: TimelineClip; track: TimelineTrack; playhead: number; playing: boolean;
  fps: number; active: boolean; audible: boolean;
  ensureContext: () => AudioContext; registerAnalyser: RegisterAnalyser;
}) {
  const {url} = useArtifactBlob(workspace, clip.media_id, audible);
  const ref = useRef<HTMLAudioElement>(null);
  const sourceRef = useRef<MediaElementAudioSourceNode | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const inClip = playhead >= clip.start_frame && playhead < clip.start_frame + clip.duration_frames;
  const volume = Math.max(0, Math.min(1, dbToGain((track.volume_db ?? 0) + (clip.volume_db ?? 0))));

  useEffect(() => {
    const audio = ref.current;
    if (!audio || !url || sourceRef.current) return;
    try {
      const context = ensureContext();
      const source = context.createMediaElementSource(audio);
      const analyser = context.createAnalyser();
      analyser.fftSize = 256;
      source.connect(analyser);
      analyser.connect(context.destination);
      sourceRef.current = source;
      analyserRef.current = analyser;
      registerAnalyser(clip.id, track.id, analyser);
    } catch {}
  }, [url, clip.id, track.id, ensureContext, registerAnalyser]);

  useEffect(() => () => {
    registerAnalyser(clip.id, track.id, null);
    sourceRef.current?.disconnect();
    analyserRef.current?.disconnect();
    sourceRef.current = null;
    analyserRef.current = null;
  }, [clip.id, track.id, registerAnalyser]);

  useEffect(() => {
    const audio = ref.current;
    if (!audio || !url) return;
    audio.volume = volume;
    if (!active || !audible || !inClip) {
      if (!audio.paused) audio.pause();
      return;
    }
    const target = (clip.source_in_frame ?? 0) / fps + ((playhead - clip.start_frame) / fps) * (clip.speed ?? 1);
    if (Math.abs(audio.currentTime - target) > 0.15) {
      try { audio.currentTime = Math.max(0, target); } catch {}
    }
    if (playing && audio.paused) {
      void ensureContext().resume().catch(() => {});
      void audio.play().catch(() => {});
    }
    if (!playing && !audio.paused) audio.pause();
  }, [url, playhead, playing, active, audible, inClip, volume, fps, clip.source_in_frame, clip.start_frame, clip.speed, ensureContext]);

  return <audio ref={ref} src={url} preload="auto" data-track-id={track.id} data-clip-id={clip.id} data-audible={audible ? 'true' : 'false'} style={{display: 'none'}} />;
}

export function TimelineAudioMixer({timeline, workspace, playhead, playing, active = true, onLevels}: {
  timeline: Timeline | null; workspace: string; playhead: number; playing: boolean; active?: boolean;
  onLevels?: (levels: Record<string, number>) => void;
}) {
  const [pageVisible, setPageVisible] = useState(true);
  const contextRef = useRef<AudioContext | null>(null);
  const analysersRef = useRef<Map<string, {trackId: string; analyser: AnalyserNode}>>(new Map());
  const ensureContext = useCallback(() => {
    if (!contextRef.current) {
      const Ctor = window.AudioContext ?? (window as any).webkitAudioContext;
      contextRef.current = new Ctor();
    }
    return contextRef.current;
  }, []);
  const registerAnalyser = useCallback<RegisterAnalyser>((clipId, trackId, analyser) => {
    if (analyser) analysersRef.current.set(clipId, {trackId, analyser});
    else analysersRef.current.delete(clipId);
  }, []);

  useEffect(() => {
    const update = () => setPageVisible(document.visibilityState === 'visible');
    update();
    document.addEventListener('visibilitychange', update);
    return () => document.removeEventListener('visibilitychange', update);
  }, []);

  useEffect(() => {
    const resume = () => { void contextRef.current?.resume().catch(() => {}); };
    document.addEventListener('pointerdown', resume, {once: true});
    return () => document.removeEventListener('pointerdown', resume);
  }, []);

  useEffect(() => {
    if (!playing) { onLevels?.({}); return; }
    let frame = 0;
    const tick = () => {
      const levels: Record<string, number> = {};
      for (const {trackId, analyser} of analysersRef.current.values()) {
        const data = new Uint8Array(analyser.fftSize);
        analyser.getByteTimeDomainData(data);
        let sum = 0;
        for (const value of data) { const sample = (value - 128) / 128; sum += sample * sample; }
        const level = Math.min(1, Math.sqrt(sum / data.length) * 3.2);
        levels[trackId] = Math.max(levels[trackId] || 0, level);
      }
      onLevels?.(levels);
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [playing, onLevels]);

  const audioTracks = timeline?.tracks.filter((track) => track.kind === 'audio') ?? [];
  const anySolo = audioTracks.some((track) => track.solo);
  const fps = Math.max(1, timeline?.timebase.num ?? 30);
  const running = active && pageVisible;
  return <div className="timeline-audio-mixer" aria-hidden="true">
    {audioTracks.map((track) => {
      const audible = anySolo ? Boolean(track.solo) : !track.muted;
      return track.clips.map((clip) => <AudioClipPlayer key={`${track.id}-${clip.id}`} workspace={workspace} clip={clip} track={track} playhead={playhead} playing={playing} fps={fps} active={running} audible={audible} ensureContext={ensureContext} registerAnalyser={registerAnalyser} />);
    })}
  </div>;
}
