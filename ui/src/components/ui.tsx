import type {ReactNode} from 'react';

export const cx = (...parts: Array<string | false | null | undefined>) =>
  parts.filter(Boolean).join(' ');

export function Badge({
  children,
  tone,
}: {
  children: ReactNode;
  tone?: 'accent' | 'ok' | 'warn' | 'err';
}) {
  return <span className={cx('badge', tone && `badge-${tone}`)}>{children}</span>;
}

export function SectionLabel({children, trailing}: {children: ReactNode; trailing?: ReactNode}) {
  return (
    <div className="section-label">
      <span>{children}</span>
      {trailing ? <span className="normal-case tracking-normal">{trailing}</span> : null}
    </div>
  );
}

export function Kv({label, value}: {label: string; value: ReactNode}) {
  return (
    <div className="kv">
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

export function Empty({title, hint}: {title: string; hint?: string}) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-1 px-6 text-center">
      <div className="text-[12.5px]" style={{color: 'var(--fg-muted)'}}>{title}</div>
      {hint ? <div className="text-[11.5px]" style={{color: 'var(--fg-dim)'}}>{hint}</div> : null}
    </div>
  );
}
