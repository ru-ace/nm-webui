const API_BASE = '/api/v1';

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers
    }
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export const api = {
  system: {
    status: () => request<SystemStatus>('/system/status'),
    refreshExternalIP: () => request<Pick<SystemStatus, 'external_ip' | 'external_ip_status' | 'external_ip_checked_at'>>('/system/external-ip/refresh', { method: 'POST' })
  },
  devices: {
    list: () => request<{ devices: DeviceInfo[] }>('/devices'),
    get: (iface: string) => request<DeviceInfo>(`/devices/${encodeURIComponent(iface)}`),
    disconnect: (iface: string) => request<{ status: string }>(`/devices/${encodeURIComponent(iface)}/disconnect`, { method: 'POST' }),
    up: (iface: string) => request<{ status: string }>(`/devices/${encodeURIComponent(iface)}/up`, { method: 'POST' })
  },
  wifi: {
    scan: (iface: string) => request<{ status: string }>(`/wifi/${encodeURIComponent(iface)}/scan`, { method: 'POST' }),
    networks: (iface: string, group = true) => request<{ networks: NetworkInfo[] }>(`/wifi/${encodeURIComponent(iface)}/networks?group=${group ? '1' : '0'}`),
    connect: (iface: string, ssid: string, password?: string) => request<{ status: string; ssid: string }>(`/wifi/${encodeURIComponent(iface)}/connect`, {
      method: 'POST',
      body: JSON.stringify({ ssid, password })
    }),
    status: (iface: string) => request<WifiStatus>(`/wifi/${encodeURIComponent(iface)}/status`)
  },
  connections: {
    list: () => request<{ connections: ConnectionInfo[] }>('/connections'),
    create: (data: ConnectionRequest) => request<ConnectionInfo>('/connections', { method: 'POST', body: JSON.stringify(data) }),
    delete: (uuid: string) => request<{ status: string }>(`/connections/${encodeURIComponent(uuid)}`, { method: 'DELETE' }),
    update: (uuid: string, data: ConnectionRequest) => request<ConnectionInfo>(`/connections/${encodeURIComponent(uuid)}`, { method: 'PUT', body: JSON.stringify(data) }),
    up: (uuid: string) => request<{ status: string; path: string }>(`/connections/${encodeURIComponent(uuid)}/up`, { method: 'POST' }),
    down: (uuid: string) => request<{ status: string }>(`/connections/${encodeURIComponent(uuid)}/down`, { method: 'POST' })
  }
};

export interface SystemStatus {
  state: number;
  state_name: string;
  hostname: string;
  networkmanager_version: string;
  connectivity: string;
  connectivity_code: number;
  networking_enabled: boolean;
  primary_gateway: string;
  external_ip: string | null;
  external_ip_status: 'fresh' | 'cached' | 'unavailable';
  external_ip_checked_at: string;
  time: string;
}

export interface DeviceInfo {
  path: string;
  interface: string;
  kind: string;
  type_name: string;
  driver: string;
  mac: string;
  mtu: number;
  state: number;
  state_name: string;
  managed: boolean;
  ipv4: IPConfig;
  ipv6: IPConfig;
  active_connection: string;
  wireless: boolean;
  autoconnect: boolean;
}

export interface IPConfig {
  addresses: Array<{ address: string; prefix: number; gateway?: string }>;
  gateway: string;
  nameservers: string[];
  domains: string[];
}

export interface NetworkInfo {
  ssid: string;
  saved: boolean;
  signal: number;
  bssid?: string;
  security: string;
  band?: string;
  channel: number;
  frequency: number;
  bssids?: NetworkInfo[];
}

export interface WifiStatus {
  ssid?: string;
  signal?: number;
  frequency?: number;
  band?: string;
  channel?: number;
  security?: string;
  bssid?: string;
  connected: boolean;
}

export interface ConnectionInfo {
  path: string;
  uuid: string;
  id: string;
  type: string;
  type_name: string;
  interface: string;
  ssid: string;
  autoconnect: boolean;
  active: boolean;
  device: string;
  ipv4_method: string;
  ipv6_method: string;
  static4: StaticConfig | null;
  static6: StaticConfig | null;
  is_wifi: boolean;
}

export interface StaticConfig {
  address: string;
  prefix: number;
  gateway: string;
  dns: string[];
}

export interface ConnectionRequest {
  id: string;
  interface: string;
  autoconnect?: boolean;
  type: string;
  ssid?: string;
  password?: string;
  ipv4?: IPConfigRequest;
  ipv6?: IPConfigRequest;
}

export interface IPConfigRequest {
  method: string;
  address?: string;
  prefix?: number;
  gateway?: string;
  dns?: string[];
}
