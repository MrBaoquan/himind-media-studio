import {useEffect, useMemo, useState} from 'react';
import {CheckCircle2, ExternalLink, Film, Grid3X3, Image as ImageIcon, Loader2, MessageSquare, Play, RefreshCw, Rows3, Send, ShieldCheck} from 'lucide-react';
import {callCapability} from '../api/capability';
import {useArtifactBlob} from './timeline/hooks';
import type {Artifact, FeedbackEntry, ProjectSummary} from '../types';


function imageMime(path: string): string {
  const lower = path.toLowerCase();
  if (lower.endsWith('.png')) return 'image/png';
  if (lower.endsWith('.webp')) return 'image/webp';
  if (lower.endsWith('.gif')) return 'image/gif';
  return 'image/jpeg';
}

type RenderJob = {
  job_id: string;
  status: string;
  progress: number;
  message?: string;
  quality_status?: string;
  quality_report?: {ok?: boolean; warnings?: string[]; checks?: Array<{id: string; status: string; detail?: string}>};
  artifact_id?: string;
  artifact_path?: string;
};

function previewArtifacts(items: Artifact[]): Artifact[] {
  return items.filter((item) => item.ready && item.kind.startsWith('preview/')).sort((a, b) => b.created_at.localeCompare(a.created_at));
}

function PreviewThumb({workspace, artifact, enabled}: {workspace: string; artifact: Artifact; enabled: boolean}) {
  const {url} = useArtifactBlob(workspace, artifact.artifact_id, enabled);
  return url
    ? <img src={url} alt="" className="h-8 w-8 shrink-0 rounded-md object-cover" style={{border: '1px solid var(--line)'}} />
    : <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md" style={{background: 'var(--ink-850)', border: '1px solid var(--line)'}}><ImageIcon size={12} style={{color: 'var(--fg-dim)'}} /></span>;
}

export function ReviewPane({
  project,
  workspace,
  artifacts,
  feedback: feedbackItems,
  generateSignal,
  onRefresh,
}: {
  project: ProjectSummary | null;
  workspace: string;
  artifacts: Artifact[];
  feedback: FeedbackEntry[];
  generateSignal: number;
  onRefresh: () => void | Promise<void>;
}) {
  const [items, setItems] = useState<Artifact[]>(() => previewArtifacts(artifacts));
  const [selectedId, setSelectedId] = useState<string>('');
  const [imageUrl, setImageUrl] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [feedbackText, setFeedbackText] = useState('');
  const [category, setCategory] = useState('motion');
  const [detBusy, setDetBusy] = useState(false);
  const [renderBusy, setRenderBusy] = useState(false);
  const [renderJob, setRenderJob] = useState<RenderJob | null>(null);
  const [renderError, setRenderError] = useState<string | null>(null);
  const [renderNotice, setRenderNotice] = useState<string | null>(null);
  const [upgradeBusy, setUpgradeBusy] = useState(false);
  const [upgradeError, setUpgradeError] = useState<string | null>(null);
  const [upgradeNotice, setUpgradeNotice] = useState<string | null>(null);
  const [qaBusy, setQaBusy] = useState(false);
  const [qaError, setQaError] = useState<string | null>(null);
  const [qaNotice, setQaNotice] = useState<string | null>(null);

  const previews = useMemo(() => previewArtifacts(items), [items]);
  const videos = useMemo(() => artifacts.filter((item) => item.ready && item.kind.startsWith('video/')).sort((a, b) => b.created_at.localeCompare(a.created_at)), [artifacts]);
  const latestVideo = videos[0] ?? null;
  const qaVideo = videos.find((item) => Boolean(item.quality_status)) ?? null;
  const pendingFeedback = feedbackItems.filter((item) => (item.status ?? 'open') === 'open');
  const selected = previews.find((item) => item.artifact_id === selectedId) ?? previews[0] ?? null;

  useEffect(() => {
    setItems(previewArtifacts(artifacts));
  }, [artifacts]);

  useEffect(() => {
    if (!selectedId && previews[0]) setSelectedId(previews[0].artifact_id);
  }, [previews, selectedId]);

  useEffect(() => {
    if (!renderJob?.job_id) return;
    if (['completed', 'failed', 'cancelled'].includes(renderJob.status)) return;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;
    const poll = async () => {
      try {
        const result = await callCapability<{job?: RenderJob}>('media.workbench.render.status', {
          workspace_root: workspace,
          job_id: renderJob.job_id,
        });
        if (cancelled) return;
        const job = result.job;
        if (!job) throw new Error('渲染任务没有返回状态');
        setRenderJob(job);
        if (job.status === 'completed') {
          setRenderNotice('成片已生成');
          await onRefresh();
          return;
        }
        if (job.status === 'failed' || job.status === 'cancelled') {
          setRenderError(job.message || '渲染未完成');
          return;
        }
      } catch (err) {
        if (!cancelled) setRenderError(err instanceof Error ? err.message : String(err));
        return;
      }
      if (!cancelled) timer = setTimeout(poll, 2000);
    };
    timer = setTimeout(poll, 1000);
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [renderJob?.job_id, renderJob?.status, workspace]);

  useEffect(() => {
    let cancelled = false;
    if (!selected) {
      setImageUrl(null);
      return;
    }
    (async () => {
      try {
        const read = await callCapability<{base64?: string}>('media.workbench.artifact.read', {
          workspace_root: workspace,
          artifact_id: selected.artifact_id,
        });
        if (!cancelled && read?.base64) {
          setImageUrl(`data:${imageMime(selected.path)};base64,${read.base64}`);
        }
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [selected, workspace]);

  function sampleTimes(fractions: number[]): string {
    const duration = Math.max(1, project?.duration_seconds ?? 10);
    return fractions.map((fraction) => Math.max(0.1, Math.round(duration * fraction * 10) / 10).toFixed(1)).join(',');
  }

  async function generate(kind: 'sheet' | 'strip' | 'stills') {
    if (!project) return;
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const params: Record<string, unknown> = {
        workspace_root: workspace,
        project_id: project.id,
        kind,
      };
      if (kind === 'sheet') {
        params.times = sampleTimes([0.05, 0.16, 0.27, 0.38, 0.49, 0.60, 0.71, 0.82, 0.93]);
        params.width = 540;
        params.cols = 3;
      } else if (kind === 'strip') {
        const duration = Math.max(1, project?.duration_seconds ?? 10);
        params.range = `${(duration * 0.2).toFixed(1)}:${(duration * 0.25).toFixed(1)}`;
        params.width = 320;
        params.cols = 6;
      } else {
        params.times = sampleTimes([0.2, 0.5, 0.8]);
      }
      const result = await callCapability<{artifacts?: Artifact[]}>('media.workbench.preview.review', params);
      const next = previewArtifacts(result.artifacts ?? []);
      if (next.length) {
        setItems((current) => [...next, ...current]);
        setSelectedId(next[0].artifact_id);
      }
      setNotice(kind === 'sheet' ? '关键帧总览已生成' : kind === 'strip' ? '连续帧审查条已生成' : '静帧已生成');
      await onRefresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  useEffect(() => {
    if (generateSignal > 0) void generate('sheet');
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [generateSignal]);

  async function resolveFeedback(id: string) {
    if (!project) return;
    setBusy(true);
    setError(null);
    try {
      await callCapability('media.workbench.feedback.resolve', {
        workspace_root: workspace,
        project_id: project.id,
        feedback_id: id,
        status: 'resolved',
      });
      await onRefresh();
      setNotice('反馈已标记为已处理');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }
  async function submitFeedback() {
    if (!project || !feedbackText.trim()) return;
    setBusy(true);
    setError(null);
    try {
      await callCapability('media.workbench.feedback.record', {
        workspace_root: workspace,
        project_id: project.id,
        category,
        feedback: feedbackText.trim(),
        accepted: false,
      });
      setFeedbackText('');
      setNotice('反馈已记录。回到 AI 对话后，让它读取 project.context 继续修改。');
      await onRefresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function openArtifact(item: Artifact | null | undefined) {
    if (!item) return;
    try {
      await callCapability('media.workbench.artifact.open', {
        workspace_root: workspace,
        artifact_id: item.artifact_id,
        mode: 'file',
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  const previewReady = project?.preview_status === 'ready';
  const determinismPassed = Boolean(project && project.determinism_status === 'passed' && project.determinism_hash && project.determinism_hash === project.source_hash);
  const rendering = renderBusy || Boolean(renderJob && !['completed', 'failed', 'cancelled'].includes(renderJob.status));

  async function checkDeterminism(): Promise<boolean> {
    if (!project) return false;
    setDetBusy(true);
    setRenderError(null);
    setRenderNotice(null);
    try {
      const result = await callCapability<{valid?: boolean}>('media.workbench.preview.determinism', {
        workspace_root: workspace,
        project_id: project.id,
        times: sampleTimes([0.15, 0.5, 0.85]),
      });
      if (!result.valid) throw new Error('确定性检查未通过；先检查随机种子、时间和外部资源');
      setRenderNotice('确定性检查通过');
      await onRefresh();
      return true;
    } catch (err) {
      setRenderError(err instanceof Error ? err.message : String(err));
      return false;
    } finally {
      setDetBusy(false);
    }
  }

  async function upgradeStyle() {
    if (!project) return;
    setUpgradeBusy(true);
    setUpgradeError(null);
    setUpgradeNotice(null);
    try {
      const result = await callCapability<{state?: string; changed?: string[]; message?: string}>('media.workbench.project.upgrade-style', {
        workspace_root: workspace,
        project_id: project.id,
      });
      const changed = result.changed ?? [];
      setUpgradeNotice(result.message || (result.state === 'current'
        ? '当前 renderer 已是最新版本'
        : `已升级 ${changed.join(', ') || 'renderer'}；请重新生成审查图并检查确定性`));
      await onRefresh();
    } catch (err) {
      setUpgradeError(err instanceof Error ? err.message : String(err));
    } finally {
      setUpgradeBusy(false);
    }
  }

  async function runVisualQA() {
    if (!project) return;
    setQaBusy(true);
    setQaError(null);
    setQaNotice(null);
    try {
      const result = await callCapability<{ok?: boolean; checks?: Array<{id: string; status: string; detail?: string}>; warnings?: string[]}>('media.workbench.quality.visual', {
        workspace_root: workspace,
        project_id: project.id,
      });
      const checks = result.checks ?? [];
      const failed = checks.filter((check) => check.status === 'failed');
      const warnings = checks.filter((check) => check.status === 'warning');
      setQaNotice(result.ok
        ? `视觉 QA 通过${warnings.length ? `（${warnings.length} 个 warning）` : ''}`
        : `视觉 QA 失败：${failed.map((check) => check.id).join(', ') || '未知检查'}`);
    } catch (err) {
      setQaError(err instanceof Error ? err.message : String(err));
    } finally {
      setQaBusy(false);
    }
  }

  async function startRender() {
    if (!project) return;
    if (!determinismPassed) {
      const checked = await checkDeterminism();
      if (!checked) return;
    }
    setRenderBusy(true);
    setRenderError(null);
    setRenderNotice(null);
    try {
      const result = await callCapability<{job?: RenderJob}>('media.workbench.render.start', {
        workspace_root: workspace,
        project_id: project.id,
      });
      if (!result.job?.job_id) throw new Error('渲染任务没有返回 job_id');
      setRenderJob(result.job);
      setRenderNotice('渲染任务已启动');
      await onRefresh();
    } catch (err) {
      setRenderError(err instanceof Error ? err.message : String(err));
    } finally {
      setRenderBusy(false);
    }
  }

  if (!project) {
    return <div className="flex h-full items-center justify-center text-[12px]" style={{color: 'var(--fg-dim)'}}>选择一个代码项目开始审查</div>;
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex items-center gap-2 border-b px-3 py-2" style={{borderColor: 'var(--line)'}}>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 text-[12px] font-medium">
            <span className="truncate">{project.name}</span>
            <span className="badge">代码项目</span>
            <span className="font-mono text-[10px]" style={{color: 'var(--fg-dim)'}}>{project.style_pack} · r{project.revision}</span>
          </div>
          <div className="mt-0.5 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
            {project.preview_status === 'ready' ? '审查图已生成' : project.preview_status === 'stale' ? '源码已变化，审查图过期' : '尚未生成审查图'}
          </div>
        </div>
        <button type="button" className="btn" onClick={() => void generate('sheet')} disabled={busy}>
          {busy ? <Loader2 size={12} className="animate-spin" /> : <Grid3X3 size={12} />}
          关键帧
        </button>
        <button type="button" className="btn" onClick={() => void generate('strip')} disabled={busy}>
          <Rows3 size={12} />连续帧
        </button>
        <button type="button" className="btn" onClick={() => void generate('stills')} disabled={busy}>
          <ImageIcon size={12} />静帧
        </button>
        <button type="button" className="btn" onClick={() => void generate('sheet')} disabled={busy}>
          <RefreshCw size={12} />重新生成
        </button>
      </div>

      <div className="grid min-h-0 flex-1" style={{gridTemplateColumns: '1fr 230px'}}>
        <div className="flex min-h-0 flex-col items-center justify-center overflow-auto p-3" style={{background: '#070b16'}}>
          {imageUrl ? (
            <img src={imageUrl} alt="审查图" className="max-h-full max-w-full rounded-md object-contain" style={{border: '1px solid var(--line)'}} />
          ) : (
            <div className="flex flex-col items-center gap-3 text-center">
              <ImageIcon size={34} style={{color: 'var(--fg-dim)'}} />
              <div className="text-[12.5px]" style={{color: 'var(--fg-muted)'}}>还没有审查图</div>
              <button type="button" className="btn" onClick={() => void generate('sheet')} disabled={busy}>
                {busy ? <Loader2 size={12} className="animate-spin" /> : <Play size={12} />}
                生成关键帧总览
              </button>
            </div>
          )}
        </div>

        <aside className="min-h-0 overflow-auto border-l p-2" style={{borderColor: 'var(--line)', background: 'var(--ink-900)'}}>
          <div className="mb-1 px-1 text-[10.5px] font-medium" style={{color: 'var(--fg-dim)'}}>审查产物</div>
          {previews.length === 0 ? (
            <div className="px-1 py-2 text-[11px]" style={{color: 'var(--fg-dim)'}}>生成后可在这里切换</div>
          ) : (
            <div className="flex flex-col gap-1">
              {previews.map((item, index) => (
                <button
                  key={item.artifact_id}
                  type="button"
                  className="nav-item"
                  data-active={item.artifact_id === selected?.artifact_id}
                  onClick={() => setSelectedId(item.artifact_id)}
                  title={item.path}
                >
                  <PreviewThumb workspace={workspace} artifact={item} enabled={index < 3} />
                  <span className="min-w-0 flex-1 truncate text-left text-[10.5px]">{item.kind.replace('preview/', '')}</span>
                  <span className="text-[9.5px]" style={{color: 'var(--fg-dim)'}}>{Math.round(item.size / 1024)}k</span>
                </button>
              ))}
            </div>
          )}
          <div className="divider my-2" />
          <button type="button" className="btn w-full" onClick={() => void openArtifact(selected ?? previews[0])} disabled={!selected}>
            <ExternalLink size={11} />打开原图
          </button>
          <button type="button" className="btn mt-1 w-full" onClick={() => void openArtifact(latestVideo)} disabled={!latestVideo}>
            <Play size={11} />打开最新成片{videos.length ? ` (${videos.length})` : ''}
          </button>
          {qaVideo?.quality_status ? (
            <div className="mt-1 px-1 text-[10px]" style={{color: qaVideo.quality_status === 'failed' || qaVideo.quality_status === 'error' ? 'var(--err)' : qaVideo.quality_status === 'warning' ? 'var(--warn)' : 'var(--ok)'}}>
              母版 QA：{qaVideo.quality_status === 'warning' ? '通过（有 warning）' : qaVideo.quality_status === 'passed' ? '通过' : qaVideo.quality_status === 'failed' ? '失败，不建议交付' : '未完成'}
            </div>
          ) : null}
          <div className="divider my-2" />
          <div className="mb-1 px-1 text-[10.5px] font-medium" style={{color: 'var(--fg-dim)'}}>成片渲染</div>
          <div className="review-render-sticky">
          <button type="button" className="btn mt-1 w-full" onClick={() => void checkDeterminism()} disabled={detBusy || rendering || !previewReady || determinismPassed}>
            {detBusy ? <Loader2 size={11} className="animate-spin" /> : determinismPassed ? <CheckCircle2 size={11} /> : <ShieldCheck size={11} />}
            {determinismPassed ? '确定性已通过' : '检查确定性'}
          </button>
          <button type="button" className="btn btn-primary mt-1 w-full" onClick={() => void startRender()} disabled={renderBusy || rendering || detBusy || !previewReady}>
            {rendering ? <Loader2 size={11} className="animate-spin" /> : detBusy ? <Loader2 size={11} className="animate-spin" /> : <Film size={11} />}
            {rendering ? `渲染中 ${renderJob?.progress ?? 0}%` : detBusy ? '正在检查确定性…' : determinismPassed ? '渲染成片' : '检查确定性并渲染'}
          </button>
          </div>
          <button type="button" className="btn mt-1 w-full" onClick={() => void upgradeStyle()} disabled={upgradeBusy || rendering || detBusy}>
            {upgradeBusy ? <Loader2 size={11} className="animate-spin" /> : <RefreshCw size={11} />}
            {upgradeBusy ? '正在升级风格…' : '升级风格渲染器'}
          </button>
          <button type="button" className="btn mt-1 w-full" onClick={() => void runVisualQA()} disabled={qaBusy || rendering || detBusy || !latestVideo}>
            {qaBusy ? <Loader2 size={11} className="animate-spin" /> : <ShieldCheck size={11} />}
            {qaBusy ? '正在检查视觉…' : '视觉 QA'}
          </button>
          {qaError ? <div className="mt-1 px-1 text-[10px]" style={{color: 'var(--err)'}}>{qaError}</div> : null}
          {qaNotice ? <div className="mt-1 px-1 text-[10px]" style={{color: 'var(--ok)'}}>{qaNotice}</div> : null}
          {upgradeError ? <div className="mt-1 px-1 text-[10px]" style={{color: 'var(--err)'}}>{upgradeError}</div> : null}
          {upgradeNotice ? <div className="mt-1 px-1 text-[10px]" style={{color: 'var(--ok)'}}>{upgradeNotice}</div> : null}
          {renderJob ? (
            <div className="mt-2 rounded-md p-2 text-[10px]" style={{background: 'var(--ink-850)', border: '1px solid var(--line)'}}>
              <div className="flex items-center gap-2">
                <span className="truncate" style={{color: 'var(--fg-muted)'}}>{renderJob.message || renderJob.status}</span>
                <span className="ml-auto font-mono" style={{color: 'var(--fg-dim)'}}>{renderJob.progress ?? 0}%</span>
              </div>
              {renderJob.quality_status ? (
                <div className="mt-1 text-[10px]" style={{color: renderJob.quality_status === 'failed' || renderJob.quality_status === 'error' ? 'var(--err)' : renderJob.quality_status === 'warning' ? 'var(--warn)' : 'var(--ok)'}}>
                  视觉 QA：{renderJob.quality_status === 'warning' ? '通过（有 warning）' : renderJob.quality_status === 'passed' ? '通过' : renderJob.quality_status === 'failed' ? '失败' : '未完成'}
                </div>
              ) : null}
              <div className="mt-1 h-1 overflow-hidden rounded-full" style={{background: 'var(--ink-800)'}}>
                <div className="h-full rounded-full transition-all" style={{width: `${Math.max(0, Math.min(100, renderJob.progress ?? 0))}%`, background: 'var(--accent)'}} />
              </div>
            </div>
          ) : null}
          {!previewReady ? <div className="mt-1 px-1 text-[9.5px]" style={{color: 'var(--fg-dim)'}}>先生成审查图</div> : null}
          {previewReady && !determinismPassed ? <div className="mt-1 px-1 text-[9.5px]" style={{color: 'var(--fg-dim)'}}>点击渲染会先自动检查确定性</div> : null}
          {renderError ? <div className="mt-1 px-1 text-[10px]" style={{color: 'var(--err)'}}>{renderError}</div> : null}
          {renderNotice ? <div className="mt-1 px-1 text-[10px]" style={{color: 'var(--ok)'}}>{renderNotice}</div> : null}
          <div className="divider my-2" />
          <div className="mb-1 px-1 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>待处理反馈 {pendingFeedback.length}</div>
          {pendingFeedback.length === 0 ? (
            <div className="px-1 py-1 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>没有待处理反馈</div>
          ) : (
            <div className="flex flex-col gap-1">
              {pendingFeedback.slice(0, 6).map((item) => (
                <div key={item.id} className="rounded-md p-2 text-[10.5px]" style={{background: 'var(--ink-850)', border: '1px solid var(--line)'}}>
                  <div className="mb-1 flex items-center gap-1">
                    <span className="badge">{item.category}</span>
                    <span className="ml-auto text-[9px]" style={{color: 'var(--fg-dim)'}}>{item.created_at.slice(11, 16)}</span>
                  </div>
                  <div style={{color: 'var(--fg-muted)'}}>{item.feedback}</div>
                  <button type="button" className="btn mt-1 w-full" onClick={() => void resolveFeedback(item.id)} disabled={busy}>
                    标记已处理
                  </button>
                </div>
              ))}
            </div>
          )}
        </aside>
      </div>

      <div className="border-t p-3" style={{borderColor: 'var(--line)', background: 'var(--ink-900)'}}>
        <div className="flex items-center gap-2">
          <MessageSquare size={12} style={{color: 'var(--fg-dim)'}} />
          <span className="text-[11px]" style={{color: 'var(--fg-muted)'}}>把修改意见写下来，AI 下一轮会读取</span>
          <select className="field ml-auto w-[110px]" value={category} onChange={(event) => setCategory(event.target.value)}>
            <option value="style">风格</option>
            <option value="layout">布局</option>
            <option value="motion">动效</option>
            <option value="narrative">叙事</option>
            <option value="quality">质量</option>
            <option value="data">数据</option>
          </select>
          <button type="button" className="btn" onClick={() => void submitFeedback()} disabled={busy || !feedbackText.trim()}>
            <Send size={11} />记录反馈
          </button>
        </div>
        <textarea
          className="field mt-2 h-14 w-full resize-none"
          placeholder="例如：第 3 个镜头标题太小，转场太快；音乐在 8 秒处断一下。"
          value={feedbackText}
          onChange={(event) => setFeedbackText(event.target.value)}
        />
        {error ? <div className="mt-1 text-[11px]" style={{color: 'var(--err)'}}>{error}</div> : null}
        {notice ? <div className="mt-1 text-[11px]" style={{color: 'var(--ok)'}}>{notice}</div> : null}
      </div>
    </div>
  );
}

type StoryboardScene = {time: string; scene: string; description: string};
type StoryboardSection = {title: string; body: string};

function parseStoryboard(markdown: string): {scenes: StoryboardScene[]; sections: StoryboardSection[]} {
  const scenes: StoryboardScene[] = [];
  const sections: StoryboardSection[] = [];
  let tableSeen = false;
  let current: StoryboardSection | null = null;
  for (const line of markdown.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (trimmed.startsWith('|')) {
      const cells = trimmed.split('|').slice(1, -1).map((cell) => cell.trim());
      if (cells.length >= 3 && cells[0].toLowerCase().includes('time')) { tableSeen = true; continue; }
      if (tableSeen && cells.every((cell) => /^:?-{2,}:?$/.test(cell))) continue;
      if (tableSeen && cells.length >= 3) {
        scenes.push({time: cells[0], scene: cells[1], description: cells.slice(2).join(' ')});
        continue;
      }
    } else {
      tableSeen = false;
    }
    if (trimmed.startsWith('## ')) {
      if (current) sections.push(current);
      current = {title: trimmed.slice(3).trim(), body: ''};
      continue;
    }
    if (current) current.body += line + '\n';
  }
  if (current) sections.push(current);
  return {scenes, sections};
}

export function CodeStoryboardPane({storyboard}: {storyboard?: string}) {
  const [selected, setSelected] = useState(0);
  if (!storyboard) {
    return <div className="flex h-full items-center justify-center text-[12px]" style={{color: 'var(--fg-dim)'}}>还没有 STORYBOARD.md</div>;
  }
  const {scenes, sections} = parseStoryboard(storyboard);
  if (!scenes.length) {
    return (
      <div className="h-full overflow-auto p-5">
        <pre className="whitespace-pre-wrap font-sans text-[12px] leading-6" style={{color: 'var(--fg-muted)'}}>{storyboard}</pre>
      </div>
    );
  }
  const active = scenes[Math.min(selected, scenes.length - 1)];
  return (
    <div className="grid h-full min-h-0" style={{gridTemplateColumns: '280px 1fr'}}>
      <aside className="flex min-h-0 flex-col" style={{borderRight: '1px solid var(--line)', background: 'var(--ink-900)'}}>
        <div className="px-3 py-2 text-[9.5px]" style={{color: 'var(--fg-dim)'}}>↑/↓ 切换场景 · Enter 定位</div>
        <div
          role="listbox"
          aria-label="场景列表"
          tabIndex={0}
          className="min-h-0 flex-1 overflow-auto px-2 pb-2"
          onKeyDown={(event) => {
            if (!scenes.length) return;
            if (event.key === 'ArrowDown') { event.preventDefault(); setSelected((value) => Math.min(scenes.length - 1, value + 1)); }
            else if (event.key === 'ArrowUp') { event.preventDefault(); setSelected((value) => Math.max(0, value - 1)); }
            else if (event.key === 'Home') { event.preventDefault(); setSelected(0); }
            else if (event.key === 'End') { event.preventDefault(); setSelected(scenes.length - 1); }
          }}
        >
          {scenes.map((scene, index) => (
            <button
              key={`${scene.time}-${index}`}
              type="button"
              role="option"
              aria-selected={selected === index}
              data-active={selected === index}
              className="mb-1 w-full rounded-md border px-2 py-2 text-left"
              style={{borderColor: selected === index ? 'var(--accent-line)' : 'transparent', background: selected === index ? 'var(--accent-soft)' : 'transparent', contentVisibility: 'auto', containIntrinsicSize: '52px'}}
              onClick={() => setSelected(index)}
            >
              <span className="flex items-center gap-2">
                <span className="badge">{scene.time}</span>
                <span className="truncate text-[11.5px] font-medium">{scene.scene}</span>
              </span>
              <span className="mt-1 block truncate text-[10px]" style={{color: 'var(--fg-dim)'}}>{scene.description}</span>
            </button>
          ))}
        </div>
      </aside>
      <section className="min-h-0 overflow-auto p-5">
        <div className="flex items-center gap-2">
          <span className="badge badge-accent">{active.time}</span>
          <strong className="text-[15px]">{active.scene}</strong>
          <span className="ml-auto text-[10px]" style={{color: 'var(--fg-dim)'}}>{selected + 1}/{scenes.length}</span>
        </div>
        <p className="mt-3 text-[12.5px] leading-6" style={{color: 'var(--fg-muted)'}}>{active.description}</p>
        {sections.map((section) => (
          <div key={section.title} className="mt-5 border-t pt-3" style={{borderColor: 'var(--line)'}}>
            <div className="text-[10px] uppercase tracking-wide" style={{color: 'var(--fg-dim)'}}>{section.title}</div>
            <pre className="mt-2 whitespace-pre-wrap font-sans text-[11.5px] leading-6" style={{color: 'var(--fg-muted)'}}>{section.body.trim()}</pre>
          </div>
        ))}
      </section>
    </div>
  );
}
