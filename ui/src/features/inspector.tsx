import {useMemo} from 'react';
import {AlertTriangle, CheckCircle2, ShieldCheck} from 'lucide-react';
import type {ConstraintPayload, ProjectDetail, Recipe} from '../types';
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
}: {
  detail: ProjectDetail | null;
  constraints: ConstraintPayload | null;
}) {
  const recipe = detail?.project.recipe;
  const isCode = detail?.project.mode === 'code';

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
    const determinismStale = detail.project.determinism_status === 'passed' && detail.project.determinism_hash !== detail.project.source_hash;
    const determinismTone = detail.project.determinism_status === 'passed' && !determinismStale ? 'ok' : 'warn';
    const determinismLabel = determinismStale ? 'stale（待重检）' : detail.project.determinism_status ?? '未检查';
    return (
      <div className="flex h-full flex-col overflow-auto pb-3">
        <SectionLabel>代码项目</SectionLabel>
        <dl className="px-3">
          <Kv label="风格包" value={<span className="font-mono text-[11px]">{detail.project.style_pack ?? '—'}</span>} />
          <Kv label="版本" value={<span className="font-mono text-[10.5px]">{detail.project.style_version ?? '—'}</span>} />
          <Kv label="渲染器" value={<Badge tone="accent">{detail.project.renderer ?? 'browser-code'}</Badge>} />
          <Kv label="源码目录" value={<span className="font-mono text-[10px]">{detail.project.source_dir ?? '—'}</span>} />
          <Kv label="入口" value={<span className="font-mono text-[10.5px]">{detail.project.entry ?? 'render.mjs'}</span>} />
          <Kv label="修订" value={`r${detail.project.revision}`} />
          <Kv label="预览" value={<Badge tone={detail.project.preview_status === 'ready' ? 'ok' : 'warn'}>{detail.project.preview_status ?? '未生成'}</Badge>} />
          <Kv label="确定性" value={<Badge tone={determinismTone}>{determinismLabel}</Badge>} />
        </dl>
        <div className="divider my-2" />
        <SectionLabel>产物</SectionLabel>
        <dl className="px-3">
          <Kv label="审查图" value={`${previews.length} 个`} />
          <Kv label="成片" value={`${videos.length} 个`} />
          <Kv label="时长" value={`${detail.project.duration_seconds ?? 0}s`} />
        </dl>
        <div className="divider my-2" />
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


