import {AlertTriangle, CheckCircle2, Clapperboard, Database, Route, Target} from 'lucide-react';
import type {VideoPlan} from '../types';
import {Badge, Empty, SectionLabel} from '../components/ui';

function tags(values?: string[]) {
  if (!values || values.length === 0) return null;
  return (
    <div className="flex flex-wrap gap-1.5">
      {values.map((value) => (
        <span key={value} className="badge">{value}</span>
      ))}
    </div>
  );
}

function kv(label: string, value?: string | number) {
  if (value === undefined || value === null || value === '') return null;
  return (
    <div className="flex items-baseline justify-between gap-3 py-0.5">
      <span className="text-[10.5px]" style={{color: 'var(--fg-dim)'}}>{label}</span>
      <span className="text-right text-[11px]" style={{color: 'var(--fg-muted)'}}>{value}</span>
    </div>
  );
}

export function PlanPane({plan}: {plan?: VideoPlan | null}) {
  if (!plan) {
    return <Empty title="还没有可审阅的 Plan" hint="让 AI 先产出叙事计划与分镜计划" />;
  }
  const shots = plan.storyboard?.shots ?? [];
  const narrative = plan.narrative ?? {};
  const brief = plan.brief ?? {};
  const presentation = plan.presentation ?? {};
  const totalSeconds = shots.reduce((sum, shot) => sum + (Number(shot.duration_seconds) || 0), 0);

  return (
    <div className="h-full overflow-auto px-4 pb-8">
      <div className="flex items-start gap-3 py-4">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg" style={{background: 'var(--accent-soft)', color: 'var(--accent)'}}>
          <Target size={17} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="text-[14px] font-semibold">{presentation.title || narrative.thesis || '未命名计划'}</div>
          <div className="mt-1 text-[11px]" style={{color: 'var(--fg-dim)'}}>
            {plan.version} · {narrative.structure || 'single-shot'} · {shots.length} 镜头 · {totalSeconds}s
          </div>
        </div>
        <Badge tone="accent">{presentation.style_id || '未指定风格'}</Badge>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <section className="surface p-3">
          <SectionLabel>Brief</SectionLabel>
          <div className="text-[12px] leading-relaxed">{brief.goal || '—'}</div>
          <div className="mt-2 space-y-1">
            {kv('受众', brief.audience)}
            {kv('平台', brief.platform)}
            {kv('语气', brief.tone)}
            {kv('目标时长', brief.duration_seconds ? `${brief.duration_seconds}s` : undefined)}
          </div>
          {tags(brief.must_include) ? <div className="mt-2">{tags(brief.must_include)}</div> : null}
        </section>

        <section className="surface p-3">
          <SectionLabel>叙事</SectionLabel>
          <div className="text-[12px] leading-relaxed">{narrative.thesis || '—'}</div>
          <div className="mt-2 flex items-center gap-2 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
            <Route size={12} />
            {narrative.structure || 'single-shot'}
          </div>
          {narrative.required_data?.length ? (
            <div className="mt-3">
              <div className="mb-1 flex items-center gap-1 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
                <Database size={11} /> 所需数据
              </div>
              {tags(narrative.required_data)}
            </div>
          ) : null}
        </section>
      </div>

      {(narrative.key_points?.length || narrative.constraints?.length || narrative.risk_notes?.length) ? (
        <section className="surface mt-3 p-3">
          <SectionLabel>论证与约束</SectionLabel>
          <div className="space-y-1.5">
            {(narrative.key_points ?? []).map((point) => (
              <div key={point.claim} className="flex items-start gap-2 text-[11.5px]">
                <CheckCircle2 size={12} className="mt-[2px] shrink-0" style={{color: 'var(--ok)'}} />
                <span className="min-w-0 flex-1">{point.claim}</span>
                {point.evidence_ref ? <span className="shrink-0 font-mono text-[9.5px]" style={{color: 'var(--fg-dim)'}}>{point.evidence_ref}</span> : null}
              </div>
            ))}
            {(narrative.constraints ?? []).map((item) => (
              <div key={item} className="text-[10.5px]" style={{color: 'var(--fg-muted)'}}>· {item}</div>
            ))}
            {(narrative.risk_notes ?? []).map((item) => (
              <div key={item} className="flex items-start gap-2 text-[10.5px]" style={{color: 'var(--warn)'}}>
                <AlertTriangle size={11} className="mt-[2px] shrink-0" />
                <span>{item}</span>
              </div>
            ))}
          </div>
        </section>
      ) : null}

      <section className="surface mt-3 p-3">
        <SectionLabel trailing={<span>{totalSeconds}s</span>}>Storyboard</SectionLabel>
        <div className="space-y-2">
          {shots.map((shot, index) => (
            <div key={`${shot.shot}-${index}`} className="rounded-lg p-2.5" style={{background: 'var(--ink-950)', border: '1px solid var(--line)'}}>
              <div className="flex items-center gap-2">
                <span className="flex h-5 w-5 items-center justify-center rounded text-[10px] font-semibold" style={{background: 'var(--accent-soft)', color: 'var(--accent)'}}>
                  {String(index + 1).padStart(2, '0')}
                </span>
                <Clapperboard size={12} style={{color: 'var(--fg-dim)'}} />
                <span className="font-mono text-[11px]">{shot.shot}</span>
                <span className="badge">{shot.role || 'shot'}</span>
                <span className="ml-auto text-[10.5px]" style={{color: 'var(--fg-dim)'}}>{shot.duration_seconds}s</span>
              </div>
              <div className="mt-1.5 flex flex-wrap gap-2 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
                {shot.data_binding ? <span>data: {shot.data_binding}</span> : null}
                {shot.title ? <span>标题: {shot.title}</span> : null}
              </div>
              {shot.rationale ? <div className="mt-1 text-[10.5px] leading-relaxed" style={{color: 'var(--fg-muted)'}}>{shot.rationale}</div> : null}
            </div>
          ))}
          {shots.length === 0 ? <div className="text-[11px]" style={{color: 'var(--fg-dim)'}}>没有镜头</div> : null}
        </div>
      </section>

      <section className="surface mt-3 p-3">
        <SectionLabel>Presentation</SectionLabel>
        <div className="grid gap-x-6 md:grid-cols-2">
          {kv('标题', presentation.title)}
          {kv('副标题', presentation.subtitle)}
          {kv('来源', presentation.source)}
          {kv('转场帧数', presentation.transition_frames)}
        </div>
      </section>
    </div>
  );
}
