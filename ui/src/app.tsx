import {useCallback, useEffect, useState} from 'react';
import {ChevronLeft, ChevronRight, ClipboardList, Film, FolderOpen, LayoutGrid, ListVideo, Plus, RefreshCw, Sparkles} from 'lucide-react';
import {callCapability, hostInvoke, hosted} from './api/capability';
import type {CatalogOption, ConstraintPayload, ProjectDetail, ProjectList, ProjectSummary} from './types';
import {ProjectTree} from './features/project-tree';
import {PreviewPane} from './features/storyboard';
import {CodeStoryboardPane, ReviewPane} from './features/review';
import {StyleLibrary} from './features/style-library';
import {StoryboardEditor} from './features/storyboard-editor';
import {PlanPane} from './features/plan';
import {DurationRuler} from './features/timeline';
import {Inspector} from './features/inspector';
import {StatusBar} from './features/status-bar';
import {CreateDialog} from './features/create-dialog';
import {Empty} from './components/ui';

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
  const [catalog, setCatalog] = useState<{templates: CatalogOption[]; styles: CatalogOption[]}>({templates: [], styles: []});
  const [leftOpen, setLeftOpen] = useState(true);
  const [rightOpen, setRightOpen] = useState(true);
  const [previewSignal, setPreviewSignal] = useState(0);
  const [mainView, setMainView] = useState<'preview' | 'plan' | 'storyboard'>('preview');
  const [styleLibraryOpen, setStyleLibraryOpen] = useState(false);

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

  return (
    <div className="grid h-full" style={{gridTemplateRows: '48px 1fr 30px'}}>
      <header
        className="flex items-center gap-3 px-3"
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

        <button type="button" className="icon-btn" onClick={() => setLeftOpen((v) => !v)} title={leftOpen ? '收起项目栏' : '展开项目栏'}>
          {leftOpen ? <ChevronLeft size={14} /> : <ChevronRight size={14} />}
        </button>

        <button
          type="button"
          className="max-w-[360px] truncate rounded-md px-2 py-1 text-left font-mono text-[10.5px]"
          style={{background: 'var(--ink-850)', color: workspace ? 'var(--fg-dim)' : 'var(--warn)', border: '1px solid var(--line)'}}
          title={workspace || '点击选择工作区'}
          onClick={() => void chooseWorkspace()}
        >
          {workspace || '未绑定工作区 · 点击选择'}
        </button>

        <div className="ml-auto flex items-center gap-2">
          <button type="button" className="btn btn-ghost" onClick={() => void loadProjects(workspace)} title="刷新项目列表">
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
          <button type="button" className="btn" onClick={() => setStyleLibraryOpen(true)} disabled={!workspace}>
            <LayoutGrid size={12} />
            风格库
          </button>
          <button type="button" className="btn btn-primary" onClick={() => { setMainView('preview'); setPreviewSignal((value) => value + 1); }} disabled={!detail}>
            <Sparkles size={12} />
            {isCode ? '生成审查图' : '生成动效预览'}
          </button>
          <button type="button" className="icon-btn" onClick={() => setRightOpen((v) => !v)} title={rightOpen ? '收起属性栏' : '展开属性栏'}>
            {rightOpen ? <ChevronRight size={14} /> : <ChevronLeft size={14} />}
          </button>
        </div>
      </header>

      <div className="grid min-h-0" style={{gridTemplateColumns: `${leftOpen ? 232 : 0}px 1fr ${rightOpen ? 296 : 0}px`}}>
        <aside className="min-h-0 overflow-hidden" style={{borderRight: leftOpen ? '1px solid var(--line)' : 'none', background: 'var(--ink-900)'}}>
          {leftOpen ? (
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

        <main className="grid min-h-0" style={{gridTemplateRows: isCode || isStoryboard || hasPlan ? '34px 1fr 112px' : '1fr 112px'}}>
          {isCode || isStoryboard || hasPlan ? (
            <div className="flex items-center gap-1 px-3" style={{borderBottom: '1px solid var(--line)', background: 'var(--ink-900)'}}>
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
              ) : null}
              <span className="ml-auto text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
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

          <div className="min-h-0">
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

          <div style={{borderTop: '1px solid var(--line)', background: 'var(--ink-900)'}}>
            <DurationRuler durationSeconds={recipe?.duration_seconds ?? detail?.project.duration_seconds ?? 0} />
          </div>
        </main>

        <aside className="min-h-0 overflow-hidden" style={{borderLeft: rightOpen ? '1px solid var(--line)' : 'none', background: 'var(--ink-900)'}}>
          {rightOpen ? <Inspector detail={detail} constraints={constraints} /> : null}
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

      <CreateDialog
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














