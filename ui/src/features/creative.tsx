import {useEffect, useMemo, useRef, useState} from 'react';
import {ThemeToggle} from '../theme';
import {
  AlertTriangle,
  ArrowRight,
  Bot,
  CheckCircle2,
  ChevronDown,
  ChevronLeft,
  ChevronUp,
  Clapperboard,
  Copy,
  ExternalLink,
  FileText,
  FlaskConical,
  GitBranch,
  History,
  Lightbulb,
  Link2,
  Loader2,
  MessageSquare,
  PanelLeft,
  PanelRight,
  Plus,
  RefreshCw,
  Save,
  Search,
  ShieldCheck,
  Sparkles,
  Trash2,
  Wand2,
  X,
} from 'lucide-react';
import {callCapability, hostInvoke} from '../api/capability';
import {ShotTable} from './shot-list';
import {AudioPanel} from './audio-panel';
import type {CreativeList, CreativePackage, CreativeSummary, CreativeValidation, ProjectDetail, ShotProduceAsset, ShotProduceResult, ShotProvider, ShotStatusResult, TimelinePlan} from '../types';

type TabID = 'brief' | 'research' | 'script' | 'storyboard' | 'production';

const TABS: Array<{id: TabID; label: string; icon: any}> = [
  {id: 'brief', label: 'Brief', icon: FileText},
  {id: 'research', label: '研究', icon: FlaskConical},
  {id: 'script', label: '脚本', icon: MessageSquare},
  {id: 'storyboard', label: '分镜', icon: Clapperboard},
  {id: 'production', label: '制作规格', icon: Sparkles},
];

const AI_ACTIONS = [
  {id: 'directions', label: '出 3 个方向', prompt: '请基于当前 Brief，给出 3 个差异化的短视频方向。每个方向包含：一句话钩子、核心反直觉点、视觉母题、风险。'},
  {id: 'research', label: '补研究证据', prompt: '请为当前主题补充 3-5 条可核实的事实，并给出具体来源 URL。区分“事实”“机制解释”“常见误解”。'},
  {id: 'script', label: '写 45 秒脚本', prompt: '请把当前研究写成 45 秒竖屏短视频脚本。按 hook / setup / mechanism / proof / twist / takeaway 结构，给出旁白、屏幕文字、视觉意图和时长。'},
  {id: 'storyboard', label: '拆分镜', prompt: '请把当前脚本拆成 6 个分镜。每个分镜给出 shot 类型、对应 beat、画面构图、屏幕文字、时长和转场意图。'},
  {id: 'critique', label: '检查事实与节奏', prompt: '请审查当前脚本：指出事实性风险、过度简化、节奏拖沓、视觉不可实现的地方，并给出具体修改建议。'},
];

function inputStyle(): Record<string, string> {
  return {background: 'var(--ink-950)', borderColor: 'var(--line)', color: 'var(--fg)'};
}
function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value ?? {}));
}
function asArray<T = any>(value: any): T[] {
  return Array.isArray(value) ? value : [];
}
function asText(value: any): string {
  return value == null ? '' : String(value);
}
function splitLines(value: string): string[] {
  return value.split('\n').map((item) => item.trim()).filter(Boolean);
}
function joinLines(value: any): string {
  return Array.isArray(value) ? value.join('\n') : asText(value);
}
function statusLabel(status: string): string {
  if (status === 'approved') return '已批准';
  if (status === 'materialized') return '已创建项目';
  return '草稿';
}
function statusColor(status: string): string {
  return status === 'approved' || status === 'materialized' ? 'var(--ok)' : 'var(--warn)';
}

function Field({label, value, onChange, placeholder, textarea = false, rows = 3, mono = false}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  textarea?: boolean;
  rows?: number;
  mono?: boolean;
}) {
  return (
    <label className="block min-w-0">
      <span className="field-label">{label}</span>
      {textarea ? (
        <textarea
          className="field-textarea"
          style={{minHeight: rows * 22, fontFamily: mono ? 'monospace' : 'inherit'}}
          value={value}
          placeholder={placeholder}
          onChange={(event) => onChange(event.target.value)}
        />
      ) : (
        <input
          className="field-input"
          value={value}
          placeholder={placeholder}
          onChange={(event) => onChange(event.target.value)}
        />
      )}
    </label>
  );
}

function Card({title, icon: Icon, children, actions}: {title: string; icon: any; children: any; actions?: any}) {
  return (
    <section className="panel">
      <div className="panel-head">
        <Icon size={12} style={{color: 'var(--accent)'}} />
        <span className="text-[11px] font-semibold">{title}</span>
        <div className="ml-auto flex items-center gap-1">{actions}</div>
      </div>
      <div className="panel-body">{children}</div>
    </section>
  );
}

export function CreativeWorkspace({workspace, onClose, onMaterialized, initialCreativeId}: {workspace: string; onClose: () => void; onMaterialized: (projectId: string) => void | Promise<void>; initialCreativeId?: string}) {
  const [items, setItems] = useState<CreativeSummary[]>([]);
  const [selectedId, setSelectedId] = useState('');
  const [detail, setDetail] = useState<CreativePackage | null>(null);
  const [validation, setValidation] = useState<CreativeValidation | null>(null);
  const [activeTab, setActiveTab] = useState<TabID>('brief');
  const [editingShotIndex, setEditingShotIndex] = useState<number | null>(null);
  const [editing, setEditing] = useState<Record<string, any> | null>(null);
  const [dirty, setDirty] = useState(false);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [prompt, setPrompt] = useState('');
  const [showPrompt, setShowPrompt] = useState(false);
  const [railOpen, setRailOpen] = useState(true);
  const [inspectorOpen, setInspectorOpen] = useState(() => (typeof window === 'undefined' ? true : window.innerWidth >= 1280));
  const promptRef = useRef<HTMLTextAreaElement | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newBrief, setNewBrief] = useState({
    title: '',
    topic: '',
    goal: '',
    audience: '泛知识用户 / 中学生',
    platform: 'douyin',
    duration_seconds: '45',
    tone: '好奇、克制、可视化解释',
    constraints: '不能用常见误解解释\n必须给出来源',
    success_criteria: '观众能复述核心机制\n没有事实性错误',
    style_pack: 'science-sketch',
    style_version: '0.1.1',
  });
  const [inspectorTab, setInspectorTab] = useState<'logic' | 'sources' | 'versions' | 'plan'>('logic');
  const [creativeQuery, setCreativeQuery] = useState('');
  const [creativeFilter, setCreativeFilter] = useState<'all' | 'draft' | 'approved' | 'materialized'>('all');
  const [projectInfo, setProjectInfo] = useState<ProjectDetail | null>(null);
  const [providers, setProviders] = useState<ShotProvider[]>([]);
  const [timelinePlan, setTimelinePlan] = useState<TimelinePlan | null>(null);
  const [planBusy, setPlanBusy] = useState(false);
  const [shotAssets, setShotAssets] = useState<Record<string, ShotProduceAsset>>({});
  const [remoteUpdate, setRemoteUpdate] = useState<{revision: number; updated_at: string} | null>(null);
  const dismissedRemoteRef = useRef(0);
  const remoteStateRef = useRef({selectedId: '', revision: 0, dirty: false, busy: false, loading: false});

  const brief = detail?.brief ?? {};
  const research = detail?.research ?? {};
  const script = detail?.script ?? {};
  const storyboard = detail?.storyboard ?? {};
  const production = detail?.production ?? {};
  const claims = asArray(research.claims);
  const sources = asArray(research.sources);
  const beats = asArray(script.beats);
  const shots = asArray(storyboard.shots);
  remoteStateRef.current = {selectedId, revision: detail?.revision ?? 0, dirty, busy, loading};

  async function load(preferred?: string) {
    setLoading(true);
    setError(null);
    setRemoteUpdate(null);
    dismissedRemoteRef.current = 0;
    try {
      const result = await callCapability<CreativeList>('media.workbench.creative.list', {workspace_root: workspace});
      const next = result.items ?? [];
      setItems(next);
      setSelectedId((current) => {
        if (preferred) return preferred;
        return current && next.some((item) => item.id === current) ? current : next[0]?.id ?? '';
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function select(id: string) {
    setSelectedId(id);
    setError(null);
    setNotice(null);
    setRemoteUpdate(null);
    dismissedRemoteRef.current = 0;
    try {
      const result = await callCapability<{creative: CreativePackage; validation?: CreativeValidation}>('media.workbench.creative.get', {
        workspace_root: workspace,
        creative_id: id,
      });
      setDetail(result.creative);
      setValidation(result.validation ?? null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function checkRemote() {
    const state = remoteStateRef.current;
    if (!state.selectedId || state.busy || state.loading) return;
    try {
      const result = await callCapability<CreativeList>('media.workbench.creative.list', {workspace_root: workspace});
      const next = result.items ?? [];
      setItems((prev) => {
        const changed = prev.length !== next.length || prev.some((item, index) => item.id !== next[index]?.id || item.revision !== next[index]?.revision || item.updated_at !== next[index]?.updated_at);
        return changed ? next : prev;
      });
      const item = next.find((entry) => entry.id === state.selectedId);
      if (!item || item.revision === state.revision) return;
      if (item.revision === dismissedRemoteRef.current) return;
      if (!remoteStateRef.current.dirty) {
        await select(state.selectedId);
        setNotice(`AI 已在客户端更新，已刷新到 r${item.revision}`);
      } else {
        setRemoteUpdate({revision: item.revision, updated_at: item.updated_at});
      }
    } catch {
      // 轮询失败不打断创作；用户仍可手动刷新
    }
  }

  useEffect(() => {
    void load(initialCreativeId);
  }, [workspace]);

  useEffect(() => {
    if (!detail?.id) {
      setShotAssets({});
      return;
    }
    void loadShotStatus(detail.id);
  }, [detail?.id, detail?.revision, workspace]);

  useEffect(() => {
    void (async () => {
      try {
        const result = await callCapability<{items: ShotProvider[]}>('media.workbench.shot.providers', {workspace_root: workspace});
        setProviders(result.items ?? []);
      } catch {
        setProviders([]);
      }
    })();
  }, [workspace]);



  useEffect(() => {
    if (selectedId) void select(selectedId);
    else setDetail(null);
  }, [selectedId]);

  useEffect(() => {
    if (!detail) {
      setEditing(null);
      setDirty(false);
      return;
    }
    setEditing(clone(detail[activeTab] ?? {}));
    setDirty(false);
  }, [detail, activeTab]);

  useEffect(() => {
    const projectId = detail?.links?.project_id;
    if (!projectId) {
      setProjectInfo(null);
      return;
    }
    let cancelled = false;
    void (async () => {
      try {
        const info = await callCapability<ProjectDetail>('media.workbench.project.get', {workspace_root: workspace, project_id: projectId});
        if (!cancelled) setProjectInfo(info);
      } catch {
        if (!cancelled) setProjectInfo(null);
      }
    })();
    return () => { cancelled = true; };
  }, [detail?.links?.project_id, workspace]);

  const filteredItems = useMemo(() => {
    const query = creativeQuery.trim().toLowerCase();
    return items.filter((item) => {
      const matchesQuery = !query || (item.title || '').toLowerCase().includes(query) || item.id.toLowerCase().includes(query);
      const matchesFilter = creativeFilter === 'all' || item.status === creativeFilter;
      return matchesQuery && matchesFilter;
    });
  }, [items, creativeQuery, creativeFilter]);

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
        event.preventDefault();
        if (dirty && detail && editing) void saveSection();
        return;
      }
      if (event.key === 'Escape') {
        if (editingShotIndex !== null) {
          setEditingShotIndex(null);
          return;
        }
        handleClose();
      }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [dirty, detail, editing, activeTab, onClose, editingShotIndex]);

  useEffect(() => {
    const timer = window.setInterval(() => { void checkRemote(); }, 4000);
    const onFocus = () => { void checkRemote(); };
    window.addEventListener('focus', onFocus);
    return () => {
      window.clearInterval(timer);
      window.removeEventListener('focus', onFocus);
    };
  }, [workspace]);

  function patch(key: string, value: any) {
    setEditing((prev) => ({...(prev ?? {}), [key]: value}));
    setDirty(true);
  }
  function patchArray(section: string, index: number, key: string, value: any) {
    setEditing((prev) => {
      const list = asArray(prev?.[section]).map((item, i) => (i === index ? {...item, [key]: value} : item));
      return {...(prev ?? {}), [section]: list};
    });
    setDirty(true);
  }
  function addArrayItem(section: string, item: any) {
    setEditing((prev) => ({...(prev ?? {}), [section]: [...asArray(prev?.[section]), item]}));
    setDirty(true);
  }
  function removeArrayItem(section: string, index: number) {
    setEditing((prev) => ({...(prev ?? {}), [section]: asArray(prev?.[section]).filter((_, i) => i !== index)}));
    setDirty(true);
  }
  function patchShotsBulk(indices: number[], key: string, value: any) {
    setEditing((prev) => {
      const list = [...asArray(prev?.shots)];
      for (const index of indices) {
        if (list[index]) list[index] = {...list[index], [key]: value};
      }
      return {...(prev ?? {}), shots: list};
    });
    setDirty(true);
  }
  function deleteShots(indices: number[]) {
    if (!indices.length) return;
    setEditing((prev) => ({...(prev ?? {}), shots: asArray(prev?.shots).filter((_, index) => !indices.includes(index))}));
    setDirty(true);
    setEditingShotIndex(null);
  }
  function addShot() {
    const index = asArray(editing?.shots).length;
    addArrayItem('shots', {id: `shot${index + 1}`, shot: '', beat_id: '', duration_seconds: 5, visual: '', text: '', provider: '', technique: '', prompt: '', fallback_provider: '', status: 'planned'});
    setEditingShotIndex(index);
  }

  function moveArrayItem(section: string, index: number, delta: number) {
    setEditing((prev) => {
      const list = [...asArray(prev?.[section])];
      const next = index + delta;
      if (next < 0 || next >= list.length) return prev ?? {};
      [list[index], list[next]] = [list[next], list[index]];
      return {...(prev ?? {}), [section]: list};
    });
    setDirty(true);
  }

  function changeTab(next: TabID) {
    if (next === activeTab) return;
    if (dirty && !window.confirm('当前修改还没有保存，切换会丢失。继续？')) return;
    setActiveTab(next);
  }

  async function saveSection() {
    if (!detail || !editing) return;
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const result = await callCapability<{creative: CreativePackage}>('media.workbench.creative.update', {
        workspace_root: workspace,
        creative_id: detail.id,
        section: activeTab,
        value: editing,
      });
      setDetail(result.creative);
      setItems((prev) => prev.map((item) => item.id === result.creative.id
        ? {...item, title: result.creative.title, status: result.creative.status, revision: result.creative.revision, updated_at: result.creative.updated_at}
        : item));
      setRemoteUpdate(null);
      dismissedRemoteRef.current = 0;
      setNotice(`${TABS.find((tab) => tab.id === activeTab)?.label ?? activeTab} 已保存为 r${result.creative.revision}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function validate() {
    if (!selectedId) return;
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const result = await callCapability<CreativeValidation>('media.workbench.creative.validate', {workspace_root: workspace, creative_id: selectedId});
      setValidation(result);
      setNotice(result.valid ? '创作包校验通过' : `还有 ${result.issues?.length ?? 0} 个问题`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function loadShotStatus(creativeId: string) {
    try {
      const result = await callCapability<ShotStatusResult>('media.workbench.shot.status', {workspace_root: workspace, creative_id: creativeId});
      const map: Record<string, ShotProduceAsset> = {};
      for (const item of result.items ?? []) {
        const candidates = item.assets?.filter((entry) => entry.exists && entry.preview_data_url) ?? [];
        const asset = candidates[candidates.length - 1];
        if (asset) {
          map[item.shot_id] = {shot_id: item.shot_id, provider: item.provider || '', path: asset.path, preview_data_url: asset.preview_data_url, mock: item.mock};
        }
      }
      setShotAssets(map);
    } catch {
      setShotAssets({});
    }
  }

  async function produceShots(indices: number[]) {
    if (!detail || !editing) return;
    if (dirty) {
      setNotice('先保存本节，再生成镜头资产');
      return;
    }
    const currentShots = asArray(editing.shots);
    const aiImageIds = indices
      .map((index) => currentShots[index])
      .filter((shot) => asText(shot?.provider) === 'ai.image')
      .map((shot) => asText(shot?.id))
      .filter(Boolean);
    if (!aiImageIds.length) {
      setNotice('选中镜头中没有 AI 图片镜头');
      return;
    }
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const result = await callCapability<ShotProduceResult>('media.workbench.shot.produce', {
        workspace_root: workspace,
        creative_id: detail.id,
        shot_ids: aiImageIds,
        mode: 'mock',
      });
      if (result.assets?.length) {
        setShotAssets((prev) => {
          const next = {...prev};
          for (const asset of result.assets ?? []) next[asset.shot_id] = asset;
          return next;
        });
      }
      await select(detail.id);
      const blockers = result.blockers?.length ?? 0;
      setNotice(`已生成 ${result.assets?.length ?? 0} 个模拟图片${blockers ? `；${blockers} 个镜头被阻断` : ''}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function loadTimelinePlan() {
    if (!selectedId) return;
    setPlanBusy(true);
    setError(null);
    try {
      const result = await callCapability<TimelinePlan>('media.workbench.timeline.plan', {workspace_root: workspace, creative_id: selectedId});
      setTimelinePlan(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setPlanBusy(false);
    }
  }

  async function approve() {
    if (!selectedId) return;
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const result = await callCapability<{state?: string; blockers?: unknown[]}>('media.workbench.creative.approve', {workspace_root: workspace, creative_id: selectedId});
      if (result.state === 'blocked') throw new Error(`批准被阻断：${result.blockers?.length ?? 0} 个问题`);
      await select(selectedId);
      setNotice('创作包已批准');
      await load(selectedId);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function materialize() {
    if (!selectedId) return;
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const result = await callCapability<{state?: string; project?: {id?: string}; blockers?: unknown[]}>('media.workbench.creative.materialize', {workspace_root: workspace, creative_id: selectedId});
      const projectId = result.project?.id;
      if (result.state === 'blocked' || !projectId) throw new Error(`创建项目被阻断：${result.blockers?.length ?? 0} 个问题`);
      await onMaterialized(projectId);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function createCreative() {
    if (!newBrief.title.trim()) {
      setError('先给创意一个标题');
      return;
    }
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const created = await callCapability<{creative: CreativePackage}>('media.workbench.creative.create', {
        workspace_root: workspace,
        title: newBrief.title.trim(),
        brief: {
          title: newBrief.title.trim(),
          topic: newBrief.topic.trim(),
          goal: newBrief.goal.trim(),
          audience: newBrief.audience.trim(),
          platform: newBrief.platform.trim(),
          duration_seconds: Number(newBrief.duration_seconds) || 45,
          tone: newBrief.tone.trim(),
          constraints: splitLines(newBrief.constraints),
          success_criteria: splitLines(newBrief.success_criteria),
          source: '',
        },
      });
      await callCapability('media.workbench.creative.update', {
        workspace_root: workspace,
        creative_id: created.creative.id,
        section: 'production',
        value: {
          mode: 'code',
          style_pack: newBrief.style_pack.trim() || 'science-sketch',
          style_version: newBrief.style_version.trim() || '0.1.1',
          renderer: 'browser-code',
          duration_seconds: Number(newBrief.duration_seconds) || 45,
          aspect: '9:16',
          platform: newBrief.platform.trim() || 'douyin',
          required_data: ['scenes', 'claims', 'sources'],
          required_assets: [],
          quality_gates: ['safe_area', 'text_bounds', 'text_overlap', 'determinism'],
        },
      });
      setShowCreate(false);
      setActiveTab('brief');
      await load(created.creative.id);
      setNotice('创意包已创建，可以把 AI 产出填进右侧各段');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  function applyAction(action: typeof AI_ACTIONS[number]) {
    const context = detail
      ? `当前创意：${detail.title}\n目标：${asText(brief.goal) || '（未填写）'}\n受众：${asText(brief.audience) || '（未填写）'}\n平台：${asText(brief.platform) || 'douyin'}\n时长：${asText(brief.duration_seconds) || 45}s\n风格：${asText(production.style_pack) || 'science-sketch'}@${asText(production.style_version) || '0.1.1'}\n当前脚本：\n${beats.map((beat: any, index: number) => `${index + 1}. ${asText(beat.narration || beat.on_screen_text)}`).join('\n') || '（尚未生成）'}`
      : '我想做一个新的科普类手绘短视频。';
    setPrompt(`${context}\n\n【这次请 AI 做的事】\n${action.prompt}\n\n请直接输出可以写入 CreativePackage 的结构化内容，不要写泛泛的建议。`);
    setNotice('提示词已生成。复制后到 HiMind AI 对话继续共创。');
  }

  async function copyPrompt() {
    if (!prompt.trim()) {
      setNotice('先选一个 AI 动作，或写下你想让 AI 做什么');
      return;
    }
    try {
      await Promise.race([
        navigator.clipboard.writeText(prompt),
        new Promise((_, reject) => setTimeout(() => reject(new Error('clipboard timeout')), 300)),
      ]);
      setNotice('提示词已复制，粘贴到 HiMind AI 对话继续共创');
      return;
    } catch {
      // WebView2 的 Clipboard API 可能被权限或焦点拦截，退回“全选 + Ctrl+C”。
    }
    const area = promptRef.current;
    if (area) {
      area.focus();
      area.select();
      setNotice('已选中提示词，按 Ctrl+C 复制');
      return;
    }
    setNotice('复制失败，请手动选中文案复制');
  }

  async function openAi() {
    try {
      await hostInvoke('show_main_window');
      handleClose();
    } catch {
      setNotice('请切换到 HiMind Agent 主窗口的 AI 对话');
    }
  }

  const draftBrief = activeTab === 'brief' && editing ? editing : brief;
  const draftResearch = activeTab === 'research' && editing ? editing : research;
  const draftScript = activeTab === 'script' && editing ? editing : script;
  const draftStoryboard = activeTab === 'storyboard' && editing ? editing : storyboard;
  const draftProduction = activeTab === 'production' && editing ? editing : production;
  const draftClaims = asArray(draftResearch.claims);
  const draftSources = asArray(draftResearch.sources);
  const draftBeats = asArray(draftScript.beats);
  const draftShots = asArray(draftStoryboard.shots);
  const beatDuration = draftBeats.reduce((sum: number, beat: any) => sum + (Number(beat.duration_seconds) || 0), 0);
  const sceneCount = asArray((draftProduction.data as any)?.scenes).length;

  const flow = useMemo(() => {
    if (!detail) return [] as Array<{id: string; label: string; detail: string; ready: boolean; tab: TabID | null}>;
    const nodes: Array<{id: string; label: string; detail: string; ready: boolean; tab: TabID | null}> = [
      {id: 'brief', label: 'Brief', detail: `${asText(draftBrief.duration_seconds) || '—'}s · ${asText(draftBrief.audience) || '未写受众'}`, ready: Boolean(asText(draftBrief.goal) && asText(draftBrief.audience)), tab: 'brief'},
      {id: 'research', label: '证据链', detail: `${draftClaims.length} claims · ${draftSources.length} sources`, ready: draftClaims.length > 0 && draftSources.length > 0, tab: 'research'},
      {id: 'script', label: '叙事脚本', detail: `${draftBeats.length} beats · ${beatDuration}s`, ready: draftBeats.length > 0, tab: 'script'},
      {id: 'storyboard', label: '分镜', detail: `${draftShots.length} shots`, ready: draftShots.length > 0, tab: 'storyboard'},
      {id: 'production', label: '制作规格', detail: `${asText(draftProduction.style_pack) || '未选风格'}@${asText(draftProduction.style_version) || '—'} · ${sceneCount} scenes`, ready: Boolean(asText(draftProduction.style_pack) && asText(draftProduction.mode)), tab: 'production'},
      {id: 'approval', label: '校验/批准', detail: validation ? (validation.valid ? '校验通过' : `${validation.issues?.length ?? 0} 个问题`) : '尚未校验', ready: Boolean(validation?.valid), tab: null},
      {id: 'project', label: '项目/成片', detail: asText(detail.links?.project_id) || '尚未创建项目', ready: Boolean(detail.links?.project_id), tab: null},
    ];
    return nodes;
  }, [detail, draftBrief, draftClaims.length, draftSources.length, draftBeats, beatDuration, draftShots, draftProduction, sceneCount, validation]);

  const logicRisks = useMemo(() => {
    if (!detail) return [] as Array<{code: string; text: string; level: 'error' | 'warn'}>;
    const risks: Array<{code: string; text: string; level: 'error' | 'warn'}> = [];
    if (!asText(draftBrief.goal)) risks.push({code: 'BRIEF_GOAL', text: 'Brief 缺少创作目标', level: 'error'});
    if (!asText(draftBrief.audience)) risks.push({code: 'BRIEF_AUDIENCE', text: 'Brief 缺少受众', level: 'warn'});
    if (!draftClaims.length) risks.push({code: 'NO_CLAIMS', text: '研究区还没有事实 claim', level: 'error'});
    if (!draftSources.length) risks.push({code: 'NO_SOURCES', text: '研究区还没有来源 URL', level: 'error'});
    if (!draftBeats.length) risks.push({code: 'NO_BEATS', text: '脚本区还没有 beat', level: 'error'});
    if (draftBeats.length && draftShots.length && draftBeats.length !== draftShots.length) risks.push({code: 'BEAT_SHOT_MISMATCH', text: `脚本 ${draftBeats.length} beats 与分镜 ${draftShots.length} shots 数量不一致`, level: 'warn'});
    if (draftBeats.length && beatDuration && Number(draftBrief.duration_seconds) && Math.abs(beatDuration - Number(draftBrief.duration_seconds)) > 1) risks.push({code: 'DURATION_DRIFT', text: `脚本总时长 ${beatDuration}s 与 Brief ${draftBrief.duration_seconds}s 不一致`, level: 'warn'});
    if (!asText(draftProduction.style_pack)) risks.push({code: 'NO_STYLE', text: '制作规格还没有选风格包', level: 'error'});
    for (const issue of validation?.issues ?? []) risks.push({code: issue.code, text: issue.message, level: 'error'});
    if (validation?.valid && !detail.approval?.approved_revision) risks.push({code: 'NOT_APPROVED', text: '当前 revision 已通过校验但尚未批准', level: 'warn'});
    return risks;
  }, [detail, draftBrief, draftClaims.length, draftSources.length, draftBeats, beatDuration, draftShots, draftProduction, validation]);

  const projectArtifacts = projectInfo?.artifacts ?? [];
  const videoArtifacts = projectArtifacts.filter((artifact) => artifact.kind.includes('video'));
  const latestVideo = [...videoArtifacts].sort((a, b) => (b.size || 0) - (a.size || 0))[0];
  const claimsWithoutEvidence = draftClaims.filter((claim: any) => !asArray(claim.evidence_refs).length);

  function providerLabel(provider: ShotProvider) {
    if (provider.source === 'agent') return `${provider.name} · 内置 AI`;
    if (provider.source === 'external') return `${provider.name} · 需安装扩展`;
    return provider.name;
  }

  function flowColor(node: {id: string; ready: boolean}) {
    if (node.ready) return 'var(--ok)';
    if (node.id === 'approval' && !validation) return 'var(--fg-dim)';
    return 'var(--warn)';
  }

  function focusNode(node: {id: string; tab: TabID | null; ready: boolean}) {
    if (node.tab) {
      changeTab(node.tab);
      return;
    }
    if (node.id === 'approval') {
      void validate();
      return;
    }
    if (node.id === 'project') {
      setNotice(node.ready ? `项目 ${detail?.links?.project_id} 已创建` : '先批准创意，再创建项目');
    }
  }

  function handleClose() {
    if (dirty && !window.confirm('当前修改还没有保存，关闭会丢失。继续？')) return;
    onClose();
  }

  const activeShot = editing && editingShotIndex !== null ? asArray(editing.shots)[editingShotIndex] : null;
  const activeShotProvider = activeShot ? providers.find((provider) => provider.id === asText(activeShot.provider)) : undefined;
  const totalShots = asArray(editing?.shots).length;

  async function loadRemote() {
    if (!remoteUpdate) return;
    const revision = remoteUpdate.revision;
    setRemoteUpdate(null);
    setDirty(false);
    await select(selectedId);
    setNotice(`已加载 AI 更新 r${revision}`);
  }

  return (
    <div className="studio-shell">
      <header className="studio-topbar">
        <button type="button" className="btn btn-ghost" onClick={handleClose}><ChevronLeft size={13} />返回</button>
        <div className="h-4 w-px" style={{background: 'var(--line)'}} />
        <div className="flex min-w-0 items-center gap-2">
          <GitBranch size={14} style={{color: 'var(--accent)'}} />
          <span className="text-[11px]" style={{color: 'var(--fg-dim)'}}>创作</span>
          <span style={{color: 'var(--fg-dim)'}}>/</span>
          <strong className="truncate text-[13px]">{detail?.title || '未选择创意'}</strong>
          {detail ? <span className="badge" style={{color: statusColor(detail.status)}}>{statusLabel(detail.status)}</span> : null}
          {detail ? <span className="badge">r{detail.revision}</span> : null}
          {busy ? (
            <span className="badge badge-accent">AI 处理中</span>
          ) : remoteUpdate ? (
            <span className="badge badge-warn">AI 已更新 r{remoteUpdate.revision}</span>
          ) : dirty ? (
            <span className="badge badge-warn">未保存</span>
          ) : (
            <span className="badge badge-ok" title="每 4 秒检查一次；窗口重新获得焦点时立即检查">AI 协作 · 已同步</span>
          )}
        </div>
        <div className="ml-auto flex items-center gap-2">
          <button type="button" className="btn" onClick={() => { setError(null); setShowCreate(true); }} disabled={busy} title="新建创意"><Plus size={12} /><span className="hidden xl:inline">新建创意</span></button>
          <button type="button" className="icon-btn" title={railOpen ? '隐藏脉络' : '显示脉络'} onClick={() => setRailOpen((value) => !value)}><PanelLeft size={14} /></button>
          <button type="button" className="icon-btn" title={inspectorOpen ? '隐藏检查' : '显示检查'} onClick={() => setInspectorOpen((value) => !value)}><PanelRight size={14} /></button>
          <button type="button" className="icon-btn" onClick={() => void load(selectedId)} disabled={busy || loading} title="刷新"><RefreshCw size={14} /></button>
          <button type="button" className="btn btn-ghost" onClick={() => void openAi()} title="在 AI 客户端继续"><ExternalLink size={12} /><span className="hidden xl:inline">AI 对话</span></button>
          <ThemeToggle />
          <button type="button" className="btn btn-primary" onClick={() => void saveSection()} disabled={busy || !dirty}><Save size={12} />保存</button>
          <button type="button" className="icon-btn" aria-label="关闭" onClick={handleClose}><X size={15} /></button>
        </div>
      </header>

      {error ? <div className="px-4 py-2 text-[11.5px]" style={{background: 'rgba(255,107,107,.08)', color: 'var(--err)', borderBottom: '1px solid rgba(255,107,107,.18)'}}>{error}</div> : null}
      {notice ? <div className="px-4 py-2 text-[11.5px]" style={{background: 'rgba(61,220,151,.06)', color: 'var(--ok)', borderBottom: '1px solid rgba(61,220,151,.16)'}}>{notice}</div> : null}

      {remoteUpdate ? (
        <div className="flex items-center gap-2 px-4 py-2 text-[11.5px]" style={{background: 'rgba(245,196,81,.07)', color: 'var(--warn)', borderBottom: '1px solid rgba(245,196,81,.18)'}}>
          <RefreshCw size={12} />
          <span>AI 已在客户端更新到 r{remoteUpdate.revision}；本地有未保存修改。</span>
          <button type="button" className="btn ml-auto" onClick={() => void loadRemote()}>加载新版</button>
          <button type="button" className="btn btn-ghost" onClick={() => { dismissedRemoteRef.current = remoteUpdate.revision; setRemoteUpdate(null); }}>保留本地</button>
        </div>
      ) : null}

        <div className="studio-body" style={{gridTemplateColumns: `${railOpen ? 300 : 0}px minmax(0,1fr) ${inspectorOpen ? 320 : 0}px`}}>
          {railOpen ? <aside className="studio-rail" style={{gridColumn: '1'}}>
            <div className="studio-section">
              <div className="studio-section-head">
                <span>创意</span>
                <span className="badge">{filteredItems.length}/{items.length}</span>
                <button type="button" className="icon-btn ml-auto" onClick={() => { setError(null); setShowCreate(true); }} title="新建创意"><Plus size={13} /></button>
              </div>
              <div className="studio-section-body space-y-2">
                <label className="relative block">
                  <Search size={12} className="absolute left-2 top-1/2 -translate-y-1/2" style={{color: 'var(--fg-dim)'}} />
                  <input className="field-input pl-7" value={creativeQuery} onChange={(event) => setCreativeQuery(event.target.value)} placeholder="搜索标题或 ID" />
                </label>
                <div className="segmented w-full">
                  {[{id: 'all', label: '全部'}, {id: 'draft', label: '草稿'}, {id: 'approved', label: '已批准'}, {id: 'materialized', label: '已创建'}].map((filter) => (
                    <button key={filter.id} type="button" className="segmented-item flex-1" data-active={creativeFilter === filter.id} onClick={() => setCreativeFilter(filter.id as any)}>{filter.label}</button>
                  ))}
                </div>
                <div className="max-h-[190px] space-y-1 overflow-y-auto pr-1">
                  {loading ? <div className="flex items-center gap-2 p-2 text-[11px]" style={{color: 'var(--fg-dim)'}}><Loader2 size={12} className="animate-spin" />读取中</div> : null}
                  {filteredItems.map((item) => (
                    <button
                      key={item.id}
                      type="button"
                      className="w-full rounded-md border px-2 py-2 text-left"
                      style={{
                        borderColor: item.id === selectedId ? 'rgba(90,169,255,.35)' : 'transparent',
                        background: item.id === selectedId ? 'var(--accent-soft)' : 'transparent',
                      }}
                      onClick={() => { if (dirty && !window.confirm('当前修改还没有保存，切换会丢失。继续？')) return; setSelectedId(item.id); }}
                    >
                      <div className="truncate text-[11.5px] font-medium">{item.title || item.id}</div>
                      <div className="mt-1 flex items-center gap-2 text-[9.5px]" style={{color: 'var(--fg-dim)'}}>
                        <span style={{color: statusColor(item.status)}}>{statusLabel(item.status)}</span>
                        <span>r{item.revision}</span>
                        {item.project_id ? <span className="truncate">→ {item.project_id}</span> : null}
                      </div>
                    </button>
                  ))}
                  {!loading && !items.length ? <div className="p-2 text-[11px]" style={{color: 'var(--fg-dim)'}}>还没有创意包，点右上角“新建创意”。</div> : null}
                  {!loading && items.length > 0 && !filteredItems.length ? <div className="p-2 text-[11px]" style={{color: 'var(--fg-dim)'}}>没有匹配的创意。</div> : null}
                </div>
              </div>
            </div>

            <div className="flex min-h-0 flex-1 flex-col">
              <div className="flex items-center gap-2 border-b px-3 py-2" style={{borderColor: 'var(--line)'}}>
                <GitBranch size={13} style={{color: 'var(--accent)'}} />
                <span className="text-[11px] font-semibold">创作脉络</span>
                <span className="badge ml-auto">{flow.filter((node) => node.ready).length}/{flow.length || 0}</span>
              </div>
              <div className="min-h-0 flex-1 overflow-y-auto p-3">
                {flow.map((node, index) => (
                  <button key={node.id} type="button" className="flex w-full gap-2 pb-3 text-left" onClick={() => focusNode(node)}>
                    <span className="flex w-5 shrink-0 flex-col items-center">
                      <span className="flex h-5 w-5 items-center justify-center rounded-full border text-[9px]" style={{borderColor: flowColor(node), color: flowColor(node)}}>{index + 1}</span>
                      {index < flow.length - 1 ? <span className="mt-1 w-px flex-1" style={{background: 'var(--line)'}} /> : null}
                    </span>
                    <span className="min-w-0 flex-1 rounded-md border px-2 py-1.5" style={{borderColor: activeTab === node.tab ? 'rgba(90,169,255,.35)' : 'var(--line)', background: activeTab === node.tab ? 'var(--accent-soft)' : 'transparent'}}>
                      <span className="flex items-center gap-2">
                        <span className="text-[11px] font-medium">{node.label}</span>
                        <span className="ml-auto text-[9px]" style={{color: flowColor(node)}}>{node.ready ? '就绪' : '待补'}</span>
                      </span>
                      <span className="mt-1 block text-[9.5px] leading-4" style={{color: 'var(--fg-dim)'}}>{node.detail}</span>
                    </span>
                  </button>
                ))}
                {!detail ? <div className="text-[11px]" style={{color: 'var(--fg-dim)'}}>选择或新建创意后，这里会显示创作脉络。</div> : null}
              </div>
              <div className="border-t p-3" style={{borderColor: 'var(--line)'}}>
                <div className="mb-2 flex items-center gap-2">
                  <Bot size={12} style={{color: 'var(--accent)'}} />
                  <span className="text-[10px] font-semibold">交给 AI</span>
                  <span className="badge ml-auto">客户端对话</span>
                </div>
                <div className="flex flex-wrap gap-1">
                  {AI_ACTIONS.slice(1, 4).map((action) => (
                    <button key={action.id} type="button" className="badge" style={{cursor: 'pointer'}} onClick={() => { applyAction(action); setShowPrompt(true); }}>{action.label}</button>
                  ))}
                  <button type="button" className="badge" style={{cursor: 'pointer'}} onClick={() => setShowPrompt((value) => !value)}>{showPrompt ? '收起任务' : '任务提示词'}</button>
                </div>
                {showPrompt ? (
                  <>
                    <textarea
                      className="mt-2 w-full rounded-md border p-2 text-[11px] leading-5"
                      style={{...inputStyle(), minHeight: 92}}
                      ref={promptRef}
                      value={prompt}
                      placeholder="例如：把第 3 段改成从观察者视角解释……"
                      onChange={(event) => setPrompt(event.target.value)}
                    />
                    <div className="mt-2 flex gap-1">
                      <button type="button" className="btn flex-1" onClick={() => void copyPrompt()}><Copy size={12} />复制/选中</button>
                      <button type="button" className="btn" onClick={() => void openAi()}><ExternalLink size={12} />AI 对话</button>
                    </div>
                  </>
                ) : null}
                <div className="mt-2 text-[9.5px] leading-4" style={{color: 'var(--fg-dim)'}}>AI 对话在 HiMind 客户端；这里只生成带当前脉络上下文的任务，不维护第二套聊天记录。</div>
              </div>
            </div>
          </aside> : null}

          <main className="studio-canvas" style={{gridColumn: '2'}}>
            <div className="studio-canvas-scroll">
            <div className="mx-auto max-w-[980px] p-5">
            {showCreate ? (
              <div className="mx-auto max-w-[760px]">
                <div className="mb-3 flex items-center gap-2">
                  <Lightbulb size={15} style={{color: 'var(--warn)'}} />
                  <div><div className="text-[15px] font-semibold">新建创意</div><div className="text-[10.5px]" style={{color: 'var(--fg-dim)'}}>先写清楚要做什么，再让 AI 在客户端补细节。</div></div>
                  <button type="button" className="icon-btn ml-auto" onClick={() => { setError(null); setShowCreate(false); }}><X size={14} /></button>
                </div>
                <div className="grid gap-3 md:grid-cols-2">
                  <Field label="标题" value={newBrief.title} onChange={(value) => setNewBrief((prev) => ({...prev, title: value}))} placeholder="例如：天空为什么是蓝色的？" />
                  <Field label="主题" value={newBrief.topic} onChange={(value) => setNewBrief((prev) => ({...prev, topic: value}))} placeholder="例如：瑞利散射" />
                  <div className="md:col-span-2"><Field label="创作目标" value={newBrief.goal} onChange={(value) => setNewBrief((prev) => ({...prev, goal: value}))} textarea rows={3} placeholder="用反直觉问题解释……" /></div>
                  <Field label="受众" value={newBrief.audience} onChange={(value) => setNewBrief((prev) => ({...prev, audience: value}))} />
                  <Field label="平台" value={newBrief.platform} onChange={(value) => setNewBrief((prev) => ({...prev, platform: value}))} />
                  <Field label="时长（秒）" value={newBrief.duration_seconds} onChange={(value) => setNewBrief((prev) => ({...prev, duration_seconds: value}))} />
                  <Field label="语气" value={newBrief.tone} onChange={(value) => setNewBrief((prev) => ({...prev, tone: value}))} />
                  <div className="md:col-span-2"><Field label="约束（每行一条）" value={newBrief.constraints} onChange={(value) => setNewBrief((prev) => ({...prev, constraints: value}))} textarea rows={3} /></div>
                  <div className="md:col-span-2"><Field label="成功标准（每行一条）" value={newBrief.success_criteria} onChange={(value) => setNewBrief((prev) => ({...prev, success_criteria: value}))} textarea rows={3} /></div>
                  <Field label="风格包" value={newBrief.style_pack} onChange={(value) => setNewBrief((prev) => ({...prev, style_pack: value}))} />
                  <Field label="风格版本" value={newBrief.style_version} onChange={(value) => setNewBrief((prev) => ({...prev, style_version: value}))} />
                </div>
                <div className="mt-4 flex items-center gap-2">
                  <button type="button" className="btn btn-primary" onClick={() => void createCreative()} disabled={busy}>{busy ? <Loader2 size={12} className="animate-spin" /> : <Plus size={12} />}创建创意包</button>
                  <button type="button" className="btn" onClick={() => { setError(null); setShowCreate(false); }}>取消</button>
                </div>
              </div>
            ) : !detail || !editing ? (
              <div className="flex h-full flex-col items-center justify-center gap-2">
                <Wand2 size={24} style={{color: 'var(--fg-dim)'}} />
                <div className="text-[12.5px]" style={{color: 'var(--fg-muted)'}}>选择一个创意，或新建一条</div>
                <div className="text-[11px]" style={{color: 'var(--fg-dim)'}}>左侧是创作脉络，中间是结构化创作稿。</div>
              </div>
            ) : (
              <>
                <div className="mb-4 flex items-end justify-between gap-3">
                  <div className="min-w-0">
                    <div className="text-[10px] uppercase tracking-[0.1em]" style={{color: 'var(--fg-dim)'}}>{TABS.find((tab) => tab.id === activeTab)?.label}</div>
                    <div className="mt-1 truncate text-[20px] font-semibold">{detail.title}</div>
                    <div className="mt-1 flex flex-wrap items-center gap-2 text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
                      <span style={{color: statusColor(detail.status)}}>{statusLabel(detail.status)}</span>
                      <span>r{detail.revision}</span>
                      <span className="truncate">{detail.id}</span>
                      {detail.approval?.approved_revision ? <span>批准 r{detail.approval.approved_revision}</span> : null}
                    </div>
                  </div>
                  <div className="shrink-0 text-right text-[10px]" style={{color: 'var(--fg-dim)'}}>{dirty ? `未保存 · 保存后生成 r${detail.revision + 1}` : `已同步 r${detail.revision}`}</div>
                </div>

                <div className="mb-4 flex items-center gap-2">
                  <div className="segmented">
                    {TABS.map((tab) => {
                      const Icon = tab.icon;
                      return <button key={tab.id} type="button" className="segmented-item flex items-center gap-1.5" data-active={tab.id === activeTab} onClick={() => changeTab(tab.id)}><Icon size={11} />{tab.label}</button>;
                    })}
                  </div>
                  <div className="ml-auto text-[10px]" style={{color: 'var(--fg-dim)'}}>Ctrl+S 保存 · Esc 返回</div>
                </div>

                {activeTab === 'brief' ? (
                  <div className="grid gap-3 md:grid-cols-2">
                    <Field label="标题" value={asText(editing.title)} onChange={(value) => patch('title', value)} />
                    <Field label="主题" value={asText(editing.topic)} onChange={(value) => patch('topic', value)} />
                    <div className="md:col-span-2"><Field label="创作目标" value={asText(editing.goal)} onChange={(value) => patch('goal', value)} textarea rows={3} /></div>
                    <Field label="受众" value={asText(editing.audience)} onChange={(value) => patch('audience', value)} />
                    <Field label="平台" value={asText(editing.platform)} onChange={(value) => patch('platform', value)} />
                    <Field label="时长（秒）" value={asText(editing.duration_seconds)} onChange={(value) => patch('duration_seconds', Number(value) || 0)} />
                    <Field label="语气" value={asText(editing.tone)} onChange={(value) => patch('tone', value)} />
                    <div className="md:col-span-2"><Field label="约束（每行一条）" value={joinLines(editing.constraints)} onChange={(value) => patch('constraints', splitLines(value))} textarea rows={3} /></div>
                    <div className="md:col-span-2"><Field label="成功标准（每行一条）" value={joinLines(editing.success_criteria)} onChange={(value) => patch('success_criteria', splitLines(value))} textarea rows={3} /></div>
                    <div className="md:col-span-2"><Field label="来源说明" value={asText(editing.source)} onChange={(value) => patch('source', value)} placeholder="来源名称 / 待补 URL" /></div>
                  </div>
                ) : null}

                {activeTab === 'research' ? (
                  <div className="space-y-3">
                    <div className="grid gap-3 md:grid-cols-2">
                      <Field label="研究模式" value={asText(editing.mode)} onChange={(value) => patch('mode', value)} placeholder="required / optional" />
                      <div className="text-[11px] leading-5" style={{color: 'var(--fg-dim)'}}>事实、机制解释、常见误解分开写；每条 claim 尽量绑定来源。</div>
                    </div>
                    <Card title={`Claims · ${asArray(editing.claims).length}`} icon={FlaskConical} actions={<button type="button" className="btn" onClick={() => addArrayItem('claims', {id: `c${asArray(editing.claims).length + 1}`, text: '', confidence: 'medium', evidence_refs: []})}><Plus size={11} />添加</button>}>
                      <div className="space-y-2">
                        {asArray(editing.claims).map((claim: any, index: number) => (
                          <div key={index} className="rounded-md border p-2" style={{borderColor: 'var(--line)'}}>
                            <div className="mb-1 flex items-center gap-2"><span className="badge">{asText(claim.id) || `c${index + 1}`}</span><button type="button" className="icon-btn ml-auto" onClick={() => removeArrayItem('claims', index)}><Trash2 size={12} /></button></div>
                            <textarea className="w-full rounded-md border p-2 text-[11.5px] leading-5" style={{...inputStyle(), minHeight: 52}} value={asText(claim.text)} onChange={(event) => patchArray('claims', index, 'text', event.target.value)} />
                            <div className="mt-2 grid gap-2 md:grid-cols-2">
                              <Field label="置信度" value={asText(claim.confidence)} onChange={(value) => patchArray('claims', index, 'confidence', value)} />
                              <Field label="证据 refs（逗号分隔）" value={asArray(claim.evidence_refs).join(', ')} onChange={(value) => patchArray('claims', index, 'evidence_refs', value.split(',').map((item) => item.trim()).filter(Boolean))} />
                            </div>
                          </div>
                        ))}
                        {!asArray(editing.claims).length ? <div className="text-[11px]" style={{color: 'var(--fg-dim)'}}>还没有 claim。让 AI 补研究，或手动添加。</div> : null}
                      </div>
                    </Card>
                    <Card title={`Sources · ${asArray(editing.sources).length}`} icon={Link2} actions={<button type="button" className="btn" onClick={() => addArrayItem('sources', {id: `s${asArray(editing.sources).length + 1}`, publisher: '', title: '', url: ''})}><Plus size={11} />添加</button>}>
                      <div className="space-y-2">
                        {asArray(editing.sources).map((source: any, index: number) => (
                          <div key={index} className="rounded-md border p-2" style={{borderColor: 'var(--line)'}}>
                            <div className="mb-1 flex items-center gap-2"><span className="badge">{asText(source.id) || `s${index + 1}`}</span><button type="button" className="icon-btn ml-auto" onClick={() => removeArrayItem('sources', index)}><Trash2 size={12} /></button></div>
                            <div className="grid gap-2 md:grid-cols-2">
                              <Field label="发布方" value={asText(source.publisher)} onChange={(value) => patchArray('sources', index, 'publisher', value)} />
                              <Field label="标题" value={asText(source.title)} onChange={(value) => patchArray('sources', index, 'title', value)} />
                            </div>
                            <div className="mt-2"><Field label="URL" value={asText(source.url)} onChange={(value) => patchArray('sources', index, 'url', value)} placeholder="https://..." /></div>
                          </div>
                        ))}
                        {!asArray(editing.sources).length ? <div className="text-[11px]" style={{color: 'var(--fg-dim)'}}>来源为空，发布前必须补齐。</div> : null}
                      </div>
                    </Card>
                    <Card title="不确定性 / 发布前复核" icon={AlertTriangle}>
                      <Field label="每行一条" value={joinLines(editing.uncertainties)} onChange={(value) => patch('uncertainties', splitLines(value))} textarea rows={3} />
                    </Card>
                  </div>
                ) : null}

                {activeTab === 'script' ? (
                  <div className="space-y-3">
                    <div className="grid gap-3 md:grid-cols-2">
                      <Field label="Logline" value={asText(editing.logline)} onChange={(value) => patch('logline', value)} />
                      <Field label="结构" value={asText(editing.structure)} onChange={(value) => patch('structure', value)} />
                    </div>
                    <Card title={`Beats · ${asArray(editing.beats).length}`} icon={MessageSquare} actions={<button type="button" className="btn" onClick={() => addArrayItem('beats', {id: `b${asArray(editing.beats).length + 1}`, purpose: 'beat', duration_seconds: 5, narration: '', on_screen_text: '', visual_intent: ''})}><Plus size={11} />添加 beat</button>}>
                      <div className="space-y-2">
                        {asArray(editing.beats).map((beat: any, index: number) => (
                          <div key={index} className="rounded-md border p-2" style={{borderColor: 'var(--line)'}}>
                            <div className="mb-1 flex items-center gap-1">
                              <span className="badge">{asText(beat.id) || `b${index + 1}`}</span>
                              <span className="badge">{asText(beat.purpose)}</span>
                              <span className="ml-auto" />
                              <button type="button" className="icon-btn" onClick={() => moveArrayItem('beats', index, -1)}><ChevronUp size={12} /></button>
                              <button type="button" className="icon-btn" onClick={() => moveArrayItem('beats', index, 1)}><ChevronDown size={12} /></button>
                              <button type="button" className="icon-btn" onClick={() => removeArrayItem('beats', index)}><Trash2 size={12} /></button>
                            </div>
                            <div className="grid gap-2 md:grid-cols-[1fr_90px]">
                              <Field label="旁白" value={asText(beat.narration)} onChange={(value) => patchArray('beats', index, 'narration', value)} textarea rows={2} />
                              <Field label="时长" value={asText(beat.duration_seconds)} onChange={(value) => patchArray('beats', index, 'duration_seconds', Number(value) || 0)} />
                            </div>
                            <div className="mt-2 grid gap-2 md:grid-cols-2">
                              <Field label="屏幕文字" value={asText(beat.on_screen_text)} onChange={(value) => patchArray('beats', index, 'on_screen_text', value)} />
                              <Field label="视觉意图" value={asText(beat.visual_intent)} onChange={(value) => patchArray('beats', index, 'visual_intent', value)} />
                            </div>
                          </div>
                        ))}
                        {!asArray(editing.beats).length ? <div className="text-[11px]" style={{color: 'var(--fg-dim)'}}>还没有 beat。让 AI 写 45 秒脚本，或手动添加。</div> : null}
                      </div>
                    </Card>
                  </div>
                ) : null}

                {activeTab === 'storyboard' ? (
                  <Card title={`Shots · ${asArray(editing.shots).length}`} icon={Clapperboard}>
                    <ShotTable
                      shots={asArray(editing.shots)}
                      providers={providers}
                      providerLabel={providerLabel}
                      assets={shotAssets}
                      onPatch={(index, key, value) => patchArray('shots', index, key, value)}
                      onBulkPatch={patchShotsBulk}
                      onDelete={deleteShots}
                      onAdd={addShot}
                      onOpen={(index) => setEditingShotIndex(index)}
                      onProduce={(indices) => void produceShots(indices)}
                    />
                  </Card>
                ) : null}

                {activeTab === 'production' ? (
                  <div className="space-y-3">
                    <div className="grid gap-3 md:grid-cols-2">
                      <Field label="Mode" value={asText(editing.mode)} onChange={(value) => patch('mode', value)} placeholder="code / template" />
                      <Field label="Renderer" value={asText(editing.renderer)} onChange={(value) => patch('renderer', value)} />
                      <Field label="Style Pack" value={asText(editing.style_pack)} onChange={(value) => patch('style_pack', value)} />
                      <Field label="Style Version" value={asText(editing.style_version)} onChange={(value) => patch('style_version', value)} />
                      <Field label="时长（秒）" value={asText(editing.duration_seconds)} onChange={(value) => patch('duration_seconds', Number(value) || 0)} />
                      <Field label="画幅" value={asText(editing.aspect)} onChange={(value) => patch('aspect', value)} />
                      <Field label="平台" value={asText(editing.platform)} onChange={(value) => patch('platform', value)} />
                      <Field label="质量门禁（逗号分隔）" value={asArray(editing.quality_gates).join(', ')} onChange={(value) => patch('quality_gates', value.split(',').map((item) => item.trim()).filter(Boolean))} />
                    </div>
                    <AudioPanel
                      workspace={workspace}
                      creativeId={detail.id}
                      projectId={asText(detail.links?.project_id)}
                      durationSeconds={Number(draftProduction.duration_seconds) || 72}
                      onNotice={setNotice}
                      onError={setError}
                    />
                    <Card title="Production Data" icon={Sparkles}>
                      <div className="text-[11px] leading-5" style={{color: 'var(--fg-muted)'}}>
                        <div>title · {asText((editing.data as any)?.title)}</div>
                        <div>scenes · {asArray((editing.data as any)?.scenes).length} 个</div>
                        <div>source · {asText((editing.data as any)?.source)}</div>
                      </div>
                      <div className="mt-2 text-[10px]" style={{color: 'var(--fg-dim)'}}>具体场景数据由 AI 写入 production.data；当前先展示摘要，避免在 UI 里误改大段 JSON。</div>
                    </Card>
                  </div>
                ) : null}
              </>
            )}
            </div>
            </div>
            {dirty && detail ? <div className="savebar"><span className="text-[11px]" style={{color: 'var(--fg-muted)'}}>未保存 · 保存后生成 r{detail.revision + 1}</span><button type="button" className="btn btn-primary ml-auto" onClick={() => void saveSection()} disabled={busy}><Save size={12} />保存本节</button></div> : null}
          </main>

          {inspectorOpen ? <aside className="studio-inspector" style={{gridColumn: '3'}}>
            <div className="border-b p-3" style={{borderColor: 'var(--line)'}}>
              <div className="flex items-center gap-2"><ShieldCheck size={13} style={{color: 'var(--accent)'}} /><span className="text-[11px] font-semibold">逻辑检查</span></div>
              <div className="mt-1 text-[10px] leading-4" style={{color: 'var(--fg-dim)'}}>依赖、断点、来源与溯源。</div>
            </div>
            <div className="inspector-tabs">
              <button type="button" className="inspector-tab" data-active={inspectorTab === 'logic'} onClick={() => setInspectorTab('logic')}>逻辑</button>
              <button type="button" className="inspector-tab" data-active={inspectorTab === 'sources'} onClick={() => setInspectorTab('sources')}>来源</button>
              <button type="button" className="inspector-tab" data-active={inspectorTab === 'versions'} onClick={() => setInspectorTab('versions')}>版本</button>
              <button type="button" className="inspector-tab" data-active={inspectorTab === 'plan'} onClick={() => { setInspectorTab('plan'); void loadTimelinePlan(); }}>计划</button>
            </div>
            <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-3">
              {inspectorTab === 'logic' ? (
                <>
                  <Card title="脉络摘要" icon={GitBranch}>
                    <div className="space-y-1.5 text-[10.5px]" style={{color: 'var(--fg-muted)'}}>
                      <div className="kv"><dt>证据 → 脚本</dt><dd>{draftClaims.length} claims · {draftSources.length} sources → {draftBeats.length} beats</dd></div>
                      <div className="kv"><dt>脚本 → 分镜</dt><dd>{draftBeats.length} beats · {beatDuration}s → {draftShots.length} shots</dd></div>
                      <div className="kv"><dt>分镜 → 场景</dt><dd>{draftShots.length} shots → {sceneCount} scenes</dd></div>
                      <div className="kv"><dt>制作</dt><dd className="truncate">{asText(draftProduction.style_pack) || '未选风格'}@{asText(draftProduction.style_version) || '—'}</dd></div>
                    </div>
                  </Card>
                  <Card title="镜头能力" icon={Clapperboard}>
                    <div className="space-y-1.5 text-[10.5px]" style={{color: 'var(--fg-muted)'}}>
                      <div className="kv"><dt>程序化</dt><dd>{draftShots.filter((shot: any) => !asText(shot.provider) || asText(shot.provider).startsWith('programmatic.')).length}</dd></div>
                      <div className="kv"><dt>AI 图片</dt><dd>{draftShots.filter((shot: any) => asText(shot.provider) === 'ai.image').length}</dd></div>
                      <div className="kv"><dt>AI 视频</dt><dd>{draftShots.filter((shot: any) => asText(shot.provider) === 'ai.video').length}</dd></div>
                      <div className="kv"><dt>缺 prompt</dt><dd>{draftShots.filter((shot: any) => (asText(shot.provider) === 'ai.image' || asText(shot.provider) === 'ai.video') && !asText(shot.prompt)).length}</dd></div>
                      <div className="kv"><dt>图片来源</dt><dd>media.image.generate</dd></div>
                      <div className="kv"><dt>视频来源</dt><dd>media.video.generate</dd></div>
                    </div>
                  </Card>
                  <Card title="风险与断点" icon={AlertTriangle}>
                    <div className="space-y-1.5">
                      {logicRisks.length ? logicRisks.slice(0, 8).map((risk, index) => (
                        <div key={`${risk.code}-${index}`} className="flex gap-2 text-[10.5px] leading-4">
                          <span style={{color: risk.level === 'error' ? 'var(--err)' : 'var(--warn)'}}>●</span>
                          <span style={{color: 'var(--fg-muted)'}}>{risk.text}</span>
                        </div>
                      )) : <div className="text-[10.5px]" style={{color: 'var(--ok)'}}>没有发现结构性断点。</div>}
                    </div>
                  </Card>
                </>
              ) : null}

              {inspectorTab === 'sources' ? (
                <>
                  <Card title={`来源 · ${draftSources.length}`} icon={Link2}>
                    <div className="space-y-2">
                      {draftSources.map((source: any, index: number) => (
                        <div key={index} className="rounded-md border p-2 text-[10.5px] leading-4" style={{borderColor: 'var(--line)'}}>
                          <div style={{color: 'var(--fg-muted)'}}>{asText(source.publisher)} · {asText(source.title)}</div>
                          {source.url ? <a href={asText(source.url)} target="_blank" rel="noreferrer" className="break-all" style={{color: 'var(--accent)'}}>{asText(source.url)}</a> : <div style={{color: 'var(--warn)'}}>缺少 URL</div>}
                        </div>
                      ))}
                      {!draftSources.length ? <div className="text-[10.5px]" style={{color: 'var(--fg-dim)'}}>还没有来源。发布前必须补齐。</div> : null}
                    </div>
                  </Card>
                  <Card title={`证据覆盖 · ${draftClaims.length - claimsWithoutEvidence.length}/${draftClaims.length}`} icon={FlaskConical}>
                    <div className="space-y-1.5">
                      {claimsWithoutEvidence.length ? claimsWithoutEvidence.map((claim: any, index: number) => <div key={index} className="text-[10.5px] leading-4" style={{color: 'var(--warn)'}}>· {asText(claim.text) || asText(claim.id)}</div>) : <div className="text-[10.5px]" style={{color: 'var(--ok)'}}>所有 claim 都绑定了证据 refs。</div>}
                    </div>
                  </Card>
                </>
              ) : null}

              {inspectorTab === 'versions' ? (
                <>
                  <Card title="校验与批准" icon={ShieldCheck}>
                    <div className="space-y-2">
                      <div className="text-[11px]" style={{color: validation?.valid ? 'var(--ok)' : validation ? 'var(--err)' : 'var(--fg-dim)'}}>{validation ? (validation.valid ? '校验通过' : `${validation.issues?.length ?? 0} 个问题`) : '尚未校验'}</div>
                      {validation?.issues?.length ? <div className="space-y-1">{validation.issues.slice(0, 6).map((issue, index) => <div key={index} className="text-[10px] leading-4" style={{color: 'var(--warn)'}}>{issue.code} · {issue.message}</div>)}</div> : null}
                    </div>
                  </Card>
                  <Card title="溯源" icon={History}>
                    <dl className="space-y-1 text-[10.5px]">
                      <div className="kv"><dt>当前 revision</dt><dd>r{detail?.revision ?? '—'}</dd></div>
                      <div className="kv"><dt>批准 revision</dt><dd>{detail?.approval?.approved_revision ? `r${detail.approval.approved_revision}` : '—'}</dd></div>
                      <div className="kv"><dt>项目</dt><dd className="truncate">{detail?.links?.project_id || '—'}</dd></div>
                    </dl>
                    <div className="mt-2 text-[9.5px] leading-4" style={{color: 'var(--fg-dim)'}}>修改任一段会生成新 revision 并清除批准；已创建项目不会自动跟随新 revision。</div>
                  </Card>
                  <Card title="项目状态" icon={Clapperboard}>
                    {detail?.links?.project_id ? (
                      <dl className="space-y-1 text-[10.5px]">
                        <div className="kv"><dt>项目</dt><dd className="truncate">{detail.links.project_id}</dd></div>
                        <div className="kv"><dt>最新成片</dt><dd>{latestVideo ? `${latestVideo.quality_status || 'unknown'} · ${Math.round((latestVideo.size || 0) / 1024)} KB` : '尚未渲染'}</dd></div>
                        <div className="kv"><dt>产物</dt><dd>{videoArtifacts.length} 个视频 / {projectArtifacts.length} 个产物</dd></div>
                      </dl>
                    ) : <div className="text-[10.5px]" style={{color: 'var(--fg-dim)'}}>创意还没有创建项目。</div>}
                  </Card>
                </>
              ) : null}

              {inspectorTab === 'plan' ? (
                <Card title="镜头制作计划" icon={Clapperboard} actions={<button type="button" className="btn" onClick={() => void loadTimelinePlan()} disabled={planBusy}>{planBusy ? <Loader2 size={11} className="animate-spin" /> : <RefreshCw size={11} />}刷新</button>}>
                  {timelinePlan ? (
                    <div className="space-y-2">
                      <div className="kv"><dt>状态</dt><dd style={{color: timelinePlan.state === 'ready' ? 'var(--ok)' : 'var(--warn)'}}>{timelinePlan.state}</dd></div>
                      <div className="kv"><dt>镜头</dt><dd>{timelinePlan.shots.length}</dd></div>
                      <div className="kv"><dt>总时长</dt><dd>{timelinePlan.duration_seconds}s / {timelinePlan.target_duration_seconds || '—'}s</dd></div>
                      <div className="space-y-1.5">
                        {timelinePlan.shots.map((shot) => (
                          <div key={shot.id} className="rounded-md border p-2" style={{borderColor: 'var(--line)'}}>
                            <div className="flex flex-wrap items-center gap-1">
                              <span className="badge">{shot.id}</span>
                              <span className="badge badge-accent">{shot.provider_name || shot.provider}</span>
                              <span className="badge">{shot.status || 'planned'}</span>
                              <span className="ml-auto text-[9.5px]" style={{color: 'var(--fg-dim)'}}>{shot.duration_seconds}s</span>
                            </div>
                            <div className="mt-1 text-[10px]" style={{color: 'var(--fg-dim)'}}>{shot.technique || '默认技法'}</div>
                            {shot.prompt ? <div className="mt-1 text-[10px]" style={{color: 'var(--fg-muted)'}}>{shot.prompt}</div> : null}
                          </div>
                        ))}
                      </div>
                      {timelinePlan.blockers.length ? <div className="space-y-1">{timelinePlan.blockers.map((blocker, index) => <div key={index} className="text-[10px]" style={{color: 'var(--err)'}}>{blocker.code} · {blocker.message}</div>)}</div> : null}
                      {timelinePlan.warnings.length ? <div className="space-y-1">{timelinePlan.warnings.map((warning, index) => <div key={index} className="text-[10px]" style={{color: 'var(--warn)'}}>{warning.code} · {warning.message}</div>)}</div> : null}
                    </div>
                  ) : <div className="text-[10.5px]" style={{color: 'var(--fg-dim)'}}>点击刷新编译镜头制作计划。</div>}
                </Card>
              ) : null}
            </div>
            <div className="border-t p-3" style={{borderColor: 'var(--line)'}}>
              <div className="flex gap-1">
                <button type="button" className="btn flex-1" onClick={() => void validate()} disabled={busy || !detail}><ShieldCheck size={11} />校验</button>
                <button type="button" className="btn flex-1" onClick={() => void approve()} disabled={busy || !detail || detail.status === 'materialized'}><CheckCircle2 size={11} />批准</button>
                <button type="button" className="btn btn-primary flex-1" onClick={() => void materialize()} disabled={busy || detail?.status !== 'approved'}><ArrowRight size={11} />创建</button>
              </div>
            </div>
          </aside> : null}
        </div>

        {activeShot && editingShotIndex !== null ? (
          <div className="fixed inset-0 z-[60] flex items-center justify-center p-4" style={{background: 'rgba(0,0,0,.62)'}}>
            <div className="flex max-h-full w-full max-w-[760px] flex-col overflow-hidden rounded-xl" style={{background: 'var(--ink-900)', border: '1px solid var(--line)'}}>
              <header className="flex items-center gap-2 border-b px-4 py-3" style={{borderColor: 'var(--line)'}}>
                <Clapperboard size={14} style={{color: 'var(--accent)'}} />
                <span className="text-[13px] font-semibold">镜头 {editingShotIndex + 1} / {totalShots}</span>
                <span className="badge">{asText(activeShot.id) || '未命名'}</span>
                <div className="ml-auto flex items-center gap-1">
                  <button type="button" className="btn" onClick={() => setEditingShotIndex(Math.max(0, editingShotIndex - 1))} disabled={editingShotIndex <= 0}><ChevronUp size={11} />上一个</button>
                  <button type="button" className="btn" onClick={() => setEditingShotIndex(Math.min(totalShots - 1, editingShotIndex + 1))} disabled={editingShotIndex >= totalShots - 1}>下一个<ChevronDown size={11} /></button>
                  <button type="button" className="icon-btn" aria-label="关闭镜头编辑" onClick={() => setEditingShotIndex(null)}><X size={14} /></button>
                </div>
              </header>
              <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-4">
                <div className="grid gap-3 md:grid-cols-[1fr_110px_100px]">
                  <Field label="镜头 ID" value={asText(activeShot.id)} onChange={(value) => patchArray('shots', editingShotIndex, 'id', value)} />
                  <Field label="镜头类型" value={asText(activeShot.shot)} onChange={(value) => patchArray('shots', editingShotIndex, 'shot', value)} />
                  <Field label="时长（秒）" value={asText(activeShot.duration_seconds)} onChange={(value) => patchArray('shots', editingShotIndex, 'duration_seconds', Number(value) || 0)} />
                </div>
                <div className="grid gap-3 md:grid-cols-2">
                  <Field label="对应 beat" value={asText(activeShot.beat_id)} onChange={(value) => patchArray('shots', editingShotIndex, 'beat_id', value)} />
                  <Field label="状态" value={asText(activeShot.status) || 'planned'} onChange={(value) => patchArray('shots', editingShotIndex, 'status', value)} />
                </div>
                <div className="grid gap-3 md:grid-cols-2">
                  <label className="block">
                    <span className="field-label">生成方式</span>
                    <select className="field-input" value={asText(activeShot.provider)} onChange={(event) => patchArray('shots', editingShotIndex, 'provider', event.target.value)}>
                      <option value="">默认（程序化）</option>
                      {providers.map((provider) => <option key={provider.id} value={provider.id}>{providerLabel(provider)}</option>)}
                    </select>
                  </label>
                  <Field label="技法" value={asText(activeShot.technique)} onChange={(value) => patchArray('shots', editingShotIndex, 'technique', value)} placeholder="hand-drawn / cinematic / whiteboard" />
                </div>
                {activeShotProvider && activeShotProvider.source === 'external' && !activeShotProvider.available ? <div className="text-[10px]" style={{color: 'var(--warn)'}}>需要安装 {activeShotProvider.requires_pack}</div> : null}
                <div className="grid gap-3 md:grid-cols-2">
                  <Field label="生成提示词" value={asText(activeShot.prompt)} onChange={(value) => patchArray('shots', editingShotIndex, 'prompt', value)} textarea rows={4} placeholder="AI 图片/视频镜头使用" />
                  <Field label="负面提示词" value={asText(activeShot.negative_prompt)} onChange={(value) => patchArray('shots', editingShotIndex, 'negative_prompt', value)} textarea rows={4} />
                </div>
                <div className="grid gap-3 md:grid-cols-2">
                  <Field label="回退生成方式" value={asText(activeShot.fallback_provider)} onChange={(value) => patchArray('shots', editingShotIndex, 'fallback_provider', value)} placeholder="programmatic.browser-code" />
                  <Field label="参考资产（逗号分隔）" value={asArray(activeShot.reference_assets).join(', ')} onChange={(value) => patchArray('shots', editingShotIndex, 'reference_assets', value.split(',').map((item) => item.trim()).filter(Boolean))} />
                </div>
                <div className="grid gap-3 md:grid-cols-2">
                  <Field label="画面" value={asText(activeShot.visual)} onChange={(value) => patchArray('shots', editingShotIndex, 'visual', value)} textarea rows={3} />
                  <Field label="屏幕文字" value={asText(activeShot.text)} onChange={(value) => patchArray('shots', editingShotIndex, 'text', value)} textarea rows={3} />
                </div>
                {shotAssets[asText(activeShot.id)]?.preview_data_url ? (
                  <div className="rounded-md border p-2" style={{borderColor: 'var(--line)'}}>
                    <div className="field-label">最近生成的模拟资产</div>
                    <img src={shotAssets[asText(activeShot.id)].preview_data_url} alt="mock shot" className="mt-2 max-h-[320px] w-full rounded object-contain" />
                    <div className="mt-1 break-all text-[9.5px]" style={{color: 'var(--fg-dim)'}}>{shotAssets[asText(activeShot.id)].path}</div>
                  </div>
                ) : null}
              </div>
              <footer className="flex items-center gap-2 border-t px-4 py-3" style={{borderColor: 'var(--line)'}}>
                <span className="text-[10px]" style={{color: 'var(--fg-dim)'}}>修改实时进入草稿；关闭后点“保存本节”生成新 revision。</span>
                {asText(activeShot.provider) === 'ai.image' ? <button type="button" className="btn btn-primary ml-auto" onClick={() => void produceShots([editingShotIndex])} disabled={busy || dirty} title={dirty ? '先保存本节' : '生成模拟图片'}>{busy ? <Loader2 size={11} className="animate-spin" /> : <Wand2 size={11} />}生成模拟图片</button> : <span className="ml-auto" />}
                <button type="button" className="btn" onClick={() => { if (window.confirm('删除这个镜头？')) { removeArrayItem('shots', editingShotIndex); setEditingShotIndex(null); } }}><Trash2 size={11} />删除</button>
              </footer>
            </div>
          </div>
        ) : null}
    </div>
  );
}
