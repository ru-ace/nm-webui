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
  kind: 'wifi' | 'ethernet' | 'modem';
  resolve: (v: ConnectionInfo | null) => void;
}

export const profileModal = writable<ProfileModalState | null>(null);

export function showProfileModal(
  iface: string,
  profiles: ConnectionInfo[],
  opts: { manage: string; kind?: 'wifi' | 'ethernet' | 'modem' } = {
    manage: `/wifi?iface=${encodeURIComponent(iface)}`,
    kind: 'wifi',
  }
): Promise<ConnectionInfo | null> {
  return new Promise((resolve) => {
    profileModal.set({ iface, profiles, manage: opts.manage, kind: opts.kind ?? 'wifi', resolve });
  });
}

/**
 * Dedicated Wi-Fi profile picker. Unlike the generic ethernet/modem modal it
 * opens *immediately* in a loading state: the backend refreshes the wireless
 * scan before answering `/devices/{iface}/connections`, which takes a few
 * seconds, and the spinner covers that round-trip. There is no single-profile
 * auto-connect for Wi-Fi — the user always picks, because "available" is only
 * meaningful after the fresh scan.
 */
export interface WifiProfileModalState {
  iface: string;
  profiles: ConnectionInfo[];
  loading: boolean;
  resolve: (v: ConnectionInfo | null) => void;
}

export const wifiProfileModal = writable<WifiProfileModalState | null>(null);

export function openWifiProfileModal(iface: string): {
  promise: Promise<ConnectionInfo | null>;
  setProfiles: (profiles: ConnectionInfo[]) => void;
  close: () => void;
} {
  let resolveFn!: (v: ConnectionInfo | null) => void;
  const promise = new Promise<ConnectionInfo | null>((r) => (resolveFn = r));
  wifiProfileModal.set({ iface, profiles: [], loading: true, resolve: resolveFn });
  return {
    promise,
    setProfiles(profiles) {
      wifiProfileModal.update((s) => (s ? { ...s, profiles, loading: false } : s));
    },
    close() {
      wifiProfileModal.update((s) => {
        s?.resolve(null);
        return null;
      });
    },
  };
}
