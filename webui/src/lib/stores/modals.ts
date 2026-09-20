import { writable } from 'svelte/store';

export const passwordModal = writable<{ ssid: string; resolve: (v: string | null) => void } | null>(null);

export function showPasswordModal(ssid: string): Promise<string | null> {
  return new Promise((resolve) => {
    passwordModal.set({ ssid, resolve });
  });
}