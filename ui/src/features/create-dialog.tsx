import {useEffect, useState} from 'react';
import {X} from 'lucide-react';
import {callCapability} from '../api/capability';
import type {CatalogOption} from '../types';


export function CreateDialog({
  open,
  workspace,
  templates,
  styles,
  onClose,
  onCreated,
}: {
  open: boolean;
  workspace: string;
  templates: CatalogOption[];
  styles: CatalogOption[];
  onClose: () => void;
  onCreated: (projectId: string) => void;
}) {
  const [name, setName] = useState('');
  const [templateId, setTemplateId] = useState('');
  const [styleId, setStyleId] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setName('');
    setError(null);
    setBusy(false);
    const firstTemplate = templates.find((t) => t.status === 'ready') ?? templates[0];
    setTemplateId(firstTemplate?.id ?? '');
    setStyleId(styles[0]?.id ?? '');
  }, [open, templates, styles]);

  if (!open) return null;

  async function submit() {
    if (!name.trim() || !templateId || !styleId) {
      setError('请填写项目名称，并选择模板与风格');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const result = await callCapability<{project?: {id?: string}}>('media.workbench.project.create', {
        workspace_root: workspace,
        name: name.trim(),
        template_id: templateId,
        style_id: styleId,
      });
      const id = result?.project?.id ?? '';
      onCreated(id);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div
      className="fixed inset-0 z-20 flex items-center justify-center"
      style={{background: 'rgba(0,0,0,.55)'}}
      onClick={onClose}
    >
      <div
        className="surface w-[440px] p-4"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        <div className="mb-3 flex items-center justify-between">
          <div className="text-[13px] font-semibold">新建短视频项目</div>
          <button type="button" className="icon-btn" onClick={onClose} title="关闭">
            <X size={14} />
          </button>
        </div>

        <label className="mb-2 block">
          <div className="mb-1 text-[11px]" style={{color: 'var(--fg-dim)'}}>项目名称</div>
          <input
            className="field"
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="例如：安徽高铁网络"
            onKeyDown={(e) => { if (e.key === 'Enter') void submit(); }}
          />
        </label>

        <div className="mb-2 grid grid-cols-2 gap-2">
          <label className="block">
            <div className="mb-1 text-[11px]" style={{color: 'var(--fg-dim)'}}>模板</div>
            <select className="field" value={templateId} onChange={(e) => setTemplateId(e.target.value)}>
              {templates.map((t) => (
                <option key={t.id} value={t.id} disabled={t.status === 'planned'}>
                  {t.name ?? t.id}{t.status && t.status !== 'ready' ? `（${t.status}）` : ''}
                </option>
              ))}
            </select>
          </label>
          <label className="block">
            <div className="mb-1 text-[11px]" style={{color: 'var(--fg-dim)'}}>风格</div>
            <select className="field" value={styleId} onChange={(e) => setStyleId(e.target.value)}>
              {styles.map((s) => (
                <option key={s.id} value={s.id}>{s.name ?? s.id}</option>
              ))}
            </select>
          </label>
        </div>

        <div className="mb-3 rounded-md px-2 py-1.5 font-mono text-[10px]" style={{background: 'var(--ink-950)', color: 'var(--fg-dim)', border: '1px solid var(--line)'}}>
          {workspace || '未绑定工作区'}
        </div>

        {error ? (
          <div className="mb-2 text-[11.5px]" style={{color: 'var(--err)'}}>{error}</div>
        ) : null}

        <div className="flex justify-end gap-2">
          <button type="button" className="btn" onClick={onClose} disabled={busy}>取消</button>
          <button type="button" className="btn btn-primary" onClick={() => void submit()} disabled={busy}>
            {busy ? '创建中…' : '创建'}
          </button>
        </div>
      </div>
    </div>
  );
}

