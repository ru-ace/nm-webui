import { writable } from 'svelte/store';
import type { ConnectionInfo } from '$lib/api/client';

export const passwordModal = writable<{ ssid: string; resolve: (v: string | null) => void } | null>(null);

export function showPasswordModal(ssid: string): Promise<string | null> {
  return new Promise((resolve) => {
    passwordModal.set({ ssid, resolve });
  });
}

export interface ProfileModalState {
  iface: string;
  profiles: ConnectionInfo[];
  /** Route the Manage button navigates to (device-type specific). */
  manage: string;
  /** Controls the header/empty-state icon and copy. */
  kind: 'wifi' | 'ethernet';
  resolve: (v: ConnectionInfo | null) => void;
}

export const profileModal = writable<ProfileModalState | null>(null);

export function showProfileModal(
  iface: string,
  profiles: ConnectionInfo[],
  opts: { manage: string; kind?: 'wifi' | 'ethernet' } = {
    manage: `/wifi?iface=${encodeURIComponent(iface)}`,
    kind: 'wifi',
  }
): Promise<ConnectionInfo | null> {
  return new Promise((resolve) => {
    profileModal.set({ iface, profiles, manage: opts.manage, kind: opts.kind ?? 'wifi', resolve });
  });
}