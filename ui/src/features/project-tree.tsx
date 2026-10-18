import {Film} from 'lucide-react';
import type {ProjectSummary} from '../types';
import {Empty, cx} from '../components/ui';

export function ProjectTree({
  projects,
  selectedId,
  onSelect,
  query,
  onQuery,
  loading,
}: {
  projects: ProjectSummary[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  query: string;
  onQuery: (value: string) => void;
  loading: boolean;
}) {
  const filtered = projects.filter((p) => {
    if (!query.trim()) return true;
    const q = query.trim().toLowerCase();
    return p.name.toLowerCase().includes(q) || p.id.toLowerCase().includes(q) || (p.template_id ?? ').includes(q) || (p.style_pack ?? ').includes(q);
  });

  return (
    <div className="flex h-full flex-col">
      <div className="px-3 pb-2 pt-3">
        <input
          className="field"
          placeholder="搜索项目 / 模板"
          value={query}
          onChange={(e) => onQuery(e.target.value)}
        />
      </div>
      <div className="min-h-0 flex-1 overflow-auto px-2 pb-3">
        {loading ? (
          <Empty title="正在读取项目…" />
        ) : filtered.length === 0 ? (
          <Empty title={query ? '没有匹配的项目' : '还没有项目'} hint={query ? '换个关键词试试' : '用「新建项目」开始'} />
        ) : (
          <div className="flex flex-col gap-0.5">
            {filtered.map((p) => (
              <button
                key={p.id}
                type="button"
                className={cx('nav-item')}
                data-active={p.id === selectedId}
                onClick={() => onSelect(p.id)}
                title={`${p.id} · ${p.mode === 'code' ? p.style_pack : p.template_id}`}
              >
                <Film size={13} className="shrink-0 opacity-70" />
                <span className="min-w-0 flex-1 truncate">{p.name}</span>
                {p.mode === 'code' ? (
                  <span className="shrink-0 rounded px-1 py-0.5 text-[9px]" style={{background: 'var(--accent-soft)', color: 'var(--fg-muted)'}}>
                    代码
                  </span>
                ) : null}
                <span className="shrink-0 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
                  r{p.revision}
                </span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
