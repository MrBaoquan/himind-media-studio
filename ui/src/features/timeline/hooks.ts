import {useCallback, useEffect, useMemo, useRef, useState} from 'react';
import {callCapability} from '../../api/capability';
import type {Artifact} from './types';

const blobCache = new Map<string, string>();
const pendingBlobCache = new Map<string, Promise<string>>();
const MAX_BLOB_CACHE = 80;

function trimBlobCache() {
  while (blobCache.size > MAX_BLOB_CACHE) {
    const first = blobCache.keys().next().value as string | undefined;
    if (!first) return;
    const url = blobCache.get(first);
    blobCache.delete(first);
    if (url) URL.revokeObjectURL(url);
  }
}

/** Persisted layout / preference state. Storage failures are non-fatal. */
export function usePersistentState<T>(key: string, initial: T): [T, (value: T | ((current: T) => T)) => void] {
  const [value, setValue] = useState<T>(() => {
    try {
      const raw = localStorage.getItem(key);
      if (!raw) return initial;
      return JSON.parse(raw) as T;
    } catch { return initial; }
  });
  const update = useCallback((next: T | ((current: T) => T)) => {
    setValue((current) => {
      const resolved = typeof next === 'function' ? (next as (current: T) => T)(current) : next;
      try { localStorage.setItem(key, JSON.stringify(resolved)); } catch {}
      return resolved;
    });
  }, [key]);
  return [value, update];
}

export function useArtifactBlob(workspace: string, artifactId?: string, enabled = true) {
  const [url, setUrl] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  useEffect(() => {
    if (!enabled || !artifactId || !workspace) { setUrl(''); setError(''); return; }
    const key = workspace + ':' + artifactId;
    const cached = blobCache.get(key);
    if (cached) { setUrl(cached); setError(''); return; }
    let cancelled = false;
    setLoading(true); setError('');
    let promise = pendingBlobCache.get(key);
    if (!promise) {
      promise = (async () => {
        const result = await callCapability<any>('media.workbench.artifact.read', {workspace_root: workspace, artifact_id: artifactId});
        if (!result?.base64) throw new Error('artifact has no content');
        const bytes = Uint8Array.from(atob(result.base64), (ch) => ch.charCodeAt(0));
        const kind = String(result.kind || '');
        const mime = kind.startsWith('audio/') ? kind : kind.startsWith('image/') ? kind : 'video/mp4';
        const objectUrl = URL.createObjectURL(new Blob([bytes], {type: mime}));
        blobCache.set(key, objectUrl);
        trimBlobCache();
        return objectUrl;
      })().finally(() => pendingBlobCache.delete(key));
      pendingBlobCache.set(key, promise);
    }
    promise.then((objectUrl) => { if (!cancelled) { setUrl(objectUrl); setLoading(false); } })
      .catch((err) => { if (!cancelled) { setError(err instanceof Error ? err.message : String(err)); setLoading(false); } });
    return () => { cancelled = true; };
  }, [artifactId, enabled, workspace]);
  return {url, error, loading};
}

export function useMediaDuration(url: string) {
  const [duration, setDuration] = useState(0);
  useEffect(() => {
    if (!url) { setDuration(0); return; }
    const media = document.createElement('video');
    media.preload = 'metadata';
    media.src = url;
    let cancelled = false;
    const done = () => { if (!cancelled) setDuration(Number.isFinite(media.duration) ? media.duration : 0); };
    media.addEventListener('loadedmetadata', done);
    media.addEventListener('durationchange', done);
    return () => { cancelled = true; media.removeAttribute('src'); media.load(); };
  }, [url]);
  return duration;
}

/** Client-side audio peaks for the A-track. Decoding failures degrade to an empty wave. */
export function useAudioPeaks(url: string, buckets = 160) {
  const [peaks, setPeaks] = useState<number[]>([]);
  useEffect(() => {
    if (!url) { setPeaks([]); return; }
    let cancelled = false;
    (async () => {
      try {
        const response = await fetch(url);
        const arrayBuffer = await response.arrayBuffer();
        const context = new AudioContext();
        const audio = await context.decodeAudioData(arrayBuffer.slice(0));
        const data = audio.getChannelData(0);
        const size = Math.max(1, buckets);
        const step = Math.max(1, Math.floor(data.length / size));
        const result: number[] = [];
        for (let index = 0; index < size; index++) {
          let peak = 0;
          const start = index * step;
          for (let cursor = start; cursor < Math.min(data.length, start + step); cursor += 1) {
            const value = Math.abs(data[cursor]);
            if (value > peak) peak = value;
          }
          result.push(peak);
        }
        await context.close();
        if (!cancelled) setPeaks(result);
      } catch { if (!cancelled) setPeaks([]); }
    })();
    return () => { cancelled = true; };
  }, [buckets, url]);
  return peaks;
}

export function useEventCallback<T extends (...args: any[]) => void>(callback: T) {
  const ref = useRef(callback);
  useEffect(() => { ref.current = callback; }, [callback]);
  return useCallback((...args: any[]) => ref.current(...args), []) as T;
}
