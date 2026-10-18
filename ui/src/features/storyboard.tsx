import {useEffect, useRef, useState} from 'react';
import {Loader2, Play} from 'lucide-react';
import {callCapability} from '../api/capability';
import type {Recipe} from '../types';

/**
 * 预览面板 —— **不自己画任何东西**。
 *
 * 画面由渲染器产出（preview.player 生成的单文件 HTML），这里只负责
 * 取回 artifact 并内嵌播放。创作台不认识"地图/线路/图表"，
 * 否则每加一个题材都要改外壳。
 */

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
  const generationRef = useRef(0);

  // 切换项目必须丢弃上一个项目的 iframe，避免"项目树已经切了，画面还是旧的"。
  useEffect(() => {
    generationRef.current += 1;
    setHtml(null);
    setError(null);
    setBusy(false);
  }, [projectId]);

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

  if (html) {
    return (
      <div className="flex h-full flex-col p-3">
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

  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-8 text-center">
      <div
        className="flex items-center justify-center rounded-xl"
        style={{width: 132, height: 234, background: 'linear-gradient(180deg,#0f1629,#131b33)', border: '1px solid var(--line)'}}
      >
        <span className="text-[11px]" style={{color: 'var(--fg-dim)'}}>9:16</span>
      </div>
      <div className="text-[12.5px]" style={{color: 'var(--fg-muted)'}}>
        {recipe ? recipe.title ?? '未命名项目' : '未选择项目'}
      </div>
      {recipe ? (
        <div className="text-[11px]" style={{color: 'var(--fg-dim)'}}>
          {recipe.template_id} · {recipe.duration_seconds}s · {recipe.canvas.width}×{recipe.canvas.height}
        </div>
      ) : null}
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


