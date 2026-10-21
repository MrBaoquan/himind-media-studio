import {useCallback, useEffect, useState} from 'react';
import {BookOpen, ChevronLeft, ChevronRight, ClipboardList, Film, FolderOpen, LayoutGrid, ListVideo, Plus, RefreshCw, Scissors, Sparkles, GitBranch} from 'lucide-react';
import {callCapability, hostInvoke, hosted} from './api/capability';
import type {CatalogOption, ConstraintPayload, ProjectDetail, ProjectList, ProjectSummary} from './types';
import {ProjectTree} from './features/project-tree';
import {PreviewPane} from './features/storyboard';
import {CodeStoryboardPane, ReviewPane} from './features/review';
import {StyleLibrary} from './features/style-library';
import {CreativeWorkspace} from './features/creative';
import {StoryboardEditor} from './features/storyboard-editor';
import {PlanPane} from './features/plan';
import {DurationRuler} from './features/timeline';
import {Inspector} from './features/inspector';
import {TimelineEditor} from './features/timeline-editor';
import {StatusBar} from './features/status-bar';
import {CreateDialog} from './features/create-dialog';
import {Empty} from './components/ui';
import {ThemeToggle} from './theme';

const WORKSPACE_KEY = 'himind.workbench.workspace';

/**
 * 解析本次操作的工作区。
 *
 * 顺序：宿主绑定的工作区 → 上次记住的目录。
 * 视频项目允许放在任意目录，所以必须由调用方显式传入 workspace_root；
 * 内核会拒绝缺少该参数的 project.list（真机联调时踩到过）。
 */
async function resolveWorkspace(): Promise<string> {
  if (hosted) {
    try {
      const ctx = await hostInvoke<{workspace_root?: string | null}>('get_plugin_view_context');
      const root = (ctx?.workspace_root ?? '').trim();
      if (root) return root;
    } catch {
      // 宿主未绑定工作区时继续走本地记忆
    }
  }
  try {
    return (localStorage.getItem(WORKSPACE_KEY) ?? '').trim();
  } catch {
    return '';
  }
}

function rememberWorkspace(root: string) {
  try {
    if (root) localStorage.setItem(WORKSPACE_KEY, root);
    else localStorage.removeItem(WORKSPACE_KEY);
  } catch {
    // 忽略存储不可用
  }
}

export function App() {
  const [projects, setProjects] = useState<ProjectSummary[]>([]);
  const [workspace, setWorkspace] = useState('');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<ProjectDetail | null>(null);
  const [constraints, setConstraints] = useState<ConstraintPayload | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [guideOpen, setGuideOpen] = useState(false);
  const [catalog, setCatalog] = useState<{templates: CatalogOption[]; styles: CatalogOption[]}>({templates: [], styles: []});
  const [leftOpen, setLeftOpen] = useState(true);
  const [rightOpen, setRightOpen] = useState(true);
  const [previewSignal, setPreviewSignal] = useState(0);
  const [mainView, setMainView] = useState<'preview' | 'plan' | 'storyboard' | 'edit'>('preview');
  const [styleLibraryOpen, setStyleLibraryOpen] = useState(false);
  const [creativeOpen, setCreativeOpen] = useState(false);

  useEffect(() => {
    const query = window.matchMedia('(max-width: 1100px)');
    const apply = () => {
      if (query.matches) {
        setLeftOpen(false);
        setRightOpen(false);
      }
    };
    apply();
    query.addEventListener('change', apply);
    return () => query.removeEventListener('change', apply);
  }, []);

  const loadProjects = useCallback(async (root: string) => {
    if (!root) {
      setProjects([]);
      setSelectedId(null);
      setDetail(null);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const list = await callCapability<ProjectList>('media.workbench.project.list', {workspace_root: root});
      setProjects(list.items ?? []);
      setSelectedId((current) => (current && list.items?.some((p) => p.id === current) ? current : list.items?.[0]?.id ?? null));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  const loadCatalog = useCallback(async () => {
    try {
      const data = await callCapability<{templates?: CatalogOption[]; styles?: CatalogOption[]}>('media.workbench.catalog.list', {});
      setCatalog({templates: data.templates ?? [], styles: data.styles ?? []});
    } catch {
      setCatalog({templates: [], styles: []});
    }
  }, []);

  const loadConstraints = useCallback(async () => {
    try {
      setConstraints(await callCapability<ConstraintPayload>('media.workbench.constraint.list', {}));
    } catch {
      setConstraints(null);
    }
  }, []);

  const refreshDetail = useCallback(async () => {
    if (!selectedId || !workspace) return;
    const data = await callCapability<ProjectDetail>('media.workbench.project.get', {
      workspace_root: workspace,
      project_id: selectedId,
    });
    setDetail(data);
  }, [selectedId, workspace]);

  useEffect(() => {
    setMainView('preview');
  }, [selectedId]);

  useEffect(() => {
    void (async () => {
      const root = await resolveWorkspace();
      setWorkspace(root);
      await loadProjects(root);
    })();
    void loadConstraints();
    void loadCatalog();
  }, [loadProjects, loadConstraints, loadCatalog]);

  useEffect(() => {
    if (!selectedId) {
      setDetail(null);
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const data = await callCapability<ProjectDetail>('media.workbench.project.get', {
          workspace_root: workspace,
          project_id: selectedId,
        });
        if (!cancelled) setDetail(data);
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [selectedId, workspace]);

  async function chooseWorkspace() {
    setNotice(null);
    try {
      const picked = await hostInvoke<string | null>('pick_workspace_directory');
      const root = typeof picked === 'string' ? picked.trim() : '';
      if (!root) return;
      setWorkspace(root);
      rememberWorkspace(root);
      setSelectedId(null);
      await loadProjects(root);
    } catch (err) {
      setNotice(`选择工作区失败：${err instanceof Error ? err.message : String(err)}`);
    }
  }


  const recipe = detail?.project.recipe;
  const isCode = detail?.project.mode === 'code';
  const isStoryboard = recipe?.template_id === 'storyboard';
  const hasPlan = Boolean(detail?.plan);
  const storyboardShots = catalog.templates.filter((template) => template.storyboard_shot);
  const isEditing = mainView === 'edit';

  if (creativeOpen) {
    return (
      <CreativeWorkspace
        workspace={workspace}        initialCreativeId={detail?.project.source_creative_id}

        onClose={() => setCreativeOpen(false)}
        onMaterialized={async (projectId) => {
          await loadProjects(workspace);
          setSelectedId(projectId);
          setMainView('preview');
          setNotice('创作需求包已创建为视频项目');
        }}
      />
    );
  }

  return (
    <div className="grid h-full" style={{gridTemplateRows: '52px 1fr 30px'}}>
      <header
        className="workbench-global-header flex items-center gap-3 px-3"
        style={{borderBottom: '1px solid var(--line)', background: 'var(--ink-900)'}}
      >
        <div className="flex items-center gap-2">
          <span
            className="flex h-6 w-6 items-center justify-center rounded-md"
            style={{background: 'linear-gradient(135deg,#5aa9ff,#3ddc97)'}}
          >
            <Film size={13} color="#06101f" />
          </span>
          <span className="text-[13px] font-semibold tracking-tight">自媒体创作台</span>
        </div>

        <button type="button" className="icon-btn" style={isEditing ? {display:'none'} : undefined} onClick={() => setLeftOpen((v) => !v)} title={leftOpen ? '收起项目栏' : '展开项目栏'}>
          {leftOpen ? <ChevronLeft size={14} /> : <ChevronRight size={14} />}
        </button>

        <button
          type="button"
          className="workspace-path-button max-w-[240px] truncate rounded-md px-2 py-1 text-left font-mono text-[10.5px]"
          style={{background: 'var(--ink-850)', color: workspace ? 'var(--fg-dim)' : 'var(--warn)', border: '1px solid var(--line)'}}
          title={workspace || '点击选择工作区'}
          onClick={() => void chooseWorkspace()}
        >
          {workspace || '未绑定工作区 · 点击选择'}
        </button>

        {detail?.project.source_creative_id ? (
          <button
            type="button"
            className="badge badge-accent header-action-creative-chip"
            title="打开来源创意包"
            style={{cursor: 'pointer'}}
            onClick={() => setCreativeOpen(true)}
          >
            <GitBranch size={11} />
            创意 r{detail.project.source_creative_revision ?? '?'}
          </button>
        ) : null}        <div className="ml-auto flex items-center gap-2">
          <button type="button" className="btn btn-ghost header-action-secondary" style={isEditing ? {display:'none'} : undefined} onClick={() => void loadProjects(workspace)} title="刷新项目列表">
            <RefreshCw size={12} />
            刷新
          </button>
          <button
            type="button"
            className="btn"
            onClick={() => (workspace ? setDialogOpen(true) : void chooseWorkspace())}
          >
            {workspace ? <Plus size={12} /> : <FolderOpen size={12} />}
            {workspace ? '新建项目' : '选择工作区'}
          </button>
          <button type="button" className="btn header-action-secondary" style={isEditing ? {display:'none'} : undefined} onClick={() => setStyleLibraryOpen(true)} disabled={!workspace}>
            <LayoutGrid size={12} />
            风格库
          </button>
          <button type="button" className="btn header-action-creative" style={isEditing ? {display:'none'} : {background:'var(--accent-soft)',borderColor:'rgba(90,169,255,.28)',color:'var(--accent)'}} onClick={() => setCreativeOpen(true)} disabled={!workspace}>
            <GitBranch size={12} />
            创作脉络
          </button>
          <button type="button" className="btn" style={isEditing ? {display:'none'} : undefined} onClick={() => { setMainView('preview'); setPreviewSignal((value) => value + 1); }} disabled={!detail}>
            <Sparkles size={12} />
            {isCode ? '生成审查图' : '生成动效预览'}
          </button>
          <button type="button" className="btn btn-ghost header-action-secondary" style={isEditing ? {display:'none'} : undefined} onClick={() => setGuideOpen(true)}><BookOpen size={12} />引导</button>
          <ThemeToggle />
          <button type="button" className="icon-btn" style={isEditing ? {display:'none'} : undefined} onClick={() => setRightOpen((v) => !v)} title={rightOpen ? '收起属性栏' : '展开属性栏'}>
            {rightOpen ? <ChevronRight size={14} /> : <ChevronLeft size={14} />}
          </button>
        </div>
      </header>

      <div className="workbench-main-grid grid min-h-0 min-w-0" style={{gridTemplateColumns: `${leftOpen && !isEditing ? 232 : 0}px minmax(0, 1fr) ${rightOpen && !isEditing ? 296 : 0}px`}}>
        <aside data-open={leftOpen} className="min-h-0 overflow-hidden" style={{borderRight: leftOpen ? '1px solid var(--line)' : 'none', background: 'var(--ink-900)'}}>
          {leftOpen && !isEditing ? (
            <ProjectTree
              projects={projects}
              selectedId={selectedId}
              onSelect={setSelectedId}
              query={query}
              onQuery={setQuery}
              loading={loading}
            />
          ) : null}
        </aside>

        <main className="grid min-h-0 min-w-0" style={{gridTemplateRows: isEditing ? '40px minmax(0, 1fr)' : (isCode || isStoryboard || hasPlan ? '40px 1fr 112px' : '1fr 112px')}}>
          {isCode || isStoryboard || hasPlan ? (
            <div className="workbench-tabbar flex items-center gap-1 px-3" style={{borderBottom: '1px solid var(--line)', background: 'var(--ink-900)'}}>
              <button
                type="button"
                className="btn"
                onClick={() => setMainView('preview')}
                style={mainView === 'preview' ? {background: 'var(--accent-soft)', borderColor: 'rgba(90,169,255,.28)', color: 'var(--fg)'} : undefined}
              >
                <Film size={12} />
                {isCode ? '审阅' : '预览'}
              </button>
              {!isCode ? (
                <button
                  type="button"
                  className="btn"
                  onClick={() => setMainView('plan')}
                  style={mainView === 'plan' ? {background: 'var(--accent-soft)', borderColor: 'rgba(90,169,255,.28)', color: 'var(--fg)'} : undefined}
                >
                  <ClipboardList size={12} />
                  计划
                </button>
              ) : null}
              {isCode || isStoryboard ? (
                <button
                  type="button"
                  className="btn"
                  onClick={() => setMainView('storyboard')}
                  style={mainView === 'storyboard' ? {background: 'var(--accent-soft)', borderColor: 'rgba(90,169,255,.28)', color: 'var(--fg)'} : undefined}
                >
                  <ListVideo size={12} />
                  分镜
                </button>
              ) : null}              <button
                type="button"
                className="btn"
                onClick={() => setMainView('edit')}
                style={mainView === 'edit' ? {background: 'var(--accent-soft)', borderColor: 'rgba(90,169,255,.28)', color: 'var(--fg)'} : undefined}
              >
                <Scissors size={12} />
                剪辑
              </button>
              <span className="ml-auto" style={{color: 'var(--fg-dim)'}}>
                {isCode
                  ? `${detail?.project.style_pack ?? 'code'} · r${detail?.project.revision ?? 1}`
                  : mainView === 'plan'
                    ? `${detail?.plan?.storyboard?.shots?.length ?? 0} 镜头 · ${detail?.plan?.narrative?.structure ?? 'single-shot'}`
                    : mainView === 'storyboard'
                      ? '镜头顺序、时长与数据'
                      : `${recipe?.duration_seconds ?? 0}s`}
              </span>
            </div>
          ) : null}

          <div className="min-h-0 min-w-0 overflow-hidden">
            {error ? (
              <div className="flex items-center justify-center px-6 text-center text-[12px]" style={{color: 'var(--err)'}}>
                {error}
              </div>
            ) : (
              <>
                {isCode ? (
                  <div className="h-full" style={{display: mainView === 'preview' ? 'block' : 'none'}}>
                    <ReviewPane
                      project={detail?.project ?? null}
                      workspace={workspace}
                      artifacts={detail?.artifacts ?? []}
                      feedback={detail?.feedback ?? []}
                      generateSignal={previewSignal}
                      onRefresh={async () => {
                        await refreshDetail();
                        await loadProjects(workspace);
                      }}
                    />
                  </div>
                ) : recipe ? (
                  <div className="h-full" style={{display: mainView === 'preview' ? 'block' : 'none'}}>
                    <PreviewPane recipe={recipe} workspace={workspace} projectId={detail?.project.id ?? null} generateSignal={previewSignal} />
                  </div>
                ) : workspace ? (
                  <Empty title="未选择项目" hint="从左侧选择一个项目，或让 AI 新建一个" />
                ) : (
                  <div className="flex h-full flex-col items-center justify-center gap-2">
                    <div className="text-[12.5px]" style={{color: 'var(--fg-muted)'}}>还没有绑定工作区</div>
                    <div className="text-[11.5px]" style={{color: 'var(--fg-dim)'}}>视频项目可以放在任意目录</div>
                    <button type="button" className="btn btn-primary mt-1" onClick={() => void chooseWorkspace()}>
                      <FolderOpen size={12} />
                      选择工作区
                    </button>
                  </div>
                )}

                {detail?.plan ? (
                  <div className="h-full" style={{display: mainView === 'plan' ? 'block' : 'none'}}>
                    <PlanPane plan={detail.plan} />
                  </div>
                ) : null}

                {isCode && detail ? (
                  <div className="h-full" style={{display: mainView === 'storyboard' ? 'block' : 'none'}}>
                    <CodeStoryboardPane storyboard={detail.storyboard} />
                  </div>
                ) : null}

                {detail ? (
                  <div className="h-full min-w-0" style={{display: mainView === 'edit' ? 'block' : 'none'}}>
                    <TimelineEditor workspace={workspace} projectId={detail.project.id} artifacts={detail.artifacts ?? []} onSaved={refreshDetail} active={mainView === 'edit'} />
                  </div>
                ) : null}
                {isStoryboard && detail ? (
                  <div className="h-full" style={{display: mainView === 'storyboard' ? 'block' : 'none'}}>
                    <StoryboardEditor
                      key={detail.project.id}
                      detail={detail}
                      workspace={workspace}
                      shots={storyboardShots}
                      onSaved={async () => {
                        await refreshDetail();
                        await loadProjects(workspace);
                      }}
                      onPreview={() => {
                        setMainView('preview');
                        setPreviewSignal((value) => value + 1);
                      }}
                    />
                  </div>
                ) : null}
              </>
            )}
          </div>

          <div style={{display: isEditing ? 'none' : 'block', borderTop: '1px solid var(--line)', background: 'var(--ink-900)'}}>
            <DurationRuler durationSeconds={recipe?.duration_seconds ?? detail?.project.duration_seconds ?? 0} />
          </div>
        </main>

        <aside data-open={rightOpen} className="min-h-0 overflow-hidden" style={{borderLeft: rightOpen ? '1px solid var(--line)' : 'none', background: 'var(--ink-900)'}}>
          {rightOpen && !isEditing ? <Inspector detail={detail} constraints={constraints} workspace={workspace} onRefresh={refreshDetail} /> : null}
        </aside>
      </div>

      <StatusBar detail={detail} constraints={constraints} hosted={hosted} />

      {styleLibraryOpen ? (
        <StyleLibrary
          workspace={workspace}
          onClose={() => setStyleLibraryOpen(false)}
          onCreated={async (id) => {
            await loadProjects(workspace);
            setSelectedId(id);
            setMainView('preview');
          }}
        />
      ) : null}


      {guideOpen ? (
        <div className="fixed inset-0 z-20 flex items-center justify-center" style={{background: 'rgba(0,0,0,.45)'}} onClick={() => setGuideOpen(false)}>
          <div className="w-[520px] rounded-xl border p-5" style={{background: 'var(--ink-900)', borderColor: 'var(--line-strong)', boxShadow: 'var(--shadow)'}} onClick={(event) => event.stopPropagation()}>
            <div className="flex items-center gap-2">
              <BookOpen size={16} style={{color: 'var(--accent)'}} />
              <strong className="text-[15px]">5 分钟做出第一版</strong>
              <button type="button" className="icon-btn ml-auto" onClick={() => setGuideOpen(false)}>×</button>
            </div>
            <div className="mt-4 space-y-2">
              <div className="flex gap-2 rounded-md border px-3 py-2" style={{borderColor: 'var(--line)'}}><span className="badge badge-accent">1</span><div><div className="text-[12px] font-medium">绑定工作区</div><div className="text-[10px] leading-4" style={{color: 'var(--fg-dim)'}}>视频项目可以放在任意目录，不要求属于扩展仓库。</div></div></div>
              <div className="flex gap-2 rounded-md border px-3 py-2" style={{borderColor: 'var(--line)'}}><span className="badge badge-accent">2</span><div><div className="text-[12px] font-medium">描述主题或选择模板</div><div className="text-[10px] leading-4" style={{color: 'var(--fg-dim)'}}>AI 会从 Brief 推进到脚本、分镜与制作规格。</div></div></div>
              <div className="flex gap-2 rounded-md border px-3 py-2" style={{borderColor: 'var(--line)'}}><span className="badge badge-accent">3</span><div><div className="text-[12px] font-medium">看审查图并按反馈迭代</div><div className="text-[10px] leading-4" style={{color: 'var(--fg-dim)'}}>体检通过后再渲染成片；失败时任务中心会给出原因。</div></div></div>
            </div>
            <div className="mt-4 flex gap-2">
              <button type="button" className="btn btn-primary" onClick={() => { setGuideOpen(false); void chooseWorkspace(); }}>绑定工作区</button>
              <button type="button" className="btn" onClick={() => { setGuideOpen(false); void hostInvoke('show_main_window').catch(() => undefined); }}>让 AI 描述主题</button>
              <button type="button" className="btn" onClick={() => { setGuideOpen(false); setDialogOpen(true); }} disabled={!workspace}>从模板开始</button>
            </div>
          </div>
        </div>
      ) : null}      <CreateDialog
        open={dialogOpen}
        workspace={workspace}
        templates={catalog.templates}
        styles={catalog.styles}
        onClose={() => setDialogOpen(false)}
        onCreated={(id) => {
          void loadProjects(workspace).then(() => {
            if (id) setSelectedId(id);
          });
        }}
      />

      {notice ? (
        <div
          className="fixed bottom-10 left-1/2 z-10 -translate-x-1/2 rounded-lg px-3 py-2 text-[11.5px] shadow-xl"
          style={{background: 'var(--ink-800)', border: '1px solid var(--line-strong)', color: 'var(--fg)'}}
          onClick={() => setNotice(null)}
          role="status"
        >
          {notice}
        </div>
      ) : null}
    </div>
  );
}
