import { writable } from 'svelte/store';
import type { ConnectionInfo } from '$lib/api/client';

export const passwordModal = writable<{ ssid: string; resolve: (v: string | null) => void } | null>(null);

export function showPasswordModal(ssid: string): Promise<string | null> {
  return new Promise((resolve) => {
    passwordModal.set({ ssid, resolve });
  });
}

export const profileModal = writable<{ iface: string; profiles: ConnectionInfo[]; resolve: (v: ConnectionInfo | null) => void } | null>(null);

export function showProfileModal(iface: string, profiles: ConnectionInfo[]): Promise<ConnectionInfo | null> {
  return new Promise((resolve) => {
    profileModal.set({ iface, profiles, resolve });
  });
}