import {useEffect, useMemo, useState} from 'react';
import {Check, ChevronRight, Download, FileAudio, FileImage, FileVideo, History, ListMusic, Package, Search, Sparkles, Trash2, Wand2, X} from 'lucide-react';
import {callCapability} from '../../api/capability';
import type {Artifact, HistoryEntry, SemanticHit, Timeline, TimelineClip, TimelineDiff, TimelineTrack} from './types';
import {useArtifactBlob, useMediaDuration} from './hooks';

function humanSize(size?: number) {
  if (!size || size <= 0) return '—';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}
function timecode(frame: number, fps: number) {
  const safeFps = Math.max(1, fps || 30);
  const total = Math.max(0, Math.round(frame));
  const ff = total % safeFps;
  const seconds = Math.floor(total / safeFps);
  return `${String(Math.floor(seconds / 60)).padStart(2, '0')}:${String(seconds % 60).padStart(2, '0')}:${String(ff).padStart(2, '0')}`;
}
function kindLabel(kind: string) {
  if (kind.startsWith('video/')) return '视频';
  if (kind.startsWith('image/')) return '图片';
  if (kind.startsWith('audio/')) return '音频';
  if (kind.includes('subtitle')) return '字幕';
  return kind.split('/')[0] || '文件';
}
function ArtifactIcon({kind}: {kind: string}) {
  if (kind.startsWith('audio/')) return <FileAudio size={12} />;
  if (kind.startsWith('image/')) return <FileImage size={12} />;
  return <FileVideo size={12} />;
}

function formatDuration(seconds: number) {
  if (!Number.isFinite(seconds) || seconds <= 0) return '';
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const whole = Math.round(seconds);
  return `${Math.floor(whole / 60)}:${String(whole % 60).padStart(2, '0')}`;
}
function MediaThumb({artifact, url}: {artifact: Artifact; url: string}) {
  if (artifact.kind.startsWith('image/') && url) return <img className="media-thumb" src={url} alt="" />;
  if (artifact.kind.startsWith('video/') && url) return <video className="media-thumb" src={url} muted preload="metadata" onLoadedMetadata={(event) => { try { event.currentTarget.currentTime = Math.min(0.15, event.currentTarget.duration || 0.15); } catch {} }} />;
  return <span className="media-thumb media-thumb-fallback"><ArtifactIcon kind={artifact.kind} /><em>{kindLabel(artifact.kind)}</em></span>;
}
function MediaLibraryItem({workspace, artifact, enabled, onInsert, onDragStart}: {
  workspace: string; artifact: Artifact; enabled: boolean;
  onInsert: (artifact: Artifact) => void;
  onDragStart: (event: React.DragEvent, artifact: Artifact) => void;
}) {
  const {url} = useArtifactBlob(workspace, artifact.artifact_id, enabled);
  const probeDuration = artifact.kind.startsWith('video/') || artifact.kind.startsWith('audio/');
  const duration = useMediaDuration(probeDuration ? url : '');
  const durationLabel = duration > 0 ? formatDuration(duration) : '';
  return <div className="media-library-item" draggable onDragStart={(event) => onDragStart(event, artifact)} onDoubleClick={() => onInsert(artifact)} tabIndex={0} role="button" aria-label={`插入 ${artifact.artifact_id}`} onKeyDown={(event) => { if (event.key === 'Enter') onInsert(artifact); }}>
    <MediaThumb artifact={artifact} url={url} />
    <div className="media-library-meta"><strong>{artifact.artifact_id}</strong><span>{kindLabel(artifact.kind)} · {durationLabel ? `${durationLabel} · ` : ''}{humanSize(artifact.size)}{artifact.quality_status ? ` · ${artifact.quality_status}` : ''}</span><span className="media-path">{artifact.path}</span></div>
    <button type="button" className="icon-btn" title="插入播放头" aria-label="插入媒体" onClick={() => onInsert(artifact)}>+</button>
  </div>;
}

export function MediaLibraryPanel({artifacts, workspace, onClose, onInsert, onDragStart, onSemantic}: {
  artifacts: Artifact[]; workspace: string; onClose: () => void; onInsert: (artifact: Artifact) => void;
  onDragStart: (event: React.DragEvent, artifact: Artifact) => void; onSemantic: (query: string) => void;
}) {
  const [query, setQuery] = useState('');
  const [filter, setFilter] = useState('all');
  const [sort, setSort] = useState<'recent' | 'name' | 'kind'>('recent');
  const filtered = useMemo(() => {
    const items = artifacts.filter((artifact) => {
      const type = artifact.kind.split('/')[0] || 'other';
      if (filter !== 'all' && type !== filter) return false;
      const haystack = `${artifact.artifact_id} ${artifact.kind} ${artifact.path}`.toLowerCase();
      return !query.trim() || haystack.includes(query.trim().toLowerCase());
    });
    return items.sort((left, right) => {
      if (sort === 'name') return left.artifact_id.localeCompare(right.artifact_id);
      if (sort === 'kind') return left.kind.localeCompare(right.kind);
      return String(right.created_at || '').localeCompare(String(left.created_at || ''));
    });
  }, [artifacts, filter, query, sort]);
  const groups = useMemo(() => {
    const map = new Map<string, Artifact[]>();
    for (const artifact of filtered) {
      const key = artifact.kind.split('/')[0] || 'other';
      map.set(key, [...(map.get(key) || []), artifact]);
    }
    return Array.from(map.entries());
  }, [filtered]);
  return <aside className="timeline-side-panel timeline-media-library" aria-label="媒体库">
    <div className="timeline-panel-head"><strong>媒体库</strong><span>{filtered.length}/{artifacts.length}</span><button type="button" className="icon-btn" onClick={onClose} aria-label="关闭媒体库"><X size={14} /></button></div>
    <div className="media-search"><Search size={13} /><input value={query} onChange={(event) => setQuery(event.target.value)} onKeyDown={(event) => { if (event.key === 'Enter' && query.trim()) onSemantic(query.trim()); }} placeholder="搜索素材，回车用 AI 语义定位" aria-label="搜索媒体" /></div>
    <div className="media-filter-tabs">{[['all', '全部'], ['video', '视频'], ['image', '图片'], ['audio', '音频']].map(([value, label]) => <button key={value} type="button" data-active={filter === value} onClick={() => setFilter(value)}>{label}</button>)}</div>
    <div className="media-sort-row"><span>排序</span><select className="field" value={sort} onChange={(event) => setSort(event.target.value as typeof sort)} aria-label="素材排序"><option value="recent">最近</option><option value="name">名称</option><option value="kind">类型</option></select></div>
    <div className="media-library-list">
      {groups.map(([kind, items]) => <section key={kind} className="media-group"><div className="media-group-head"><span>{kindLabel(kind + '/x')}</span><span>{items.length}</span></div>
        {items.map((artifact, index) => <MediaLibraryItem key={artifact.artifact_id} workspace={workspace} artifact={artifact} enabled={index < 12} onInsert={onInsert} onDragStart={onDragStart} />)}
      </section>)}
      {!filtered.length ? <div className="timeline-panel-empty">{query ? '没有匹配的素材；按回车尝试 AI 语义定位。' : '项目还没有素材。'}</div> : null}
    </div>
    <div className="media-library-foot"><Sparkles size={11} /> 双击或拖拽到轨道；回车按语义找镜头</div>
  </aside>;
}

function NumberField({label, value, onCommit, min, max, step = 0.01, suffix, testId}: {label: string; value: number; onCommit: (value: number) => void; min?: number; max?: number; step?: number; suffix?: string; testId?: string}) {
  return <label className="inspector-field"><span>{label}</span><div className="inspector-number"><input data-testid={testId} className="field" key={String(value)} type="number" defaultValue={Number.isFinite(value) ? value : 0} min={min} max={max} step={step} onBlur={(event) => { const next = Number(event.target.value); if (Number.isFinite(next)) onCommit(next); }} />{suffix ? <em>{suffix}</em> : null}</div></label>;
}

export function ClipInspectorPanel({clip, timeline, selectedIds, workspace, playhead, onClose, onUpdate, onMove, onTrim, onRetime, onSpeedCurve, onPip, onDelete, onBatch, onSeek, onApply}: {
  clip: TimelineClip | null; timeline: Timeline; selectedIds: string[]; workspace: string; playhead: number;
  onClose: () => void; onUpdate: (patch: Partial<TimelineClip>) => void;
  onMove: (atFrame: number) => void; onTrim: (edge: 'left' | 'right', delta: number) => void;
  onRetime: (speed: number) => void; onSpeedCurve: () => void; onPip: () => void; onDelete: () => void;
  onBatch: (action: 'delete' | 'retime-0.5' | 'retime-1' | 'retime-2' | 'warm' | 'cool') => void;
  onSeek: (frame: number) => void;
  onApply: (commands: Array<Record<string, unknown>>) => Promise<void> | void;
}) {
  const [tab, setTab] = useState('clip');
  if (!clip) return <aside className="timeline-side-panel timeline-inspector-panel" aria-label="片段属性"><div className="timeline-panel-head"><strong>属性</strong><button type="button" className="icon-btn" onClick={onClose} aria-label="关闭属性"><X size={14} /></button></div><div className="timeline-panel-empty">在时间轴中选择片段；支持多选后批量精修。</div></aside>;
  const track = timeline.tracks.find((item) => item.id === clip.track_id);
  const transform = clip.transform || {};
  const color = clip.color || {};
  const isSubtitle = track?.kind === 'subtitle';
  const localFrame = Math.max(0, Math.min(clip.duration_frames, Math.round(playhead - clip.start_frame)));
  const keyframes = color.keyframes || [];
  const multi = selectedIds.length > 1;
  return <aside className="timeline-side-panel timeline-inspector-panel" aria-label="片段属性">
    <div className="timeline-panel-head"><strong>属性</strong><span>{multi ? `${selectedIds.length} 个片段` : clip.id}</span><button type="button" className="icon-btn" onClick={onClose} aria-label="关闭属性"><X size={14} /></button></div>
    <div className="inspector-tabs" role="tablist" aria-label="属性分类">{([['clip', '片段'], ['transform', '变换'], ['speed', '速度'], ['audio', '音频'], ['color', '调色'], ['transition', '转场']] as const).map(([value, label]) => <button key={value} type="button" role="tab" data-testid={`inspector-tab-${value}`} data-active={tab === value} onClick={() => setTab(value)}>{label}</button>)}</div>
    <div className="inspector-scroll">
      {multi ? <div className="inspector-section inspector-multi"><div className="inspector-section-title">批量操作</div><div className="inspector-hint">已选择 {selectedIds.length} 个片段。批量命令会形成一次 revision，可 Ctrl+Z 撤销。</div><div className="inspector-button-row"><button type="button" className="btn" onClick={() => onBatch('retime-0.5')}>批量 0.5×</button><button type="button" className="btn" onClick={() => onBatch('retime-1')}>批量 1×</button><button type="button" className="btn" onClick={() => onBatch('retime-2')}>批量 2×</button><button type="button" className="btn" onClick={() => onBatch('warm')}>批量暖调</button><button type="button" className="btn" onClick={() => onBatch('cool')}>批量冷调</button><button type="button" className="btn" onClick={() => onBatch('delete')}>批量删除</button></div></div> : null}
      {tab === 'clip' ? <div className="inspector-section"><div className="inspector-section-title">片段</div><dl className="inspector-kv"><dt>轨道</dt><dd>{clip.track_id} · {track?.kind}</dd><dt>入点</dt><dd>{timecode(clip.start_frame, timeline.timebase.num)}</dd><dt>时长</dt><dd>{clip.duration_frames}f · {(clip.duration_frames / timeline.timebase.num).toFixed(2)}s</dd><dt>源区间</dt><dd>{clip.source_in_frame ?? 0}–{clip.source_out_frame ?? 0}f</dd><dt>媒体</dt><dd className="truncate">{clip.media_id || '文字轨道'}</dd><dt>速度</dt><dd>{clip.speed ?? 1}×</dd></dl>
        <div className="inspector-grid"><NumberField label="开始帧" value={clip.start_frame} step={1} onCommit={(value) => onMove(Math.max(0, Math.round(value)))} testId="inspector-start-frame" /><NumberField label="时长帧" value={clip.duration_frames} step={1} onCommit={(value) => { const next = Math.max(1, Math.round(value)); if (next !== clip.duration_frames) onTrim('right', next - clip.duration_frames); }} testId="inspector-duration-frame" /></div>
        {isSubtitle ? <label className="inspector-field inspector-field-block"><span>字幕文本</span><textarea className="field" defaultValue={clip.text || ''} onBlur={(event) => { if (event.target.value !== (clip.text || '')) onUpdate({text: event.target.value}); }} /></label> : null}
        <div className="inspector-actions"><button type="button" className="btn" onClick={onPip}>画中画</button><button type="button" className="btn" onClick={() => onSeek(clip.start_frame)}>定位播放头</button><button type="button" className="btn btn-danger" onClick={onDelete}>删除片段</button></div>
        {isSubtitle ? <div className="inspector-hint">字幕与文稿双向联动：在“文稿”页签改字，会写回同一片段。</div> : null}
      </div> : null}
      {tab === 'transform' ? <div className="inspector-section"><div className="inspector-section-title">变换</div><div className="inspector-grid"><NumberField label="缩放" value={Number(transform.scale ?? 1)} step={0.05} onCommit={(value) => onUpdate({transform: {...transform, scale: value}})} /><NumberField label="水平" value={Number(transform.position_x ?? 0)} onCommit={(value) => onUpdate({transform: {...transform, position_x: value}})} /><NumberField label="垂直" value={Number(transform.position_y ?? 0)} onCommit={(value) => onUpdate({transform: {...transform, position_y: value}})} /><NumberField label="旋转" value={Number(transform.rotation ?? 0)} suffix="°" onCommit={(value) => onUpdate({transform: {...transform, rotation: value}})} /><NumberField label="不透明度" value={Number(clip.opacity ?? 1)} min={0} max={1} step={0.05} onCommit={(value) => onUpdate({opacity: value})} /></div><label className="inspector-field inspector-field-block"><span>适配</span><select className="field" value={transform.fit || 'cover'} onChange={(event) => onUpdate({transform: {...transform, fit: event.target.value}})}><option value="cover">填充画面</option><option value="contain">完整入画</option></select></label><div className="inspector-button-row"><button type="button" className="btn" onClick={() => onUpdate({transform: {scale: 1, fit: 'cover', position_x: 0, position_y: 0}})}>九宫格填充</button><button type="button" className="btn" onClick={() => onUpdate({transform: {scale: 0.72, fit: 'contain', position_x: 0.14, position_y: 0.14}})}>右上画中画</button></div></div> : null}
      {tab === 'speed' ? <div className="inspector-section"><div className="inspector-section-title">速度</div><div className="inspector-button-row"><button type="button" className="btn" onClick={() => onRetime(0.5)}>0.5×</button><button type="button" className="btn" onClick={() => onRetime(1)}>1×</button><button type="button" className="btn" onClick={() => onRetime(2)}>2×</button><button type="button" className="btn" onClick={onSpeedCurve}>速度曲线</button></div><NumberField label="自定义速度" value={Number(clip.speed ?? 1)} min={0.25} max={4} step={0.05} suffix="×" onCommit={onRetime} /><div className="inspector-hint">变速会按源区间重算时长；开启 ripple 时后续片段跟随。</div></div> : null}
      {tab === 'audio' ? <div className="inspector-section"><div className="inspector-section-title">音频</div><NumberField label="音量" value={Number(clip.volume_db ?? 0)} min={-60} max={12} step={0.5} suffix="dB" onCommit={(value) => onUpdate({volume_db: value})} /><div className="inspector-button-row"><button type="button" className="btn" onClick={() => onUpdate({volume_db: -6})}>人声优先 −6</button><button type="button" className="btn" onClick={() => onUpdate({volume_db: -12})}>BGM 压低 −12</button><button type="button" className="btn" onClick={() => onUpdate({volume_db: 0})}>还原 0</button></div><div className="inspector-section-title">淡入淡出</div><div className="inspector-button-row"><button type="button" className="btn" onClick={() => onApply([{op: 'update_clip', track_id: clip.track_id, clip_id: clip.id, clip: {...clip, volume_db: Number(clip.volume_db ?? 0)}}])}>应用音量</button><button type="button" className="btn" onClick={() => onUpdate({transition_in: {type: 'fade', duration_frames: 12}})}>淡入 0.4s</button><button type="button" className="btn" onClick={() => onUpdate({transition_out: {type: 'fade', duration_frames: 12}})}>淡出 0.4s</button></div></div> : null}
      {tab === 'color' ? <div className="inspector-section"><div className="inspector-section-title">基础调色</div><div className="inspector-grid"><NumberField label="亮度" value={Number(color.brightness ?? 0)} min={-1} max={1} step={0.01} onCommit={(value) => onUpdate({color: {...color, brightness: value}})} /><NumberField label="对比" value={Number(color.contrast ?? 1)} step={0.01} onCommit={(value) => onUpdate({color: {...color, contrast: value}})} /><NumberField label="饱和" value={Number(color.saturation ?? 1)} step={0.01} onCommit={(value) => onUpdate({color: {...color, saturation: value}})} /><NumberField label="Gamma" value={Number(color.gamma ?? 1)} step={0.01} onCommit={(value) => onUpdate({color: {...color, gamma: value}})} /></div><label className="inspector-field inspector-field-block"><span>LUT</span><input className="field" defaultValue={color.lut || ''} placeholder="workspace relative .cube" onBlur={(event) => onUpdate({color: {...color, lut: event.target.value}})} /></label><div className="inspector-section-title">颜色关键帧 · {keyframes.length}</div><div className="inspector-button-row"><button type="button" className="btn" onClick={() => { const next = [...keyframes, {frame: localFrame, brightness: Number(color.brightness ?? 0), contrast: Number(color.contrast ?? 1), saturation: Number(color.saturation ?? 1), gamma: Number(color.gamma ?? 1)}].sort((a, b) => a.frame - b.frame); onUpdate({color: {...color, keyframes: next}}); }}>在播放头加关键帧</button><button type="button" className="btn" onClick={() => onUpdate({color: {...color, keyframes: [{frame: 0, brightness: -0.04, contrast: 1, saturation: 1, gamma: 1}, {frame: Math.floor(clip.duration_frames / 2), brightness: 0.08, contrast: 1.06, saturation: 1.12, gamma: 1.02}, {frame: clip.duration_frames, brightness: -0.04, contrast: 1, saturation: 1, gamma: 1}]}})}>呼吸曲线</button>{keyframes.length ? <button type="button" className="btn" onClick={() => onUpdate({color: {...color, keyframes: undefined}})}>清空关键帧</button> : null}</div>{keyframes.length ? <div className="inspector-keyframe-list">{keyframes.map((keyframe, index) => <button key={index} type="button" className="inspector-keyframe-row" onClick={() => onSeek(clip.start_frame + keyframe.frame)}><span>◆ {keyframe.frame}f</span><span>亮度 {(keyframe.brightness ?? 0).toFixed(2)}</span><ChevronRight size={11} /></button>)}</div> : null}</div> : null}
      {tab === 'transition' ? <div className="inspector-section"><div className="inspector-section-title">转场</div><dl className="inspector-kv"><dt>淡入</dt><dd>{clip.transition_in ? `${clip.transition_in.type} · ${clip.transition_in.duration_frames ?? 0}f` : '无'}</dd><dt>淡出</dt><dd>{clip.transition_out ? `${clip.transition_out.type} · ${clip.transition_out.duration_frames ?? 0}f` : '无'}</dd></dl><div className="inspector-button-row"><button type="button" className="btn" onClick={() => onUpdate({transition_in: {type: 'fade', duration_frames: 10}})}>淡入 10f</button><button type="button" className="btn" onClick={() => onUpdate({transition_out: {type: 'fade', duration_frames: 10}})}>淡出 10f</button><button type="button" className="btn" onClick={() => onUpdate({transition_in: undefined, transition_out: undefined})}>清除转场</button></div><div className="inspector-hint">转场时长不能超过片段时长；渲染时由 FFmpeg xfade 执行。</div></div> : null}
    </div>
  </aside>;
}

export function HistoryPanel({entries, currentRevision, diff, onCompare, onRestore, onClose}: {
  entries: HistoryEntry[]; currentRevision: number; diff: TimelineDiff | null;
  onCompare: (entry: HistoryEntry) => void; onRestore: (entry: HistoryEntry) => void; onClose: () => void;
}) {
  const [active, setActive] = useState('');
  return <aside className="timeline-side-panel timeline-history-panel" aria-label="编辑历史">
    <div className="timeline-panel-head"><strong>历史与版本</strong><span>r{currentRevision}</span><button type="button" className="icon-btn" onClick={onClose} aria-label="关闭历史"><X size={14} /></button></div>
    <div className="history-scroll">
      {entries.length ? entries.slice().reverse().map((entry) => <div key={entry.id} className="history-item" data-active={active === entry.id}>
        <button type="button" className="history-main" onClick={() => { setActive(entry.id); onCompare(entry); }}>
          <span className="history-revision">r{entry.revision}</span>
          <span className="history-label">{entry.label}</span>
          <span className="history-time">{new Date(entry.createdAt).toLocaleTimeString()}</span>
        </button>
        <button type="button" className="btn h-6" onClick={() => onRestore(entry)}>恢复</button>
      </div>) : <div className="timeline-panel-empty">还没有可比较的历史 revision。每次命令、AI 精修或拖动都会记录快照。</div>}
    </div>
    {diff ? <div className="history-diff">
      <div className="inspector-section-title">r{diff.from_revision} → r{diff.to_revision}</div>
      <div className="history-diff-summary"><span className="diff-added">+{diff.summary.added}</span><span className="diff-removed">−{diff.summary.removed}</span><span className="diff-changed">~{diff.summary.changed}</span></div>
      <div className="history-diff-items">
        {diff.added.map((item) => <span key={`a-${item.clip_id}`} className="diff-chip diff-added">+ {item.clip_id}</span>)}
        {diff.removed.map((item) => <span key={`r-${item.clip_id}`} className="diff-chip diff-removed">− {item.clip_id}</span>)}
        {diff.changed.map((item) => <span key={`c-${item.clip_id}`} className="diff-chip diff-changed">~ {item.clip_id}</span>)}
      </div>
    </div> : <div className="history-foot">选择任一历史 revision 查看差异。</div>}
  </aside>;
}

export function TranscriptPanel({timeline, selectedIds, onClose, onSeek, onSelect, onUpdateText}: {
  timeline: Timeline; selectedIds: string[]; onClose: () => void; onSeek: (frame: number) => void;
  onSelect: (id: string) => void; onUpdateText: (clip: TimelineClip, text: string) => void;
}) {
  const [query, setQuery] = useState('');
  const subtitleTracks = timeline.tracks.filter((track) => track.kind === 'subtitle');
  const clips = subtitleTracks.flatMap((track) => track.clips.map((clip) => ({clip, track}))).sort((left, right) => left.clip.start_frame - right.clip.start_frame);
  const filtered = clips.filter(({clip}) => !query.trim() || (clip.text || '').toLowerCase().includes(query.trim().toLowerCase()));
  return <aside className="timeline-side-panel timeline-transcript-panel" aria-label="文稿">
    <div className="timeline-panel-head"><strong>文稿 / 字幕</strong><span>{clips.length}</span><button type="button" className="icon-btn" onClick={onClose} aria-label="关闭文稿"><X size={14} /></button></div>
    <div className="media-search"><Search size={13} /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索文稿" aria-label="搜索文稿" /></div>
    <div className="transcript-list">
      {filtered.map(({clip, track}) => <div key={clip.id} className="transcript-item" data-active={selectedIds.includes(clip.id)}>
        <button type="button" className="transcript-time" onClick={() => { onSelect(clip.id); onSeek(clip.start_frame); }}>{timecode(clip.start_frame, timeline.timebase.num)}<span>{track.id}</span></button>
        <textarea className="field transcript-text" defaultValue={clip.text || ''} onBlur={(event) => { if (event.target.value !== (clip.text || '')) onUpdateText(clip, event.target.value); }} aria-label={`编辑字幕 ${clip.id}`} />
      </div>)}
      {!filtered.length ? <div className="timeline-panel-empty">没有字幕片段。先在简易模式“智能字幕”挂载文稿。</div> : null}
    </div>
    <div className="media-library-foot"><ListMusic size={11} /> 点时间码定位播放头；改字写回同一字幕片段</div>
  </aside>;
}

export function AssetBrowserPanel({workspace, onClose, onApplyTemplate, onToast}: {
  workspace: string; onClose: () => void; onApplyTemplate: (templateId: string) => void; onToast: (message: string) => void;
}) {
  const [tab, setTab] = useState<'templates' | 'assets'>('templates');
  const [templates, setTemplates] = useState<any[]>([]);
  const [assets, setAssets] = useState<any[]>([]);
  const [query, setQuery] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  async function load() {
    setBusy(true); setError('');
    try {
      const templateResult = await callCapability<any>('media.workbench.timeline.templates.list', {});
      setTemplates(templateResult?.templates ?? []);
      const assetResult = await callCapability<any>('media.workbench.asset.market.list', {workspace_root: workspace});
      setAssets(assetResult?.assets ?? []);
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  useEffect(() => { void load(); }, [workspace]);
  async function install(asset: any) {
    if (asset.status !== 'published') { onToast?.(`${asset.name || asset.id} 当前状态为 ${asset.status}，尚不可安装`); return; }
    setBusy(true); setError('');
    try {
      await callCapability('media.workbench.asset.market.install', {workspace_root: workspace, pack_id: asset.id, version: asset.version});
      onToast?.(`已安装资产包 ${asset.name || asset.id}`);
      await load();
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  const templateMatches = templates.filter((item) => !query.trim() || `${item.name} ${item.description} ${(item.tags || []).join(' ')}`.toLowerCase().includes(query.trim().toLowerCase()));
  const assetMatches = assets.filter((item) => !query.trim() || `${item.name} ${item.id} ${(item.tags || []).join(' ')}`.toLowerCase().includes(query.trim().toLowerCase()));
  return <aside className="timeline-side-panel timeline-asset-panel" aria-label="素材市场">
    <div className="timeline-panel-head"><strong>素材市场</strong><span>{templates.length + assets.length}</span><button type="button" className="icon-btn" onClick={onClose} aria-label="关闭市场"><X size={14} /></button></div>
    <div className="media-filter-tabs"><button type="button" data-active={tab === 'templates'} onClick={() => setTab('templates')}>剪辑模板</button><button type="button" data-active={tab === 'assets'} onClick={() => setTab('assets')}>资产包</button></div>
    <div className="media-search"><Search size={13} /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索模板、风格或资产" aria-label="搜索市场" /></div>
    <div className="asset-browser-list">
      {error ? <div className="asset-error">{error}</div> : null}
      {tab === 'templates' ? templateMatches.map((template) => <div key={template.id} className="asset-card">
        <div className="asset-card-head"><strong>{template.name || template.id}</strong><span className="badge">{template.slots?.length ?? 0} 槽位</span></div>
        <p>{template.description || '无描述'}</p>
        <div className="asset-tags">{(template.tags || []).map((tag: string) => <span key={tag}>{tag}</span>)}</div>
        <button type="button" className="btn btn-primary h-7" onClick={() => onApplyTemplate(template.id)}>套用模板</button>
      </div>) : assetMatches.map((asset) => <div key={asset.id} className="asset-card">
        <div className="asset-card-head"><strong>{asset.name || asset.id}</strong><span className="badge">{asset.version}</span></div>
        <p>{asset.status === 'published' ? '已发布' : `状态：${asset.status || 'unknown'}`} · {asset.license || 'license unknown'}</p>
        <div className="asset-tags">{(asset.tags || []).map((tag: string) => <span key={tag}>{tag}</span>)}</div>
        <div className="asset-card-foot"><span>{humanSize(asset.file_size)}</span>{asset.installed ? <span className="badge" style={{color: 'var(--ok)'}}><Check size={10} /> 已安装 {asset.installed_version}</span> : <button type="button" className="btn h-7" disabled={busy || asset.status !== 'published'} onClick={() => void install(asset)}><Download size={11} /> 安装</button>}</div>
      </div>)}
      {!busy && !templateMatches.length && tab === 'templates' ? <div className="timeline-panel-empty">没有匹配模板。</div> : null}
      {!busy && !assetMatches.length && tab === 'assets' ? <div className="timeline-panel-empty">没有匹配资产包。</div> : null}
      {busy ? <div className="timeline-panel-empty">正在读取市场…</div> : null}
    </div>
  </aside>;
}

export function DiffPanel({diff, onApply, onDiscard, onClose}: {
  diff: TimelineDiff; onApply: (selected: Set<string>) => void; onDiscard: () => void; onClose: () => void;
}) {
  const [selected, setSelected] = useState<Set<string>>(() => new Set([...diff.added, ...diff.changed].map((item) => item.clip_id)));
  function toggle(id: string) {
    setSelected((current) => { const next = new Set(current); if (next.has(id)) next.delete(id); else next.add(id); return next; });
  }
  return <div className="timeline-modal-backdrop" onClick={onClose}>
    <div className="timeline-diff-panel" onClick={(event) => event.stopPropagation()} role="dialog" aria-label="AI 修改预览">
      <div className="timeline-panel-head"><strong>AI 修改预览</strong><span>r{diff.from_revision} → r{diff.to_revision}</span><button type="button" className="icon-btn" onClick={onClose} aria-label="关闭预览"><X size={14} /></button></div>
      <div className="diff-summary-bar"><span className="diff-added">新增 {diff.summary.added}</span><span className="diff-removed">删除 {diff.summary.removed}</span><span className="diff-changed">修改 {diff.summary.changed}</span></div>
      <div className="diff-columns">
        {([['added', '新增', diff.added], ['changed', '修改', diff.changed], ['removed', '删除', diff.removed]] as const).map(([kind, label, items]) => <div key={kind} className="diff-column">
          <div className="inspector-section-title">{label} · {items.length}</div>
          {items.length ? items.map((item) => <label key={`${kind}-${item.clip_id}`} className={`diff-row diff-${kind}`}><input type="checkbox" checked={selected.has(item.clip_id)} onChange={() => toggle(item.clip_id)} /><span>{item.clip_id}</span><em>{item.track_id}</em></label>) : <div className="diff-empty">无</div>}
        </div>)}
      </div>
      <div className="diff-foot"><span>默认选中新增与修改项；取消勾选可局部应用。</span><button type="button" className="btn" onClick={onDiscard}>放弃修改</button><button type="button" className="btn btn-primary" disabled={!selected.size} onClick={() => onApply(selected)}>应用选中 {selected.size}</button></div>
    </div>
  </div>;
}

export function SemanticPanel({workspace, timeline, mediaPaths, initialQuery, onClose, onSeek, onSelect}: {
  workspace: string; timeline: Timeline; mediaPaths: Record<string, string>; initialQuery: string;
  onClose: () => void; onSeek: (frame: number) => void; onSelect: (id: string) => void;
}) {
  const [query, setQuery] = useState(initialQuery);
  const [hits, setHits] = useState<SemanticHit[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  async function search(next = query) {
    if (!next.trim()) return;
    setBusy(true); setError('');
    try {
      const result = await callCapability<any>('media.workbench.timeline.semantic-search', {workspace_root: workspace, timeline, media_paths: mediaPaths, query: next, limit: 30});
      setHits(result?.hits ?? []);
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  }
  useEffect(() => { void search(initialQuery); /* eslint-disable-next-line react-hooks/exhaustive-deps */ }, []);
  return <aside className="timeline-side-panel timeline-semantic-panel" aria-label="AI 语义检索">
    <div className="timeline-panel-head"><strong>AI 语义检索</strong><span>{hits.length}</span><button type="button" className="icon-btn" onClick={onClose} aria-label="关闭检索"><X size={14} /></button></div>
    <div className="media-search"><Sparkles size={13} /><input value={query} onChange={(event) => setQuery(event.target.value)} onKeyDown={(event) => { if (event.key === 'Enter') void search(); }} placeholder="例如：找讨论磁场的镜头" aria-label="语义检索" /><button type="button" className="btn h-6" disabled={busy || !query.trim()} onClick={() => void search()}>检索</button></div>
    {error ? <div className="asset-error">{error}</div> : null}
    <div className="semantic-list">
      {hits.map((hit, index) => <button key={`${hit.kind}-${hit.clip_id || hit.media_id}-${index}`} type="button" className="semantic-hit" onClick={() => { if (hit.clip_id) { onSelect(hit.clip_id); onSeek(hit.start_frame || 0); } }}>
        <span className="semantic-score">{hit.score.toFixed(1)}</span>
        <span className="semantic-main"><strong>{hit.clip_id || hit.media_id}</strong><em>{hit.reason}</em><span>{hit.kind === 'clip' ? `${timecode(hit.start_frame || 0, timeline.timebase.num)} · ${hit.track_id}` : '素材'}</span></span>
        <ChevronRight size={12} />
      </button>)}
      {!busy && !hits.length ? <div className="timeline-panel-empty">输入自然语言查找素材或片段。结果会定位到播放头。</div> : null}
      {busy ? <div className="timeline-panel-empty">正在检索…</div> : null}
    </div>
  </aside>;
}
