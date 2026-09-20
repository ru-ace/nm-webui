import { writable, derived, get } from 'svelte/store';
import { api, type SystemStatus, type DeviceInfo, type NetworkInfo, type ConnectionInfo, type WifiStatus } from '$lib/api/client';

export const systemStatus = writable<SystemStatus | null>(null);
export const devices = writable<DeviceInfo[]>([]);
export const wifiNetworks = writable<NetworkInfo[]>([]);
export const connections = writable<ConnectionInfo[]>([]);
export const wifiStatusMap = writable<Record<string, WifiStatus>>({});
export const loading = writable<Record<string, boolean>>({});
export const error = writable<string | null>(null);
export const toast = writable<{ message: string; type: 'success' | 'error' | 'info' } | null>(null);

export const selectedDevice = writable<string | null>(null);

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

export async function loadDevices() {
  try {
    setLoading('devices', true);
    const { devices: devs } = await api.devices.list();
    devices.set(devs);
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
    'connections_changed',
    'wifi_connected',
    'wifi_failed',
    'manager_state_changed',
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

  return es;
}

function handleEvent(type: string, data: any) {
  switch (type) {
    case 'connectivity_changed':
      systemStatus.update(($s) => $s ? { ...$s, connectivity: data.status, connectivity_code: data.connectivity } : null);
      break;
    case 'device_state_changed':
      devices.update(($d) => $d.map(d => d.interface === data.iface ? { ...d, state: data.state, state_name: data.state_name } : d));
      loadSystemStatus();
      break;
    case 'scan_done':
      loadWifiNetworks(data.iface);
      const pending = pendingScans.get(data.iface);
      if (pending) {
        pendingScans.delete(data.iface);
        pending.resolve();
      }
      break;
    case 'connections_changed':
      loadConnections();
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