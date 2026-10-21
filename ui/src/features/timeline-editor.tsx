import {useEffect, useRef, useState} from 'react';
import {History as HistoryIcon, Library, MessageSquareText, PanelLeft, SlidersHorizontal, Sparkles, X} from 'lucide-react';
import {callCapability} from '../api/capability';
import type {Artifact, HistoryEntry, StudioMode, Timeline, TimelineClip, TimelineDiff, TimelineMediaRef, TimelineTool, TimelineTrack} from './timeline/types';
import {usePersistentState} from './timeline/hooks';
import {TimelineCanvas} from './timeline/canvas';
import {ProgramMonitor} from './timeline/program-monitor';
import {MediaLibraryPanel, ClipInspectorPanel, HistoryPanel, TranscriptPanel, AssetBrowserPanel, DiffPanel, SemanticPanel} from './timeline/panels-media-inspector';
import {TimelineAudioMixer} from './timeline/audio-mixer';

export function TimelineEditor({workspace, projectId, artifacts = [], onSaved, active = true}: {workspace: string; projectId: string; artifacts?: Array<{artifact_id: string; kind: string; path: string; size?: number; created_at?: string; quality_status?: string}>; onSaved?: () => void; active?: boolean}) {
  const [timeline, setTimeline] = useState<Timeline | null>(null);
  const timelineRef = useRef<Timeline | null>(null);
  useEffect(() => { timelineRef.current = timeline; }, [timeline]);
  const [selected, setSelected] = useState('');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const selectionRef = useRef<{ids: string[]; primary: string}>({ids: [], primary: ''});
  const [busy, setBusy] = useState(false);
  const [tool, setTool] = usePersistentState<TimelineTool>('himind.timeline.tool', 'select');
  const [playing, setPlaying] = useState(false);
  const [audioLevels, setAudioLevels] = useState<Record<string, number>>({});
  const [historyEntries, setHistoryEntries] = useState<HistoryEntry[]>([]);
  const [historyDiff, setHistoryDiff] = useState<TimelineDiff | null>(null);
  const [commandText, setCommandText] = useState('把第 1 段缩短到 4 秒');
  const [commandBusy, setCommandBusy] = useState(false);
  const [pendingCommands, setPendingCommands] = useState<{commands: Array<Record<string, unknown>>; diff: TimelineDiff; label: string} | null>(null);
  const [transcriptOpen, setTranscriptOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [assetOpen, setAssetOpen] = useState(false);
  const [semanticOpen, setSemanticOpen] = useState(false);
  const [semanticQuery, setSemanticQuery] = useState('');
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [savedRevision, setSavedRevision] = useState(0);
  const [playhead, setPlayhead] = useState(0);

  const [renderResult, setRenderResult] = useState<any>(null);
  const [missingMedia, setMissingMedia] = useState<string[]>([]);
  const [relinkMap, setRelinkMap] = useState<Record<string, string>>({});
  const [skipMissing, setSkipMissing] = useState<Record<string, boolean>>({});
  const [mediaUrl, setMediaUrl] = useState('');
  const [mediaError, setMediaError] = useState('');
  const [sourceIn, setSourceIn] = useState(0);
  const [sourceOut, setSourceOut] = useState(90);

  const [undoStack, setUndoStack] = useState<Timeline[]>([]);
  const [redoStack, setRedoStack] = useState<Timeline[]>([]);
  const [snap, setSnap] = usePersistentState('himind.timeline.snap', true);
  const [shortcutsOpen, setShortcutsOpen] = useState(false);
  const [mode, setMode] = useState<StudioMode>('easy');
  const [selectedMedia, setSelectedMedia] = useState<string[]>([]);
  const [targetDuration, setTargetDuration] = useState(15);
  const [deliveryPreset, setDeliveryPreset] = useState('douyin');
  const [burnSubtitles, setBurnSubtitles] = useState(false);
  const [qaResult, setQaResult] = useState<any>(null);
  const [mediaCatalog, setMediaCatalog] = useState<TimelineMediaRef[]>([]);
  const [proxyBusy, setProxyBusy] = useState(false);
  const [templateId, setTemplateId] = useState('hook-proof-payoff');
  const [searchQuery, setSearchQuery] = useState('合肥');
  const [searchHits, setSearchHits] = useState<any[]>([]);
  const [roughCutBusy, setRoughCutBusy] = useState(false);
  const [commentsOpen, setCommentsOpen] = useState(false);
  const [comments, setComments] = useState<any[]>([]);
  const [commentText, setCommentText] = useState('');
  const [beatPath, setBeatPath] = useState('');
  const [aspect, setAspect] = useState('9:16');
  const [mediaPanelOpen, setMediaPanelOpen] = usePersistentState('himind.timeline.layout.mediaOpen', false);
  const [inspectorPanelOpen, setInspectorPanelOpen] = usePersistentState('himind.timeline.layout.inspectorOpen', false);
  const [mediaPanelWidth, setMediaPanelWidth] = usePersistentState('himind.timeline.layout.mediaWidth', 280);
  const [inspectorPanelWidth, setInspectorPanelWidth] = usePersistentState('himind.timeline.layout.inspectorWidth', 310);
  const [timelineHeight, setTimelineHeight] = usePersistentState('himind.timeline.layout.timelineHeight', 0);

  useEffect(() => {
    if (mode === 'pro') setInspectorPanelOpen(true);
  }, [mode]);
  function beginPanelResize(kind: 'media' | 'inspector' | 'timeline', event: React.PointerEvent) {
    event.preventDefault();
    const startX = event.clientX;
    const startY = event.clientY;
    const mediaStart = mediaPanelWidth;
    const inspectorStart = inspectorPanelWidth;
    const timelineStart = timelineHeight || 220;
    const move = (moveEvent: PointerEvent) => {
      if (kind === 'media') setMediaPanelWidth(Math.max(220, Math.min(420, mediaStart + moveEvent.clientX - startX)));
      else if (kind === 'inspector') setInspectorPanelWidth(Math.max(260, Math.min(460, inspectorStart - (moveEvent.clientX - startX))));
      else setTimelineHeight(Math.max(140, Math.min(460, timelineStart - (moveEvent.clientY - startY))));
    };
    const up = () => { window.removeEventListener('pointermove', move); window.removeEventListener('pointerup', up); };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
  }
  function selectClips(ids: string[], primary?: string) {
    const nextPrimary = primary ?? ids[0] ?? '';
    selectionRef.current = {ids, primary: nextPrimary};
    setSelectedIds(ids);
    setSelected(nextPrimary);
  }
  function selectClip(id: string) { selectClips(id ? [id] : []); }
  function pushHistory(label: string, snapshot: Timeline) {
    setHistoryEntries((items) => [...items, {id: `hist-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`, label, revision: snapshot.revision, timeline: snapshot, createdAt: Date.now()}].slice(-60));
  }
  async function load(attempt = 0) {
    setError('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.get', {workspace_root: workspace, project_id: projectId});
      const loaded = result?.state === 'ready' ? result.timeline : null;
      timelineRef.current = loaded;
      setTimeline(loaded);
      setSavedRevision(result?.state === 'ready' ? result.timeline.revision : 0);
      selectionRef.current = {ids: [], primary: ''};
      setSelected(''); setSelectedIds([]);
      setPlayhead(0);
      setPlaying(false);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      // 从审阅页快速切到剪辑页时，上一页的图片读取可能还在占用宿主调用配额。
      // 这是一次性拥塞，不把用户留在不可恢复的错误态；短暂等待后只重试一次。
      if (attempt < 1 && /invocation limit/i.test(message)) {
        setNotice('宿主正在收尾上一页请求，正在重试…');
        window.setTimeout(() => { void load(attempt + 1); }, 700);
        return;
      }
      setError(message);
    }
  }
  useEffect(() => { void load(); /* eslint-disable-next-line react-hooks/exhaustive-deps */ }, [workspace, projectId]);
  useEffect(() => {
    if (!timeline) return;
    const max = Math.max(0, timeline.duration_frames - 1);
    setPlayhead((frame) => Math.min(frame, max));
  }, [timeline]);

  async function apply(commands: Array<Record<string, unknown>>) {
    const currentTimeline = timelineRef.current ?? timeline;
    if (!currentTimeline) return;
    setBusy(true); setError(''); setNotice('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.apply', {workspace_root: workspace, project_id: projectId, timeline: currentTimeline, commands});
      if (result?.ok) {
        setUndoStack((stack) => [...stack, currentTimeline].slice(-50));
        setRedoStack([]);
        pushHistory(commands.map((command) => String(command.op)).join(' + ') || '时间轴命令', currentTimeline);
        timelineRef.current = result.timeline;
        setTimeline(result.timeline);
        const nextIDs = new Set<string>((result.timeline?.tracks ?? []).flatMap((track: TimelineTrack) => track.clips.map((clip) => clip.id)));
        const current = selectionRef.current;
        const alive = current.ids.filter((id) => nextIDs.has(id));
        const nextPrimary = alive.includes(current.primary) ? current.primary : (alive[0] ?? '');
        selectionRef.current = {ids: alive, primary: nextPrimary};
        setSelectedIds(alive);
        setSelected(nextPrimary);
        setNotice(`revision ${result.revision}`);
      } else {
        setError((result?.blockers ?? []).join('；') || '命令被拒绝');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally { setBusy(false); }
  }

  function undo() {
    if (!undoStack.length || !timeline) return;
    const previous = undoStack[undoStack.length - 1];
    setUndoStack((stack) => stack.slice(0, -1));
    setRedoStack((stack) => [...stack, timeline].slice(-50));
    setTimeline(previous);
    setNotice('已撤销');
  }
  function redo() {
    if (!redoStack.length || !timeline) return;
    const next = redoStack[redoStack.length - 1];
    setRedoStack((stack) => stack.slice(0, -1));
    setUndoStack((stack) => [...stack, timeline].slice(-50));
    setTimeline(next);
    setNotice('已重做');
  }
  async function save() {
    if (!timeline) return;
    setBusy(true); setError(''); setNotice('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.save', {
        workspace_root: workspace, project_id: projectId,
        timeline, expected_revision: savedRevision,
      });
      if (result?.ok) { setTimeline(result.timeline); setSavedRevision(result.revision ?? result.timeline.revision); setNotice(`已保存 revision ${result.revision}`); await onSaved?.(); }
      else setError((result?.blockers ?? []).join('；') || '保存被拒绝');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally { setBusy(false); }
  }

  function insertWithRange() {
    if (!selectedClip || !timeline) return;
    const inFrame = Math.max(0, Math.min(sourceIn, sourceOut - 1));
    const outFrame = Math.max(inFrame + 1, sourceOut);
    void apply([{
      op: 'insert', track_id: selectedClip.track_id, at_frame: playhead,
      clip: {
        id: `clip-${Date.now()}`, media_id: selectedClip.media_id, track_id: selectedClip.track_id,
        start_frame: playhead, duration_frames: outFrame - inFrame, source_in_frame: inFrame, source_out_frame: outFrame, speed: 1,
      },
    }]);
  }
  function insertArtifact(artifact: {artifact_id: string; kind?: string}, trackId = 'V1', atFrame = playhead) {
    if (!timeline) return;
    void apply([{
      op: 'insert', track_id: trackId, at_frame: atFrame,
      clip: {
        id: `clip-${Date.now()}`, media_id: artifact.artifact_id, track_id: trackId,
        start_frame: atFrame, duration_frames: 90, source_in_frame: 0, source_out_frame: 90, speed: 1,
      },
    }]);
  }
  function dropArtifact(event: React.DragEvent, track: TimelineTrack) {
    event.preventDefault();
    event.stopPropagation();
    const artifactId = event.dataTransfer.getData('application/x-himind-artifact');
    const artifact = artifacts.find((item) => item.artifact_id === artifactId);
    if (!artifact) return;
    const rect = event.currentTarget.getBoundingClientRect();
    let atFrame = Math.max(0, Math.round(((event.clientX - rect.left) / rect.width) * duration));
    if (snap) {
      const boundaries = timeline?.tracks.flatMap((item) => item.clips.flatMap((clip) => [clip.start_frame, clip.start_frame + clip.duration_frames])) ?? [];
      const nearest = boundaries.reduce((best, value) => Math.abs(value - atFrame) < Math.abs(best - atFrame) ? value : best, atFrame);
      if (Math.abs(nearest - atFrame) <= 15) atFrame = nearest;
    }
    insertArtifact(artifact, track.id, atFrame);
  }
  const videoArtifacts = artifacts.filter((item) => item.kind.startsWith('video/'));
  useEffect(() => {
    setSelectedMedia((current) => {
      const alive = current.filter((id) => videoArtifacts.some((item) => item.artifact_id === id));
      return alive.length ? alive : videoArtifacts.slice(0, 3).map((item) => item.artifact_id);
    });
  }, [artifacts]);
  function selectedArtifactIds() {
    return selectedMedia.length ? selectedMedia : videoArtifacts.slice(0, 3).map((item) => item.artifact_id);
  }
  function mediaPathMap(ids?: string[]) {
    const paths: Record<string, string> = {};
    const chosen = ids ? new Set(ids) : null;
    for (const artifact of artifacts) {
      if (!chosen || chosen.has(artifact.artifact_id)) paths[artifact.artifact_id] = artifact.path;
    }
    return paths;
  }
  function replaceTimeline(nextTimeline: Timeline, message: string) {
    const before = timelineRef.current ?? timeline;
    if (before) { setUndoStack((stack) => [...stack, before].slice(-50)); pushHistory(message, before); }
    setRedoStack([]);
    timelineRef.current = nextTimeline;
    setTimeline(nextTimeline);
    const nextIDs = new Set<string>(nextTimeline.tracks.flatMap((track) => track.clips.map((clip) => clip.id)));
    const current = selectionRef.current;
    const alive = current.ids.filter((id) => nextIDs.has(id));
    const nextPrimary = alive.includes(current.primary) ? current.primary : (alive[0] ?? '');
    selectionRef.current = {ids: alive, primary: nextPrimary};
    setSelectedIds(alive);
    setSelected(nextPrimary);
    setNotice(message);
  }
  async function autoEdit() {
    const ids = selectedArtifactIds();
    if (!ids.length) { setError('至少选择一个视频产物'); return; }
    setBusy(true); setError(''); setNotice('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.autoedit', {
        workspace_root: workspace, project_id: projectId, media_ids: ids,
        media_paths: mediaPathMap(ids), duration_seconds: targetDuration,
        transition: 'fade', transition_frames: 6,
        timeline: timeline ? {timebase: timeline.timebase, canvas: timeline.canvas} : {timebase: {num: 30, den: 1}, canvas: {width: 1080, height: 1920}},
      });
      if (result?.state === 'ready' && result.timeline) replaceTimeline(result.timeline, '已生成一键成片时间轴，可预览或精修');
      else setError((result?.blockers ?? []).join('；') || '一键成片失败');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally { setBusy(false); }
  }
  async function reframeTimeline() {
    if (!timeline) return;
    const dims = aspect === '16:9' ? {width: 1920, height: 1080} : aspect === '1:1' ? {width: 1080, height: 1080} : {width: 1080, height: 1920};
    setBusy(true); setError('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.reframe', {workspace_root: workspace, timeline, ...dims, fit_mode: 'cover'});
      if (result?.state === 'ready' && result.timeline) replaceTimeline(result.timeline, `已重构为 ${aspect}`);
      else setError((result?.blockers ?? []).join('；') || '画布重构失败');
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  async function addBeatMarkers() {
    if (!timeline || !beatPath.trim()) { setError('填写工作区内的音频路径后再检测卡点'); return; }
    setBusy(true); setError('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.beats.markers', {workspace_root: workspace, path: beatPath.trim(), timeline});
      if (result?.state === 'ready') { setTimeline(result.timeline); setNotice(`已写入 ${result.total} 个节拍标记`); }
      else setError((result?.blockers ?? []).join('；') || '卡点检测失败');
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  async function removeSilence() {
    const id = selectedArtifactIds()[0];
    const artifact = artifacts.find((item) => item.artifact_id === id);
    if (!artifact) { setError('先选择一个视频产物'); return; }
    setBusy(true); setError(''); setNotice('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.silence.trim', {
        workspace_root: workspace, path: artifact.path, media_id: artifact.artifact_id,
        noise_db: -35, min_silence_seconds: 0.35, padding_seconds: 0.08,
        timeline: timeline ? {timebase: timeline.timebase, canvas: timeline.canvas} : {timebase: {num: 30, den: 1}, canvas: {width: 1080, height: 1920}},
      });
      if (result?.state === 'ready' && result.timeline) replaceTimeline(result.timeline, `已去除 ${result.removed_seconds.toFixed(1)}s 静音`);
      else setError((result?.blockers ?? []).join('；') || '没有可去除的静音区间');
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  async function applyTemplate(nextTemplateId = templateId) {
    const ids = selectedArtifactIds();
    if (!ids.length) { setError('至少选择一个视频产物'); return; }
    setBusy(true); setError(''); setNotice('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.template.apply', {
        workspace_root: workspace, template_id: nextTemplateId, media_ids: ids, duration_seconds: targetDuration,
        transition: 'fade', timeline: timeline ? {timebase: timeline.timebase, canvas: timeline.canvas} : {timebase: {num: 30, den: 1}, canvas: {width: 1080, height: 1920}},
      });
      if (result?.state === 'ready' && result.timeline) replaceTimeline(result.timeline, `已套用模板 ${result.template_id}`);
      else setError((result?.blockers ?? []).join('；') || '模板套用失败');
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  async function searchTranscript() {
    if (!timeline) return;
    setBusy(true); setError('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.transcript.search', {workspace_root: workspace, timeline, query: searchQuery, match_mode: 'any'});
      setSearchHits(result?.hits ?? []);
      setNotice(`文本命中 ${result?.total ?? 0} 段`);
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  async function roughCut() {
    if (!timeline) return;
    setRoughCutBusy(true); setError('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.roughcut', {workspace_root: workspace, timeline, query: searchQuery, match_mode: 'any', padding_seconds: 0.15, duration_seconds: targetDuration});
      if (result?.state === 'ready' && result.timeline) { replaceTimeline(result.timeline, `AI 粗剪完成：${result.selected_cues} 段文本`); setSearchHits([]); }
      else setError((result?.blockers ?? []).join('；') || '没有命中可粗剪的文本');
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setRoughCutBusy(false); }
  }
  async function attachSubtitles() {
    if (!timeline) return;
    setBusy(true); setError(''); setNotice('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.subtitle.attach', {
        workspace_root: workspace, project_id: projectId, timeline,
      });
      if (result?.state === 'ready' && result.timeline) { replaceTimeline(result.timeline, `已挂载 ${result.cue_count} 条字幕`); setBurnSubtitles(true); }
      else setError((result?.blockers ?? []).join('；') || result?.message || '没有可挂载的字幕');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally { setBusy(false); }
  }
  async function inspectMedia() {
    setBusy(true); setError('');
    try {
      const ids = selectedArtifactIds();
      const result = await callCapability<any>('media.workbench.timeline.media.catalog', {workspace_root: workspace, media_paths: mediaPathMap(ids)});
      setMediaCatalog(result?.items ?? []);
      setNotice(`已探测 ${result?.total ?? 0} 个媒体`);
      if ((result?.errors ?? []).length) setError(result.errors.map((item: any) => `${item.id}: ${item.message}`).join('；'));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally { setBusy(false); }
  }
  async function buildProxies() {
    setProxyBusy(true); setError('');
    try {
      const ids = selectedArtifactIds();
      const result = await callCapability<any>('media.workbench.timeline.proxy.build', {workspace_root: workspace, media_paths: mediaPathMap(ids), proxy_width: 540});
      const failed = (result?.items ?? []).filter((item: any) => item.state !== 'ready').length;
      setNotice(`代理已生成：${result?.total ?? 0} 个${failed ? `，失败 ${failed} 个` : ''}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally { setProxyBusy(false); }
  }
  async function loadComments() {
    setBusy(true); setError('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.comment.list', {workspace_root: workspace, project_id: projectId});
      setComments(result?.items ?? []); setCommentsOpen(true);
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  async function addComment() {
    if (!commentText.trim()) return;
    setBusy(true); setError('');
    try {
      const fps = timeline?.timebase.num ?? 30;
      const result = await callCapability<any>('media.workbench.timeline.comment.add', {workspace_root: workspace, project_id: projectId, time: playhead / fps, note: commentText.trim(), category: 'edit', timeline});
      if (result?.state === 'ready') { setComments(result.items ?? []); setCommentText(''); setNotice(`已添加批注 ${result.comment?.timecode ?? ''}`); }
      else setError((result?.blockers ?? []).join('；') || '批注写入失败');
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  async function resolveComment(id: string) {
    setBusy(true); setError('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.comment.resolve', {workspace_root: workspace, project_id: projectId, feedback_id: id, status: 'resolved'});
      if (result?.state === 'ready') { setComments(result.items ?? []); setNotice('批注已解决'); }
      else setError((result?.blockers ?? []).join('；') || '批注解决失败');
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  async function runQA() {
    if (!timeline) return;
    setBusy(true); setError('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.qa', {workspace_root: workspace, timeline, media_paths: mediaPathMap()});
      setQaResult(result);
      if (result?.passed) setNotice('时间轴 QA 通过'); else setError((result?.blockers ?? []).join('；') || '时间轴 QA 失败');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally { setBusy(false); }
  }
  function missingMediaIds(input: Timeline) {
    const available = new Set(Object.keys(mediaPathMap()));
    const missing = new Set<string>();
    for (const track of input.tracks) {
      for (const clip of track.clips) {
        if (clip.media_id && !available.has(clip.media_id)) missing.add(clip.media_id);
      }
    }
    return Array.from(missing);
  }
  async function performRender(input: Timeline, missing: string[]) {
    setBusy(true); setError(''); setNotice('');
    try {
      const paths = mediaPathMap();
      for (const id of missing) {
        const artifact = artifacts.find((item) => item.artifact_id === relinkMap[id]);
        if (artifact) paths[id] = artifact.path;
      }
      const renderDoc = {
        ...input,
        tracks: input.tracks.map((track) => ({
          ...track,
          clips: track.clips.filter((clip) => !(clip.media_id && skipMissing[clip.media_id])),
        })),
      };
      const result = await callCapability<any>('media.workbench.render.start', {
        workspace_root: workspace, timeline: renderDoc, media_paths: paths,
        delivery_preset: mode === 'deliver' ? deliveryPreset : 'master',
        burn_subtitles: burnSubtitles,
      });
      setRenderResult(result);
      if (result?.state === 'completed') {
        setNotice(`已渲染 ${result.artifact?.path ?? ''}`);
        setMissingMedia([]);
        setRelinkMap({});
        setSkipMissing({});
        if (result.artifact?.path) {
          const qa = await callCapability<any>('media.workbench.delivery.qa', {workspace_root: workspace, path: result.artifact.path, delivery_preset: mode === 'deliver' ? deliveryPreset : 'master'});
          setQaResult(qa);
        }
      } else setError((result?.blockers ?? []).join('；') || '渲染被拒绝');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally { setBusy(false); }
  }
  async function renderTimeline() {
    if (!timeline) return;
    const missing = missingMediaIds(timeline);
    if (!missing.length) { await performRender(timeline, []); return; }
    if (missing.every((id) => relinkMap[id] || skipMissing[id])) { await performRender(timeline, missing); return; }
    setMissingMedia(missing);
    setError('');
    setNotice('');
  }
  function updateSelected(patch: Partial<TimelineClip>) {
    if (!selectedClip) return;
    void apply([{op: 'update_clip', track_id: selectedClip.track_id, clip_id: selectedClip.id, clip: {...selectedClip, ...patch}}]);
  }
  async function applySpeedCurve() {
    if (!selectedClip || !timeline) return;
    setBusy(true); setError('');
    try {
      const mid = Math.floor(selectedClip.duration_frames / 2);
      const result = await callCapability<any>('media.workbench.timeline.speed.curve', {workspace_root: workspace, timeline, clip_id: selectedClip.id, speed_keyframes: [{frame: 0, speed: 1}, {frame: mid, speed: 2}, {frame: selectedClip.duration_frames, speed: 1}]});
      if (result?.state === 'ready' && result.timeline) { replaceTimeline(result.timeline, `速度曲线已拆分 ${result.segments} 段`); selectClip(`${selectedClip.id}-sc01`); }
      else setError((result?.blockers ?? []).join('；') || '速度曲线生成失败');
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  async function autoReframeTimeline() {
    if (!timeline) return;
    const dims = aspect === '16:9' ? {width: 1920, height: 1080} : aspect === '1:1' ? {width: 1080, height: 1080} : {width: 1080, height: 1920};
    setBusy(true); setError('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.reframe.auto', {workspace_root: workspace, timeline, media_paths: mediaPathMap(), ...dims, fit_mode: 'cover'});
      if (result?.state === 'ready' && result.timeline) replaceTimeline(result.timeline, `内容感知重构完成：${result.updated_clips} 个片段`);
      else setError((result?.blockers ?? []).join('；') || '内容感知重构失败');
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  async function addPictureInPicture() {
    if (!selectedClip || !timeline) return;
    setBusy(true); setError('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.pip.apply', {workspace_root: workspace, timeline, clip_id: selectedClip.id});
      if (result?.state === 'ready' && result.timeline) { replaceTimeline(result.timeline, `已添加画中画 ${result.clip_id}`); selectClip(result.clip_id); }
      else setError((result?.blockers ?? []).join('；') || '画中画创建失败');
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  function linkSelected() {
    if (!selectedClip || !timeline) return;
    const candidates = timeline.tracks.flatMap((track) => track.clips.map((clip) => ({track, clip}))).filter(({track, clip}) => track.id !== selectedClip.track_id && clip.start_frame < selectedClip.start_frame + selectedClip.duration_frames && clip.start_frame + clip.duration_frames > selectedClip.start_frame);
    if (!candidates.length) { setError('当前片段附近没有可联动的音视频片段'); return; }
    const other = candidates[0].clip;
    void apply([{op: 'link', clip_ids: [selectedClip.id, other.id], linked_group: `av-${selectedClip.id}`}]);
  }

  const selectedClip = timeline?.tracks.flatMap((track) => track.clips).find((clip) => clip.id === selected) ?? null;
  const selectedClips = timeline?.tracks.flatMap((track) => track.clips).filter((clip) => selectedIds.includes(clip.id)) ?? [];
  const duration = Math.max(
    timeline?.duration_frames ?? 0,
    ...(timeline?.tracks.flatMap((track) => track.clips.map((clip) => clip.start_frame + clip.duration_frames)) ?? [0]),
    1,
  );
  async function batchApply(action: 'delete' | 'retime-0.5' | 'retime-1' | 'retime-2' | 'warm' | 'cool') {
    const clips = selectedClips.length ? selectedClips : selectedClip ? [selectedClip] : [];
    if (!clips.length) return;
    const commands: Array<Record<string, unknown>> = [];
    for (const clip of clips) {
      if (action === 'delete') commands.push({op: 'lift', track_id: clip.track_id, clip_id: clip.id});
      else if (action.startsWith('retime-')) commands.push({op: 'retime', track_id: clip.track_id, clip_id: clip.id, speed: Number(action.split('-')[1]), ripple: true});
      else if (action === 'warm' || action === 'cool') commands.push({op: 'update_clip', track_id: clip.track_id, clip_id: clip.id, clip: {...clip, color: {brightness: action === 'warm' ? 0.02 : -0.01, contrast: 1.06, saturation: action === 'warm' ? 1.12 : 0.92, gamma: 1.02}}});
    }
    if (commands.length) await apply(commands);
  }
  async function previewCommandPlan() {
    const currentTimeline = timelineRef.current ?? timeline;
    if (!currentTimeline || !commandText.trim()) return;
    setCommandBusy(true); setError(''); setNotice('');
    try {
      const plan = await callCapability<any>('media.workbench.timeline.command.plan', {workspace_root: workspace, timeline: currentTimeline, command_text: commandText.trim()});
      if (plan?.state !== 'ready' || !plan?.preview) { setError((plan?.blockers ?? []).join('；') || '没有可执行的精剪命令'); return; }
      const diff = await callCapability<any>('media.workbench.timeline.diff', {base_timeline: currentTimeline, target_timeline: plan.preview});
      setPendingCommands({commands: plan.commands ?? [], diff, label: plan.formatted || 'AI 精修'});
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setCommandBusy(false); }
  }
  function applyPendingCommands(selected: Set<string>) {
    if (!pendingCommands) return;
    const commands = pendingCommands.commands.filter((command) => {
      const clip = command.clip as any;
      const id = String(command.clip_id ?? clip?.id ?? '');
      return !id || selected.has(id);
    });
    setPendingCommands(null);
    if (commands.length) void apply(commands);
  }
  async function compareHistory(entry: HistoryEntry) {
    if (!timeline) return;
    try {
      const diff = await callCapability<any>('media.workbench.timeline.diff', {base_timeline: entry.timeline, target_timeline: timeline});
      setHistoryDiff(diff);
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
  }
  function restoreHistory(entry: HistoryEntry) {
    if (!timeline) return;
    replaceTimeline(entry.timeline, `已恢复 r${entry.revision} 的画布快照`);
  }
  function insertArtifactAt(trackId: string, frame: number, artifactId?: string) {
    const artifact = artifactId ? artifacts.find((item) => item.artifact_id === artifactId) : undefined;
    if (artifact) insertArtifact(artifact, trackId, frame);
  }
  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      const target = event.target as HTMLElement | null;
      if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT' || target.isContentEditable)) return;
      const key = event.key.toLowerCase();
      const mod = event.metaKey || event.ctrlKey;
      if (mod && key === 'z') {
        event.preventDefault();
        if (event.shiftKey) redo(); else undo();
        return;
      }
      if (mod && key === 'y') { event.preventDefault(); redo(); return; }
      if (key === ' ' && !mod) { event.preventDefault(); setPlaying((value) => !value); return; }
      if (key === 'arrowleft') { event.preventDefault(); setPlayhead((frame) => Math.max(0, frame - (event.shiftKey ? 10 : 1))); return; }
      if (key === 'arrowright') { event.preventDefault(); setPlayhead((frame) => Math.min(duration - 1, frame + (event.shiftKey ? 10 : 1))); return; }
      if (key === 'home') { event.preventDefault(); setPlayhead(0); return; }
      if (key === 'end') { event.preventDefault(); setPlayhead(duration - 1); return; }
      if (key === 'escape') { selectClips([]); return; }
      if (!selectedClip || busy) return;
      if (key === 'b') { event.preventDefault(); void apply([{op: 'blade', track_id: selectedClip.track_id, clip_id: selectedClip.id, at_frame: selectedClip.start_frame + Math.floor(selectedClip.duration_frames / 2)}]); }
      else if (key === 'delete' && event.shiftKey) { event.preventDefault(); void batchApply('delete'); }
      else if (key === 'delete' || key === 'backspace') { event.preventDefault(); void batchApply('delete'); }
      else if (key === 'i') { event.preventDefault(); void apply([{op: 'insert', track_id: selectedClip.track_id, at_frame: playhead, clip: {...selectedClip, id: selectedClip.id + '-i' + playhead}}]); }
      else if (key === 'o') { event.preventDefault(); void apply([{op: 'overwrite', track_id: selectedClip.track_id, at_frame: playhead, clip: {...selectedClip, id: selectedClip.id + '-o' + playhead}}]); }
      else if (key === 'n') { event.preventDefault(); setSnap((value) => !value); }
      else if (key === '?') { event.preventDefault(); setShortcutsOpen(true); }
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [timeline, undoStack, redoStack, selectedClip, selectedClips, selectedIds, busy, playhead, duration]);
  useEffect(() => {
    if (!selectedClip) return;
    setSourceIn(selectedClip.source_in_frame ?? 0);
    setSourceOut(selectedClip.source_out_frame ?? selectedClip.duration_frames);
  }, [selectedClip?.id]);
  useEffect(() => {
    let url = '';
    let cancelled = false;
    async function loadPreview() {
      const artifact = artifacts.find((item) => item.artifact_id === selectedClip?.media_id);
      if (!artifact) { setMediaUrl(''); setMediaError(''); return; }
      try {
        const result = await callCapability<any>('media.workbench.artifact.read', {workspace_root: workspace, artifact_id: artifact.artifact_id});
        if (cancelled || !result?.base64) return;
        const bytes = Uint8Array.from(atob(result.base64), (char) => char.charCodeAt(0));
        const mime = artifact.kind.startsWith('image/') ? 'image/png' : 'video/mp4';
        url = URL.createObjectURL(new Blob([bytes], {type: mime}));
        setMediaUrl(url); setMediaError('');
      } catch (err) {
        if (!cancelled) setMediaError(err instanceof Error ? err.message : String(err));
      }
    }
    void loadPreview();
    return () => { cancelled = true; if (url) URL.revokeObjectURL(url); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedClip?.media_id, workspace, artifacts]);

  if (!timeline) {
    return (
      <div className="timeline-editor flex h-full flex-col" data-mode={mode} data-media-open={mediaPanelOpen} data-inspector-open={inspectorPanelOpen} style={{background: 'var(--ink-950)', '--media-panel-width': `${mediaPanelWidth}px`, '--inspector-panel-width': `${inspectorPanelWidth}px`} as React.CSSProperties}>
        <div className="flex items-center gap-2 px-3 py-2" style={{borderBottom: '1px solid var(--line)', background: 'var(--ink-900)'}}>
          <span className="text-[12px] font-semibold">剪辑</span>
          <span className="badge">还没有时间轴</span>
          <span className="ml-auto text-[10px]" style={{color: 'var(--fg-dim)'}}>从项目产物开始，不需要手写 Timeline</span>
        </div>
        <div className="flex flex-1 items-center justify-center p-6">
          <div className="w-[720px] rounded-xl border p-5" style={{borderColor: 'var(--line)', background: 'var(--ink-900)'}}>
            <div className="flex items-center gap-2">
              <span className="badge badge-accent">简易模式</span>
              <strong className="text-[15px]">从素材一键成片</strong>
            </div>
            <div className="mt-3 flex flex-wrap items-center gap-2 text-[10.5px]" style={{color: 'var(--fg-muted)'}}>
              <label>目标时长 <input className="field ml-1 w-16" type="number" min={3} max={180} value={targetDuration} onChange={(event) => setTargetDuration(Number(event.target.value) || 15)} aria-label="目标时长" /> 秒</label>
              <span>· {videoArtifacts.length} 个视频产物</span>
            </div>
            <div className="mt-3 flex flex-wrap gap-1.5">
              {videoArtifacts.slice(0, 12).map((artifact) => {
                const checked = selectedMedia.includes(artifact.artifact_id);
                return <label key={artifact.artifact_id} className="badge cursor-pointer" style={{borderColor: checked ? 'var(--accent)' : 'var(--line)', color: checked ? 'var(--accent)' : 'var(--fg-muted)'}}><input type="checkbox" className="mr-1" checked={checked} onChange={() => setSelectedMedia((current) => current.includes(artifact.artifact_id) ? current.filter((id) => id !== artifact.artifact_id) : [...current, artifact.artifact_id])} />{artifact.artifact_id.slice(-10)}</label>;
              })}
              {!videoArtifacts.length ? <span className="text-[10.5px]" style={{color: 'var(--fg-dim)'}}>项目还没有视频产物；先在创作/审阅区生成视频，或通过媒体池导入。</span> : null}
            </div>
            <div className="mt-4 flex gap-2">
              <button type="button" className="btn btn-primary" disabled={busy || !videoArtifacts.length} onClick={() => void autoEdit()}>生成可精修时间轴</button>
              <button type="button" className="btn" disabled={busy} onClick={() => void attachSubtitles()}>只挂载项目字幕</button>
            </div>
            {error ? <div className="mt-3 text-[10.5px]" style={{color: 'var(--err)'}}>{error}</div> : null}
            {notice ? <div className="mt-3 text-[10.5px]" style={{color: 'var(--ok)'}}>{notice}</div> : null}
          </div>
        </div>
      </div>
    );
  }

  function toggleSidePanel(panel: 'media' | 'inspector' | 'transcript' | 'history' | 'asset' | 'semantic', next: boolean) {
    const narrow = typeof window !== 'undefined' && window.matchMedia('(max-width: 1199px)').matches;
    if (next && narrow) {
      setMediaPanelOpen(panel === 'media');
      setInspectorPanelOpen(panel === 'inspector');
      setTranscriptOpen(panel === 'transcript');
      setHistoryOpen(panel === 'history');
      setAssetOpen(panel === 'asset');
      setSemanticOpen(panel === 'semantic');
      return;
    }
    if (panel === 'media') setMediaPanelOpen(next);
    else if (panel === 'inspector') setInspectorPanelOpen(next);
    else if (panel === 'transcript') setTranscriptOpen(next);
    else if (panel === 'history') setHistoryOpen(next);
    else if (panel === 'asset') setAssetOpen(next);
    else setSemanticOpen(next);
  }
  return (
    <div className="timeline-editor flex h-full flex-col" data-mode={mode} data-media-open={mediaPanelOpen} data-inspector-open={inspectorPanelOpen} style={{background: 'var(--ink-950)', '--media-panel-width': `${mediaPanelWidth}px`, '--inspector-panel-width': `${inspectorPanelWidth}px`} as React.CSSProperties}>
      <div className="timeline-editor-header">
        <div className="timeline-title-block">
          <div>
            <div className="timeline-kicker">EDIT WORKSPACE</div>
            <div className="timeline-title">剪辑工作台</div>
          </div>
          <div className="timeline-meta">
            <span>r{timeline.revision}</span>
            <span>{timeline.timebase.num}/{timeline.timebase.den} fps</span>
            <span>{timeline.canvas.width}×{timeline.canvas.height}</span>
            {selectedIds.length > 1 ? <span className="timeline-meta-selected">{selectedIds.length} selected</span> : selectedClip ? <span className="timeline-meta-selected">{selectedClip.id}</span> : null}
          </div>
        </div>
        <div className="timeline-mode-switch" role="tablist" aria-label="剪辑模式">
          {([['easy', '简易'], ['pro', '专业'], ['deliver', '交付']] as const).map(([value, label]) => (
            <button key={value} type="button" data-testid={`timeline-mode-${value}`} className="timeline-mode-item" data-active={mode === value} onClick={() => setMode(value)}>{label}</button>
          ))}
        </div>
        <div className="timeline-header-actions">
          <button type="button" data-testid="timeline-media-toggle" className="btn btn-ghost" data-active={mediaPanelOpen} onClick={() => toggleSidePanel('media', !mediaPanelOpen)}><PanelLeft size={13} />媒体</button>
          <button type="button" data-testid="timeline-inspector-toggle" className="btn btn-ghost" data-active={inspectorPanelOpen} onClick={() => toggleSidePanel('inspector', !inspectorPanelOpen)}><SlidersHorizontal size={13} />属性</button>
          <button type="button" className="btn btn-ghost" data-active={transcriptOpen} onClick={() => toggleSidePanel('transcript', !transcriptOpen)}><MessageSquareText size={13} />文稿</button>
          <button type="button" className="btn btn-ghost" data-active={historyOpen} onClick={() => toggleSidePanel('history', !historyOpen)}><HistoryIcon size={13} />历史</button>
          <button type="button" className="btn btn-ghost" data-active={assetOpen} onClick={() => toggleSidePanel('asset', !assetOpen)}><Library size={13} />市场</button>
          <button type="button" className="btn btn-ghost" data-active={semanticOpen} onClick={() => { setSemanticQuery(searchQuery); toggleSidePanel('semantic', true); }}><Sparkles size={13} />AI 找</button>
          <span className="timeline-action-divider" />
          <button type="button" data-testid="timeline-undo" className="btn btn-ghost" disabled={!undoStack.length || busy} onClick={undo} title="撤销 Ctrl+Z">撤销</button>
          <button type="button" data-testid="timeline-redo" className="btn btn-ghost" disabled={!redoStack.length || busy} onClick={redo} title="重做 Ctrl+Shift+Z">重做</button>
          <button type="button" className="btn btn-ghost" onClick={() => void loadComments()} disabled={busy}>批注</button>
          <button type="button" data-testid="timeline-shortcuts" className="icon-btn" onClick={() => setShortcutsOpen((value) => !value)} title="快捷键（?）">?</button>
          <span className="timeline-action-divider" />
          <button type="button" data-testid="timeline-save" className="btn" disabled={busy} onClick={() => void save()}>保存</button>
          <button type="button" className={mode === 'deliver' ? 'btn' : 'btn btn-primary'} disabled={busy} onClick={() => void renderTimeline()}>渲染</button>
        </div>
      </div>
      {error || notice ? <div className="timeline-toast-stack">{error ? <div className="timeline-toast timeline-toast-error">{error}</div> : null}{notice ? <div className="timeline-toast timeline-toast-ok">{notice}</div> : null}</div> : null}
      {mode === 'easy' ? (
        <div className="timeline-commandbar timeline-commandbar-easy" style={{borderBottom: '1px solid var(--line)'}}>
          <span className="badge badge-accent">一键成片</span>
          <label className="flex items-center gap-1 text-[10.5px]" style={{color: 'var(--fg-muted)'}}>目标时长
            <input className="field w-16" type="number" min={3} max={180} value={targetDuration} onChange={(event) => setTargetDuration(Number(event.target.value) || 15)} aria-label="目标时长" />
            秒
          </label>
          <span className="text-[10px]" style={{color: 'var(--fg-dim)'}}>{selectedMedia.length || Math.min(3, videoArtifacts.length)} 段素材</span>
          <button type="button" className="btn" disabled={busy || !videoArtifacts.length} onClick={() => void autoEdit()}>生成时间轴</button>
          <button type="button" className="btn" disabled={busy} onClick={() => void attachSubtitles()}>智能字幕</button>
          <button type="button" className="btn" disabled={busy} onClick={() => void runQA()}>质量检查</button>
          <button type="button" className="btn" disabled={proxyBusy} onClick={() => void buildProxies()}>{proxyBusy ? '生成代理…' : '生成代理'}</button>
          <button type="button" className="btn" disabled={busy} onClick={() => void removeSilence()}>去静音</button>
          <input className="field w-36" value={beatPath} onChange={(event) => setBeatPath(event.target.value)} placeholder="BGM 路径" aria-label="卡点音频" />
          <button type="button" className="btn" disabled={busy || !beatPath.trim()} onClick={() => void addBeatMarkers()}>卡点标记</button>
          <button type="button" className="btn" disabled={busy} onClick={() => void inspectMedia()}>探测素材</button>
          <span style={{width: 1, height: 18, background: 'var(--line)'}} />
          <select className="field" value={templateId} onChange={(event) => setTemplateId(event.target.value)} aria-label="剪辑模板">
            <option value="hook-proof-payoff">钩子-论证-收束</option><option value="six-beat-story">六拍故事</option><option value="explainer-five">五段式讲解</option>
          </select>
          <button type="button" className="btn" disabled={busy || !videoArtifacts.length} onClick={() => void applyTemplate()}>套模板</button>
          <input className="field w-32" value={searchQuery} onChange={(event) => setSearchQuery(event.target.value)} placeholder="搜索转写" aria-label="搜索转写" />
          <button type="button" className="btn" disabled={busy} onClick={() => void searchTranscript()}>查文本</button>
          <button type="button" className="btn" disabled={roughCutBusy || busy} onClick={() => void roughCut()}>{roughCutBusy ? '粗剪中…' : 'AI 粗剪命中'}</button>
          {searchHits.length ? <span className="badge">{searchHits.length} 命中</span> : null}
          <div className="media-choice-strip ml-auto flex gap-1">
            {videoArtifacts.slice(0, 8).map((artifact) => {
              const checked = selectedMedia.includes(artifact.artifact_id);
              return <label key={artifact.artifact_id} className="badge cursor-pointer" style={{borderColor: checked ? 'var(--accent)' : 'var(--line)', color: checked ? 'var(--accent)' : 'var(--fg-muted)'}}>
                <input type="checkbox" className="mr-1" checked={checked} onChange={() => setSelectedMedia((current) => current.includes(artifact.artifact_id) ? current.filter((id) => id !== artifact.artifact_id) : [...current, artifact.artifact_id])} />
                {artifact.artifact_id.slice(-8)}
              </label>;
            })}
          </div>
        </div>
      ) : null}
      {mode === 'deliver' ? (
        <div className="timeline-commandbar timeline-commandbar-deliver" style={{borderBottom: '1px solid var(--line)'}}>
          <span className="badge badge-accent">交付</span>
          <select className="field" value={deliveryPreset} onChange={(event) => setDeliveryPreset(event.target.value)} aria-label="交付预设">
            <option value="master">母版</option><option value="web">Web</option><option value="douyin">抖音竖屏</option><option value="bilibili">B 站</option><option value="youtube">YouTube</option><option value="xigua">西瓜视频</option>
          </select>
          <label className="flex items-center gap-1 text-[10.5px]" style={{color: 'var(--fg-muted)'}}><input type="checkbox" checked={burnSubtitles} onChange={(event) => setBurnSubtitles(event.target.checked)} /> 烧录字幕</label>
          <button type="button" className="btn" disabled={busy} onClick={() => void runQA()}>渲染前 QA</button>
          <button type="button" className="btn btn-primary" disabled={busy} onClick={() => void renderTimeline()}>按预设渲染并 QC</button>
          {qaResult ? <span className="badge" style={{color: qaResult.passed ? 'var(--ok)' : 'var(--err)'}}>{qaResult.passed ? 'QC passed' : 'QC blocked'} · {qaResult.checks?.length ?? 0} 项</span> : null}
          {renderResult?.artifact?.path ? <span className="font-mono text-[9.5px] truncate" style={{color: 'var(--fg-dim)'}}>{renderResult.artifact.path}</span> : null}
        </div>
      ) : null}
      {mode === 'pro' ? (
        <div className="timeline-commandbar timeline-commandbar-pro" style={{borderBottom: '1px solid var(--line)'}}>
          <div className="command-group command-group-ai"><span className="command-group-label">AI 精修</span>
            <input className="field w-56" value={commandText} onChange={(event) => setCommandText(event.target.value)} onKeyDown={(event) => { if (event.key === 'Enter') void previewCommandPlan(); }} placeholder="例如：把第 1 段缩短到 4 秒" aria-label="AI 精修指令" />
            <button type="button" className="btn" data-testid="timeline-ai-preview" disabled={commandBusy || busy || !commandText.trim()} onClick={() => void previewCommandPlan()}>{commandBusy ? '分析中…' : '预览修改'}</button>
          </div>
          {selectedClip ? (
            <>
              <div className="command-group"><span className="command-group-label">编辑</span>
                <button type="button" data-testid="timeline-blade" className="btn" disabled={busy} onClick={() => void apply([{op: 'blade', track_id: selectedClip.track_id, clip_id: selectedClip.id, at_frame: selectedClip.start_frame + Math.floor(selectedClip.duration_frames / 2)}])}>切开</button>
                <button type="button" data-testid="timeline-lift" className="btn" disabled={busy} onClick={() => void apply([{op: 'lift', track_id: selectedClip.track_id, clip_id: selectedClip.id}])}>删除</button>
                <button type="button" data-testid="timeline-ripple-delete" className="btn" disabled={busy} onClick={() => void apply([{op: 'ripple_delete', track_id: selectedClip.track_id, clip_id: selectedClip.id}])}>波纹删除</button>
                <button type="button" data-testid="timeline-insert" className="btn" disabled={busy} onClick={() => void apply([{op: 'insert', track_id: selectedClip.track_id, at_frame: playhead, clip: {...selectedClip, id: selectedClip.id + '-i' + playhead}}])}>插入</button>
                <button type="button" data-testid="timeline-overwrite" className="btn" disabled={busy} onClick={() => void apply([{op: 'overwrite', track_id: selectedClip.track_id, at_frame: playhead, clip: {...selectedClip, id: selectedClip.id + '-o' + playhead}}])}>覆盖</button>
              </div>
              <div className="command-group"><span className="command-group-label">修剪</span>
                <button type="button" className="btn" disabled={busy || selectedClip.duration_frames <= 16} onClick={() => void apply([{op: 'ripple_trim', track_id: selectedClip.track_id, clip_id: selectedClip.id, duration_frames: selectedClip.duration_frames - 15}])}>−0.5s</button>
                <button type="button" className="btn" disabled={busy} onClick={() => void apply([{op: 'ripple_trim', track_id: selectedClip.track_id, clip_id: selectedClip.id, duration_frames: selectedClip.duration_frames + 15}])}>+0.5s</button>
                <button type="button" className="btn" disabled={busy} onClick={() => void apply([{op: 'roll', track_id: selectedClip.track_id, clip_id: selectedClip.id, delta_frames: -10}])}>Roll ←</button>
                <button type="button" className="btn" disabled={busy} onClick={() => void apply([{op: 'roll', track_id: selectedClip.track_id, clip_id: selectedClip.id, delta_frames: 10}])}>Roll →</button>
                <button type="button" className="btn" disabled={busy} onClick={() => void apply([{op: 'slip', track_id: selectedClip.track_id, clip_id: selectedClip.id, delta_frames: -10}])}>Slip ←</button>
                <button type="button" className="btn" disabled={busy} onClick={() => void apply([{op: 'slip', track_id: selectedClip.track_id, clip_id: selectedClip.id, delta_frames: 10}])}>Slip →</button>
              </div>
              <div className="command-group"><span className="command-group-label">速度</span>
                <button type="button" className="btn" disabled={busy} onClick={() => void apply([{op: 'retime', track_id: selectedClip.track_id, clip_id: selectedClip.id, speed: 0.5, ripple: true}])}>0.5×</button>
                <button type="button" className="btn" disabled={busy} onClick={() => void apply([{op: 'retime', track_id: selectedClip.track_id, clip_id: selectedClip.id, speed: 1, ripple: true}])}>1×</button>
                <button type="button" className="btn" disabled={busy} onClick={() => void apply([{op: 'retime', track_id: selectedClip.track_id, clip_id: selectedClip.id, speed: 2, ripple: true}])}>2×</button>
                <button type="button" className="btn" disabled={busy} onClick={() => void applySpeedCurve()}>速度曲线</button>
              </div>
              <div className="command-group"><span className="command-group-label">构图</span>
                <select className="field" value={aspect} onChange={(event) => setAspect(event.target.value)} aria-label="画布比例"><option value="9:16">9:16</option><option value="1:1">1:1</option><option value="16:9">16:9</option></select>
                <button type="button" className="btn" disabled={busy} onClick={() => void reframeTimeline()}>重构画布</button>
                <button type="button" className="btn" disabled={busy} onClick={() => void autoReframeTimeline()}>内容感知重构</button>
                <button type="button" className="btn" disabled={busy} onClick={() => updateSelected({transform: {scale: 1, fit: 'cover', position_x: 0, position_y: 0}})}>填充</button>
                <button type="button" className="btn" disabled={busy} onClick={() => updateSelected({transform: {scale: 1, fit: 'contain', position_x: 0, position_y: 0}})}>入画</button>
              </div>
              <div className="command-group"><span className="command-group-label">调色</span>
                <button type="button" className="btn" disabled={busy} onClick={() => updateSelected({color: {brightness: 0.02, contrast: 1.05, saturation: 1.12, gamma: 1.03}})}>暖调</button>
                <button type="button" className="btn" disabled={busy} onClick={() => updateSelected({color: {brightness: -0.01, contrast: 1.08, saturation: 0.92, gamma: 1}})}>冷调</button>
                <button type="button" className="btn" disabled={busy} onClick={() => updateSelected({color: {keyframes: [{frame: 0, brightness: -0.04, contrast: 1, saturation: 1, gamma: 1}, {frame: Math.floor(selectedClip.duration_frames / 2), brightness: 0.08, contrast: 1.06, saturation: 1.12, gamma: 1.02}, {frame: selectedClip.duration_frames, brightness: -0.04, contrast: 1, saturation: 1, gamma: 1}]}})}>呼吸调色</button>
                <button type="button" className="btn" disabled={busy} onClick={() => updateSelected({color: undefined})}>还原</button>
              </div>
              <div className="command-group"><span className="command-group-label">转场</span>
                <button type="button" className="btn" disabled={busy} onClick={() => updateSelected({transition_in: {type: 'fade', duration_frames: 10}})}>淡入</button>
                <button type="button" className="btn" disabled={busy} onClick={() => updateSelected({transition_out: {type: 'fade', duration_frames: 10}})}>淡出</button>
                <button type="button" className="btn" disabled={busy} onClick={() => void addPictureInPicture()}>画中画</button>
              </div>
              <div className="command-group"><span className="command-group-label">联动</span>
                <button type="button" className="btn" disabled={busy} onClick={linkSelected}>联动 A/V</button>
                <button type="button" className="btn" disabled={busy} onClick={() => void apply([{op: 'unlink', clip_ids: [selectedClip.id]}])}>解除</button>
              </div>
            </>
          ) : <div className="command-empty">在时间轴中选择一个片段，进入精修。</div>}
        </div>
      ) : null}
      {shortcutsOpen ? (
        <div className="fixed inset-0 z-30 flex items-center justify-center" style={{background: 'rgba(0,0,0,.42)'}} onClick={() => setShortcutsOpen(false)}>
          <div className="w-[420px] rounded-xl border p-4" style={{background: 'var(--ink-900)', borderColor: 'var(--line-strong)', boxShadow: 'var(--shadow)'}} onClick={(event) => event.stopPropagation()} role="dialog" aria-label="时间轴快捷键">
            <div className="flex items-center justify-between">
              <strong className="text-[13px]">时间轴快捷键</strong>
              <button type="button" className="icon-btn" aria-label="关闭快捷键" onClick={() => setShortcutsOpen(false)}>×</button>
            </div>
            <div className="mt-3 grid grid-cols-[1fr_auto] gap-x-4 gap-y-1.5 text-[11px]">
              <span>撤销 / 重做</span><kbd>Ctrl Z / Ctrl Shift Z</kbd>
              <span>切开选中片段</span><kbd>B</kbd>
              <span>删除 / 波纹删除</span><kbd>Delete / Shift Delete</kbd>
              <span>插入 / 覆盖</span><kbd>I / O</kbd>
              <span>吸附开关</span><kbd>N</kbd>
              <span>打开快捷键表</span><kbd>?</kbd>
            </div>
            <div className="mt-3 text-[10px]" style={{color: 'var(--fg-dim)'}}>快捷键作用于当前选中的 clip；输入框中不会触发。</div>
          </div>
        </div>
      ) : null}
      {commentsOpen ? (
        <div className="fixed inset-0 z-30 flex items-center justify-center" style={{background: 'rgba(0,0,0,.42)'}} onClick={() => setCommentsOpen(false)}>
          <div className="w-[520px] max-h-[70vh] overflow-auto rounded-xl border p-4" style={{background: 'var(--ink-900)', borderColor: 'var(--line-strong)', boxShadow: 'var(--shadow)'}} onClick={(event) => event.stopPropagation()} role="dialog" aria-label="时间轴批注">
            <div className="flex items-center justify-between">
              <strong className="text-[13px]">时间轴批注</strong>
              <button type="button" className="icon-btn" aria-label="关闭批注" onClick={() => setCommentsOpen(false)}>×</button>
            </div>
            <div className="mt-3 flex gap-2">
              <input className="field flex-1" value={commentText} onChange={(event) => setCommentText(event.target.value)} placeholder={`在播放头 ${(playhead / (timeline?.timebase.num ?? 30)).toFixed(1)}s 添加批注`} aria-label="批注内容" />
              <button type="button" className="btn" disabled={busy || !commentText.trim()} onClick={() => void addComment()}>添加</button>
            </div>
            <div className="mt-3 space-y-2">
              {comments.length ? comments.map((comment: any) => (
                <div key={comment.id} className="flex items-start gap-2 rounded-md border px-2.5 py-2" style={{borderColor: 'var(--line)', opacity: comment.status === 'resolved' ? 0.55 : 1}}>
                  <span className="badge badge-accent">{comment.timecode}</span>
                  <div className="min-w-0 flex-1 text-[10.5px]"><div>{comment.text}</div><div className="mt-0.5 text-[9px]" style={{color: 'var(--fg-dim)'}}>{comment.category} · {comment.status}</div></div>
                  {comment.status !== 'resolved' ? <button type="button" className="btn h-6" onClick={() => void resolveComment(comment.id)}>解决</button> : null}
                </div>
              )) : <div className="text-[10.5px]" style={{color: 'var(--fg-dim)'}}>还没有批注。拖动播放头到需要讨论的位置再添加。</div>}
            </div>
          </div>
        </div>
      ) : null}
      <TimelineAudioMixer timeline={timeline} workspace={workspace} playhead={playhead} playing={playing} active={active} onLevels={setAudioLevels} />
      <div className="timeline-viewers">
        <div className="timeline-source-monitor">
          <div className="timeline-viewer-head"><span className="timeline-viewer-kicker">SOURCE</span><strong>源监视器</strong></div>
          {selectedClip ? (
            <div className="space-y-0.5 text-[10px]">
              <div className="font-mono">{selectedClip.id}</div>
              <div style={{color: 'var(--fg-muted)'}}>media {selectedClip.media_id}</div>
              <div style={{color: 'var(--fg-muted)'}}>source {selectedClip.source_in_frame}–{selectedClip.source_out_frame}f · {selectedClip.duration_frames}f</div>
              <div className="flex items-center gap-1 pt-1">
                <input className="field w-14" type="number" min={0} value={sourceIn} onChange={(event) => setSourceIn(Number(event.target.value) || 0)} aria-label="源入点" />
                <span style={{color: 'var(--fg-dim)'}}>–</span>
                <input className="field w-14" type="number" min={1} value={sourceOut} onChange={(event) => setSourceOut(Number(event.target.value) || 1)} aria-label="源出点" />
                <button type="button" className="btn h-6" disabled={busy} onClick={insertWithRange}>三点插入</button>
              </div>
              {mediaUrl ? (selectedClip.media_id?.startsWith('video') || artifacts.find((item) => item.artifact_id === selectedClip.media_id)?.kind.startsWith('video/')
                ? <video src={mediaUrl} controls muted style={{maxHeight: 160, width: '100%', marginTop: 6}} />
                : <img src={mediaUrl} alt="source preview" style={{maxHeight: 160, maxWidth: '100%', marginTop: 6}} />) : null}
              {mediaError ? <div style={{color: 'var(--err)'}}>{mediaError}</div> : null}
            </div>
          ) : <div className="timeline-viewer-empty">选中一个片段查看源信息</div>}
        </div>
        <ProgramMonitor timeline={timeline} artifacts={artifacts as Artifact[]} workspace={workspace} playhead={playhead} onPlayhead={setPlayhead} playing={playing} onPlaying={setPlaying} height={188} />
      </div>
      <div className="timeline-media-shelf" style={{borderBottom: '1px solid var(--line)'}}>
        <span className="text-[10px] uppercase tracking-wide" style={{color: 'var(--fg-dim)'}}>媒体池</span>
        {artifacts.slice(0, 10).map((artifact) => (
          <button key={artifact.artifact_id} type="button" draggable className="badge" style={{cursor: 'grab', maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis'}} disabled={busy} title={artifact.kind + ' · ' + artifact.artifact_id} onDragStart={(event) => { event.dataTransfer.setData('application/x-himind-artifact', artifact.artifact_id); event.dataTransfer.effectAllowed = 'copy'; }} onClick={() => insertArtifact(artifact)}>
            {artifact.kind} · {artifact.artifact_id.slice(-8)} +
          </button>
        ))}
        <span className="text-[9.5px]" style={{color: 'var(--fg-dim)'}}>{artifacts.length} 个产物</span>
      </div>
      <div className="timeline-height-handle" onPointerDown={(event) => beginPanelResize('timeline', event)} title="拖动调整时间轴高度" />
      <div className="timeline-canvas-host" style={timelineHeight ? {flex: `0 0 ${timelineHeight}px`} : {flex: '1 1 auto', minHeight: 190}}>
        <TimelineCanvas
          timeline={timeline}
          artifacts={artifacts as Artifact[]}
          workspace={workspace}
          playhead={playhead}
          onPlayhead={(frame) => { setPlaying(false); setPlayhead(frame); }}
          selectedIds={selectedIds.length ? selectedIds : selectedClip ? [selectedClip.id] : []}
          onSelect={selectClips}
          tool={tool}
          onTool={setTool}
          snap={snap}
          onSnap={setSnap}
          onApply={apply}
          onInsertAt={insertArtifactAt}
          diff={pendingCommands?.diff ?? historyDiff}
          audioLevels={audioLevels}
          onToast={(message) => setNotice(message)}
        />
      </div>
      {mediaPanelOpen || inspectorPanelOpen || historyOpen || transcriptOpen || assetOpen || semanticOpen ? <div className="timeline-panel-backdrop" onClick={() => { setMediaPanelOpen(false); setInspectorPanelOpen(false); setHistoryOpen(false); setTranscriptOpen(false); setAssetOpen(false); setSemanticOpen(false); }} /> : null}
      {mediaPanelOpen ? <MediaLibraryPanel artifacts={artifacts as Artifact[]} workspace={workspace} onClose={() => setMediaPanelOpen(false)} onInsert={(artifact) => insertArtifact(artifact)} onDragStart={(event, artifact) => { event.dataTransfer.setData('application/x-himind-artifact', artifact.artifact_id); event.dataTransfer.effectAllowed = 'copy'; }} onSemantic={(query) => { setSemanticQuery(query); setSemanticOpen(true); }} /> : null}
      {inspectorPanelOpen ? <ClipInspectorPanel clip={selectedClip} timeline={timeline} selectedIds={selectedIds.length ? selectedIds : selectedClip ? [selectedClip.id] : []} workspace={workspace} playhead={playhead} onClose={() => setInspectorPanelOpen(false)} onUpdate={updateSelected} onMove={(atFrame) => { if (selectedClip) void apply([{op: 'move', track_id: selectedClip.track_id, clip_id: selectedClip.id, at_frame: Math.max(0, Math.round(atFrame))}]); }} onTrim={(edge, delta) => { if (selectedClip && delta !== 0) void apply([{op: 'trim_edge', track_id: selectedClip.track_id, clip_id: selectedClip.id, edge, delta_frames: Math.round(delta)}]); }} onRetime={(speed) => { if (selectedClip) void apply([{op: 'retime', track_id: selectedClip.track_id, clip_id: selectedClip.id, speed, ripple: true}]); }} onSpeedCurve={() => void applySpeedCurve()} onPip={() => void addPictureInPicture()} onDelete={() => { if (selectedClip) void apply([{op: 'lift', track_id: selectedClip.track_id, clip_id: selectedClip.id}]); }} onBatch={(action) => void batchApply(action)} onSeek={(frame) => { setPlaying(false); setPlayhead(frame); }} onApply={apply} /> : null}
      {historyOpen ? <HistoryPanel entries={historyEntries} currentRevision={timeline.revision} diff={historyDiff} onCompare={(entry) => void compareHistory(entry)} onRestore={restoreHistory} onClose={() => setHistoryOpen(false)} /> : null}
      {transcriptOpen ? <TranscriptPanel timeline={timeline} selectedIds={selectedIds.length ? selectedIds : selectedClip ? [selectedClip.id] : []} onClose={() => setTranscriptOpen(false)} onSeek={(frame) => { setPlaying(false); setPlayhead(frame); }} onSelect={selectClip} onUpdateText={(clip, text) => { void apply([{op: 'update_clip', track_id: clip.track_id, clip_id: clip.id, clip: {...clip, text}}]); }} /> : null}
      {assetOpen ? <AssetBrowserPanel workspace={workspace} onClose={() => setAssetOpen(false)} onApplyTemplate={(id) => { setTemplateId(id); setAssetOpen(false); void applyTemplate(id); }} onToast={(message) => setNotice(message)} /> : null}
      {semanticOpen ? <SemanticPanel workspace={workspace} timeline={timeline} mediaPaths={mediaPathMap()} initialQuery={semanticQuery} onClose={() => setSemanticOpen(false)} onSeek={(frame) => { setPlaying(false); setPlayhead(frame); }} onSelect={selectClip} /> : null}
      {pendingCommands ? <DiffPanel diff={pendingCommands.diff} onApply={applyPendingCommands} onDiscard={() => setPendingCommands(null)} onClose={() => setPendingCommands(null)} /> : null}
      {mediaPanelOpen ? <div className="panel-resize-handle media-panel-handle" onPointerDown={(event) => beginPanelResize('media', event)} /> : null}
      {inspectorPanelOpen ? <div className="panel-resize-handle inspector-panel-handle" onPointerDown={(event) => beginPanelResize('inspector', event)} /> : null}
      {missingMedia.length ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center" style={{background: 'rgba(0,0,0,.48)'}} onClick={() => setMissingMedia([])}>
          <div className="w-[560px] max-h-[80vh] overflow-auto rounded-xl border p-4" style={{background: 'var(--ink-900)', borderColor: 'var(--line-strong)', boxShadow: 'var(--shadow)'}} onClick={(event) => event.stopPropagation()} role="dialog" aria-label="渲染前预检">
            <div className="flex items-center justify-between">
              <strong className="text-[14px]">渲染前预检</strong>
              <button type="button" className="icon-btn" aria-label="关闭预检" onClick={() => setMissingMedia([])}>×</button>
            </div>
            <div className="mt-2 text-[12px]" style={{color: 'var(--fg-muted)'}}>有 {missingMedia.length} 个片段缺少媒体路径。请重新链接或跳过后再渲染。</div>
            <div className="mt-3 space-y-2">
              {missingMedia.map((id) => {
                const track = timeline?.tracks.find((item) => item.clips.some((clip) => clip.media_id === id));
                const candidates = artifacts.filter((artifact) => track?.kind === 'audio' ? artifact.kind.startsWith('audio/') : artifact.kind.startsWith('video/') || artifact.kind.startsWith('image/'));
                return (
                  <div key={id} className="rounded-md border p-2" style={{borderColor: 'var(--line)'}}>
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-[12px]">{id}</span>
                      <span className="badge">{track?.kind ?? 'media'}</span>
                    </div>
                    <div className="mt-1 flex items-center gap-2">
                      <select className="field flex-1" value={relinkMap[id] ?? ''} onChange={(event) => { const value = event.target.value; setRelinkMap((current) => ({...current, [id]: value})); if (value) setSkipMissing((current) => ({...current, [id]: false})); }}>
                        <option value="">选择项目素材重新链接</option>
                        {candidates.map((artifact) => <option key={artifact.artifact_id} value={artifact.artifact_id}>{artifact.artifact_id} · {artifact.kind}</option>)}
                      </select>
                      <label className="flex items-center gap-1 text-[11px]">
                        <input type="checkbox" checked={Boolean(skipMissing[id])} onChange={(event) => { const checked = event.target.checked; setSkipMissing((current) => ({...current, [id]: checked})); if (checked) setRelinkMap((current) => ({...current, [id]: ''})); }} />
                        跳过
                      </label>
                    </div>
                  </div>
                );
              })}
            </div>
            <div className="mt-2 text-[11px]" style={{color: 'var(--fg-dim)'}}>跳过只影响本次渲染，不修改已保存的时间轴。</div>
            <div className="mt-3 flex justify-end gap-2">
              <button type="button" className="btn" onClick={() => setMissingMedia([])}>取消</button>
              <button type="button" className="btn btn-primary" disabled={!missingMedia.every((id) => relinkMap[id] || skipMissing[id])} onClick={() => { if (timeline) void performRender(timeline, missingMedia); }}>继续渲染</button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
