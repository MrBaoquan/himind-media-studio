import {Activity, Boxes, ShieldCheck} from 'lucide-react';
import type {ConstraintPayload, ProjectDetail} from '../types';

export function StatusBar({
  detail,
  constraints,
  hosted,
}: {
  detail: ProjectDetail | null;
  constraints: ConstraintPayload | null;
  hosted: boolean;
}) {
  const job = detail?.jobs?.[0];
  const artifacts = detail?.artifacts?.length ?? 0;
  const summary = constraints?.summary;

  return (
    <footer
      className="flex items-center gap-4 px-3 text-[10.5px]"
      style={{borderTop: '1px solid var(--line)', background: 'var(--ink-900)', color: 'var(--fg-dim)'}}
    >
      <span className="flex items-center gap-1.5">
        <Activity size={11} />
        {job ? `${job.status}${job.stage ? ' · ' + job.stage : ''} ${job.progress}%` : '无任务'}
      </span>
      <span className="flex items-center gap-1.5">
        <Boxes size={11} />
        产物 {artifacts}
      </span>
      <span className="flex items-center gap-1.5">
        <ShieldCheck size={11} />
        约束 {summary ? `${summary.resolved}/${summary.rules}` : '—'}
      </span>
      <span className="ml-auto flex items-center gap-1.5">
        <span
          className="inline-block h-1.5 w-1.5 rounded-full"
          style={{background: hosted ? 'var(--ok)' : 'var(--warn)'}}
        />
        {hosted ? '已连接宿主' : '浏览器预览（mock 数据）'}
      </span>
    </footer>
  );
}
