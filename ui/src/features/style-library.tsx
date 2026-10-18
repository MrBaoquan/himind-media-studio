import {useEffect, useMemo, useState} from 'react';
import {CheckCircle2, Clock3, Download, Image as ImageIcon, LayoutGrid, Loader2, Plus, RefreshCw, Search, Star, Trash2, X} from 'lucide-react';
import {callCapability} from '../api/capability';
import type {AssetMarket, AssetMarketEntry, AssetPack} from '../types';

const FAVORITES_KEY = 'himind.workbench.style.favorites';
const RECENT_KEY = 'himind.workbench.style.recent';

const STYLE_FILTERS = [
  {id: 'all', label: '全部', tags: []},
  {id: 'favorites', label: '收藏', tags: []},
  {id: 'recent', label: '最近', tags: []},
  {id: 'data', label: '数据', tags: ['data', 'chart', 'dashboard', 'kinetic']},
  {id: 'map', label: '地图路线', tags: ['map', 'route', 'transit']},
  {id: 'brand', label: '品牌编辑', tags: ['brand', 'editorial', 'minimal', 'identity', 'palette']},
  {id: 'illustration', label: '插画手绘', tags: ['painted', 'paper', 'whiteboard', 'isometric', 'collage', 'hand-drawn']},
  {id: 'retro', label: '复古', tags: ['retro', 'vhs', 'arcade', 'comic', 'pixel']},
  {id: '3d', label: '3D', tags: ['3d', 'threejs']},
  {id: 'science', label: '科学', tags: ['astronomy', 'science', 'space']},
  {id: 'tech', label: '技术', tags: ['blueprint', 'technical', 'diagram', 'engineering']},
  {id: 'type', label: '字体歌词', tags: ['typography', 'lyric', 'karaoke', 'kinetic']},
];

function styleKey(pack: AssetPack): string {
  return `${pack.id}@${pack.version}`;
}

function readStoredList(key: string): string[] {
  try {
    const raw = localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as string[]) : [];
  } catch {
    return [];
  }
}

function sourceLabel(source?: string): string {
  if (source === 'workspace') return '工作区';
  if (source === 'repository') return '仓库';
  if (source === 'installed') return '已安装';
  return source || '未知来源';
}

function imageMime(path: string): string {
  const lower = path.toLowerCase();
  if (lower.endsWith('.png')) return 'image/png';
  if (lower.endsWith('.webp')) return 'image/webp';
  return 'image/jpeg';
}

export function StyleLibrary({
  workspace,
  onClose,
  onCreated,
}: {
  workspace: string;
  onClose: () => void;
  onCreated: (projectId: string) => void | Promise<void>;
}) {
  const [packs, setPacks] = useState<AssetPack[]>([]);
  const [previews, setPreviews] = useState<Record<string, string>>({});
  const [market, setMarket] = useState<Record<string, AssetMarketEntry>>({});
  const [selected, setSelected] = useState<AssetPack | null>(null);
  const [name, setName] = useState('');
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [filter, setFilter] = useState('all');
  const [favorites, setFavorites] = useState<string[]>(() => readStoredList(FAVORITES_KEY));
  const [recent, setRecent] = useState<string[]>(() => readStoredList(RECENT_KEY));

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const result = await callCapability<{items?: AssetPack[]}>('media.workbench.asset.list', {workspace_root: workspace});
      const items = result.items ?? [];
      setPacks(items);
      setSelected((current) => items.find((pack) => pack.id === current?.id && pack.version === current?.version) ?? current);
      try {
        const marketResult = await callCapability<AssetMarket>('media.workbench.asset.market.list', {workspace_root: workspace});
        setMarket(Object.fromEntries((marketResult.assets ?? []).map((entry) => [entry.id, entry])));
      } catch {
        setMarket({});
      }
      const entries = await Promise.all(items.map(async (pack) => {
        const preview = pack.resources?.find((resource) => resource.id === 'preview_loop' || resource.role === 'preview_loop')
          ?? pack.resources?.find((resource) => resource.type === 'image' && (resource.role === 'preview' || resource.id === 'preview'));
        if (!preview) return null;
        try {
          const read = await callCapability<{base64?: string}>('media.workbench.asset.read', {
            workspace_root: workspace,
            pack_id: pack.id,
            version: pack.version,
            resource_id: preview.id,
          });
          return read.base64 ? [pack.id, `data:${preview.mime || imageMime(preview.path)};base64,${read.base64}`] as const : null;
        } catch {
          return null;
        }
      }));
      setPreviews(Object.fromEntries(entries.filter(Boolean) as Array<readonly [string, string]>));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [workspace]);

  useEffect(() => {
    try { localStorage.setItem(FAVORITES_KEY, JSON.stringify(favorites)); } catch {}
  }, [favorites]);

  useEffect(() => {
    try { localStorage.setItem(RECENT_KEY, JSON.stringify(recent)); } catch {}
  }, [recent]);

  async function installPack() {
    if (!selected) return;
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const output = `dist/asset-packages/${selected.id}-${selected.version}.hmpack`;
      const packaged = await callCapability<{package_path?: string}>('media.workbench.asset.pack', {
        workspace_root: workspace,
        pack_id: selected.id,
        version: selected.version,
        output,
      });
      if (!packaged.package_path) throw new Error('打包没有返回 .hmpack 路径');
      await callCapability('media.workbench.asset.install', {package_path: packaged.package_path});
      await load();
      setNotice('已安装到本机资产库');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function uninstallPack() {
    if (!selected) return;
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      await callCapability('media.workbench.asset.uninstall', {pack_id: selected.id, version: selected.version});
      await load();
      setNotice('已移除本机安装副本');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function installFromMarket() {
    if (!selected || !selectedMarket) return;
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      await callCapability('media.workbench.asset.market.install', {
        workspace_root: workspace,
        pack_id: selected.id,
        version: selectedMarket.version,
      });
      await load();
      setNotice('已从市场安装');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function createProject() {
    if (!selected) return;
    setBusy(true);
    setError(null);
    try {
      const result = await callCapability<{project?: {id?: string}}>('media.workbench.stylepack.scaffold', {
        workspace_root: workspace,
        pack_id: selected.id,
        version: selected.version,
        name: name.trim() || `${selected.name} 项目`,
      });
      const projectId = result.project?.id;
      if (!projectId) throw new Error('创作台没有返回项目 ID');
      markRecent(selectedKey);
      await onCreated(projectId);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  const selectedMarket = selected ? market[selected.id] : undefined;
  const selectedKey = selected ? styleKey(selected) : '';
  const selectedFavorite = favorites.includes(selectedKey);
  const filteredPacks = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    const filterTags = STYLE_FILTERS.find((item) => item.id === filter)?.tags ?? [];
    return packs.filter((pack) => {
      const key = styleKey(pack);
      if (filter === 'favorites' && !favorites.includes(key)) return false;
      if (filter === 'recent' && !recent.includes(key)) return false;
      if (filter !== 'all' && filter !== 'favorites' && filter !== 'recent') {
        const tags = (pack.tags ?? []).map((tag) => tag.toLowerCase());
        if (!filterTags.some((tag) => tags.includes(tag))) return false;
      }
      if (!normalized) return true;
      return [pack.name, pack.description ?? '', pack.id, ...(pack.tags ?? [])].join(' ').toLowerCase().includes(normalized);
    });
  }, [packs, query, filter, favorites, recent]);

  function markRecent(key: string) {
    setRecent((current) => [key, ...current.filter((item) => item !== key)].slice(0, 8));
  }

  function toggleFavorite(key: string) {
    setFavorites((current) => current.includes(key) ? current.filter((item) => item !== key) : [key, ...current]);
  }


  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-6" style={{background: 'rgba(0,0,0,0.62)', backdropFilter: 'blur(4px)'}}>
      <div className="flex max-h-full w-full max-w-[1080px] flex-col overflow-hidden rounded-xl" style={{background: 'var(--ink-900)', border: '1px solid var(--line)', boxShadow: '0 24px 90px rgba(0,0,0,0.55)'}}>
        <header className="flex items-center gap-3 border-b px-4 py-3" style={{borderColor: 'var(--line)'}}>
          <span className="flex h-7 w-7 items-center justify-center rounded-md" style={{background: 'var(--accent-soft)'}}>
            <LayoutGrid size={14} />
          </span>
          <div className="min-w-0 flex-1">
            <div className="text-[13px] font-semibold">资产库 · 视频风格</div>
            <div className="text-[10.5px]" style={{color: 'var(--fg-dim)'}}>{packs.length} 个风格 · 搜索、分类、收藏与最近使用</div>
          </div>
          <button type="button" className="icon-btn" onClick={() => void load()} title="刷新资产库">
            <RefreshCw size={13} />
          </button>
          <button type="button" className="icon-btn" onClick={onClose} title="关闭">
            <X size={14} />
          </button>
        </header>

        <div className="grid min-h-0 flex-1" style={{gridTemplateColumns: '1fr 300px'}}>
          <div className="min-h-0 overflow-auto p-4">
            <div className="mb-3 flex flex-col gap-2">
              <div className="relative">
                <Search size={12} className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2" style={{color: 'var(--fg-dim)'}} />
                <input
                  className="field w-full pl-8"
                  placeholder="搜索风格、标签或用途"
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                />
              </div>
              <div className="flex flex-wrap gap-1">
                {STYLE_FILTERS.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    className="rounded-full px-2.5 py-1 text-[10px] transition"
                    onClick={() => setFilter(item.id)}
                    style={{
                      background: filter === item.id ? 'var(--accent-soft)' : 'var(--ink-850)',
                      border: '1px solid var(--line)',
                      color: filter === item.id ? 'var(--fg)' : 'var(--fg-dim)',
                    }}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
              <div className="flex items-center justify-between text-[10px]" style={{color: 'var(--fg-dim)'}}>
                <span>{filteredPacks.length} / {packs.length} 个风格</span>
                {filter !== 'all' || query ? (
                  <button type="button" onClick={() => { setFilter('all'); setQuery(''); }} style={{color: 'var(--fg-muted)'}}>清除筛选</button>
                ) : null}
              </div>
            </div>
            {loading ? (
              <div className="flex h-56 items-center justify-center gap-2 text-[12px]" style={{color: 'var(--fg-dim)'}}>
                <Loader2 size={14} className="animate-spin" />正在读取资产包…
              </div>
            ) : packs.length === 0 ? (
              <div className="flex h-56 flex-col items-center justify-center gap-2 text-center">
                <LayoutGrid size={30} style={{color: 'var(--fg-dim)'}} />
                <div className="text-[12.5px]" style={{color: 'var(--fg-muted)'}}>还没有可用的视频风格</div>
                <div className="text-[11px]" style={{color: 'var(--fg-dim)'}}>把 asset-pack.json + template 放到 style-packs/ 后刷新</div>
              </div>
            ) : filteredPacks.length === 0 ? (
              <div className="flex h-56 flex-col items-center justify-center gap-2 text-center">
                <Search size={28} style={{color: 'var(--fg-dim)'}} />
                <div className="text-[12.5px]" style={{color: 'var(--fg-muted)'}}>没有匹配的风格</div>
                <button type="button" className="btn mt-1" onClick={() => { setFilter('all'); setQuery(''); }}>清除筛选</button>
              </div>
            ) : (
              <div className="grid gap-3" style={{gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))'}}>
                {filteredPacks.map((pack) => {
                  const marketInfo = market[pack.id];
                  const hasLoop = (pack.resources ?? []).some((resource) => resource.id === 'preview_loop' || resource.role === 'preview_loop');
                  return (
                  <button
                    key={pack.id + pack.version}
                    type="button"
                    className="overflow-hidden rounded-lg text-left transition"
                    onClick={() => {
                      setSelected(pack);
                      setName(`${pack.name} 项目`);
                      markRecent(styleKey(pack));
                    }}
                    style={{
                      background: 'var(--ink-850)',
                      border: selected?.id === pack.id ? '1px solid rgba(90,169,255,.65)' : '1px solid var(--line)',
                      boxShadow: selected?.id === pack.id ? '0 0 0 2px rgba(90,169,255,.12)' : undefined,
                    }}
                  >
                    <div className="relative flex h-[140px] items-center justify-center overflow-hidden" style={{background: '#070b16'}}>
                      {hasLoop ? <span className="absolute left-2 top-2 rounded bg-black/60 px-1.5 py-0.5 text-[9px] text-white">动态预览</span> : null}
                      {previews[pack.id] ? (
                        <img src={previews[pack.id]} alt={pack.name} className="max-h-full max-w-full object-contain" />
                      ) : (
                        <ImageIcon size={26} style={{color: 'var(--fg-dim)'}} />
                      )}
                    </div>
                    <div className="p-3">
                      <div className="flex items-center gap-2">
                        <span className="truncate text-[12.5px] font-medium">{pack.name}</span>
                        <span className="ml-auto font-mono text-[9.5px]" style={{color: 'var(--fg-dim)'}}>v{pack.version}</span>
                      </div>
                      <div className="mt-1 line-clamp-2 min-h-[30px] text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
                        {pack.description ?? '可复用的视频风格资源包'}
                      </div>
                      <div className="mt-2 flex flex-wrap gap-1">
                        <span className="badge">{sourceLabel(pack.source)}</span>
                        {pack.installed ? <span className="badge" style={{color: 'var(--ok)'}}><CheckCircle2 size={9} />已安装</span> : null}
                        {favorites.includes(styleKey(pack)) ? <span className="badge" style={{color: 'var(--warn)'}}><Star size={9} fill="currentColor" />收藏</span> : null}
                        {recent.includes(styleKey(pack)) ? <span className="badge"><Clock3 size={9} />最近</span> : null}
                        {marketInfo ? <span className="badge">市场 v{marketInfo.version}</span> : null}
                        {marketInfo?.status === 'prepared' ? <span className="badge">待发布</span> : null}
                        {marketInfo?.update_available ? <span className="badge" style={{color: 'var(--warn)'}}>可更新</span> : null}
                        {(pack.tags ?? []).slice(0, 3).map((tag) => <span key={tag} className="badge">{tag}</span>)}
                      </div>
                    </div>
                  </button>
                  );
                })}
              </div>
            )}
            {error ? <div className="mt-3 text-[11px]" style={{color: 'var(--err)'}}>{error}</div> : null}
          </div>

          <aside className="border-l p-4" style={{borderColor: 'var(--line)', background: 'var(--ink-850)'}}>
            <div className="text-[11px] font-medium" style={{color: 'var(--fg-dim)'}}>创建项目</div>
            {selected ? (
              <>
                <div className="mt-2 text-[13px] font-medium">{selected.name}</div>
                <div className="mt-1 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
                  {selected.license ?? '未声明许可证'} · {selected.resources?.length ?? 0} 个资源
                </div>
                <div className="mt-2 flex flex-wrap gap-1">
                  <span className="badge">{sourceLabel(selected.source)}</span>
                  <span className="badge" style={{color: selected.installed ? 'var(--ok)' : 'var(--fg-dim)'}}>{selected.installed ? '本机已安装' : '未安装'}</span>
                </div>
                <button type="button" className="btn mt-2 w-full" onClick={() => toggleFavorite(selectedKey)} disabled={busy}>
                  <Star size={11} fill={selectedFavorite ? 'currentColor' : 'none'} />
                  {selectedFavorite ? '已收藏' : '收藏风格'}
                </button>
                {selected.installed ? (
                  <button type="button" className="btn mt-2 w-full" onClick={() => void uninstallPack()} disabled={busy}>
                    {busy ? <Loader2 size={11} className="animate-spin" /> : <Trash2 size={11} />}卸载本机副本
                  </button>
                ) : (
                  <button type="button" className="btn mt-2 w-full" onClick={() => void installPack()} disabled={busy}>
                    {busy ? <Loader2 size={11} className="animate-spin" /> : <Download size={11} />}安装到本机
                  </button>
                )}
                {selected.install_path ? <div className="mt-1 break-all text-[9px]" style={{color: 'var(--fg-dim)'}}>{selected.install_path}</div> : null}
                {selectedMarket ? (
                  <div className="mt-2 rounded-md p-2 text-[10px]" style={{background: 'var(--ink-800)', border: '1px solid var(--line)'}}>
                    <div className="flex items-center gap-1">
                      <span>市场 v{selectedMarket.version}</span>
                      <span className="ml-auto" style={{color: selectedMarket.status === 'published' ? 'var(--ok)' : 'var(--fg-dim)'}}>
                        {selectedMarket.status === 'published'
                          ? selectedMarket.update_available ? '可更新' : selectedMarket.installed ? '已是最新' : '可安装'
                          : '待发布'}
                      </span>
                    </div>
                    {selectedMarket.status === 'published' ? (
                      <button type="button" className="btn mt-2 w-full" onClick={() => void installFromMarket()} disabled={busy || (selected.installed && !selectedMarket.update_available)}>
                        {busy ? <Loader2 size={11} className="animate-spin" /> : <Download size={11} />}
                        {selectedMarket.update_available ? `更新到 v${selectedMarket.version}` : selected.installed ? '市场版本已安装' : '从市场安装'}
                      </button>
                    ) : (
                      <div className="mt-1" style={{color: 'var(--fg-dim)'}}>发布 GitHub Release 后可从市场安装</div>
                    )}
                  </div>
                ) : null}
                <div className="mt-2 flex flex-wrap gap-1">
                  {(selected.resources ?? []).filter((resource) => resource.type !== 'docs').slice(0, 6).map((resource) => (
                    <span key={resource.id} className="badge">{resource.type}</span>
                  ))}
                </div>
                <label className="mt-3 block text-[10.5px]" style={{color: 'var(--fg-dim)'}}>项目名称</label>
                <input className="field mt-1 w-full" value={name} onChange={(event) => setName(event.target.value)} />
                <button type="button" className="btn btn-primary mt-3 w-full" onClick={() => void createProject()} disabled={busy || !name.trim()}>
                  {busy ? <Loader2 size={12} className="animate-spin" /> : <Plus size={12} />}
                  {busy ? '正在创建…' : '创建项目并打开审阅'}
                </button>
                <div className="mt-3 text-[10px] leading-relaxed" style={{color: 'var(--fg-dim)'}}>
                  自动复制模板到工作区，调用 project.import，然后进入审阅页。
                </div>
                {notice ? <div className="mt-2 text-[10px]" style={{color: 'var(--ok)'}}>{notice}</div> : null}
              </>
            ) : (
              <div className="mt-4 text-[11px] leading-6" style={{color: 'var(--fg-dim)'}}>从左侧选择一个风格。</div>
            )}
          </aside>
        </div>
      </div>
    </div>
  );
}
