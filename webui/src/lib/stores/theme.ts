import { writable } from 'svelte/store';

export type ThemeMode = 'system' | 'light' | 'dark';
export type ActiveTheme = 'light' | 'dark';

const STORAGE_KEY = 'nm-webui-theme';

export const themeMode = writable<ThemeMode>('system');
export const activeTheme = writable<ActiveTheme>('light');

let mediaQuery: MediaQueryList | null = null;
let initialized = false;

function getActiveTheme(mode: ThemeMode): ActiveTheme {
  if (mode !== 'system') return mode;
  return mediaQuery?.matches ? 'dark' : 'light';
}

function applyTheme(mode: ThemeMode) {
  const active = getActiveTheme(mode);
  document.documentElement.dataset.theme = active;
  activeTheme.set(active);
}

function handleSystemThemeChange() {
  if (get(themeMode) === 'system') applyTheme('system');
}

export function initTheme() {
  if (initialized) return () => {};
  initialized = true;

  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
  const stored = window.localStorage.getItem(STORAGE_KEY);
  const mode: ThemeMode = stored === 'light' || stored === 'dark' ? stored : 'system';
  themeMode.set(mode);
  applyTheme(mode);
  mediaQuery.addEventListener('change', handleSystemThemeChange);

  return () => mediaQuery?.removeEventListener('change', handleSystemThemeChange);
}

export function setTheme(mode: ThemeMode) {
  themeMode.set(mode);
  window.localStorage.setItem(STORAGE_KEY, mode);
  applyTheme(mode);
}

export function cycleTheme() {
  const current = get(themeMode);
  setTheme(current === 'system' ? 'dark' : current === 'dark' ? 'light' : 'system');
}

function get<T>(store: { subscribe: (run: (value: T) => void) => () => void }): T {
  let value!: T;
  const unsubscribe = store.subscribe((current) => { value = current; });
  unsubscribe();
  return value;
}
