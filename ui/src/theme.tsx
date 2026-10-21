import {useEffect, useState} from 'react';
import {Moon, Sun} from 'lucide-react';

export type Theme = 'dark' | 'light';
const THEME_KEY = 'himind.theme';

export function readTheme(): Theme {
  try {
    const stored = localStorage.getItem(THEME_KEY);
    if (stored === 'dark' || stored === 'light') return stored;
  } catch {
    // localStorage may be unavailable in restricted hosts
  }
  try {
    return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
  } catch {
    return 'dark';
  }
}

export function applyTheme(theme: Theme) {
  document.documentElement.dataset.theme = theme;
  try {
    localStorage.setItem(THEME_KEY, theme);
  } catch {
    // ignore persistence failures
  }
}

export function ThemeToggle({className = 'icon-btn'}: {className?: string}) {
  const [theme, setTheme] = useState<Theme>(() => readTheme());
  useEffect(() => {
    applyTheme(theme);
  }, [theme]);
  const next: Theme = theme === 'dark' ? 'light' : 'dark';
  const label = next === 'light' ? '切换到浅色主题' : '切换到深色主题';
  return (
    <button
      type="button"
      className={className}
      title={label}
      aria-label={label}
      onClick={() => setTheme(next)}
    >
      {theme === 'dark' ? <Sun size={14} /> : <Moon size={14} />}
    </button>
  );
}
