import {useEffect, useRef, useState} from 'react';
import {ExternalLink, FileVideo, Loader2, Play} from 'lucide-react';
import {callCapability} from '../api/capability';
import type {Recipe} from '../types';

/**
 * 预览面板 —— **不自己画任何东西**。
 *
 * 画面由渲染器产出（preview.player 生成的单文件 HTML），这里只负责
 * 取回 artifact 并内嵌播放。创作台不认识"地图/线路/图表"，
 * 否则每加一个题材都要改外壳。
 */

type ProjectArtifact = {
  artifact_id: string;
  kind: string;
  path: string;
  size: number;
  sha256: string;
  created_at?: string;
  renderer?: string;
  ready?: boolean;
};

type ProjectDetailLite = {
  latest_video?: ProjectArtifact | null;
  artifacts?: ProjectArtifact[];
};

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0 B';
  if (value >= 1024 * 1024) return `${(value / 1024 / 1024).toFixed(1)} MB`;
  if (value >= 1024) return `${(value / 1024).toFixed(0)} KB`;
  return `${value} B`;
}
function aspectRatioLabel(width: number, height: number): string {
  const gcd = (a: number, b: number): number => (b === 0 ? Math.abs(a) : gcd(b, a % b));
  const divisor = gcd(width, height) || 1;
  return `${width / divisor}:${height / divisor}`;
}
function decodeBase64Utf8(base64: string): string {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return new TextDecoder('utf-8').decode(bytes);
}

export function PreviewPane({
  recipe,
  workspace,
  projectId,
  generateSignal,
}: {
  recipe: Recipe | null;
  workspace: string;
  projectId: string | null;
  /** 页头按钮与面板按钮共用同一条生成链路；每次自增触发一次生成。 */
  generateSignal: number;
}) {
  const [html, setHtml] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [latestVideo, setLatestVideo] = useState<ProjectArtifact | null>(null);
  const [artifacts, setArtifacts] = useState<ProjectArtifact[]>([]);
  const [openingVideo, setOpeningVideo] = useState(false);
  const generationRef = useRef(0);

  // 切换项目必须丢弃上一个项目的 iframe，避免项目树已经切了、画面还是旧的。
  useEffect(() => {
    generationRef.current += 1;
    setHtml(null);
    setError(null);
    setBusy(false);
    setLatestVideo(null);
    setArtifacts([]);
  }, [projectId]);

  useEffect(() => {
    if (!projectId || !workspace) return;
    let cancelled = false;
    (async () => {
      try {
        const detail = await callCapability<ProjectDetailLite>('media.workbench.project.get', {
          workspace_root: workspace,
          project_id: projectId,
        });
        if (cancelled) return;
        setLatestVideo(detail.latest_video ?? null);
        setArtifacts(detail.artifacts ?? []);
      } catch {
        if (!cancelled) {
          setLatestVideo(null);
          setArtifacts([]);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [projectId, workspace]);

  async function openLatestVideo() {
    if (!latestVideo) return;
    setOpeningVideo(true);
    setError(null);
    try {
      await callCapability('media.workbench.artifact.open', {
        workspace_root: workspace,
        artifact_id: latestVideo.artifact_id,
        open_mode: 'file',
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setOpeningVideo(false);
    }
  }

  useEffect(() => {
    if (generateSignal > 0) void generate();
    // generate 读取的是当前 projectId/workspace；这里只跟随外部触发信号。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [generateSignal]);

  async function generate() {
    if (!projectId) return;
    const token = ++generationRef.current;
    setBusy(true);
    setError(null);
    try {
      const preview = await callCapability<{artifact?: {artifact_id?: string}}>('media.workbench.preview.player', {
        workspace_root: workspace,
        project_id: projectId,
      });
      if (token !== generationRef.current) return;
      const artifactId = preview?.artifact?.artifact_id ?? '';
      if (!artifactId) throw new Error('渲染器没有返回预览产物');
      const read = await callCapability<{base64?: string}>('media.workbench.artifact.read', {
        workspace_root: workspace,
        artifact_id: artifactId,
      });
      if (!read?.base64) throw new Error('预览产物读取失败');
      if (token !== generationRef.current) return;
      setHtml(decodeBase64Utf8(read.base64));
    } catch (err) {
      if (token === generationRef.current) setError(err instanceof Error ? err.message : String(err));
    } finally {
      if (token === generationRef.current) setBusy(false);
    }
  }

  const latestVideoCard = latestVideo ? (
    <div className="mb-2 flex w-full items-center justify-between gap-3 rounded-lg px-3 py-2 text-left" style={{border: '1px solid var(--line)', background: 'var(--ink-850)'}}>
      <div className="flex min-w-0 items-center gap-2">
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md" style={{background: 'var(--accent-soft)', color: 'var(--accent)'}}>
          <FileVideo size={15} />
        </div>
        <div className="min-w-0">
          <div className="truncate text-[11.5px] font-medium">最新成片 · {latestVideo.kind}</div>
          <div className="truncate font-mono text-[10px]" style={{color: 'var(--fg-dim)'}}>
            {latestVideo.artifact_id} · {formatBytes(latestVideo.size)} · {latestVideo.renderer ?? 'renderer'} · 产物 {artifacts.length}
          </div>
        </div>
      </div>
      <button type="button" className="btn shrink-0" onClick={() => void openLatestVideo()} disabled={openingVideo}>
        {openingVideo ? <Loader2 size={11} className="animate-spin" /> : <ExternalLink size={11} />}
        {openingVideo ? '打开中…' : '打开成片'}
      </button>
    </div>
  ) : null;


  if (html) {
    return (
      <div className="flex h-full flex-col p-3">
        {latestVideoCard}
        <div className="mb-2 flex items-center justify-between">
          <span className="text-[11px]" style={{color: 'var(--fg-dim)'}}>动效预览（渲染器产出）</span>
          <button type="button" className="btn btn-ghost" onClick={() => void generate()} disabled={busy}>
            <Play size={11} />
            重新生成
          </button>
        </div>
        <iframe
          title="动效预览"
          srcDoc={html}
          sandbox="allow-scripts allow-same-origin"
          className="min-h-0 flex-1 rounded-lg"
          style={{border: '1px solid var(--line)', background: '#0a0f1f'}}
        />
        {error ? <div className="mt-2 text-[11px]" style={{color: 'var(--err)'}}>{error}</div> : null}
      </div>
    );
  }

  const canvas = recipe?.canvas;
  const isWide = (canvas?.width ?? 1080) >= (canvas?.height ?? 1920);
  const ratio = canvas ? aspectRatioLabel(canvas.width, canvas.height) : '9:16';
  const previewSize = isWide ? {width: 208, height: 117} : {width: 132, height: 234};

  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-8 text-center">
      <div
        className="flex items-center justify-center rounded-xl"
        style={{...previewSize, background: 'linear-gradient(180deg,#0f1629,#131b33)', border: '1px solid var(--line)'}}
      >
        <span className="text-[11px]" style={{color: 'var(--fg-dim)'}}>{ratio}</span>
      </div>
      <div className="text-[12.5px]" style={{color: 'var(--fg-muted)'}}>
        {recipe ? recipe.title ?? '未命名项目' : '未选择项目'}
      </div>
      {recipe ? (
        <div className="text-[11px]" style={{color: 'var(--fg-dim)'}}>
          {recipe.template_id} · {recipe.duration_seconds}s · {recipe.canvas.width}×{recipe.canvas.height}
        </div>
      ) : null}
      {latestVideoCard}
      <button type="button" className="btn btn-primary mt-1" onClick={() => void generate()} disabled={!projectId || busy}>
        {busy ? <Loader2 size={12} className="animate-spin" /> : <Play size={12} />}
        {busy ? '生成中…' : '生成动效预览'}
      </button>
      {error ? <div className="max-w-[420px] text-[11px]" style={{color: 'var(--err)'}}>{error}</div> : null}
      <div className="max-w-[380px] text-[10.5px] leading-relaxed" style={{color: 'var(--fg-dim)'}}>
        预览由渲染器生成后回传，创作台不绘制画面。
      </div>
    </div>
  );
}
