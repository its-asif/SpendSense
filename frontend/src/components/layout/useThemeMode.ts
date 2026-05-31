import { useEffect, useState } from 'react';

const THEME_KEY = 'spendsense-theme';

export function readThemeMode() {
  if (typeof window === 'undefined') {
    return 'dark' as const;
  }

  const saved = window.localStorage.getItem(THEME_KEY);
  if (saved === 'light' || saved === 'dark') {
    return saved;
  }

  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export function applyThemeMode(mode: 'light' | 'dark') {
  if (typeof document === 'undefined') {
    return;
  }

  document.documentElement.setAttribute('data-theme', mode);
}

export function useThemeMode() {
  const [mode, setMode] = useState<'light' | 'dark'>(() => readThemeMode());

  useEffect(() => {
    applyThemeMode(mode);
    window.localStorage.setItem(THEME_KEY, mode);
  }, [mode]);

  const toggleMode = () => {
    setMode((current) => (current === 'dark' ? 'light' : 'dark'));
  };

  return { mode, toggleMode };
}