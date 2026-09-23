import { writable, derived, get } from 'svelte/store';
import { api, type SystemStatus, type DeviceInfo, type NetworkInfo, type ConnectionInfo, type WifiStatus, type CaptivePortalStatus, type SystemFeatures } from '$lib/api/client';
import { showProfileModal } from './modals';

export const systemStatus = writable<SystemStatus | null>(null);
export const devices = writable<DeviceInfo[]>([]);
export const wifiNetworks = writable<NetworkInfo[]>([]);
export const connections = writable<ConnectionInfo[]>([]);
export const wifiStatusMap = writable<Record<string, WifiStatus>>({});
export const captivePortal = writable<CaptivePortalStatus | null>(null);
export const features = writable<SystemFeatures | null>(null);
export const loading = writable<Record<string, boolean>>({});
export const error = writable<string | null>(null);
export const toast = writable<{ message: string; type: 'success' | 'error' | 'info' } | null>(null);

export const selectedDevice = writable<string | null>(null);

/** Interface with a "card Connect" profile-picker flow currently in flight. */
export const connectingIface = writable<string | null>(null);

const pendingScans = new Map<string, { resolve: () => void; reject: (e: Error) => void }>();

export const connectivityStatus = derived(systemStatus, ($s) => $s?.connectivity || 'unknown');
export const isOnline = derived(connectivityStatus, ($c) => $c === 'online');

function setLoading(key: string, value: boolean) {
  loading.update(($l) => ({ ...$l, [key]: value }));
}

export async function loadSystemStatus() {
  try {
    const status = await api.system.status();
    systemStatus.set(status);
  } catch (e) {
    console.error('Failed to load system status:', e);
  }
}

export async function loadFeatures() {
  try {
    const feat = await api.system.features();
    features.set(feat);
  } catch (e) {
    console.error('Failed to load features:', e);
  }
}

export async function loadCaptivePortal() {
  try {
    const status = await api.captivePortal.status();
    captivePortal.set(status);
  } catch (e) {
    console.error('Failed to load captive portal status:', e);
  }
}

export async function recheckCaptivePortal() {
  try {
    setLoading('portal', true);
    const status = await api.captivePortal.check();
    captivePortal.set(status);
    await loadSystemStatus();
    if (status.state === 'portal') {
      showToast('Captive portal detected — sign in to continue', 'info');
    } else {
      showToast(`Connectivity: ${status.state}`, 'success');
    }
  } catch (e: any) {
    console.error('Failed to recheck captive portal:', e);
    showToast(`Recheck failed: ${e?.message || e}`, 'error');
  } finally {
    setLoading('portal', false);
  }
}

export async function refreshExternalIP() {
  try {
    setLoading('external-ip', true);
    await api.system.refreshExternalIP();
    await loadSystemStatus();
    showToast('External IP cache refreshed', 'success');
  } catch (e: any) {
    console.error('Failed to refresh external IP:', e);
    showToast(`Failed to refresh external IP: ${e?.message || e}`, 'error');
  } finally {
    setLoading('external-ip', false);
  }
}

export async function loadDevices() {
  try {
    setLoading('devices', true);
    const { devices: devs } = await api.devices.list();
    devices.set(devs);
    await Promise.all(devs.filter((d) => d.wireless).map((d) => loadWifiStatus(d.interface)));
  } catch (e) {
    console.error('Failed to load devices:', e);
    showToast('Failed to load devices', 'error');
  } finally {
    setLoading('devices', false);
  }
}

export async function loadWifiNetworks(iface: string) {
  try {
    setLoading(`wifi-${iface}`, true);
    const { networks } = await api.wifi.networks(iface);
    wifiNetworks.set(networks);
  } catch (e) {
    console.error('Failed to load wifi networks:', e);
    showToast('Failed to scan networks', 'error');
  } finally {
    setLoading(`wifi-${iface}`, false);
  }
}

async function refreshWifiNetworksAfterScan(iface: string) {
  await new Promise((resolve) => setTimeout(resolve, 500));
  await loadWifiNetworks(iface);
}

export async function triggerScan(iface: string) {
  setLoading(`wifi-${iface}`, true);
  showToast('Scanning...', 'info');

  try {
    await api.wifi.scan(iface);
  } catch (e) {
    setLoading(`wifi-${iface}`, false);
    showToast('Scan failed: ' + ((e as any)?.message || e), 'error');
    throw e;
  }

  return new Promise<void>((resolve, reject) => {
    const timeout = setTimeout(() => {
      pendingScans.delete(iface);
      setLoading(`wifi-${iface}`, false);
      reject(new Error('Scan timeout'));
    }, 30000);

    pendingScans.set(iface, {
      resolve: () => {
        clearTimeout(timeout);
        setLoading(`wifi-${iface}`, false);
        resolve();
      },
      reject: (e) => {
        clearTimeout(timeout);
        setLoading(`wifi-${iface}`, false);
        reject(e);
      },
    });
  });
}

export async function connectWifi(iface: string, ssid: string, password?: string) {
  try {
    setLoading(`connect-${iface}`, true);
    await api.wifi.connect(iface, ssid, password);
    showToast(`Connecting to ${ssid}...`, 'info');
  } catch (e) {
    console.error('Connect failed:', e);
    showToast(`Failed to connect: ${e}`, 'error');
  } finally {
    setLoading(`connect-${iface}`, false);
  }
}

export async function loadConnections() {
  try {
    setLoading('connections', true);
    const { connections: conns } = await api.connections.list();
    connections.set(conns);
  } catch (e) {
    console.error('Failed to load connections:', e);
    showToast('Failed to load connections', 'error');
  } finally {
    setLoading('connections', false);
  }
}

export async function loadWifiStatus(iface: string) {
  try {
    const status = await api.wifi.status(iface);
    wifiStatusMap.update(($m) => ({ ...$m, [iface]: status }));
  } catch (e) {
    console.error('Failed to load wifi status:', e);
  }
}

export async function forgetConnection(uuid: string) {
  try {
    await api.connections.delete(uuid);
    showToast('Connection forgotten', 'success');
    await loadConnections();
  } catch (e) {
    console.error('Delete failed:', e);
    showToast('Failed to forget connection', 'error');
  }
}

export async function toggleAutoconnect(uuid: string, enabled: boolean) {
  try {
    await api.connections.update(uuid, { autoconnect: enabled } as any);
    showToast(enabled ? 'Auto-connect enabled' : 'Auto-connect disabled', 'success');
    await loadConnections();
  } catch (e) {
    console.error('Toggle failed:', e);
    showToast('Failed to update', 'error');
  }
}

export async function activateConnection(uuid: string) {
  try {
    await api.connections.up(uuid);
    showToast('Connecting...', 'info');
    await loadConnections();
    await loadDevices();
  } catch (e) {
    console.error('Activate failed:', e);
    showToast('Failed to activate', 'error');
  }
}

export async function deactivateConnection(uuid: string) {
  try {
    await api.connections.down(uuid);
    showToast('Disconnecting...', 'info');
    await loadConnections();
    await loadDevices();
  } catch (e) {
    console.error('Deactivate failed:', e);
    showToast('Failed to deactivate', 'error');
  }
}

export async function createConnection(data: any) {
  try {
    await api.connections.create(data);
    showToast('Connection created', 'success');
    await loadConnections();
  } catch (e) {
    console.error('Create failed:', e);
    showToast('Failed to create connection', 'error');
  }
}

export async function updateConnection(uuid: string, data: any) {
  try {
    await api.connections.update(uuid, data);
    showToast('Connection updated', 'success');
    await loadConnections();
  } catch (e: any) {
    console.error('Update failed:', e);
    showToast('Failed to update connection: ' + (e?.message || e), 'error');
  }
}

export async function disconnectDevice(iface: string) {
  try {
    setLoading(`disconnect-${iface}`, true);
    await api.devices.disconnect(iface);
    showToast(`Disconnecting ${iface}...`, 'info');
    await loadDevices();
  } catch (e: any) {
    console.error('Disconnect failed:', e);
    showToast(`Failed to disconnect: ${e?.message || e}`, 'error');
  } finally {
    setLoading(`disconnect-${iface}`, false);
  }
}

export async function connectDevice(iface: string) {
  try {
    setLoading(`connect-${iface}`, true);
    await api.devices.up(iface);
    showToast(`Connecting ${iface}...`, 'info');
    await loadDevices();
  } catch (e: any) {
    console.error('Connect failed:', e);
    showToast(`Failed to connect: ${e?.message || e}`, 'error');
  } finally {
    setLoading(`connect-${iface}`, false);
  }
}

// Card "Connect": ask NetworkManager which saved profiles are applicable to
// this interface and either activate directly (single profile) or let the
// user pick one via the shared profile modal. With no applicable profiles the
// modal falls back to a Manage route (Wi-Fi section for wireless cards, the
// Connections page for wired and modem ones, where profiles are created).
// `connectingIface` stays set while the flow runs so cards can disable their
// Connect button.
export async function connectWithPicker(
  iface: string,
  manage: string,
  kind: 'wifi' | 'ethernet' | 'modem'
) {
  connectingIface.set(iface);
  try {
    const { connections: profiles } = await api.devices.connections(iface);
    if (profiles.length === 0) {
      await showProfileModal(iface, profiles, { manage, kind });
    } else if (profiles.length === 1) {
      await activateConnection(profiles[0].uuid);
    } else {
      const selected = await showProfileModal(iface, profiles, { manage, kind });
      if (selected) {
        await activateConnection(selected.uuid);
      }
    }
  } catch (e: any) {
    console.error('Failed to load device profiles:', e);
    showToast(`Failed to load profiles: ${e?.message || e}`, 'error');
  } finally {
    connectingIface.set(null);
  }
}

export function showToast(message: string, type: 'success' | 'error' | 'info' = 'info') {
  toast.set({ message, type });
  setTimeout(() => toast.set(null), 4000);
}

let activeEventSource: EventSource | null = null;

export function initEventSource() {
  if (activeEventSource) {
    activeEventSource.close();
  }
  const es = new EventSource('/api/v1/events');
  activeEventSource = es;

  const eventTypes = [
    'connectivity_changed',
    'device_state_changed',
    'scan_done',
    'wifi_networks_changed',
    'connections_changed',
    'connection_state_changed',
    'wifi_connected',
    'wifi_failed',
    'manager_state_changed',
    'devices_changed',
  ];

  for (const type of eventTypes) {
    es.addEventListener(type, (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data);
        handleEvent(type, data);
      } catch (err) {
        console.error(`Failed to parse SSE event ${type}:`, err);
      }
    });
  }

  es.onmessage = (e) => {
    try {
      const event = JSON.parse(e.data);
      if (event && event.type) {
        handleEvent(event.type, event.data || event);
      }
    } catch {
    }
  };

  es.onerror = () => {
    // Browser EventSource automatically reconnects with Last-Event-ID
  };

  // Refresh on (re)connect: a dropped stream may have missed events outside
  // the hub replay window, leaving profile cards stale.
  es.onopen = () => {
    loadConnections();
  };

  return es;
}

// isCurrentWifiIface reports whether iface is still present as a managed
// wireless device in the latest known device list.
function isCurrentWifiIface(iface: string) {
  if (!iface) return false;
  return get(devices).some((d) => d.wireless && d.interface === iface);
}

function handleEvent(type: string, data: any) {
  switch (type) {
    case 'connectivity_changed':
      systemStatus.update(($s) => $s ? { ...$s, connectivity: data.status, connectivity_code: data.connectivity } : null);
      void loadCaptivePortal();
      break;
    case 'device_state_changed':
      devices.update(($d) => $d.map(d => d.interface === data.iface ? { ...d, state: data.state, state_name: data.state_name } : d));
      loadSystemStatus();
      break;
    case 'scan_done':
      // Ignore stale events for interfaces that no longer exist (e.g. after
      // unplugging a USB Wi-Fi adapter), otherwise the subsequent network
      // request fails with a misleading error toast.
      if (!isCurrentWifiIface(data.iface)) {
        const stale = pendingScans.get(data.iface);
        if (stale) {
          pendingScans.delete(data.iface);
          stale.reject(new Error('Wi-Fi device was removed'));
        }
        break;
      }
      const pending = pendingScans.get(data.iface);
      if (pending) {
        pendingScans.delete(data.iface);
        void refreshWifiNetworksAfterScan(data.iface).then(pending.resolve, pending.reject);
      } else {
        void refreshWifiNetworksAfterScan(data.iface);
      }
      break;
    case 'wifi_networks_changed':
      if (isCurrentWifiIface(data.iface)) {
        void loadWifiNetworks(data.iface);
      }
      break;
    case 'connections_changed':
      loadConnections();
      break;
    case 'connection_state_changed':
      // Targeted patch from NM ActiveConnection StateChanged signals: covers
      // both UI actions (the local reload races the async teardown) and
      // external changes (nmcli, autoconnect, other tabs and clients).
      connections.update(($c) => $c.map((co) => (co.uuid === data.uuid ? { ...co, active: !!data.active } : co)));
      break;
    case 'devices_changed':
      loadDevices();
      break;
    case 'wifi_connected':
      showToast(`Connected to ${data.ssid}`, 'success');
      loadDevices();
      loadConnections();
      break;
    case 'wifi_failed':
      showToast(`Connection failed: ${data.error}`, 'error');
      break;
    case 'manager_state_changed':
      loadSystemStatus();
      break;
  }
}
