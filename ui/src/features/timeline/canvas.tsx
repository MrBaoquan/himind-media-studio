import {useEffect, useMemo, useRef, useState} from 'react';
import {AudioLines, ChevronDown, Eye, EyeOff, Film, GripVertical, Lock, Magnet, MousePointer2, Plus, Scissors, SlidersHorizontal, Sparkles, Trash2, Unlock, Volume2, VolumeX, ZoomIn, ZoomOut} from 'lucide-react';
import type {Artifact, Timeline, TimelineClip, TimelineDiff, TimelineTool, TimelineTrack} from './types';
import {useArtifactBlob, useAudioPeaks} from './hooks';

type DragState = {clipId: string; trackId: string; mode: 'move' | 'trim-left' | 'trim-right'; startX: number; originalStart: number; originalDuration: number; delta: number};
type Marquee = {x: number; y: number; width: number; height: number};
type ContextMenuState = {x: number; y: number; clip?: TimelineClip; track?: TimelineTrack; frame: number};
type CanvasProps = {
  timeline: Timeline;
  artifacts: Artifact[];
  workspace: string;
  playhead: number;
  onPlayhead: (frame: number) => void;
  selectedIds: string[];
  onSelect: (ids: string[], primary?: string) => void;
  tool: TimelineTool;
  onTool: (tool: TimelineTool) => void;
  snap: boolean;
  onSnap: (value: boolean) => void;
  onApply: (commands: Array<Record<string, unknown>>) => Promise<void> | void;
  onInsertAt?: (trackId: string, frame: number, artifactId?: string) => void;
  diff?: TimelineDiff | null;
  onToast?: (message: string) => void;
  audioLevels?: Record<string, number>;
  height?: number;
};

const GUTTER = 220;
const ROW_HEIGHT: Record<string, number> = {video: 54, audio: 56, subtitle: 46};

function clamp(value: number, min: number, max: number) { return Math.max(min, Math.min(max, value)); }
function roleLabel(role?: string) {
  if (!role) return '';
  const key = role.toLowerCase();
  if (key === 'voiceover' || key === 'narration' || key === 'voice') return 'VO';
  if (key === 'bgm' || key === 'music') return 'BGM';
  if (key === 'overlay') return 'OVR';
  if (key === 'program') return 'PGM';
  return role.toUpperCase();
}
function rulerLabel(frame: number, fps: number) {
  const total = Math.round(frame / Math.max(1, fps));
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`;
}
function clipEnd(clip: TimelineClip) { return clip.start_frame + clip.duration_frames; }
function timecode(frame: number, fps: number) {
  const safeFps = Math.max(1, fps || 30);
  const total = Math.max(0, Math.round(frame));
  const ff = total % safeFps;
  const seconds = Math.floor(total / safeFps);
  return `${String(Math.floor(seconds / 3600)).padStart(2, '0')}:${String(Math.floor(seconds / 60) % 60).padStart(2, '0')}:${String(seconds % 60).padStart(2, '0')}:${String(ff).padStart(2, '0')}`;
}

function diffState(diff: TimelineDiff | null | undefined, clipId: string) {
  if (!diff) return '';
  if (diff.added.some((item) => item.clip_id === clipId)) return 'added';
  if (diff.changed.some((item) => item.clip_id === clipId)) return 'changed';
  return '';
}

function chooseTick(pxPerFrame: number, fps: number) {
  const candidates = [1, 2, 5, 10, 15, 30, 60, 120, 300, 600, 900, 1800].map((seconds) => seconds * Math.max(1, fps));
  for (const frames of candidates) if (frames * pxPerFrame >= 68) return frames;
  return candidates[candidates.length - 1];
}

function Filmstrip({workspace, artifact, width, height}: {workspace: string; artifact?: Artifact; width: number; height: number}) {
  const {url} = useArtifactBlob(workspace, artifact?.artifact_id, Boolean(artifact));
  const ref = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    const canvas = ref.current;
    if (!canvas || !url) return;
    let cancelled = false;
    const dpr = Math.min(2, window.devicePixelRatio || 1);
    const cssWidth = Math.max(48, Math.round(width));
    const cssHeight = Math.max(24, Math.round(height));
    canvas.width = Math.round(cssWidth * dpr);
    canvas.height = Math.round(cssHeight * dpr);
    const context = canvas.getContext('2d');
    if (!context) return;
    context.scale(dpr, dpr);
    context.fillStyle = '#0a0f17';
    context.fillRect(0, 0, cssWidth, cssHeight);
    const drawCover = (source: CanvasImageSource, sourceWidth: number, sourceHeight: number, x: number, y: number, w: number, h: number) => {
      const scale = Math.max(w / Math.max(1, sourceWidth), h / Math.max(1, sourceHeight));
      const sw = w / scale, sh = h / scale;
      const sx = Math.max(0, (sourceWidth - sw) / 2), sy = Math.max(0, (sourceHeight - sh) / 2);
      context.drawImage(source, sx, sy, sw, sh, x, y, w, h);
    };
    const count = clamp(Math.floor(cssWidth / 46), 2, 7);
    if (artifact?.kind?.startsWith('image/')) {
      const image = new Image();
      image.onload = () => { if (!cancelled) for (let index = 0; index < count; index++) drawCover(image, image.naturalWidth, image.naturalHeight, index * (cssWidth / count), 0, cssWidth / count, cssHeight); };
      image.src = url;
      return () => { cancelled = true; };
    }
    const video = document.createElement('video');
    video.muted = true; video.preload = 'auto'; video.src = url;
    let index = 0;
    let seekHandler: (() => void) | null = null;
    const seekNext = () => {
      if (cancelled) return;
      if (index >= count) return;
      const at = ((index + 0.5) / count) * Math.max(0.1, Math.min(video.duration || 1, 12));
      try { video.currentTime = at; } catch { drawFallback(); }
    };
    const drawFallback = () => {
      context.fillStyle = 'rgba(143,180,255,.18)';
      for (let i = 0; i < 6; i++) context.fillRect(i * (cssWidth / 6) + 2, 4, cssWidth / 6 - 4, cssHeight - 8);
    };
    video.addEventListener('loadedmetadata', seekNext);
    video.addEventListener('loadeddata', () => { if (video.readyState >= 2) seekNext(); });
    seekHandler = () => {
      if (cancelled || index >= count) return;
      drawCover(video, video.videoWidth || 16, video.videoHeight || 9, index * (cssWidth / count), 0, cssWidth / count, cssHeight);
      context.fillStyle = 'rgba(0,0,0,.12)';
      context.fillRect(index * (cssWidth / count), 0, 1, cssHeight);
      index += 1;
      if (index < count) seekNext();
    };
    video.addEventListener('seeked', seekHandler);
    video.addEventListener('error', drawFallback);
    return () => { cancelled = true; video.removeAttribute('src'); video.load(); };
  }, [artifact?.kind, height, url, width]);
  return <canvas ref={ref} className="timeline-filmstrip" aria-hidden="true" style={{width: '100%', height: '100%'}} />;
}

function Waveform({workspace, artifact, color}: {workspace: string; artifact?: Artifact; color: string}) {
  const {url} = useArtifactBlob(workspace, artifact?.artifact_id, Boolean(artifact));
  const peaks = useAudioPeaks(url, 96);
  return <div className="timeline-waveform" aria-hidden="true" style={{display: 'flex', alignItems: 'center', gap: 1}}>
    {(peaks.length ? peaks : Array.from({length: 24}, (_, index) => 0.12 + (index % 5) * 0.05)).map((peak, index) => <span key={index} style={{flex: '1 1 0', height: `${clamp(peak * 100, 8, 100)}%`, background: color, opacity: .9, borderRadius: 1}} />)}
  </div>;
}

function ClipView({clip, track, timeline, artifacts, workspace, pxPerFrame, selected, primary, diff, dragPreview, onPointerDown, onSelect, onContextMenu, onBlade, onSeek, onTransition}: {
  clip: TimelineClip; track: TimelineTrack; timeline: Timeline; artifacts: Artifact[]; workspace: string; pxPerFrame: number;
  selected: boolean; primary: boolean; diff: TimelineDiff | null | undefined;
  dragPreview: {clipId: string; mode: DragState['mode']; delta: number} | null;
  onPointerDown: (event: React.PointerEvent, clip: TimelineClip, mode: DragState['mode']) => void;
  onSelect: (ids: string[], primary?: string) => void; onContextMenu: (event: React.MouseEvent, clip: TimelineClip) => void;
  onBlade: (frame: number, clip: TimelineClip) => void; onSeek: (frame: number) => void;
  onTransition: (edge: 'in' | 'out', frames: number) => void;
}) {
  const preview = dragPreview?.clipId === clip.id ? dragPreview : null;
  const start = preview?.mode === 'move' ? clip.start_frame + preview.delta : preview?.mode === 'trim-left' ? clip.start_frame + preview.delta : clip.start_frame;
  const duration = preview?.mode === 'trim-left' ? Math.max(1, clip.duration_frames - preview.delta) : preview?.mode === 'trim-right' ? Math.max(1, clip.duration_frames + preview.delta) : clip.duration_frames;
  const width = Math.max(12, duration * pxPerFrame);
  const left = GUTTER + start * pxPerFrame;
  const artifact = artifacts.find((item) => item.artifact_id === clip.media_id);
  const state = diffState(diff, clip.id);
  const isAudio = track.kind === 'audio' || artifact?.kind?.startsWith('audio/');
  const isSubtitle = track.kind === 'subtitle';
  const isPip = track.index > 1 && track.kind === 'video';
  const [transitionDrag, setTransitionDrag] = useState<{edge: 'in' | 'out'; startX: number; base: number; frames: number} | null>(null);
  const transitionIn = transitionDrag?.edge === 'in' ? transitionDrag.frames : (clip.transition_in?.duration_frames ?? 0);
  const transitionOut = transitionDrag?.edge === 'out' ? transitionDrag.frames : (clip.transition_out?.duration_frames ?? 0);
  const beginTransition = (edge: 'in' | 'out') => (event: React.PointerEvent<HTMLSpanElement>) => {
    event.stopPropagation(); event.preventDefault();
    try { event.currentTarget.setPointerCapture(event.pointerId); } catch {}
    const base = edge === 'in' ? transitionIn : transitionOut;
    setTransitionDrag({edge, startX: event.clientX, base, frames: base});
  };
  const moveTransition = (event: React.PointerEvent<HTMLSpanElement>) => {
    if (!transitionDrag) return;
    const frames = clamp(Math.round(transitionDrag.base + (event.clientX - transitionDrag.startX) / pxPerFrame), 0, Math.max(0, Math.min(60, clip.duration_frames - 1)));
    if (frames !== transitionDrag.frames) setTransitionDrag({...transitionDrag, frames});
  };
  const endTransition = () => {
    if (!transitionDrag) return;
    onTransition(transitionDrag.edge, transitionDrag.frames);
    setTransitionDrag(null);
  };
  return <div
    className="timeline-clip"
    data-testid={`timeline-clip-${clip.id}`}
    data-timeline-clip={clip.id}
    data-active={selected}
    data-primary={primary}
    data-diff={state}
    data-kind={track.kind}
    data-pip={isPip}
    role="button"
    tabIndex={0}
    aria-label={`片段 ${clip.id}，${timecode(clip.start_frame, timeline.timebase.num)}，${clip.duration_frames} 帧`}
    title={`${clip.id} · ${clip.start_frame}–${clipEnd(clip)}f · ${clip.media_id || 'text'}`}
    style={{left, width, background: selected ? 'var(--accent)' : isSubtitle ? 'color-mix(in srgb, var(--warn) 16%, var(--ink-850))' : isAudio ? 'color-mix(in srgb, var(--ok) 14%, var(--ink-850))' : 'var(--accent-soft)', borderColor: selected ? 'var(--accent)' : isSubtitle ? 'color-mix(in srgb, var(--warn) 45%, var(--line))' : 'var(--accent-line)', color: selected ? 'var(--accent-contrast)' : 'var(--fg)'}}
    onPointerDown={(event) => { event.stopPropagation(); onPointerDown(event, clip, 'move'); }}
    onContextMenu={(event) => { event.preventDefault(); event.stopPropagation(); onContextMenu(event, clip); }}
    onKeyDown={(event) => {
      if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); onSelect([clip.id], clip.id); }
      if (event.key === 'b') { event.preventDefault(); onBlade(clip.start_frame + Math.floor(clip.duration_frames / 2), clip); }
    }}
    onDoubleClick={() => onSeek(clip.start_frame)}
  >
    <div className="timeline-clip-media" aria-hidden="true">
      {isSubtitle ? null : isAudio ? <Waveform workspace={workspace} artifact={artifact} color="var(--ok)" /> : artifact ? <Filmstrip workspace={workspace} artifact={artifact} width={Math.max(48, width)} height={30} /> : <span className="timeline-clip-empty"><Film size={12} /> {clip.media_id || 'media'}</span>}
    </div>
    <div className="timeline-clip-label">
      <span className="timeline-clip-name">{isSubtitle ? (clip.text || clip.id) : clip.id}</span>
      <span className="timeline-clip-meta">{timecode(start, timeline.timebase.num)} · {duration}f{clip.speed && clip.speed !== 1 ? ` · ${clip.speed}×` : ''}</span>
    </div>
    {isPip ? <span className="timeline-clip-badge">PIP</span> : null}
    {clip.color?.keyframes?.length ? <span className="timeline-clip-badge timeline-clip-badge-kf">KF {clip.color.keyframes.length}</span> : null}
    {transitionIn > 0 ? <span className="timeline-transition-band timeline-transition-band-in" style={{width: Math.min(width, transitionIn * pxPerFrame)}} aria-hidden="true" /> : null}
    {transitionOut > 0 ? <span className="timeline-transition-band timeline-transition-band-out" style={{width: Math.min(width, transitionOut * pxPerFrame)}} aria-hidden="true" /> : null}
    <span className="timeline-transition-handle timeline-transition-in" data-testid={`timeline-transition-in-${clip.id}`} title={`转场入 ${transitionIn}f`} aria-label="转场入手柄" onPointerDown={beginTransition('in')} onPointerMove={moveTransition} onPointerUp={endTransition} onPointerCancel={endTransition} />
    <span className="timeline-transition-handle timeline-transition-out" data-testid={`timeline-transition-out-${clip.id}`} title={`转场出 ${transitionOut}f`} aria-label="转场出手柄" onPointerDown={beginTransition('out')} onPointerMove={moveTransition} onPointerUp={endTransition} onPointerCancel={endTransition} />
    <span className="timeline-trim-handle timeline-trim-left" aria-hidden="true" onPointerDown={(event) => { event.stopPropagation(); onPointerDown(event, clip, 'trim-left'); }} />
    <span className="timeline-trim-handle timeline-trim-right" aria-hidden="true" onPointerDown={(event) => { event.stopPropagation(); onPointerDown(event, clip, 'trim-right'); }} />
  </div>;
}

function KeyframeLane({clip, timeline, pxPerFrame, onSeek}: {clip: TimelineClip; timeline: Timeline; pxPerFrame: number; onSeek: (frame: number) => void}) {
  const keyframes = clip.color?.keyframes ?? [];
  if (!keyframes.length) return null;
  return <div className="timeline-keyframe-lane" style={{left: GUTTER + clip.start_frame * pxPerFrame, width: Math.max(16, clip.duration_frames * pxPerFrame)}}>
    <span className="timeline-keyframe-label">KF</span>
    {keyframes.map((keyframe, index) => <button key={index} type="button" className="timeline-keyframe-dot" style={{left: `${(keyframe.frame / Math.max(1, clip.duration_frames)) * 100}%`}} title={`关键帧 ${keyframe.frame}f · 亮度 ${keyframe.brightness ?? 0}`} aria-label={`关键帧 ${keyframe.frame}`} onClick={(event) => { event.stopPropagation(); onSeek(clip.start_frame + keyframe.frame); }} />)}
  </div>;
}

export function TimelineCanvas({timeline, artifacts, workspace, playhead, onPlayhead, selectedIds, onSelect, tool, onTool, snap, onSnap, onApply, onInsertAt, diff, onToast, audioLevels = {}}: CanvasProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const innerRef = useRef<HTMLDivElement>(null);
  const [viewport, setViewport] = useState({width: 900, scrollLeft: 0});
  const [zoom, setZoom] = useState(1);
  const [marquee, setMarquee] = useState<Marquee | null>(null);
  const [menu, setMenu] = useState<ContextMenuState | null>(null);
  const [trackMenuOpen, setTrackMenuOpen] = useState(false);
  const [trackHeightPreview, setTrackHeightPreview] = useState<{trackId: string; height: number} | null>(null);
  const [draggedTrackId, setDraggedTrackId] = useState('');
  const [dropTargetId, setDropTargetId] = useState('');
  const [editingTrackId, setEditingTrackId] = useState('');
  const [editingName, setEditingName] = useState('');
  const [volumeDraft, setVolumeDraft] = useState<Record<string, number>>({});
  const [dragPreview, setDragPreview] = useState<{clipId: string; mode: DragState['mode']; delta: number} | null>(null);
  const dragRef = useRef<DragState | null>(null);
  const marqueeRef = useRef<{startX: number; startY: number; additive: boolean} | null>(null);
  const fps = Math.max(1, timeline.timebase.num);
  const duration = Math.max(1, timeline.duration_frames, ...timeline.tracks.flatMap((track) => track.clips.map(clipEnd)));
  useEffect(() => {
    const element = scrollRef.current;
    if (!element) return;
    const measure = () => setViewport((current) => ({...current, width: element.clientWidth}));
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);
  const basePxPerFrame = Math.max(0.22, (viewport.width - GUTTER - 24) / duration);
  const pxPerFrame = basePxPerFrame * zoom;
  const windowStart = Math.max(0, Math.floor((viewport.scrollLeft - GUTTER) / pxPerFrame) - 40);
  const windowEnd = Math.min(duration, Math.ceil((viewport.scrollLeft + viewport.width) / pxPerFrame) + 40);
  const tickFrames = chooseTick(pxPerFrame, fps);
  const ticks = useMemo(() => {
    const result: number[] = [];
    for (let frame = Math.floor(windowStart / tickFrames) * tickFrames; frame <= windowEnd; frame += tickFrames) result.push(frame);
    return result;
  }, [tickFrames, windowEnd, windowStart]);
  const visibleClips = timeline.tracks.flatMap((track) => track.clips.filter((clip) => clipEnd(clip) >= windowStart && clip.start_frame <= windowEnd));
  const primaryId = selectedIds[0] || '';
  const selectedSet = useMemo(() => new Set(selectedIds), [selectedIds]);

  function nextTrackID(kind: 'video' | 'audio' | 'subtitle') {
    const prefix = kind === 'video' ? 'V' : kind === 'audio' ? 'A' : 'S';
    const used = new Set(timeline.tracks.map((track) => track.id));
    let index = 1;
    while (used.has(prefix + index)) index += 1;
    return prefix + index;
  }
  function addTrack(kind: 'video' | 'audio' | 'subtitle') {
    void onApply([{op: 'add_track', track_id: nextTrackID(kind), track_kind: kind}]);
  }
  function setTrackState(track: TimelineTrack, patch: {muted?: boolean; locked?: boolean; solo?: boolean; name?: string; color?: string; volume_db?: number}) {
    const command: Record<string, unknown> = {op: 'set_track_state', track_id: track.id};
    if (patch.muted !== undefined) command.muted = patch.muted;
    if (patch.locked !== undefined) command.locked = patch.locked;
    if (patch.solo !== undefined) command.solo = patch.solo;
    if (patch.name !== undefined) command.track_name = patch.name;
    if (patch.color !== undefined) command.track_color = patch.color;
    if (patch.volume_db !== undefined) command.track_volume_db = patch.volume_db;
    void onApply([command]);
  }
  function commitVolume(track: TimelineTrack) {
    const draft = volumeDraft[track.id];
    if (draft === undefined) return;
    setTrackState(track, {volume_db: draft});
    setVolumeDraft((current) => { const next = {...current}; delete next[track.id]; return next; });
  }
  function defaultTrackColor(kind: string) {
    return kind === 'audio' ? '#34c38f' : kind === 'subtitle' ? '#f5b942' : '#5aa9ff';
  }
  function commitTrackName(track: TimelineTrack) {
    const name = editingName.trim();
    setEditingTrackId('');
    if (!name || name === (track.name || track.id)) return;
    setTrackState(track, {name});
  }
  function dropTrack(target: TimelineTrack) {
    const sourceId = draggedTrackId;
    setDraggedTrackId('');
    setDropTargetId('');
    if (!sourceId || sourceId === target.id) return;
    const source = timeline.tracks.find((track) => track.id === sourceId);
    if (!source || source.kind !== target.kind) return;
    const sameKind = timeline.tracks.filter((track) => track.kind === target.kind);
    const targetIndex = sameKind.findIndex((track) => track.id === target.id);
    if (targetIndex >= 0) void onApply([{op: 'reorder_track', track_id: sourceId, target_index: targetIndex}]);
  }
  function removeTrack(track: TimelineTrack) {
    void onApply([{op: 'remove_track', track_id: track.id}]);
  }
  function beginTrackResize(event: React.PointerEvent, track: TimelineTrack, currentHeight: number) {
    event.preventDefault();
    event.stopPropagation();
    const startY = event.clientY;
    let nextHeight = currentHeight;
    const move = (moveEvent: PointerEvent) => {
      nextHeight = clamp(currentHeight + moveEvent.clientY - startY, 28, 160);
      setTrackHeightPreview({trackId: track.id, height: nextHeight});
    };
    const up = () => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
      setTrackHeightPreview(null);
      if (nextHeight !== currentHeight) void onApply([{op: 'set_track_height', track_id: track.id, height: nextHeight}]);
    };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
  }
  function applyDensity(density: 'compact' | 'standard' | 'high') {
    const sizes = density === 'compact' ? {video: 44, audio: 46, subtitle: 38} : density === 'high' ? {video: 72, audio: 72, subtitle: 58} : {video: 54, audio: 56, subtitle: 46};
    void onApply(timeline.tracks.map((track) => ({op: 'set_track_height', track_id: track.id, height: sizes[track.kind as keyof typeof sizes] ?? 48})));
  }
  function frameAtClientX(clientX: number) {
    const rect = innerRef.current?.getBoundingClientRect();
    if (!rect) return 0;
    return clamp(Math.round((clientX - rect.left - GUTTER) / pxPerFrame), 0, duration);
  }
  function snapFrame(frame: number, ignoreId?: string) {
    if (!snap) return frame;
    let best = frame;
    let bestDistance = 8;
    const candidates = [0, playhead, ...timeline.tracks.flatMap((track) => track.clips.filter((clip) => clip.id !== ignoreId).flatMap((clip) => [clip.start_frame, clipEnd(clip)]))];
    for (const candidate of candidates) {
      const distance = Math.abs(candidate - frame);
      if (distance < bestDistance) { best = candidate; bestDistance = distance; }
    }
    return best;
  }
  function beginClipPointer(event: React.PointerEvent, clip: TimelineClip, mode: DragState['mode']) {
    if (event.button !== 0) return;
    event.preventDefault();
    if (tool === 'blade') {
      const at = frameAtClientX(event.clientX);
      if (at > clip.start_frame && at < clipEnd(clip)) void onApply([{op: 'blade', track_id: clip.track_id, clip_id: clip.id, at_frame: at}]);
      return;
    }
    if (tool === 'trim' && mode === 'move') mode = 'trim-right';
    if (event.shiftKey || event.metaKey || event.ctrlKey) {
      const next = selectedSet.has(clip.id) ? selectedIds.filter((id) => id !== clip.id) : [...selectedIds, clip.id];
      onSelect(next, clip.id);
      return;
    }
    onSelect([clip.id], clip.id);
    const state: DragState = {clipId: clip.id, trackId: clip.track_id, mode, startX: event.clientX, originalStart: clip.start_frame, originalDuration: clip.duration_frames, delta: 0};
    dragRef.current = state;
    setDragPreview({clipId: clip.id, mode, delta: 0});
    const move = (moveEvent: PointerEvent) => {
      const current = dragRef.current;
      if (!current) return;
      let delta = Math.round((moveEvent.clientX - current.startX) / pxPerFrame);
      if (current.mode === 'move') delta = snapFrame(current.originalStart + delta, current.clipId) - current.originalStart;
      if (current.mode === 'trim-left') delta = clamp(delta, -current.originalStart, current.originalDuration - 1);
      if (current.mode === 'trim-right') delta = clamp(delta, 1 - current.originalDuration, Math.max(1, duration - clipEnd(clip)));
      current.delta = delta;
      setDragPreview({clipId: current.clipId, mode: current.mode, delta});
    };
    const up = () => {
      const current = dragRef.current;
      dragRef.current = null;
      setDragPreview(null);
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
      if (!current || current.delta === 0) return;
      if (current.mode === 'move') void onApply([{op: 'move', track_id: current.trackId, clip_id: current.clipId, at_frame: Math.max(0, current.originalStart + current.delta)}]);
      else void onApply([{op: 'trim_edge', track_id: current.trackId, clip_id: current.clipId, edge: current.mode === 'trim-left' ? 'left' : 'right', delta_frames: current.delta}]);
    };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
  }
  function beginMarquee(event: React.PointerEvent) {
    if (event.button !== 0 || tool !== 'select') return;
    const additive = event.shiftKey || event.metaKey || event.ctrlKey;
    marqueeRef.current = {startX: event.clientX, startY: event.clientY, additive};
    setMarquee({x: event.clientX, y: event.clientY, width: 0, height: 0});
    const move = (moveEvent: PointerEvent) => {
      const start = marqueeRef.current;
      if (!start) return;
      setMarquee(normalizeRect(start.startX, start.startY, moveEvent.clientX, moveEvent.clientY));
    };
    const up = (upEvent: PointerEvent) => {
      const start = marqueeRef.current;
      marqueeRef.current = null;
      setMarquee(null);
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
      if (!start) return;
      const rect = normalizeRect(start.startX, start.startY, upEvent.clientX, upEvent.clientY);
      if (rect.width < 4 && rect.height < 4) { onPlayhead(frameAtClientX(upEvent.clientX)); return; }
      const found: string[] = [];
      document.querySelectorAll('[data-timeline-clip]').forEach((element) => {
        const box = element.getBoundingClientRect();
        if (box.right >= rect.x && box.left <= rect.x + rect.width && box.bottom >= rect.y && box.top <= rect.y + rect.height) found.push(element.getAttribute('data-timeline-clip') || '');
      });
      const ids = found.filter(Boolean);
      onSelect(start.additive ? Array.from(new Set([...selectedIds, ...ids])) : ids, ids[0]);
    };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
  }
  function beginRuler(event: React.PointerEvent) {
    if (event.button !== 0) return;
    event.preventDefault();
    const seek = (clientX: number) => onPlayhead(frameAtClientX(clientX));
    seek(event.clientX);
    const move = (moveEvent: PointerEvent) => seek(moveEvent.clientX);
    const up = () => { window.removeEventListener('pointermove', move); window.removeEventListener('pointerup', up); };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
  }
  useEffect(() => {
    const close = () => setMenu(null);
    const escape = (event: KeyboardEvent) => { if (event.key === 'Escape') setMenu(null); };
    window.addEventListener('pointerdown', close);
    window.addEventListener('keydown', escape);
    return () => { window.removeEventListener('pointerdown', close); window.removeEventListener('keydown', escape); };
  }, []);
  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      const target = event.target as HTMLElement | null;
      if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT' || target.isContentEditable)) return;
      const key = event.key.toLowerCase();
      if (key === 'v') { event.preventDefault(); onTool('select'); }
      else if (key === 'c') { event.preventDefault(); onTool('blade'); }
      else if (key === 't') { event.preventDefault(); onTool('trim'); }
      else if (key === 'r') { event.preventDefault(); onTool('ripple'); }
      else if (key === '+' || key === '=') { event.preventDefault(); setZoom((value) => clamp(value * 1.25, 0.5, 12)); }
      else if (key === '-') { event.preventDefault(); setZoom((value) => clamp(value / 1.25, 0.5, 12)); }
      else if (key === 'f') { event.preventDefault(); setZoom(1); }
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [onTool]);
  const topLine = GUTTER + playhead * pxPerFrame;
  const menuClip = menu?.clip;
  return <div className="timeline-canvas" data-tool={tool} data-testid="timeline-canvas">
    <div className="timeline-canvas-toolbar">
      <div className="timeline-tool-group" role="toolbar" aria-label="时间轴工具">
        {([['select', '选择', MousePointer2, 'V'], ['blade', '刀片', Scissors, 'C'], ['trim', '修剪', SlidersHorizontal, 'T'], ['ripple', '波纹', Sparkles, 'R']] as const).map(([value, label, Icon, shortcut]) => <button key={value} type="button" className="timeline-tool" data-active={tool === value} data-testid={`timeline-tool-${value}`} onClick={() => onTool(value)} title={`${label} (${shortcut})`} aria-label={label}><Icon size={13} /><span>{label}</span><kbd>{shortcut}</kbd></button>)}
      </div>
      <div className="timeline-tool-spacer" />
      <button type="button" className="timeline-tool" data-active={snap} onClick={() => onSnap(!snap)} title="吸附 (N)"><Magnet size={13} /><span>吸附</span></button>
      <button type="button" className="timeline-tool" onClick={() => setZoom((value) => clamp(value / 1.25, 0.5, 12))} title="缩小" aria-label="缩小"><ZoomOut size={13} /></button>
      <span className="timeline-zoom-label">{Math.round(zoom * 100)}%</span>
      <button type="button" className="timeline-tool" onClick={() => setZoom((value) => clamp(value * 1.25, 0.5, 12))} title="放大" aria-label="放大"><ZoomIn size={13} /></button>
      <button type="button" className="timeline-tool" onClick={() => setZoom(1)} title="适应" aria-label="适应时间轴"><ChevronDown size={13} /></button>
      <span className="timeline-canvas-stats">{visibleClips.length}/{timeline.tracks.reduce((sum, track) => sum + track.clips.length, 0)} clips · {timecode(playhead, fps)}</span>
    </div>
    <div className="timeline-canvas-scroll" ref={scrollRef} onScroll={(event) => setViewport((current) => ({...current, scrollLeft: event.currentTarget.scrollLeft}))}>
      <div className="timeline-canvas-inner" ref={innerRef} style={{width: GUTTER + duration * pxPerFrame + 24, minHeight: '100%'}} onPointerDown={(event) => { if (event.target === event.currentTarget) beginMarquee(event); }}>
        <div className="timeline-canvas-ruler-row">
          <div className="timeline-canvas-gutter"><span>TRACKS</span><button type="button" className="timeline-track-add" data-testid="timeline-track-add" title="添加轨道" aria-label="添加轨道" onClick={() => setTrackMenuOpen((value) => !value)}><Plus size={11} /></button></div>
          <div className="timeline-canvas-ruler" style={{width: duration * pxPerFrame}} onPointerDown={beginRuler} role="slider" aria-label="播放头" aria-valuemin={0} aria-valuemax={duration} aria-valuenow={playhead} tabIndex={0}>
            {ticks.map((frame) => <span key={frame} className="timeline-canvas-tick" style={{left: frame * pxPerFrame}}>{rulerLabel(frame, fps)}</span>)}
            <span className="timeline-canvas-playhead-handle" style={{left: playhead * pxPerFrame}} title={timecode(playhead, fps)} />
          </div>
        </div>
        {timeline.tracks.map((track) => {
          const defaultHeight = ROW_HEIGHT[track.kind] ?? 50;
          const rowHeight = trackHeightPreview?.trackId === track.id ? trackHeightPreview.height : (track.height || defaultHeight);
          return <div key={track.id} className="timeline-canvas-track" data-track-id={track.id} data-kind={track.kind} data-muted={track.muted} data-drop-target={dropTargetId === track.id} style={{height: rowHeight}}>
            <div className="timeline-canvas-track-head" data-kind={track.kind} data-muted={track.muted} data-locked={track.locked} data-drop-target={dropTargetId === track.id} onContextMenu={(event) => { event.preventDefault(); event.stopPropagation(); setMenu({x: event.clientX, y: event.clientY, track, frame: playhead}); }} onDragOver={(event) => { if (!draggedTrackId) return; const source = timeline.tracks.find((item) => item.id === draggedTrackId); if (!source || source.kind !== track.kind) return; event.preventDefault(); event.dataTransfer.dropEffect = 'move'; setDropTargetId(track.id); }} onDragLeave={() => { if (dropTargetId === track.id) setDropTargetId(''); }} onDrop={(event) => { event.preventDefault(); dropTrack(track); }}>
              <span className="timeline-track-grip" draggable title="拖动排序" onDragStart={(event) => { setDraggedTrackId(track.id); event.dataTransfer.effectAllowed = 'move'; event.dataTransfer.setData('text/plain', track.id); }} onDragEnd={() => { setDraggedTrackId(''); setDropTargetId(''); }}><GripVertical size={11} /></span>
              <label className="timeline-track-color" style={{background: track.color || defaultTrackColor(track.kind)}} title="轨道颜色">
                <input type="color" value={track.color || defaultTrackColor(track.kind)} onChange={(event) => setTrackState(track, {color: event.target.value})} aria-label="轨道颜色" />
              </label>
              <div className="timeline-track-main">
                {editingTrackId === track.id ? <input className="timeline-track-name-input" autoFocus value={editingName} onChange={(event) => setEditingName(event.target.value)} onBlur={() => commitTrackName(track)} onKeyDown={(event) => { if (event.key === 'Enter') { event.preventDefault(); commitTrackName(track); } else if (event.key === 'Escape') { setEditingTrackId(''); } }} /> : <span className="timeline-track-name" title="双击重命名" onDoubleClick={() => { setEditingTrackId(track.id); setEditingName(track.name || track.id); }}>{track.name || track.id}</span>}
                {track.kind === 'audio' ? <div className="timeline-track-volume-row" title={`轨道音量 ${volumeDraft[track.id] ?? track.volume_db ?? 0} dB`}>
                  <input type="range" min={-60} max={12} step={1} value={volumeDraft[track.id] ?? track.volume_db ?? 0} onChange={(event) => setVolumeDraft((current) => ({...current, [track.id]: Number(event.target.value)}))} onPointerUp={() => commitVolume(track)} onBlur={() => commitVolume(track)} onKeyUp={() => commitVolume(track)} aria-label="轨道音量" />
                  <span>{volumeDraft[track.id] ?? track.volume_db ?? 0}</span>
                </div> : null}
                <span className="timeline-track-kind" title={`${track.id} · ${track.kind}${track.role ? ` · ${track.role}` : ''}`}>{track.name ? `${track.id} · ` : ''}{track.role ? roleLabel(track.role) : (track.kind === 'video' ? 'VIDEO' : track.kind === 'audio' ? 'AUDIO' : 'SUBTITLE')}</span>
              </div>
              {track.kind === 'audio' ? <span className="timeline-track-meter" title={`电平 ${Math.round((audioLevels[track.id] ?? 0) * 100)}%`}><span style={{height: `${Math.max(2, Math.round((audioLevels[track.id] ?? 0) * 100))}%`}} /></span> : null}
              <div className="timeline-track-controls">
                <button type="button" className="timeline-track-btn" data-active={track.muted} title={track.kind === 'audio' ? (track.muted ? '取消静音' : '静音轨道') : (track.muted ? '显示轨道' : '隐藏轨道')} aria-label={track.kind === 'audio' ? '静音轨道' : '隐藏轨道'} onClick={() => setTrackState(track, {muted: !track.muted})}>{track.kind === 'audio' ? (track.muted ? <VolumeX size={11} /> : <Volume2 size={11} />) : (track.muted ? <EyeOff size={11} /> : <Eye size={11} />)}</button>
                {track.kind === 'audio' ? <button type="button" className="timeline-track-btn timeline-track-btn-solo" data-active={track.solo} title={track.solo ? '取消 Solo' : 'Solo 独听'} aria-label="Solo" onClick={() => setTrackState(track, {solo: !track.solo})}>S</button> : null}
                <button type="button" className="timeline-track-btn" data-active={track.locked} title={track.locked ? '解锁轨道' : '锁定轨道'} aria-label="锁定轨道" onClick={() => setTrackState(track, {locked: !track.locked})}>{track.locked ? <Lock size={11} /> : <Unlock size={11} />}</button>
              </div>
              <span className="timeline-track-resize" title="拖动调整轨道高度" onPointerDown={(event) => beginTrackResize(event, track, rowHeight)} />
            </div>
            <div className="timeline-canvas-lane" style={{left: GUTTER, width: duration * pxPerFrame}} onPointerDown={beginMarquee} onContextMenu={(event) => { event.preventDefault(); setMenu({x: event.clientX, y: event.clientY, track, frame: frameAtClientX(event.clientX)}); }} onDragOver={(event) => { event.preventDefault(); event.dataTransfer.dropEffect = 'copy'; }} onDrop={(event) => { event.preventDefault(); const artifactId = event.dataTransfer.getData('application/x-himind-artifact'); if (artifactId) onInsertAt?.(track.id, frameAtClientX(event.clientX), artifactId); }}>
              {track.clips.filter((clip) => clipEnd(clip) >= windowStart && clip.start_frame <= windowEnd).map((clip) => <ClipView key={clip.id} clip={clip} track={track} timeline={timeline} artifacts={artifacts} workspace={workspace} pxPerFrame={pxPerFrame} selected={selectedSet.has(clip.id)} primary={primaryId === clip.id} diff={diff} dragPreview={dragPreview} onPointerDown={beginClipPointer} onSelect={onSelect} onContextMenu={(event, target) => setMenu({x: event.clientX, y: event.clientY, clip: target, track, frame: frameAtClientX(event.clientX)})} onBlade={(frame, target) => { if (frame > target.start_frame && frame < clipEnd(target)) void onApply([{op: 'blade', track_id: target.track_id, clip_id: target.id, at_frame: frame}]); }} onTransition={(edge, frames) => { const transition = frames > 0 ? {type: 'fade', duration_frames: frames} : undefined; const patch = edge === 'in' ? {transition_in: transition} : {transition_out: transition}; void onApply([{op: 'update_clip', track_id: clip.track_id, clip_id: clip.id, clip: {...clip, ...patch}}]); }} onSeek={onPlayhead} />)}
              {selectedIds.map((id) => { const clip = track.clips.find((item) => item.id === id); return clip ? <KeyframeLane key={`kf-${id}`} clip={clip} timeline={timeline} pxPerFrame={pxPerFrame} onSeek={onPlayhead} /> : null; })}
              {selectedIds.length === 1 && track.clips.some((clip) => clip.id === selectedIds[0]) ? <span className="timeline-selection-band" style={{left: (track.clips.find((clip) => clip.id === selectedIds[0])?.start_frame || 0) * pxPerFrame, width: Math.max(12, (track.clips.find((clip) => clip.id === selectedIds[0])?.duration_frames || 0) * pxPerFrame)}} /> : null}
            </div>
          </div>;
        })}
        <div className="timeline-canvas-playhead" style={{left: topLine}} aria-hidden="true" />
        {marquee ? <div className="timeline-marquee" style={{left: marquee.x - (innerRef.current?.getBoundingClientRect().left || 0), top: marquee.y - (innerRef.current?.getBoundingClientRect().top || 0), width: marquee.width, height: marquee.height}} /> : null}
      </div>
    </div>
    {trackMenuOpen ? <div className="timeline-track-menu" role="menu" aria-label="添加轨道">
      <button type="button" role="menuitem" onClick={() => { setTrackMenuOpen(false); addTrack('video'); }}>视频轨 V</button>
      <button type="button" role="menuitem" onClick={() => { setTrackMenuOpen(false); addTrack('audio'); }}>音频轨 A</button>
      <button type="button" role="menuitem" onClick={() => { setTrackMenuOpen(false); addTrack('subtitle'); }}>字幕轨 S</button>
      <div className="timeline-track-menu-sep" />
      <button type="button" role="menuitem" onClick={() => { setTrackMenuOpen(false); applyDensity('compact'); }}>紧凑高度</button>
      <button type="button" role="menuitem" onClick={() => { setTrackMenuOpen(false); applyDensity('standard'); }}>标准高度</button>
      <button type="button" role="menuitem" onClick={() => { setTrackMenuOpen(false); applyDensity('high'); }}>高轨道</button>
    </div> : null}
    {menu ? <div className="timeline-context-menu" style={{left: menu.x, top: menu.y}} onPointerDown={(event) => event.stopPropagation()} role="menu" aria-label="时间轴操作">
      <div className="timeline-context-title">{menuClip ? menuClip.id : menu.track ? `${menu.track.id} · ${timecode(menu.frame, fps)}` : '时间轴'}</div>
      {menuClip ? <>
        <button type="button" role="menuitem" onClick={() => { setMenu(null); if (playhead > menuClip.start_frame && playhead < clipEnd(menuClip)) void onApply([{op: 'blade', track_id: menuClip.track_id, clip_id: menuClip.id, at_frame: playhead}]); else onToast?.('播放头不在片段内'); }}>在播放头切开</button>
        <button type="button" role="menuitem" onClick={() => { setMenu(null); void onApply([{op: 'lift', track_id: menuClip.track_id, clip_id: menuClip.id}]); }}>删除（留空）</button>
        <button type="button" role="menuitem" onClick={() => { setMenu(null); void onApply([{op: 'ripple_delete', track_id: menuClip.track_id, clip_id: menuClip.id}]); }}>波纹删除</button>
        <div className="timeline-context-sep" />
        <button type="button" role="menuitem" onClick={() => { setMenu(null); void onApply([{op: 'retime', track_id: menuClip.track_id, clip_id: menuClip.id, speed: 0.5, ripple: true}]); }}>变速 0.5×</button>
        <button type="button" role="menuitem" onClick={() => { setMenu(null); void onApply([{op: 'retime', track_id: menuClip.track_id, clip_id: menuClip.id, speed: 1, ripple: true}]); }}>恢复 1×</button>
        <button type="button" role="menuitem" onClick={() => { setMenu(null); void onApply([{op: 'retime', track_id: menuClip.track_id, clip_id: menuClip.id, speed: 2, ripple: true}]); }}>变速 2×</button>
        <div className="timeline-context-sep" />
        <button type="button" role="menuitem" onClick={() => { setMenu(null); void onApply([{op: 'link', clip_ids: selectedIds.length ? selectedIds : [menuClip.id], linked_group: 'av-' + Date.now()}]); }}>联动所选 A/V</button>
        <button type="button" role="menuitem" onClick={() => { setMenu(null); void onApply([{op: 'unlink', clip_ids: selectedIds.length ? selectedIds : [menuClip.id]}]); }}>解除联动</button>
      </> : menu.track ? <>
        <button type="button" role="menuitem" onClick={() => { setMenu(null); setTrackState(menu.track!, {muted: !menu.track!.muted}); }}>{menu.track.muted ? '显示 / 取消静音' : '隐藏 / 静音轨道'}</button>
        <button type="button" role="menuitem" onClick={() => { setMenu(null); setTrackState(menu.track!, {locked: !menu.track!.locked}); }}>{menu.track.locked ? '解锁轨道' : '锁定轨道'}</button>
        <button type="button" role="menuitem" onClick={() => { setMenu(null); void onApply([{op: 'set_track_height', track_id: menu.track!.id, height: 0}]); }}>重置轨道高度</button>
        <div className="timeline-context-sep" />
        <button type="button" role="menuitem" onClick={() => { setMenu(null); onInsertAt?.(menu.track!.id, menu.frame); }}>在播放头插入素材</button>
        <button type="button" role="menuitem" disabled={menu.track.clips.length > 0} title={menu.track.clips.length > 0 ? '仅空轨道可删除' : '删除空轨道'} onClick={() => { setMenu(null); removeTrack(menu.track!); }}>删除空轨道</button>
      </> : null}
    </div> : null}
  </div>;
}

function normalizeRect(x1: number, y1: number, x2: number, y2: number): Marquee {
  return {x: Math.min(x1, x2), y: Math.min(y1, y2), width: Math.abs(x2 - x1), height: Math.abs(y2 - y1)};
}
