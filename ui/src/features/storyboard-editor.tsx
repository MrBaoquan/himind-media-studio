import {useEffect, useMemo, useRef, useState} from 'react';
import {AlertTriangle, ArrowDown, ArrowUp, Loader2, Play, Plus, Save, Trash2} from 'lucide-react';
import {callCapability} from '../api/capability';
import type {CatalogOption, ProjectDetail, Recipe, TemplateSchemaPayload} from '../types';
import {Empty, SectionLabel} from '../components/ui';

/**
 * Storyboard 可视化编辑器。
 *
 * 外壳不知道有哪些镜头，也不知道某个镜头有哪些题材字段：
 * - 镜头清单来自 catalog.list 的 storyboard_shot 声明；
 * - 默认数据和字段结构来自 template.schema；
 * - 没有声明 schema 的镜头退化为 JSON 编辑，而不是静默失败。
 */
interface StoryboardItem {
  shot: string;
  duration_seconds: number;
  title?: string;
  subtitle?: string;
  data?: Record<string, unknown>;
}

interface SchemaBundle {
  template: CatalogOption;
  schema: Record<string, any> | null;
  defaultData: Record<string, unknown>;
}

const clone = <T,>(value: T): T => (value === null || value === undefined ? value : JSON.parse(JSON.stringify(value)));

function orderedProperties(properties: Record<string, Record<string, any>>, required: string[]): Array<[string, Record<string, any>]> {
  return Object.entries(properties).sort(([a], [b]) => {
    const requiredDelta = Number(required.includes(b)) - Number(required.includes(a));
    return requiredDelta !== 0 ? requiredDelta : a.localeCompare(b);
  });
}

function readStoryboard(recipe: Recipe): StoryboardItem[] {
  const raw = recipe.data?.storyboard;
  if (!Array.isArray(raw)) return [];
  return raw.map((entry) => {
    const item = (entry ?? {}) as Record<string, unknown>;
    return {
      shot: typeof item.shot === 'string' ? item.shot : '',
      duration_seconds: Number(item.duration_seconds) || 3,
      title: typeof item.title === 'string' ? item.title : '',
      subtitle: typeof item.subtitle === 'string' ? item.subtitle : '',
      data: item.data && typeof item.data === 'object' && !Array.isArray(item.data)
        ? (item.data as Record<string, unknown>)
        : {},
    };
  });
}

function sumDuration(items: StoryboardItem[]): number {
  return items.reduce((total, item) => total + (Number(item.duration_seconds) || 0), 0);
}

function errorLines(value: unknown): string {
  if (!Array.isArray(value)) return String(value ?? '未知错误');
  return value
    .map((item) => {
      if (!item || typeof item !== 'object') return String(item);
      const issue = item as {code?: string; message?: string};
      return [issue.code, issue.message].filter(Boolean).join('：');
    })
    .join('；');
}

function JsonField({
  value,
  onChange,
  compact = false,
}: {
  value: unknown;
  onChange: (next: unknown) => void;
  compact?: boolean;
}) {
  const [text, setText] = useState(() => JSON.stringify(value ?? {}, null, 2));
  const [error, setError] = useState<string | null>(null);

  function commit() {
    try {
      onChange(JSON.parse(text));
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <div className="flex flex-col gap-1">
      <textarea
        className="w-full resize-y rounded-md p-2 font-mono leading-relaxed"
        style={{background: 'var(--ink-950)', border: `1px solid ${error ? 'var(--err)' : 'var(--line)'}`, color: 'var(--fg)', minHeight: compact ? 62 : 130, fontSize: 10.5}}
        value={text}
        onChange={(event) => setText(event.target.value)}
        onBlur={commit}
        spellCheck={false}
      />
      {error ? <span className="text-[10.5px]" style={{color: 'var(--err)'}}>JSON：{error}</span> : null}
    </div>
  );
}

function DataField({
  name,
  schema,
  value,
  onChange,
  required,
  depth,
}: {
  name: string;
  schema: Record<string, any>;
  value: unknown;
  onChange: (next: unknown) => void;
  required: boolean;
  depth: number;
}) {
  const type = typeof schema.type === 'string' ? schema.type : '';
  const description = typeof schema.description === 'string' ? schema.description : '';
  const label = (
    <div className="flex items-baseline gap-1">
      <span className="text-[10.5px]" style={{color: 'var(--fg-muted)'}}>{name}{required ? <span style={{color: 'var(--err)'}}> *</span> : null}</span>
      {description ? <span className="truncate text-[9.5px]" style={{color: 'var(--fg-dim)'}}>{description}</span> : null}
    </div>
  );

  if (type === 'object' && schema.properties && typeof schema.properties === 'object' && depth < 2) {
    const objectValue = value && typeof value === 'object' && !Array.isArray(value) ? (value as Record<string, unknown>) : {};
    const properties = schema.properties as Record<string, Record<string, any>>;
    const requiredKeys = Array.isArray(schema.required) ? schema.required.filter((item: unknown) => typeof item === 'string') : [];
    return (
      <div className="rounded-md p-2" style={{border: '1px solid var(--line)', background: 'rgba(255,255,255,.015)'}}>
        <div className="mb-1 flex items-baseline gap-1">
          <span className="text-[10.5px]" style={{color: 'var(--fg-muted)'}}>{name}{required ? <span style={{color: 'var(--err)'}}> *</span> : null}</span>
          <span className="text-[9.5px]" style={{color: 'var(--fg-dim)'}}>{Object.keys(properties).length} 个字段</span>
        </div>
        <div className="flex flex-col gap-2">
          {orderedProperties(properties, requiredKeys).map(([key, child]) => (
            <DataField
              key={key}
              name={key}
              schema={child}
              value={objectValue[key]}
              required={requiredKeys.includes(key)}
              depth={depth + 1}
              onChange={(next) => onChange({...objectValue, [key]: next})}
            />
          ))}
        </div>
      </div>
    );
  }

  if (type === 'boolean') {
    return (
      <label className="flex items-center gap-2">
        <input type="checkbox" checked={value === true} onChange={(event) => onChange(event.target.checked)} />
        {label}
      </label>
    );
  }

  if (type === 'number' || type === 'integer') {
    return (
      <label className="flex flex-col gap-1">
        {label}
        <input
          className="field"
          type="number"
          step={type === 'integer' ? 1 : 'any'}
          value={typeof value === 'number' ? value : Number(value) || 0}
          onChange={(event) => onChange(type === 'integer' ? Math.round(Number(event.target.value) || 0) : Number(event.target.value) || 0)}
        />
      </label>
    );
  }

  if (type === 'string') {
    return (
      <label className="flex flex-col gap-1">
        {label}
        <input className="field" value={typeof value === 'string' ? value : ''} onChange={(event) => onChange(event.target.value)} />
      </label>
    );
  }

  // 数组、自由对象、没有 properties 的对象：保留 JSON 编辑，避免外壳猜结构。
  return (
    <div className="flex flex-col gap-1">
      {label}
      <JsonField value={value ?? (type === 'array' ? [] : {})} onChange={onChange} compact />
    </div>
  );
}

export function StoryboardEditor({
  detail,
  workspace,
  shots,
  onSaved,
  onPreview,
}: {
  detail: ProjectDetail;
  workspace: string;
  shots: CatalogOption[];
  onSaved: () => Promise<void>;
  onPreview: () => void;
}) {
  const recipe = detail.project.recipe as Recipe;
  const projectId = detail.project.id;
  const orderedShots = useMemo(
    () => [...shots].sort((a, b) => (a.storyboard_order ?? 999) - (b.storyboard_order ?? 999)),
    [shots],
  );
  const [items, setItems] = useState<StoryboardItem[]>(() => readStoryboard(recipe));
  const [selected, setSelected] = useState(items.length ? 0 : -1);
  const [addShot, setAddShot] = useState(orderedShots[0]?.id ?? '');
  const [busy, setBusy] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const cache = useRef(new Map<string, SchemaBundle>());
  const [schemaVersion, setSchemaVersion] = useState(0);

  const selectedItem = items[selected] ?? null;
  const total = sumDuration(items);

  async function loadBundle(shotId: string): Promise<SchemaBundle | null> {
    if (cache.current.has(shotId)) return cache.current.get(shotId) ?? null;
    const fallback = orderedShots.find((shot) => shot.id === shotId);
    if (!fallback) return null;
    try {
      const payload = await callCapability<TemplateSchemaPayload>('media.workbench.template.schema', {
        template_id: shotId,
        style_id: recipe.style_id,
      });
      const bundle: SchemaBundle = {
        template: payload.template ?? fallback,
        schema: (payload.schema as Record<string, any> | null | undefined) ?? null,
        defaultData: (payload.default_data as Record<string, unknown> | undefined) ?? {},
      };
      cache.current.set(shotId, bundle);
      setSchemaVersion((value) => value + 1);
      return bundle;
    } catch (err) {
      setError(`读取镜头「${shotId}」失败：${err instanceof Error ? err.message : String(err)}`);
      return null;
    }
  }

  useEffect(() => {
    if (selectedItem?.shot) void loadBundle(selectedItem.shot);
    // 只在选中镜头变化时加载；loadBundle 内部有缓存。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedItem?.shot]);

  function markDirty() {
    setDirty(true);
    setNotice(null);
    setError(null);
  }

  function updateItem(index: number, patch: Partial<StoryboardItem>) {
    setItems((current) => current.map((item, i) => (i === index ? {...item, ...patch} : item)));
    markDirty();
  }

  async function add() {
    if (!addShot || busy) return;
    setBusy(true);
    const bundle = await loadBundle(addShot);
    const next: StoryboardItem = {
      shot: addShot,
      duration_seconds: bundle?.template.duration_seconds ?? 3,
      title: '',
      subtitle: '',
      data: clone(bundle?.defaultData ?? {}),
    };
    setItems((current) => [...current, next]);
    setSelected(items.length);
    markDirty();
    setBusy(false);
  }

  async function changeType(index: number, shotId: string) {
    if (busy) return;
    setBusy(true);
    const bundle = await loadBundle(shotId);
    updateItem(index, {
      shot: shotId,
      duration_seconds: bundle?.template.duration_seconds ?? 3,
      data: clone(bundle?.defaultData ?? {}),
    });
    setBusy(false);
  }

  function move(index: number, delta: number) {
    const target = index + delta;
    if (target < 0 || target >= items.length) return;
    const next = [...items];
    [next[index], next[target]] = [next[target], next[index]];
    setItems(next);
    setSelected(target);
    markDirty();
  }

  function remove(index: number) {
    const next = items.filter((_, i) => i !== index);
    setItems(next);
    setSelected(next.length ? Math.min(index, next.length - 1) : -1);
    markDirty();
  }

  async function save(andPreview: boolean) {
    if (busy) return;
    if (!items.length) {
      setError('至少需要一个镜头才能保存分镜');
      return;
    }
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const nextRecipe: Recipe = {
        ...recipe,
        duration_seconds: Math.max(1, Math.round(total)),
        data: {...(recipe.data ?? {}), storyboard: items},
      };
      const result = await callCapability<{state?: string; blockers?: unknown; project?: unknown}>('media.workbench.project.update', {
        workspace_root: workspace,
        project_id: projectId,
        expected_updated_at: detail.project.updated_at,
        recipe: nextRecipe,
      });
      if (result.state === 'blocked') {
        setError(errorLines(result.blockers));
        return;
      }
      setDirty(false);
      setNotice('分镜已保存');
      await onSaved();
      if (andPreview) onPreview();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  const bundle = selectedItem ? cache.current.get(selectedItem.shot) : null;
  const schema = bundle?.schema ?? null;
  const properties = schema?.properties && typeof schema.properties === 'object'
    ? (schema.properties as Record<string, Record<string, any>>)
    : null;
  const requiredKeys = schema && Array.isArray(schema.required)
    ? schema.required.filter((item: unknown) => typeof item === 'string')
    : [];

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex items-center gap-2 px-3 py-2" style={{borderBottom: '1px solid var(--line)', background: 'var(--ink-900)'}}>
        <div className="text-[12px] font-semibold">分镜编辑</div>
        <span className="text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
          {items.length} 个镜头 · {Math.round(total)}s
        </span>
        {dirty ? <span className="badge badge-warn">未保存</span> : null}
        <div className="ml-auto flex items-center gap-2">
          {notice ? <span className="text-[10.5px]" style={{color: 'var(--ok)'}}>{notice}</span> : null}
          {error ? <span className="max-w-[320px] truncate text-[10.5px]" style={{color: 'var(--err)'}} title={error}>{error}</span> : null}
          <button type="button" className="btn" onClick={() => void save(false)} disabled={busy || !dirty}>
            {busy ? <Loader2 size={12} className="animate-spin" /> : <Save size={12} />}
            保存
          </button>
          <button type="button" className="btn btn-primary" onClick={() => void save(true)} disabled={busy || !dirty}>
            <Play size={12} />
            保存并预览
          </button>
        </div>
      </div>

      <div className="grid min-h-0 flex-1" style={{gridTemplateColumns: '320px 1fr'}}>
        <aside className="flex min-h-0 flex-col" style={{borderRight: '1px solid var(--line)', background: 'var(--ink-900)'}}>
          <div className="flex-1 overflow-auto p-2">
            {items.length ? items.map((item, index) => {
              const template = orderedShots.find((shot) => shot.id === item.shot);
              return (
                <div
                  key={`${item.shot}-${index}`}
                  role="button"
                  tabIndex={0}
                  className="mb-1 flex w-full items-start gap-2 rounded-md px-2 py-2 text-left"
                  data-active={selected === index}
                  style={{background: selected === index ? 'var(--accent-soft)' : 'transparent', border: `1px solid ${selected === index ? 'rgba(90,169,255,.22)' : 'transparent'}`}}
                  onClick={() => setSelected(index)}
                  onKeyDown={(event) => { if (event.key === 'Enter' || event.key === ' ') setSelected(index); }}
                >
                  <span className="mt-[1px] flex h-5 w-5 shrink-0 items-center justify-center rounded-md text-[10px] font-semibold" style={{background: 'rgba(255,255,255,.07)', color: 'var(--fg-muted)'}}>
                    {index + 1}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-[11.5px]">{item.title || template?.name || item.shot || '未命名镜头'}</span>
                    <span className="mt-[2px] block truncate font-mono text-[9.5px]" style={{color: 'var(--fg-dim)'}}>
                      {item.shot || '未选择类型'} · {Number(item.duration_seconds) || 0}s
                    </span>
                  </span>
                  <span className="flex shrink-0 flex-col gap-0.5">
                    <span className="flex gap-0.5">
                      <button type="button" className="icon-btn h-5 w-5" title="上移" onClick={(event) => { event.stopPropagation(); move(index, -1); }}>
                        <ArrowUp size={10} />
                      </button>
                      <button type="button" className="icon-btn h-5 w-5" title="下移" onClick={(event) => { event.stopPropagation(); move(index, 1); }}>
                        <ArrowDown size={10} />
                      </button>
                    </span>
                    <button type="button" className="icon-btn h-5 w-5" title="删除镜头" onClick={(event) => { event.stopPropagation(); remove(index); }}>
                      <Trash2 size={10} />
                    </button>
                  </span>
                </div>
              );
            }) : (
              <Empty title="还没有镜头" hint="从下方选择镜头类型后添加" />
            )}
          </div>
          <div className="flex items-center gap-1.5 p-2" style={{borderTop: '1px solid var(--line)'}}>
            <select className="field min-w-0 flex-1" value={addShot} onChange={(event) => setAddShot(event.target.value)}>
              {orderedShots.map((shot) => (
                <option key={shot.id} value={shot.id} disabled={shot.status !== 'ready'}>
                  {shot.name ?? shot.id}{shot.status !== 'ready' ? `（${shot.status}）` : ''}
                </option>
              ))}
            </select>
            <button type="button" className="btn" onClick={() => void add()} disabled={busy || !addShot}>
              <Plus size={12} />
              添加
            </button>
          </div>
        </aside>

        <section className="min-h-0 overflow-auto p-3">
          {!selectedItem ? (
            <Empty title="选择一个镜头" hint="左侧选择镜头后编辑标题、时长与数据" />
          ) : (
            <div className="flex flex-col gap-3">
              <div className="grid grid-cols-2 gap-2">
                <label className="flex flex-col gap-1">
                  <span className="text-[10.5px]" style={{color: 'var(--fg-muted)'}}>镜头类型</span>
                  <select className="field" value={selectedItem.shot} onChange={(event) => void changeType(selected, event.target.value)} disabled={busy}>
                    {orderedShots.map((shot) => (
                      <option key={shot.id} value={shot.id}>{shot.name ?? shot.id}</option>
                    ))}
                  </select>
                </label>
                <label className="flex flex-col gap-1">
                  <span className="text-[10.5px]" style={{color: 'var(--fg-muted)'}}>时长（秒）</span>
                  <input
                    className="field"
                    type="number"
                    min={1}
                    max={60}
                    step={0.5}
                    value={selectedItem.duration_seconds}
                    onChange={(event) => updateItem(selected, {duration_seconds: Math.max(1, Number(event.target.value) || 1)})}
                  />
                </label>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <label className="flex flex-col gap-1">
                  <span className="text-[10.5px]" style={{color: 'var(--fg-muted)'}}>镜头标题（可选）</span>
                  <input className="field" value={selectedItem.title ?? ''} onChange={(event) => updateItem(selected, {title: event.target.value})} placeholder="覆盖镜头内的标题" />
                </label>
                <label className="flex flex-col gap-1">
                  <span className="text-[10.5px]" style={{color: 'var(--fg-muted)'}}>镜头副标题（可选）</span>
                  <input className="field" value={selectedItem.subtitle ?? ''} onChange={(event) => updateItem(selected, {subtitle: event.target.value})} placeholder="覆盖镜头内的副标题" />
                </label>
              </div>

              <div>
                <SectionLabel trailing={<span className="font-mono text-[9.5px]">{schema ? 'schema' : 'JSON'}</span>}>镜头数据</SectionLabel>
                {!schema || !properties ? (
                  <div className="flex flex-col gap-2">
                    <div className="flex items-center gap-1.5 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
                      <AlertTriangle size={11} />
                      该镜头没有声明可渲染的字段结构，使用原始 JSON 编辑。
                    </div>
                    <JsonField key={`${selected}-${selectedItem.shot}-raw`} value={selectedItem.data ?? {}} onChange={(next) => updateItem(selected, {data: (next as Record<string, unknown>) ?? {}})} />
                  </div>
                ) : (
                  <div className="flex flex-col gap-2">
                    {orderedProperties(properties, requiredKeys).map(([key, child]) => (
                      <DataField
                        key={`${selectedItem.shot}-${key}`}
                        name={key}
                        schema={child}
                        value={(selectedItem.data ?? {})[key]}
                        required={requiredKeys.includes(key)}
                        depth={0}
                        onChange={(next) => updateItem(selected, {data: {...(selectedItem.data ?? {}), [key]: next}})}
                      />
                    ))}
                    <details className="mt-1">
                      <summary className="cursor-pointer text-[10.5px]" style={{color: 'var(--fg-dim)'}}>原始 JSON（高级）</summary>
                      <div className="mt-2">
                        <JsonField key={`${selected}-${selectedItem.shot}-json-${schemaVersion}`} value={selectedItem.data ?? {}} onChange={(next) => updateItem(selected, {data: (next as Record<string, unknown>) ?? {}})} />
                      </div>
                    </details>
                  </div>
                )}
              </div>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}






