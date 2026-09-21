import { writable } from 'svelte/store';

export const currentPath = writable('/');

// Fired on every programmatic or browser-driven navigation with the full
// destination URL (including query string), so components can react to query
// changes (e.g. the portal page reading ?url=) even when the pathname is the
// same.
export const NAVIGATE_EVENT = 'nm:navigate';

function emit(url: URL) {
  window.dispatchEvent(new CustomEvent(NAVIGATE_EVENT, { detail: url }));
}

export function navigate(path: string) {
  window.history.pushState({}, '', path);
  const url = new URL(path, window.location.origin);
  currentPath.set(url.pathname);
  emit(url);
}

// Keep the store in sync with browser back/forward so the SPA router reacts
// to popstate too (not only programmatic navigate() calls).
if (typeof window !== 'undefined') {
  window.addEventListener('popstate', () => {
    currentPath.set(window.location.pathname);
    emit(new URL(window.location.href));
  });
}