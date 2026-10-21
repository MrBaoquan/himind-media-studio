import {useMemo, useState} from 'react';
import {ChevronLeft, ChevronRight, Edit3, Plus, Search, Trash2, Wand2} from 'lucide-react';
import type {ShotProduceAsset, ShotProvider} from '../types';

export interface ShotTableProps {
  shots: any[];
  providers: ShotProvider[];
  providerLabel: (provider: ShotProvider) => string;
  assets: Record<string, ShotProduceAsset>;
  onPatch: (index: number, key: string, value: any) => void;
  onBulkPatch: (indices: number[], key: string, value: any) => void;
  onDelete: (indices: number[]) => void;
  onAdd: () => void;
  onOpen: (index: number) => void;
  onProduce: (indices: number[]) => void;
}

function shotKey(shot: any, index: number): string {
  return String(shot?.id || `index:${index}`);
}

export function ShotTable({shots, providers, providerLabel, assets, onPatch, onBulkPatch, onDelete, onAdd, onOpen, onProduce}: ShotTableProps) {
  const [query, setQuery] = useState('');
  const [providerFilter, setProviderFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [pageSize, setPageSize] = useState(25);
  const [page, setPage] = useState(0);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [bulkProvider, setBulkProvider] = useState('');
  const [bulkStatus, setBulkStatus] = useState('');

  const indexed = useMemo(() => shots.map((shot, index) => ({shot, index})), [shots]);
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return indexed.filter(({shot}) => {
      const text = [shot.id, shot.shot, shot.beat_id, shot.visual, shot.text, shot.prompt, shot.technique].map((value) => String(value ?? '')).join(' ').toLowerCase();
      const provider = String(shot.provider ?? '');
      const status = String(shot.status ?? 'planned');
      return (!q || text.includes(q)) && (providerFilter === 'all' || provider === providerFilter) && (statusFilter === 'all' || status === statusFilter);
    });
  }, [indexed, query, providerFilter, statusFilter]);
  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
  const currentPage = Math.min(page, pageCount - 1);
  const pageItems = filtered.slice(currentPage * pageSize, (currentPage + 1) * pageSize);
  const pageKeys = pageItems.map(({shot, index}) => shotKey(shot, index));
  const allPageSelected = pageKeys.length > 0 && pageKeys.every((key) => selected.has(key));
  const selectedIndices = indexed.filter(({shot, index}) => selected.has(shotKey(shot, index))).map(({index}) => index);

  function toggle(key: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key); else next.add(key);
      return next;
    });
  }
  function togglePage() {
    setSelected((prev) => {
      const next = new Set(prev);
      if (allPageSelected) pageKeys.forEach((key) => next.delete(key));
      else pageKeys.forEach((key) => next.add(key));
      return next;
    });
  }
  function applyBulk(key: 'provider' | 'status') {
    if (!selectedIndices.length) return;
    const value = key === 'provider' ? bulkProvider : bulkStatus;
    if (!value) return;
    onBulkPatch(selectedIndices, key, value);
    setSelected(new Set());
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <label className="relative min-w-[180px] flex-1">
          <Search size={12} className="absolute left-2 top-1/2 -translate-y-1/2" style={{color: 'var(--fg-dim)'}} />
          <input className="field-input pl-7" value={query} onChange={(event) => { setQuery(event.target.value); setPage(0); }} placeholder="搜索镜头 ID / 画面 / 提示词" />
        </label>
        <select className="field-input w-[150px]" value={providerFilter} onChange={(event) => { setProviderFilter(event.target.value); setPage(0); }}>
          <option value="all">全部生成方式</option>
          {providers.map((provider) => <option key={provider.id} value={provider.id}>{provider.name}</option>)}
        </select>
        <select className="field-input w-[120px]" value={statusFilter} onChange={(event) => { setStatusFilter(event.target.value); setPage(0); }}>
          <option value="all">全部状态</option>
          <option value="planned">planned</option>
          <option value="ready">ready</option>
          <option value="generated">generated</option>
          <option value="approved">approved</option>
          <option value="failed">failed</option>
        </select>
        <select className="field-input w-[90px]" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setPage(0); }}>
          <option value={25}>25 / 页</option>
          <option value={50}>50 / 页</option>
          <option value={100}>100 / 页</option>
        </select>
        <button type="button" className="btn btn-primary" onClick={onAdd}><Plus size={12} />新增镜头</button>
      </div>

      <div className="flex flex-wrap items-center gap-2 rounded-md border px-2 py-1.5 text-[10.5px]" style={{borderColor: 'var(--line)', color: 'var(--fg-dim)'}}>
        <span>共 {shots.length} 个镜头 · 当前筛选 {filtered.length} 个</span>
        {selected.size > 0 ? (
          <>
            <span className="badge badge-accent">已选 {selected.size}</span>
            <select className="field-input w-[150px]" value={bulkProvider} onChange={(event) => setBulkProvider(event.target.value)}>
              <option value="">批量设置生成方式</option>
              {providers.map((provider) => <option key={provider.id} value={provider.id}>{provider.name}</option>)}
            </select>
            <button type="button" className="btn" onClick={() => applyBulk('provider')} disabled={!bulkProvider}>应用</button>
            <select className="field-input w-[110px]" value={bulkStatus} onChange={(event) => setBulkStatus(event.target.value)}>
              <option value="">批量设置状态</option>
              <option value="planned">planned</option>
              <option value="ready">ready</option>
              <option value="generated">generated</option>
              <option value="approved">approved</option>
              <option value="failed">failed</option>
            </select>
            <button type="button" className="btn" onClick={() => applyBulk('status')} disabled={!bulkStatus}>应用</button>
            <button type="button" className="btn btn-primary" onClick={() => onProduce(selectedIndices)}><Wand2 size={11} />生成选中</button>
            <button type="button" className="btn" onClick={() => { onDelete(selectedIndices); setSelected(new Set()); }}><Trash2 size={11} />删除选中</button>
            <button type="button" className="btn btn-ghost ml-auto" onClick={() => setSelected(new Set())}>清除选择</button>
          </>
        ) : <span className="ml-auto">支持搜索、筛选、分页与批量操作</span>}
      </div>

      <div className="overflow-x-auto rounded-md border" style={{borderColor: 'var(--line)'}}>
        <table className="w-full min-w-[880px] border-collapse text-left text-[10.5px]">
          <thead>
            <tr style={{background: 'var(--ink-850)', color: 'var(--fg-dim)'}}>
              <th className="w-8 px-2 py-2"><input type="checkbox" checked={allPageSelected} onChange={togglePage} /></th>
              <th className="w-10 px-2 py-2">#</th>
              <th className="px-2 py-2">镜头</th>
              <th className="w-20 px-2 py-2">Beat</th>
              <th className="w-[170px] px-2 py-2">生成方式</th>
              <th className="w-[100px] px-2 py-2">状态</th>
              <th className="w-16 px-2 py-2 text-right">时长</th>
              <th className="w-12 px-2 py-2">资产</th>
              <th className="px-2 py-2">内容</th>
              <th className="w-20 px-2 py-2" />
            </tr>
          </thead>
          <tbody>
            {pageItems.map(({shot, index}) => {
              const key = shotKey(shot, index);
              const provider = String(shot.provider || '');
              const status = String(shot.status || 'planned');
              return (
                <tr key={key} className="border-t" style={{borderColor: 'var(--line)', background: selected.has(key) ? 'rgba(90,169,255,.06)' : 'transparent'}}>
                  <td className="px-2 py-2"><input type="checkbox" checked={selected.has(key)} onChange={() => toggle(key)} /></td>
                  <td className="px-2 py-2" style={{color: 'var(--fg-dim)'}}>{index + 1}</td>
                  <td className="px-2 py-2">
                    <div className="font-medium">{String(shot.id || '—')}</div>
                    <div className="text-[9.5px]" style={{color: 'var(--fg-dim)'}}>{String(shot.shot || '—')}</div>
                  </td>
                  <td className="px-2 py-2" style={{color: 'var(--fg-dim)'}}>{String(shot.beat_id || '—')}</td>
                  <td className="px-2 py-2">
                    <select className="field-input h-7" value={provider} onChange={(event) => onPatch(index, 'provider', event.target.value)}>
                      <option value="">默认（程序化）</option>
                      {providers.map((item) => <option key={item.id} value={item.id}>{providerLabel(item)}</option>)}
                    </select>
                  </td>
                  <td className="px-2 py-2">
                    <select className="field-input h-7" value={status} onChange={(event) => onPatch(index, 'status', event.target.value)}>
                      <option value="planned">planned</option>
                      <option value="ready">ready</option>
                      <option value="generated">generated</option>
                      <option value="approved">approved</option>
                      <option value="failed">failed</option>
                    </select>
                  </td>
                  <td className="px-2 py-2 text-right" style={{fontVariantNumeric: 'tabular-nums'}}>{Number(shot.duration_seconds) || 0}s</td>
                  <td className="px-2 py-2">{assets[String(shot.id)]?.preview_data_url ? <img src={assets[String(shot.id)].preview_data_url} alt="" className="h-9 w-9 rounded object-cover" /> : <span className="text-[9px]" style={{color: 'var(--fg-dim)'}}>—</span>}</td>
                  <td className="max-w-[260px] px-2 py-2">
                    <div className="truncate" style={{color: 'var(--fg-muted)'}}>{String(shot.visual || shot.text || shot.prompt || '—')}</div>
                  </td>
                  <td className="px-2 py-2 text-right">
                    {String(shot.provider || '') === 'ai.image' ? <button type="button" className="icon-btn" title="生成模拟图片" onClick={() => onProduce([index])}><Wand2 size={12} /></button> : null}
                    <button type="button" className="icon-btn" title="编辑镜头" onClick={() => onOpen(index)}><Edit3 size={12} /></button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
        {!pageItems.length ? <div className="p-4 text-center text-[11px]" style={{color: 'var(--fg-dim)'}}>{shots.length ? '没有匹配的镜头' : '还没有镜头，点击“新增镜头”。'}</div> : null}
      </div>

      <div className="flex items-center gap-2 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
        <span>第 {currentPage + 1} / {pageCount} 页 · 共 {filtered.length} 个匹配</span>
        <button type="button" className="btn ml-auto" onClick={() => setPage((value) => Math.max(0, value - 1))} disabled={currentPage <= 0}><ChevronLeft size={11} />上一页</button>
        <button type="button" className="btn" onClick={() => setPage((value) => Math.min(pageCount - 1, value + 1))} disabled={currentPage >= pageCount - 1}>下一页<ChevronRight size={11} /></button>
      </div>
    </div>
  );
}
