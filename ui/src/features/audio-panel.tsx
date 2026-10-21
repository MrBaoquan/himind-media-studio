import {useCallback, useEffect, useMemo, useState} from 'react';
import {AlertTriangle, Captions, Download, Mic2, RefreshCw, ShieldCheck, Volume2} from 'lucide-react';
import {callCapability} from '../api/capability';

export interface AudioVoicePack {
  id: string;
  name: string;
  language: string;
  quality?: string;
  license: string;
  license_url?: string;
  commercial_use: boolean;
  demo_only?: boolean;
  available: boolean;
  note?: string;
}

export interface AudioRuntimeStatus {
  id: string;
  version: string;
  installed: boolean;
  path?: string;
  executable?: string;
  license: string;
  voices: AudioVoicePack[];
  missing?: string[];
  next?: string;
}

export interface AudioCatalogProvider {
  id: string;
  name: string;
  available: boolean;
  network: boolean;
  offline?: boolean;
  license?: string;
  note?: string;
  runtime?: AudioRuntimeStatus;
}

export interface AudioCatalog {
  providers: AudioCatalogProvider[];
  voices: Array<{name: string; culture: string; gender?: string; default?: boolean}>;
  bgm_presets: Array<{id: string; name: string; description?: string; mood?: string[]; license: string; loop?: boolean}>;
  defaults: {provider: string; voice: string; rate: number; volume: number; bgm_preset: string; voice_gain_db: number; bgm_gain_db: number; ducking: boolean};
  warnings?: string[];
}

export interface AudioQA {
  ok: boolean;
  status: string;
  duration_seconds: number;
  expected_duration_seconds?: number;
  integrated_lufs?: number;
  true_peak_dbtp?: number;
  max_volume_db?: number;
  warnings?: string[];
  blockers?: string[];
}

export interface AudioManifest {
  version: string;
  creative_id: string;
  project_id?: string;
  voice_provider: string;
  voice: string;
  voice_license?: string;
  commercial_voice?: boolean;
  rate: number;
  bgm_preset: string;
  bgm_gain_db: number;
  voice_gain_db: number;
  ducking: boolean;
  duration_seconds: number;
  clips: Array<{beat_id: string; voice: string; duration_seconds: number; effective_duration_seconds: number; tempo?: number; license?: string}>;
  subtitles: Array<{start: number; end: number; text: string; beat_id?: string}>;
  bgm?: {id: string; name: string; license: string; duration_seconds: number};
  mix_path: string;
  qa?: AudioQA;
  source?: string;
}

interface AudioPanelProps {
  workspace: string;
  creativeId: string;
  projectId?: string;
  durationSeconds: number;
  onNotice?: (message: string) => void;
  onError?: (message: string) => void;
}

function Panel({title, icon: Icon, children, actions}: {title: string; icon: any; children: any; actions?: any}) {
  return (
    <section className="panel">
      <div className="panel-head">
        <Icon size={12} style={{color: 'var(--accent)'}} />
        <span className="text-[11px] font-semibold">{title}</span>
        <div className="ml-auto flex items-center gap-1">{actions}</div>
      </div>
      <div className="panel-body space-y-3">{children}</div>
    </section>
  );
}

function DataRow({label, value, tone}: {label: string; value: any; tone?: 'ok' | 'warn' | 'bad'}) {
  const color = tone === 'bad' ? 'var(--danger)' : tone === 'warn' ? 'var(--warn)' : 'var(--fg)';
  return (
    <div className="flex items-center justify-between gap-3 text-[11px]">
      <span style={{color: 'var(--fg-dim)'}}>{label}</span>
      <span className="truncate font-mono" style={{color, maxWidth: 360}} title={String(value ?? '')}>{String(value ?? '—')}</span>
    </div>
  );
}

export function AudioPanel({workspace, creativeId, projectId, durationSeconds, onNotice, onError}: AudioPanelProps) {
  const [catalog, setCatalog] = useState<AudioCatalog | null>(null);
  const [provider, setProvider] = useState('piper');
  const [voice, setVoice] = useState('');
  const [rate, setRate] = useState(4);
  const [bgmPreset, setBgmPreset] = useState('documentary-drone');
  const [bgmGain, setBgmGain] = useState(-20);
  const [ducking, setDucking] = useState(true);
  const [allowRestricted, setAllowRestricted] = useState(false);
  const [manifest, setManifest] = useState<AudioManifest | null>(null);
  const [busy, setBusy] = useState(false);
  const [busyText, setBusyText] = useState('');
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!workspace || !creativeId) return;
    try {
      const next = await callCapability<AudioCatalog>('audio.catalog', {workspace_root: workspace});
      setCatalog(next);
      setProvider(next.defaults?.provider || 'piper');
      setVoice(next.defaults?.voice || '');
      setRate(next.defaults?.rate ?? 4);
      setBgmPreset(next.defaults?.bgm_preset || 'documentary-drone');
      setBgmGain(next.defaults?.bgm_gain_db ?? -20);
      setDucking(next.defaults?.ducking ?? true);
      const status = await callCapability<{state: string; manifest?: AudioManifest}>('audio.manifest.get', {
        workspace_root: workspace,
        creative_id: creativeId,
      });
      setManifest(status.manifest ?? null);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(message);
      onError?.(message);
    }
  }, [workspace, creativeId, onError]);

  useEffect(() => {
    void load();
  }, [load]);

  const piper = catalog?.providers.find((item) => item.id === 'piper')?.runtime;
  const piperVoices = piper?.voices ?? [];
  const sapiVoices = catalog?.voices ?? [];
  const selectedPiperVoice = useMemo(() => piperVoices.find((item) => item.id === voice), [piperVoices, voice]);
  const selectedSapiVoice = useMemo(() => sapiVoices.find((item) => item.name === voice), [sapiVoices, voice]);
  const voiceOptions = provider === 'piper'
    ? piperVoices.map((item) => ({id: item.id, label: `${item.name} · ${item.license}`, disabled: !item.available}))
    : sapiVoices.map((item) => ({id: item.name, label: `${item.name} · ${item.culture}`, disabled: false}));
  const restricted = provider === 'piper' && selectedPiperVoice && !selectedPiperVoice.commercial_use;

  async function installRuntime() {
    setBusy(true);
    setBusyText('正在下载并校验 Piper 运行时与声音（约 150 MB）…');
    setError(null);
    try {
      const result = await callCapability<{state: string; runtime: AudioRuntimeStatus}>('audio.runtime.install', {
        workspace_root: workspace,
        runtime_id: 'piper',
        dry_run: false,
      });
      onNotice?.(`Piper 运行时已就绪：${result.runtime.version}`);
      await load();
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(message);
      onError?.(message);
    } finally {
      setBusy(false);
      setBusyText('');
    }
  }

  async function build() {
    setBusy(true);
    setBusyText('正在合成旁白、切分字幕、生成 BGM 并混音…');
    setError(null);
    try {
      const result = await callCapability<{state: string; manifest: AudioManifest; warnings?: string[]; attached_path?: string; blockers?: any[]}>('audio.pipeline.render', {
        workspace_root: workspace,
        creative_id: creativeId,
        project_id: projectId || undefined,
        voice_provider: provider,
        voice,
        voice_rate: rate,
        allow_restricted_voice: allowRestricted,
        bgm_preset: bgmPreset,
        bgm_gain_db: bgmGain,
        ducking,
        duration_seconds: durationSeconds,
      });
      if ((result as any).blockers?.length) {
        const message = (result as any).blockers.map((item: any) => item.message).join('；');
        setError(message);
        onError?.(message);
        return;
      }
      let cues = result.manifest.subtitles;
      let alignmentNote = '';
      try {
        await callCapability('audio.align', {workspace_root: workspace, creative_id: creativeId});
        const aligned = await callCapability<{subtitles?: Array<{start: number; end: number; text: string; beat_id?: string}>}>('audio.subtitle.from-alignment', {
          workspace_root: workspace,
          creative_id: creativeId,
          max_chars: 24,
        });
        if (aligned.subtitles?.length) {
          cues = aligned.subtitles;
          alignmentNote = '，已使用逐词对齐字幕';
        }
      } catch {
        alignmentNote = '，逐词对齐不可用，使用 cue 切分字幕';
      }
      let attachNote = '';
      if (projectId) {
        try {
          const attached = await callCapability<{state: string; attached_path?: string}>('media.workbench.audio.attach', {
            workspace_root: workspace,
            project_id: projectId,
            creative_id: creativeId,
            audio_path: result.manifest.mix_path,
            subtitles: cues,
          });
          attachNote = attached.attached_path ? '，字幕已挂载到项目' : '';
        } catch (attachErr) {
          attachNote = '，但字幕挂载失败：' + (attachErr instanceof Error ? attachErr.message : String(attachErr));
        }
      }
      setManifest(result.manifest);
      const warningCount = result.warnings?.length ?? 0;
      onNotice?.(`音频底座已生成：${result.manifest.clips.length} 段旁白 / ${cues.length} 条字幕${warningCount ? ` / ${warningCount} 条提示` : ''}${alignmentNote}${attachNote}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(message);
      onError?.(message);
    } finally {
      setBusy(false);
      setBusyText('');
    }
  }

  async function recheck() {
    if (!manifest?.mix_path) return;
    setBusy(true);
    setBusyText('正在复查响度与峰值…');
    try {
      const result = await callCapability<{qa: AudioQA}>('audio.qa.signal', {
        workspace_root: workspace,
        path: manifest.mix_path,
        expected_duration_seconds: durationSeconds,
      });
      setManifest({...manifest, qa: result.qa});
      onNotice?.(`音频 QA：${result.qa.status} · ${result.qa.integrated_lufs ?? '—'} LUFS · ${result.qa.true_peak_dbtp ?? '—'} dBTP`);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(message);
      onError?.(message);
    } finally {
      setBusy(false);
      setBusyText('');
    }
  }

  const qa = manifest?.qa;
  const qaTone = qa?.status === 'passed' ? 'ok' : qa?.status === 'warning' ? 'warn' : qa?.status === 'failed' ? 'bad' : undefined;

  return (
    <Panel title="旁白 / 字幕 / BGM" icon={Volume2} actions={
      <>
        <button type="button" className="btn btn-ghost" onClick={() => void load()} disabled={busy} title="刷新音频状态"><RefreshCw size={11} />刷新</button>
        {piper && !piper.installed ? <button type="button" className="btn" onClick={() => void installRuntime()} disabled={busy}><Download size={11} />安装 Piper</button> : null}
        <button type="button" className="btn btn-primary" onClick={() => void build()} disabled={busy || !catalog}>
          <Mic2 size={11} />生成音频底座
        </button>
      </>
    }>
      {busy ? <div className="rounded-md border px-3 py-2 text-[11px]" style={{borderColor: 'rgba(90,169,255,.35)', background: 'var(--accent-soft)', color: 'var(--accent)'}}>{busyText}</div> : null}
      {error ? <div className="flex items-start gap-2 rounded-md border px-3 py-2 text-[11px]" style={{borderColor: 'rgba(255,120,120,.35)', background: 'rgba(255,120,120,.08)', color: 'var(--danger)'}}><AlertTriangle size={12} className="mt-0.5 shrink-0" /><span>{error}</span></div> : null}

      <div className="grid gap-2 md:grid-cols-2">
        <label className="text-[10px]" style={{color: 'var(--fg-dim)'}}>
          TTS Provider
          <select className="mt-1 w-full rounded-md border px-2 py-1.5 text-[11px]" style={{borderColor: 'var(--line)', background: 'var(--ink-850)', color: 'var(--fg)'}} value={provider} onChange={(event) => { const next = event.target.value; setProvider(next); setVoice(next === 'piper' ? (piperVoices.find((item) => item.available)?.id ?? '') : (sapiVoices.find((item) => item.default)?.name ?? sapiVoices[0]?.name ?? '')); }}>
            {(catalog?.providers ?? []).map((item) => <option key={item.id} value={item.id} disabled={!item.available && item.id !== 'piper'}>{item.name}{item.available ? '' : '（未就绪）'}</option>)}
          </select>
        </label>
        <label className="text-[10px]" style={{color: 'var(--fg-dim)'}}>
          声音
          <select className="mt-1 w-full rounded-md border px-2 py-1.5 text-[11px]" style={{borderColor: 'var(--line)', background: 'var(--ink-850)', color: 'var(--fg)'}} value={voice} onChange={(event) => setVoice(event.target.value)}>
            {voiceOptions.length === 0 ? <option value="">（没有可用声音）</option> : null}
            {voiceOptions.map((item) => <option key={item.id} value={item.id} disabled={item.disabled}>{item.label}{item.disabled ? '（不兼容）' : ''}</option>)}
          </select>
        </label>
        <label className="text-[10px]" style={{color: 'var(--fg-dim)'}}>
          语速（-10…10）
          <input type="range" min={-10} max={10} step={1} value={rate} onChange={(event) => setRate(Number(event.target.value))} className="mt-2 w-full" />
          <span className="font-mono text-[10px]">{rate > 0 ? `+${rate}` : rate}</span>
        </label>
        <label className="text-[10px]" style={{color: 'var(--fg-dim)'}}>
          BGM 预设
          <select className="mt-1 w-full rounded-md border px-2 py-1.5 text-[11px]" style={{borderColor: 'var(--line)', background: 'var(--ink-850)', color: 'var(--fg)'}} value={bgmPreset} onChange={(event) => setBgmPreset(event.target.value)}>
            {(catalog?.bgm_presets ?? []).map((item) => <option key={item.id} value={item.id}>{item.name} · {item.license}</option>)}
          </select>
        </label>
        <label className="text-[10px]" style={{color: 'var(--fg-dim)'}}>
          BGM 音量（dB）
          <input type="number" min={-60} max={0} step={1} value={bgmGain} onChange={(event) => setBgmGain(Number(event.target.value))} className="mt-1 w-full rounded-md border px-2 py-1.5 font-mono text-[11px]" style={{borderColor: 'var(--line)', background: 'var(--ink-850)', color: 'var(--fg)'}} />
        </label>
        <label className="mt-5 flex items-center gap-2 text-[11px]" style={{color: 'var(--fg-muted)'}}>
          <input type="checkbox" checked={ducking} onChange={(event) => setDucking(event.target.checked)} />
          旁白时自动压低 BGM（ducking）
        </label>
      </div>

      {restricted ? (
        <label className="flex items-start gap-2 rounded-md border px-3 py-2 text-[11px]" style={{borderColor: 'rgba(230,189,62,.35)', background: 'rgba(230,189,62,.07)', color: 'var(--warn)'}}>
          <input type="checkbox" className="mt-0.5" checked={allowRestricted} onChange={(event) => setAllowRestricted(event.target.checked)} />
          <span>当前声音「{selectedPiperVoice?.name}」的许可为「{selectedPiperVoice?.license}」，仅供开发/演示。确认后勾选，正式商用请替换为可商用 voice pack。</span>
        </label>
      ) : null}

      {manifest ? (
        <div className="space-y-2 rounded-md border p-3" style={{borderColor: 'var(--line)', background: 'rgba(255,255,255,.015)'}}>
          <div className="flex items-center gap-2 text-[11px] font-semibold"><ShieldCheck size={12} style={{color: 'var(--accent)'}} />音频 manifest</div>
          <DataRow label="Provider / Voice" value={`${manifest.voice_provider} / ${manifest.voice}`} />
          <DataRow label="Voice License" value={manifest.voice_license || '—'} tone={manifest.commercial_voice ? 'ok' : 'warn'} />
          <DataRow label="旁白片段" value={`${manifest.clips.length} 段`} />
          <DataRow label="字幕 cues" value={`${manifest.subtitles.length} 条`} />
          <DataRow label="BGM" value={manifest.bgm ? `${manifest.bgm.name} · ${manifest.bgm.license}` : '—'} />
          <DataRow label="混音" value={manifest.mix_path} />
          <DataRow label="音频时长" value={`${manifest.duration_seconds.toFixed(2)} s`} />
          {qa ? <>
            <DataRow label="QA 状态" value={qa.status} tone={qaTone} />
            <DataRow label="整体响度" value={qa.integrated_lufs != null ? `${qa.integrated_lufs.toFixed(2)} LUFS` : '—'} />
            <DataRow label="真峰值" value={qa.true_peak_dbtp != null ? `${qa.true_peak_dbtp.toFixed(2)} dBTP` : '—'} />
            <DataRow label="采样峰值" value={qa.max_volume_db != null ? `${qa.max_volume_db.toFixed(2)} dB` : '—'} />
          </> : null}
          {(manifest.qa?.warnings ?? []).length ? <div className="text-[10px]" style={{color: 'var(--warn)'}}>{manifest.qa?.warnings?.join('；')}</div> : null}
          <div className="flex flex-wrap items-center gap-2 pt-1">
            <button type="button" className="btn btn-ghost" onClick={() => void recheck()} disabled={busy}><ShieldCheck size={11} />复查音频 QA</button>
            <span className="text-[10px]" style={{color: 'var(--fg-dim)'}}>字幕已写入项目 data.json；重新预览后再渲染成片。</span>
          </div>
        </div>
      ) : (
        <div className="rounded-md border px-3 py-2 text-[11px]" style={{borderColor: 'var(--line)', color: 'var(--fg-dim)'}}>
          <Captions size={12} className="mr-1 inline" />尚未生成音频。生成后会得到逐 beat 旁白、字幕 cues、BGM 与一条混音 M4A。
        </div>
      )}
      {catalog?.warnings?.length ? <div className="text-[10px]" style={{color: 'var(--fg-dim)'}}>{catalog.warnings.join('；')}</div> : null}
    </Panel>
  );
}
