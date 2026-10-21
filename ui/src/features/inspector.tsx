import {useMemo} from 'react';
import {useEffect, useState} from 'react';
import {AlertTriangle, CheckCircle2, ShieldCheck} from 'lucide-react';
import {callCapability, hostInvoke} from '../api/capability';
import type {ConstraintPayload, FeedbackEntry, Job, ProjectDetail, Recipe} from '../types';
import {Badge, Kv, SectionLabel} from '../components/ui';

const stageNames: Record<string, string> = {
  input: '输入',
  recipe: '配方',
  preview: '预览',
  geometry: '几何',
  layout: '布局',
  render: '渲染',
  output: '产物',
  extension: '扩展',
};

function levelTone(level: string): 'err' | 'warn' | 'accent' | 'ok' {
  if (level === 'error') return 'err';
  if (level === 'warn') return 'warn';
  if (level === 'gate') return 'accent';
  return 'ok';
}

/** 用类型无关的方式描述一个数据字段，避免外壳认识任何题材结构。 */
function describeValue(value: unknown): string {
  if (Array.isArray(value)) return `${value.length} 项`;
  if (value && typeof value === 'object') return `${Object.keys(value as object).length} 个字段`;
  if (value === null || value === undefined) return '—';
  const text = String(value);
  return text.length > 18 ? `${text.slice(0, 18)}…` : text;
}

export function Inspector({
  detail,
  constraints,
  workspace,
  onRefresh,
}: {
  detail: ProjectDetail | null;
  constraints: ConstraintPayload | null;
  workspace: string;
  onRefresh: () => void | Promise<void>;
}) {
  const recipe = detail?.project.recipe;
  const isCode = detail?.project.mode === 'code';
  const [showTech, setShowTech] = useState(false);
  const [showAllChecks, setShowAllChecks] = useState(false);

  const grouped = useMemo(() => {
    const map = new Map<string, number>();
    for (const rule of constraints?.resolved ?? []) {
      map.set(rule.stage, (map.get(rule.stage) ?? 0) + 1);
    }
    return [...map.entries()];
  }, [constraints]);

  if (!recipe && !isCode) {
    return (
      <div className="flex h-full items-center justify-center px-6 text-center text-[11.5px]" style={{color: 'var(--fg-dim)'}}>
        选择一个项目查看属性与质量
      </div>
    );
  }

  const summary = constraints?.summary;
  const rules = constraints?.resolved ?? [];

  if (isCode && detail) {
    const previews = (detail.artifacts ?? []).filter((item) => item.kind.startsWith('preview/'));
    const videos = (detail.artifacts ?? []).filter((item) => item.kind.startsWith('video/'));
    const qualityArtifact = [...(detail.artifacts ?? [])].filter((item) => item.quality_report).sort((left, right) => (right.created_at || '').localeCompare(left.created_at || ''))[0];
    const qualityChecks: any[] = qualityArtifact?.quality_report?.checks ?? [];
    const qualityFindings: any[] = qualityArtifact?.quality_report?.findings ?? [];
    const determinismStale = detail.project.determinism_status === 'passed' && detail.project.determinism_hash !== detail.project.source_hash;
    const determinismTone = detail.project.determinism_status === 'passed' && !determinismStale ? 'ok' : 'warn';
    const determinismLabel = determinismStale ? 'stale（待重检）' : detail.project.determinism_status ?? '未检查';
    return (
      <div className="flex h-full flex-col overflow-auto pb-3">
        <SectionLabel>代码项目</SectionLabel>
        <div className="px-3 pb-2">
          <div className="flex flex-wrap items-center gap-1.5 text-[11px]" style={{color: 'var(--fg-dim)'}}>
            <span className="font-mono">{detail.project.style_pack ?? '—'}</span>
            <span>·</span>
            <span>r{detail.project.revision}</span>
            <span>·</span>
            <span>{detail.project.renderer ?? 'browser-code'}</span>
          </div>
          <dl className="mt-2">
            <Kv label="预览" value={<Badge tone={detail.project.preview_status === 'ready' ? 'ok' : 'warn'}>{detail.project.preview_status ?? '未生成'}</Badge>} />
            <Kv label="确定性" value={<Badge tone={determinismTone}>{determinismLabel}</Badge>} />
            <Kv label="审查图" value={`${previews.length} 个`} />
            <Kv label="成片" value={`${videos.length} 个`} />
            <Kv label="时长" value={`${detail.project.duration_seconds ?? 0}s`} />
          </dl>
          <button type="button" className="btn btn-ghost mt-2 w-full" style={{justifyContent: 'space-between'}} onClick={() => setShowTech((value) => !value)}>
            <span>技术详情</span>
            <span style={{color: 'var(--fg-dim)'}}>{showTech ? '收起' : '展开'}</span>
          </button>
          {showTech ? (
            <dl className="mt-1 rounded-md border px-2 py-1" style={{borderColor: 'var(--line)'}}>
              <Kv label="渲染器" value={<Badge tone="accent">{detail.project.renderer ?? 'browser-code'}</Badge>} />
              <Kv label="源码目录" value={<span className="font-mono text-[11px]">{detail.project.source_dir ?? '—'}</span>} />
              <Kv label="入口" value={<span className="font-mono text-[11px]">{detail.project.entry ?? 'render.mjs'}</span>} />
              {detail.project.source_creative_id ? <Kv label="来源创意包" value={<span className="font-mono text-[11px]">{detail.project.source_creative_id}@r{detail.project.source_creative_revision ?? '?'}</span>} /> : null}
            </dl>
          ) : null}
        </div>
        <div className="divider my-2" />
        <SectionLabel
          trailing={qualityArtifact ? (
            <span className="flex items-center gap-2">
              <span className="text-[11px]" style={{color: 'var(--fg-dim)'}}>{qualityChecks.filter((check: any) => check.status === 'passed').length} 通过 · {qualityChecks.filter((check: any) => check.status === 'skipped').length} 跳过</span>
              <Badge tone={qualityArtifact.quality_status === 'passed' ? 'ok' : qualityArtifact.quality_status === 'failed' ? 'err' : 'warn'}>
                {qualityArtifact.quality_status ?? '未检查'}
              </Badge>
            </span>
          ) : null}
        >
          体检报告
        </SectionLabel>
        {qualityArtifact ? (
          <div className="space-y-1.5 px-3 pb-2">
            {(showAllChecks ? qualityChecks : qualityChecks.slice(0, 3)).map((check: any) => {
              const tone = check.status === 'passed' ? 'var(--ok)' : check.status === 'failed' ? 'var(--err)' : 'var(--warn)';
              return (
                <div key={check.id} className="flex items-start gap-2 text-[10.5px]">
                  {check.status === 'passed' ? <CheckCircle2 size={12} style={{color: tone, marginTop: 2}} /> : <AlertTriangle size={12} style={{color: tone, marginTop: 2}} />}
                  <div className="min-w-0 flex-1">
                    <div className="truncate">{check.detail ?? check.id}</div>
                    <div className="text-[9.5px]" style={{color: 'var(--fg-dim)'}}>{check.id} · {check.status}</div>
                  </div>
                </div>
              );
            })}
            {qualityChecks.length > 3 ? (
              <button type="button" className="btn btn-ghost w-full" onClick={() => setShowAllChecks((value) => !value)}>
                {showAllChecks ? '收起明细' : `查看全部 ${qualityChecks.length} 项`}
              </button>
            ) : null}
            {qualityFindings.map((finding: any, index: number) => (
              <div key={`${finding.code}-${index}`} className="rounded-md border px-2 py-1.5" style={{borderColor: 'var(--err-soft)', background: 'var(--err-soft)'}}>
                <div className="text-[10px] font-semibold" style={{color: 'var(--err)'}}>{finding.code}</div>
                <div className="mt-0.5 text-[10.5px] leading-4">{finding.message}</div>
                {finding.remediation ? <div className="mt-1 text-[9.5px] leading-4" style={{color: 'var(--fg-dim)'}}>{finding.remediation}</div> : null}
              </div>
            ))}
            {!qualityChecks.length && !qualityFindings.length ? (
              <div className="text-[10.5px]" style={{color: 'var(--fg-dim)'}}>这份产物没有结构化体检明细。</div>
            ) : null}
          </div>
        ) : (
          <div className="px-3 pb-2 text-[11px]" style={{color: 'var(--fg-dim)'}}>
            还没有体检报告；生成审查图或成片后显示。
          </div>
        )}
        <SectionLabel
          trailing={summary ? <span className="flex items-center gap-1"><ShieldCheck size={11} />{summary.resolved}/{summary.rules}</span> : null}
        >
          质量门禁
        </SectionLabel>
        {summary ? (
          <div className="flex flex-wrap gap-1.5 px-3 pb-2">
            {Object.entries(summary.by_level).map(([level, count]) => (
              <Badge key={level} tone={levelTone(level)}>{level} {count}</Badge>
            ))}
          </div>
        ) : null}
        <div className="px-3 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
          修改代码后先 project.sync，再生成审查图；质量门禁通过后才能最终渲染。
        </div>
        <HandoffSummary detail={detail} />
        <SubtitleTrack workspace={workspace} detail={detail} />
        <JobCenter workspace={workspace} detail={detail} onRefresh={onRefresh} />
        <FeedbackInbox workspace={workspace} detail={detail} onRefresh={onRefresh} />
      </div>
    );
  }

  const templateRecipe = recipe as Recipe;
  const dataEntries = Object.entries(templateRecipe.data ?? {});

  return (
    <div className="flex h-full flex-col overflow-auto pb-3">
      <SectionLabel>模板</SectionLabel>
      <dl className="px-3">
        <Kv label="模板" value={<span className="font-mono text-[11px]">{templateRecipe.template_id}</span>} />
        <Kv label="版本" value={<span className="font-mono text-[10.5px]">{templateRecipe.template_version ?? '—'}</span>} />
        <Kv label="Plan" value={<span className="font-mono text-[10.5px]">{detail?.plan?.version ?? '未生成'}</span>} />
        <Kv label="风格" value={<span className="font-mono text-[11px]">{templateRecipe.style_id}</span>} />
        <Kv label="适配器" value={<Badge tone="accent">{templateRecipe.adapter ?? 'remotion'}</Badge>} />
        <Kv label="修订" value={`r${detail?.project.revision ?? 1}`} />
        {detail?.project.source_creative_id ? <Kv label="来源创意包" value={<span className="font-mono text-[10px]">{detail.project.source_creative_id}@r{detail.project.source_creative_revision ?? "?"}</span>} /> : null}
      </dl>

      <div className="divider my-2" />

      <SectionLabel>画布</SectionLabel>
      <dl className="px-3">
        <Kv label="时长" value={`${templateRecipe.duration_seconds}s`} />
        <Kv label="尺寸" value={`${templateRecipe.canvas.width}×${templateRecipe.canvas.height} @${templateRecipe.canvas.fps}fps`} />
        {templateRecipe.safe_area ? (
          <Kv
            label="安全区"
            value={`${templateRecipe.safe_area.top}/${templateRecipe.safe_area.right}/${templateRecipe.safe_area.bottom}/${templateRecipe.safe_area.left}`}
          />
        ) : null}
      </dl>

      <div className="divider my-2" />

      {/* 数据区只做"结构概览"：不认识 lines / items / media，只按类型呈现。 */}
      <SectionLabel trailing={<span>{dataEntries.length} 个键</span>}>数据</SectionLabel>
      {dataEntries.length === 0 ? (
        <div className="px-3 py-1 text-[11px]" style={{color: 'var(--fg-dim)'}}>该配方没有数据</div>
      ) : (
        <dl className="px-3">
          {dataEntries.slice(0, 12).map(([key, value]) => (
            <Kv key={key} label={key} value={<span className="font-mono text-[10.5px]">{describeValue(value)}</span>} />
          ))}
          {dataEntries.length > 12 ? (
            <div className="pt-1 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>还有 {dataEntries.length - 12} 个键…</div>
          ) : null}
        </dl>
      )}

      <div className="divider my-2" />

      <SectionLabel
        trailing={
          summary ? (
            <span className="flex items-center gap-1">
              <ShieldCheck size={11} />
              {summary.resolved}/{summary.rules}
            </span>
          ) : null
        }
      >
        质量门禁
      </SectionLabel>

      {!summary ? (
        <div className="px-3 py-2 text-[11.5px]" style={{color: 'var(--fg-dim)'}}>
          未读取到约束清单
        </div>
      ) : (
        <>
          <div className="flex flex-wrap gap-1.5 px-3 pb-2">
            {Object.entries(summary.by_level).map(([level, count]) => (
              <Badge key={level} tone={levelTone(level)}>
                {level} {count}
              </Badge>
            ))}
          </div>
          <div className="flex flex-wrap gap-1.5 px-3 pb-2">
            {grouped.map(([stage, count]) => (
              <span key={stage} className="badge">
                {stageNames[stage] ?? stage} {count}
              </span>
            ))}
          </div>
          <div className="flex flex-col gap-0.5 px-3">
            {rules.map((rule) => (
              <div key={rule.code} className="flex items-start gap-2 py-1" title={rule.detail ?? ''}>
                {rule.level === 'error' ? (
                  <AlertTriangle size={11} className="mt-[2px] shrink-0" style={{color: 'var(--err)'}} />
                ) : rule.level === 'warn' ? (
                  <AlertTriangle size={11} className="mt-[2px] shrink-0" style={{color: 'var(--warn)'}} />
                ) : (
                  <CheckCircle2 size={11} className="mt-[2px] shrink-0" style={{color: 'var(--ok)'}} />
                )}
                <span className="min-w-0 flex-1 truncate text-[11px]" style={{color: 'var(--fg-muted)'}}>
                  {rule.code}
                </span>
                <span className="shrink-0 text-[9.5px]" style={{color: 'var(--fg-dim)'}}>
                  {rule.enforced_by.kind}
                </span>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
function feedbackStatus(entry: FeedbackEntry): 'open' | 'resolved' | 'ignored' {
  if (entry.status === 'resolved' || entry.status === 'ignored' || entry.status === 'open') return entry.status;
  return entry.accepted ? 'resolved' : 'open';
}

function FeedbackInbox({workspace, detail, onRefresh}: {workspace: string; detail: ProjectDetail; onRefresh: () => void | Promise<void>}) {
  const [filter, setFilter] = useState<'all' | 'open' | 'resolved' | 'ignored'>('all');
  const [busyId, setBusyId] = useState('');
  const items = [...(detail.feedback ?? [])].reverse();
  const openCount = items.filter((entry) => feedbackStatus(entry) === 'open').length;
  const filtered = filter === 'all' ? items : items.filter((entry) => feedbackStatus(entry) === filter);
  async function resolve(id: string, status: 'resolved' | 'ignored') {
    setBusyId(id);
    try {
      await callCapability('media.workbench.feedback.resolve', {
        workspace_root: workspace,
        project_id: detail.project.id,
        feedback_id: id,
        status,
      });
      await onRefresh();
    } finally {
      setBusyId('');
    }
  }
  return (
    <>
      <div className="divider my-2" />
      <SectionLabel trailing={<span>{openCount} 待处理</span>}>反馈收件箱</SectionLabel>
      <div className="flex flex-wrap gap-1 px-3 pb-2">
        {(['all', 'open', 'resolved', 'ignored'] as const).map((value) => (
          <button key={value} type="button" className="badge" style={{cursor: 'pointer', background: filter === value ? 'var(--accent-soft)' : undefined, color: filter === value ? 'var(--accent)' : undefined}} onClick={() => setFilter(value)}>
            {value === 'all' ? '全部' : value === 'open' ? '待处理' : value === 'resolved' ? '已解决' : '已忽略'}
          </button>
        ))}
      </div>
      {filtered.length === 0 ? (
        <div className="px-3 pb-2 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>还没有反馈；在 AI 对话中提出修改后会出现在这里。</div>
      ) : filtered.map((entry) => {
        const status = feedbackStatus(entry);
        return (
          <div key={entry.id} className="mx-3 mb-1 rounded-md border px-2 py-1.5" style={{borderColor: 'var(--line)'}}>
            <div className="flex items-center gap-1.5">
              <Badge tone={status === 'open' ? 'warn' : status === 'resolved' ? 'ok' : 'accent'}>{status === 'open' ? '待处理' : status === 'resolved' ? '已解决' : '已忽略'}</Badge>
              <span className="text-[9px]" style={{color: 'var(--fg-dim)'}}>{entry.category} · {entry.created_at}</span>
            </div>
            <div className="mt-1 text-[10.5px] leading-4">{entry.feedback}</div>
            {entry.status_note ? <div className="mt-1 text-[9.5px]" style={{color: 'var(--fg-dim)'}}>{entry.status_note}</div> : null}
            {status === 'open' ? (
              <div className="mt-1 flex gap-1">
                <button type="button" className="btn h-6" disabled={busyId === entry.id} onClick={() => void resolve(entry.id, 'resolved')}>标记解决</button>
                <button type="button" className="btn btn-ghost h-6" disabled={busyId === entry.id} onClick={() => void resolve(entry.id, 'ignored')}>忽略</button>
              </div>
            ) : null}
          </div>
        );
      })}
    </>
  );
}
function JobCenter({workspace, detail, onRefresh}: {workspace: string; detail: ProjectDetail; onRefresh: () => void | Promise<void>}) {
  const [busyId, setBusyId] = useState('');
  const [message, setMessage] = useState<{id: string; text: string} | null>(null);
  const jobs = (detail.jobs ?? []).filter((job) => job.job_id);
  async function retry(job: Job) {
    if (!job.job_id.startsWith('render')) return;
    setBusyId(job.job_id);
    setMessage(null);
    try {
      const result = await callCapability<any>('media.workbench.render.start', {
        workspace_root: workspace,
        project_id: detail.project.id,
      });
      const blockers = Array.isArray(result?.blockers) ? result.blockers : [];
      const text = blockers.length ? `${blockers[0].code ?? 'BLOCKED'}: ${blockers[0].message ?? ''}` : result?.job?.job_id ? `已提交 ${result.job.job_id}` : '已提交';
      setMessage({id: job.job_id, text});
      await onRefresh();
    } catch (error) {
      setMessage({id: job.job_id, text: error instanceof Error ? error.message : String(error)});
    } finally {
      setBusyId('');
    }
  }
  return (
    <>
      <div className="divider my-2" />
      <SectionLabel trailing={<span>{jobs.length} 条</span>}>任务中心</SectionLabel>
      {jobs.length === 0 ? (
        <div className="px-3 pb-2 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>还没有任务；预览、渲染或音频生成后会出现在这里。</div>
      ) : jobs.slice(0, 6).map((job) => {
        const tone = job.status === 'completed' ? 'var(--ok)' : job.status === 'failed' || job.status === 'cancelled' ? 'var(--err)' : 'var(--accent)';
        return (
          <div key={job.job_id} className="mx-3 mb-1 rounded-md border px-2 py-1.5" style={{borderColor: 'var(--line)'}}>
            <div className="flex items-center gap-1.5">
              <Badge tone={job.status === 'completed' ? 'ok' : job.status === 'failed' || job.status === 'cancelled' ? 'err' : 'accent'}>{job.status}</Badge>
              <span className="min-w-0 truncate font-mono text-[9px]" style={{color: 'var(--fg-dim)'}}>{job.job_id}</span>
              <span className="ml-auto text-[9px]">{job.progress}%</span>
            </div>
            <div className="mt-1 h-[2px] w-full rounded" style={{background: 'var(--line)'}}>
              <div style={{width: `${Math.max(0, Math.min(100, Number(job.progress) || 0))}%`, height: 2, background: tone}} />
            </div>
            <div className="mt-1 truncate text-[9.5px]" style={{color: 'var(--fg-muted)'}}>{job.stage ?? 'task'} · {job.message ?? ''}</div>
            {job.status === 'failed' || job.status === 'cancelled' ? (
              <button type="button" className="btn mt-1 h-6" disabled={busyId === job.job_id} onClick={() => void retry(job)}>重新渲染</button>
            ) : null}
            {message?.id === job.job_id ? <div className="mt-1 text-[9.5px]" style={{color: 'var(--warn)'}}>{message.text}</div> : null}
          </div>
        );
      })}
    </>
  );
}
type SubtitleCue = {start: number; end: number; text: string};

function formatCueTime(value: number): string {
  const seconds = Math.max(0, Number(value) || 0);
  const minute = Math.floor(seconds / 60);
  const rest = (seconds % 60).toFixed(1);
  return `${minute}:${rest.padStart(4, '0')}`;
}

function SubtitleTrack({workspace, detail}: {workspace: string; detail: ProjectDetail}) {
  const [cues, setCues] = useState<SubtitleCue[]>([]);
  const [duration, setDuration] = useState(1);
  const [selected, setSelected] = useState(0);
  const [issues, setIssues] = useState<Array<{code: string; message: string}>>([]);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const result = await callCapability<any>('media.workbench.subtitle.list', {workspace_root: workspace, project_id: detail.project.id});
        if (cancelled) return;
        setCues(Array.isArray(result?.subtitles) ? result.subtitles : []);
        setDuration(Number(result?.duration_seconds) || Number(detail.project.duration_seconds) || 1);
        setIssues(Array.isArray(result?.issues) ? result.issues : []);
        setSelected(0);
      } catch (loadError) {
        if (!cancelled) setError(loadError instanceof Error ? loadError.message : String(loadError));
      }
    })();
    return () => { cancelled = true; };
  }, [workspace, detail.project.id, detail.project.duration_seconds]);

  const active = cues[Math.min(selected, Math.max(0, cues.length - 1))] ?? null;
  function updateActive(patch: Partial<SubtitleCue>) {
    setCues((current) => current.map((cue, index) => index === selected ? {...cue, ...patch} : cue));
  }
  async function save() {
    setBusy(true);
    setError('');
    try {
      const result = await callCapability<any>('media.workbench.subtitle.update', {workspace_root: workspace, project_id: detail.project.id, subtitles: cues});
      setIssues([]);
      if (result?.saved !== undefined) setError(`已保存 ${result.saved} 条字幕`);
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : String(saveError));
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <div className="divider my-2" />
      <SectionLabel trailing={<span>{cues.length} 条</span>}>字幕轨</SectionLabel>
      {error ? <div className="px-3 pb-1 text-[9.5px]" style={{color: error.startsWith('已保存') ? 'var(--ok)' : 'var(--err)'}}>{error}</div> : null}
      {issues.length ? (
        <div className="px-3 pb-1 text-[9.5px]" style={{color: 'var(--warn)'}}>{issues.slice(0, 2).map((issue) => issue.message).join('；')}</div>
      ) : null}
      {cues.length ? (
        <>
          <div className="relative mx-3 h-6 rounded" style={{background: 'var(--line)'}}>
            {cues.map((cue, index) => (
              <button
                key={`${cue.start}-${index}`}
                type="button"
                title={`${formatCueTime(cue.start)} ${cue.text}`}
                onClick={() => setSelected(index)}
                className="absolute top-0 h-6 rounded"
                style={{left: `${Math.max(0, Math.min(100, (cue.start / duration) * 100))}%`, width: `${Math.max(0.6, Math.min(100, ((cue.end - cue.start) / duration) * 100))}%`, background: selected === index ? 'var(--accent)' : 'var(--accent-soft)', border: '1px solid var(--accent-line)'}}
              />
            ))}
          </div>
          <div className="mx-3 mt-2 max-h-[120px] space-y-1 overflow-auto">
            {cues.map((cue, index) => (
              <button key={`row-${cue.start}-${index}`} type="button" className="flex w-full items-center gap-2 rounded px-1 py-0.5 text-left" style={{background: selected === index ? 'var(--accent-soft)' : 'transparent'}} onClick={() => setSelected(index)}>
                <span className="font-mono text-[9px]" style={{color: 'var(--fg-dim)'}}>{formatCueTime(cue.start)}</span>
                <span className="min-w-0 flex-1 truncate text-[10.5px]">{cue.text}</span>
              </button>
            ))}
          </div>
          {active ? (
            <div className="mx-3 mt-2 rounded-md border p-2" style={{borderColor: 'var(--line)'}}>
              <input className="field" value={active.text} onChange={(event) => updateActive({text: event.target.value})} aria-label="字幕文本" />
              <div className="mt-1 flex gap-1">
                <input className="field" type="number" step="0.1" value={active.start} onChange={(event) => updateActive({start: Number(event.target.value)})} aria-label="字幕开始" />
                <input className="field" type="number" step="0.1" value={active.end} onChange={(event) => updateActive({end: Number(event.target.value)})} aria-label="字幕结束" />
              </div>
              <button type="button" className="btn mt-1 h-6" disabled={busy} onClick={() => void save()}>保存字幕</button>
            </div>
          ) : null}
        </>
      ) : (
        <div className="px-3 pb-2 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>还没有字幕；生成旁白或对齐字幕后显示。</div>
      )}
    </>
  );
}
function HandoffSummary({detail}: {detail: ProjectDetail}) {
  const [notice, setNotice] = useState('');
  const latest = [...(detail.artifacts ?? [])].sort((left, right) => (right.created_at || '').localeCompare(left.created_at || ''))[0];
  const openFeedback = (detail.feedback ?? []).filter((entry) => (entry.status ?? (entry.accepted ? 'resolved' : 'open')) === 'open');
  const lines = [
    '## 交接摘要',
    `- 项目：${detail.project.name}（${detail.project.id}） r${detail.project.revision}`,
    detail.project.source_creative_id ? `- 来源创意包：${detail.project.source_creative_id}@r${detail.project.source_creative_revision ?? '?'}` : '',
    `- 风格：${detail.project.style_pack ?? '—'}@${detail.project.style_version ?? '—'} · ${detail.project.renderer ?? '—'}`,
    `- 预览：${detail.project.preview_status ?? '未生成'} · 确定性：${detail.project.determinism_status ?? '未检查'}`,
    `- 产物：${(detail.artifacts ?? []).length} 个 · 最新 QA：${latest?.quality_status ?? '—'}`,
    `- 待处理反馈：${openFeedback.length}`,
  ].filter(Boolean);
  const pending = [
    openFeedback.length ? `${openFeedback.length} 条反馈待处理` : '',
    detail.project.preview_status !== 'ready' ? '预览未就绪' : '',
    detail.project.determinism_status !== 'passed' ? '确定性未通过' : '',
    latest?.quality_status === 'failed' ? '最新产物 QA 失败' : '',
  ].filter(Boolean);
  async function copy() {
    try {
      await navigator.clipboard.writeText(lines.join('\n'));
      setNotice('已复制交接摘要');
    } catch {
      setNotice('复制失败，请手动选择文本');
    }
  }
  async function openAi() {
    try {
      await hostInvoke('show_main_window');
    } catch {
      setNotice('请切换到 HiMind Agent 主窗口的 AI 对话');
    }
  }
  return (
    <>
      <div className="divider my-2" />
      <SectionLabel trailing={<span>{pending.length ? `${pending.length} 待确认` : '已就绪'}</span>}>交接摘要</SectionLabel>
      <pre className="mx-3 max-h-[140px] overflow-auto whitespace-pre-wrap rounded-md border p-2 font-sans text-[9.5px] leading-4" style={{borderColor: 'var(--line)', color: 'var(--fg-muted)'}}>{lines.join('\n')}</pre>
      {pending.length ? <div className="px-3 pt-1 text-[9.5px]" style={{color: 'var(--warn)'}}>{pending.join(' · ')}</div> : null}
      <div className="flex gap-1 px-3 pt-2">
        <button type="button" className="btn h-6" onClick={() => void copy()}>复制摘要</button>
        <button type="button" className="btn btn-ghost h-6" onClick={() => void openAi()}>打开 AI 对话</button>
      </div>
      {notice ? <div className="px-3 pt-1 text-[9px]" style={{color: 'var(--fg-dim)'}}>{notice}</div> : null}
    </>
  );
}
