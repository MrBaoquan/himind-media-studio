import {useMemo} from 'react';

/**
 * 时长标尺 —— 题材无关。
 *
 * 真正的"分镜/年份/事件"时间轴属于模板能力，不应该写进外壳。
 * 这里只呈现渲染器会用到的时间范围。
 */
export function DurationRuler({durationSeconds}: {durationSeconds: number}) {
  const ticks = useMemo(() => {
    if (!durationSeconds || durationSeconds <= 0) return [];
    const step = durationSeconds <= 12 ? 2 : durationSeconds <= 30 ? 5 : 10;
    const out: number[] = [];
    for (let t = 0; t <= durationSeconds; t += step) out.push(t);
    if (out[out.length - 1] !== durationSeconds) out.push(durationSeconds);
    return out;
  }, [durationSeconds]);

  if (!ticks.length) {
    return (
      <div className="flex h-full items-center justify-center text-[11.5px]" style={{color: 'var(--fg-dim)'}}>
        未选择项目
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col justify-center gap-3 px-4">
      <div className="flex items-center justify-between text-[10.5px]" style={{color: 'var(--fg-dim)'}}>
        <span>时长</span>
        <span>{durationSeconds}s</span>
      </div>
      <div className="relative h-6">
        <div className="absolute inset-x-0 top-3 h-[2px] rounded" style={{background: 'var(--line-strong)'}} />
        {ticks.map((t) => {
          const left = (t / durationSeconds) * 100;
          return (
            <div key={t} className="absolute top-0 flex flex-col items-center" style={{left: `${left}%`, transform: 'translateX(-50%)'}}>
              <div className="h-2 w-px" style={{background: 'var(--line-strong)'}} />
              <span className="mt-1 text-[9.5px]" style={{color: 'var(--fg-dim)'}}>{t}s</span>
            </div>
          );
        })}
      </div>
      <div className="text-[9.5px]" style={{color: 'var(--fg-dim)'}}>
        时间轴结构由模板定义
      </div>
    </div>
  );
}
