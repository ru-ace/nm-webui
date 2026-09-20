import { writable } from 'svelte/store';

export const currentPath = writable('/');

export function navigate(path: string) {
  window.history.pushState({}, '', path);
  // Only store pathname for routing, query params handled by components
  const url = new URL(path, window.location.origin);
  currentPath.set(url.pathname);
}