// Mock dataset for documentation screenshots — a fictitious "travel router".
// Tweak values here to change what the screenshots show. No real data, no
// NetworkManager required.

export const SYSTEM_STATUS = {
  state: 100,
  state_name: 'connected',
  hostname: 'travel-router',
  networkmanager_version: '1.46.0',
  connectivity: 'online',
  connectivity_code: 4,
  networking_enabled: true,
  primary_gateway: '192.168.1.1',
  external_ip: '203.0.113.42',
  external_ip_status: 'fresh',
  external_ip_checked_at: new Date().toISOString(),
  external_ip_country: 'Germany',
  external_ip_city: 'Frankfurt',
  external_ip_region: 'HE',
  external_ip_isp: 'Tretek Telecom',
  external_ip_org: 'Tretek Telecom DS',
  external_ip_asn: 'AS200782',
  external_ip_timezone: 'Europe/Berlin',
  time: new Date().toISOString(),
};

export const DEVICES = [
  {
    path: '/org/freedesktop/NetworkManager/Devices/1',
    interface: 'eth0',
    kind: 'ethernet',
    type_name: 'ethernet',
    driver: 'r8169',
    mac: '00:1a:2b:3c:4d:01',
    mtu: 1500,
    state: 100,
    state_name: 'connected',
    managed: true,
    ipv4: { addresses: [{ address: '192.168.1.100', prefix: 24 }], gateway: '192.168.1.1', nameservers: ['192.168.1.1', '1.1.1.1'], domains: ['lan'] },
    ipv6: { addresses: [], gateway: '', nameservers: [], domains: [] },
    active_connection: 'Wired connection 1',
    wireless: false,
    autoconnect: true,
  },
  {
    path: '/org/freedesktop/NetworkManager/Devices/2',
    interface: 'wlan0',
    kind: 'wifi',
    type_name: 'wifi',
    driver: 'mt7921e',
    mac: '00:1a:2b:3c:4d:02',
    mtu: 1500,
    state: 100,
    state_name: 'connected',
    managed: true,
    ipv4: { addresses: [{ address: '10.0.0.42', prefix: 24 }], gateway: '10.0.0.1', nameservers: ['10.0.0.1'], domains: ['hotel.local'] },
    ipv6: { addresses: [], gateway: '', nameservers: [], domains: [] },
    active_connection: 'HomeWiFi',
    wireless: true,
    autoconnect: true,
  },
  {
    path: '/org/freedesktop/NetworkManager/Devices/3',
    interface: 'wlan1',
    kind: 'wifi',
    type_name: 'wifi',
    driver: 'mt7921e',
    mac: '00:1a:2b:3c:4d:03',
    mtu: 1500,
    state: 30,
    state_name: 'disconnected',
    managed: true,
    ipv4: { addresses: [], gateway: '', nameservers: [], domains: [] },
    ipv6: { addresses: [], gateway: '', nameservers: [], domains: [] },
    active_connection: '',
    wireless: true,
    autoconnect: true,
  },
  {
    path: '/org/freedesktop/NetworkManager/Devices/4',
    interface: 'wwan0',
    kind: 'modem',
    type_name: 'gsm',
    driver: 'qmi_wwan',
    mac: '00:1a:2b:3c:4d:04',
    mtu: 1500,
    state: 70,
    state_name: 'connecting',
    managed: true,
    ipv4: { addresses: [], gateway: '', nameservers: [], domains: [] },
    ipv6: { addresses: [], gateway: '', nameservers: [], domains: [] },
    active_connection: 'MTS Internet',
    wireless: false,
    autoconnect: true,
    modem: {
      apn: 'internet',
      operator_code: '26201',
      operator_name: 'Tretek Mobile',
      capabilities: 7,
      capabilities_text: 'GSM/UMTS/LTE',
      signal: { percent: 78, recent: true },
      state: 8,
      state_name: 'registered',
      access_tech: 14,
      access_tech_name: 'LTE',
      manufacturer: 'Quectel',
      model: 'EG25-G',
      imei: '863000000000001',
      firmware: 'EG25GGRR07A08M2G',
      sim: {
        operator_name: 'Tretek Mobile',
        operator_code: '26201',
        iccid: '8970101000000000001',
        imsi: '262011234567890',
        active: true,
      },
    },
  },
];

export const WIFI_STATUS = {
  wlan0: {
    ssid: 'HomeWiFi',
    signal: 72,
    frequency: 5180,
    band: '5',
    channel: 36,
    security: 'wpa2',
    bssid: '60:45:bd:00:00:01',
    connected: true,
  },
  wlan1: { connected: false },
};

export const WIFI_NETWORKS = [
  {
    ssid: 'HomeWiFi',
    saved: true,
    signal: 72,
    bssid: '60:45:bd:00:00:01',
    security: 'wpa2',
    band: '5',
    channel: 36,
    frequency: 5180,
    bssids: [
      { ssid: 'HomeWiFi', saved: true, signal: 72, bssid: '60:45:bd:00:00:01', security: 'wpa2', band: '5', channel: 36, frequency: 5180 },
      { ssid: 'HomeWiFi', saved: true, signal: 55, bssid: '60:45:bd:00:00:02', security: 'wpa2', band: '5', channel: 100, frequency: 5500 },
    ],
  },
  { ssid: 'Café_Central', saved: false, signal: 64, bssid: 'a4:3c:0e:11:22:33', security: 'wpa2', band: '2', channel: 6, frequency: 2437 },
  { ssid: 'MTS-Guest', saved: false, signal: 89, bssid: 'c8:3e:a7:44:55:66', security: 'open', band: '2', channel: 1, frequency: 2412 },
  { ssid: 'NETGEAR-6G', saved: false, signal: 52, bssid: '8c:54:1b:77:88:9a', security: 'wpa3', band: '6', channel: 37, frequency: 6135 },
  { ssid: 'NETGEAR-5G', saved: false, signal: 45, bssid: '8c:54:1b:77:88:99', security: 'wpa3', band: '5', channel: 44, frequency: 5220 },
  { ssid: 'Office-Enterprise', saved: true, signal: 30, bssid: '10:3d:1c:dd:ee:ff', security: 'enterprise', band: '2', channel: 3, frequency: 2422 },
  { ssid: 'Hotel-WiFi', saved: false, signal: 41, bssid: 'f0:9f:c2:12:34:56', security: 'wpa2', band: '5', channel: 149, frequency: 5745 },
];

export const CONNECTIONS = [
  {
    path: '/org/freedesktop/NetworkManager/Settings/1',
    uuid: '11111111-1111-1111-1111-111111111111',
    id: 'HomeWiFi',
    type: '802-11-wireless',
    type_name: 'wifi',
    interface: 'wlan0',
    ssid: 'HomeWiFi',
    autoconnect: true,
    active: true,
    device: 'wlan0',
    ipv4_method: 'auto',
    ipv6_method: 'auto',
    static4: null,
    static6: null,
    is_wifi: true,
  },
  {
    path: '/org/freedesktop/NetworkManager/Settings/2',
    uuid: '22222222-2222-2222-2222-222222222222',
    id: 'Office-Guest',
    type: '802-11-wireless',
    type_name: 'wifi',
    interface: 'wlan0',
    ssid: 'Office-Guest',
    autoconnect: true,
    active: false,
    device: '',
    ipv4_method: 'auto',
    ipv6_method: 'auto',
    static4: null,
    static6: null,
    is_wifi: true,
  },
  {
    path: '/org/freedesktop/NetworkManager/Settings/3',
    uuid: '33333333-3333-3333-3333-333333333333',
    id: 'Wired connection 1',
    type: '802-3-ethernet',
    type_name: 'ethernet',
    interface: 'eth0',
    autoconnect: true,
    active: true,
    device: 'eth0',
    ipv4_method: 'auto',
    ipv6_method: 'auto',
    static4: null,
    static6: null,
    is_wifi: false,
  },
  {
    path: '/org/freedesktop/NetworkManager/Settings/4',
    uuid: '44444444-4444-4444-4444-444444444444',
    id: 'MTS Internet',
    type: 'gsm',
    type_name: 'gsm',
    interface: 'wwan0',
    apn: 'internet',
    number: '*99#',
    username: 'internet',
    autoconnect: true,
    active: true,
    device: 'wwan0',
    ipv4_method: 'auto',
    ipv6_method: 'auto',
    static4: null,
    static6: null,
    is_wifi: false,
    is_modem: true,
  },
  {
    path: '/org/freedesktop/NetworkManager/Settings/5',
    uuid: '55555555-5555-5555-5555-555555555555',
    id: 'MyVPN',
    type: 'vpn',
    type_name: 'vpn',
    interface: '',
    autoconnect: false,
    active: false,
    device: '',
    ipv4_method: 'auto',
    ipv6_method: 'auto',
    static4: null,
    static6: null,
    is_wifi: false,
  },
];

export const CAPTIVE_PORTAL = {
  state: 'online',
  portal_url: null,
  origin: null,
  probe_url: null,
  status: 'fresh',
  checked_at: new Date().toISOString(),
  nm_connectivity: 4,
  nm_connectivity_text: 'online',
};

// Portal-mode variants — served when the mock is switched to `portal` mode
// (POST /__mock/mode). Mirrors what the real backend reports: only partial
// connectivity, no external IP, and the probe-resolved sign-in page URL.
export const SYSTEM_STATUS_PORTAL = {
  ...SYSTEM_STATUS,
  connectivity: 'portal',
  connectivity_code: 2,
  external_ip: null,
  external_ip_status: 'unavailable',
  external_ip_checked_at: new Date().toISOString(),
  external_ip_country: null,
  external_ip_city: null,
  external_ip_region: null,
  external_ip_isp: null,
  external_ip_org: null,
  external_ip_asn: null,
  external_ip_timezone: null,
};

export const CAPTIVE_PORTAL_PORTAL = {
  state: 'portal',
  portal_url: 'http://10.0.0.1/login',
  origin: 'http://10.0.0.1',
  probe_url: 'http://captive.apple.com/hotspot-detect.html',
  status: 'fresh',
  checked_at: new Date().toISOString(),
  nm_connectivity: 2,
  nm_connectivity_text: 'portal',
};

// Fake hotel captive-portal landing page returned by /api/v1/captive-portal/proxy.
export const PORTAL_PAGE = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Hotel Central — Wi-Fi Sign In</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
  :root { color-scheme: light; }
  * { box-sizing: border-box; margin: 0; }
  body { font-family: -apple-system, Segoe UI, Roboto, Helvetica, Arial, sans-serif; background: #f4f1ec; color: #31261d; min-height: 100vh; }
  header { background: linear-gradient(135deg, #1f3a5f, #16324f); color: #fff; padding: 28px 20px; text-align: center; }
  header img { display: block; margin: 0 auto 10px; width: 56px; height: 56px; object-fit: contain; }
  header h1 { font-size: 22px; font-weight: 600; letter-spacing: .3px; }
  header p { color: #c7d6e8; font-size: 13px; margin-top: 6px; }
  main { max-width: 420px; margin: 0 auto; padding: 28px 20px 40px; }
  .panel { background: #fff; border: 1px solid #e5dcd0; border-radius: 12px; padding: 22px 20px; box-shadow: 0 6px 18px rgba(49,38,29,.08); }
  .panel h2 { font-size: 16px; margin-bottom: 4px; }
  .panel .sub { color: #8a7a68; font-size: 13px; margin-bottom: 18px; }
  label { display: block; font-size: 12px; font-weight: 600; color: #6b5b48; margin: 12px 0 5px; text-transform: uppercase; letter-spacing: .4px; }
  input[type=text], input[type=password] { width: 100%; padding: 11px 12px; border: 1px solid #d8cbbc; border-radius: 8px; font-size: 15px; background: #fdfcfa; }
  input:focus { outline: 2px solid #1f3a5f; border-color: transparent; }
  .terms { display: flex; gap: 8px; align-items: flex-start; margin: 16px 0 18px; font-size: 12.5px; color: #7d6d5c; }
  .terms input { margin-top: 2px; }
  button { width: 100%; background: #1f3a5f; color: #fff; border: 0; border-radius: 8px; padding: 13px; font-size: 15px; font-weight: 600; cursor: pointer; }
  button:hover { background: #16324f; }
  footer { text-align: center; color: #a08f7c; font-size: 11.5px; margin-top: 22px; }
</style>
</head>
<body>
  <header>
    <img src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 64 64'%3E%3Ccircle cx='32' cy='32' r='30' fill='%232a4d73'/%3E%3Cpath d='M18 40a20 20 0 0 1 28 0l-4 4a16 16 0 0 0-20 0zM24 47a8 8 0 0 1 16 0l-3 4-5-6-5 6z' fill='%23ffffff'/%3E%3Ccircle cx='32' cy='57' r='3' fill='%23ffffff'/%3E%3C/svg%3E" alt="Hotel Central">
    <h1>Hotel Central</h1>
    <p>Safe &amp; fast Wi-Fi for guests · 2.4/5 GHz</p>
  </header>
  <main>
    <div class="panel">
      <h2>Connect to the Internet</h2>
      <p class="sub">Please enter your room details to activate your session.</p>
      <form method="post" action="/api/v1/captive-portal/proxy">
        <input type="hidden" name="url" value="http://10.0.0.1/login">
        <label for="room">Room number</label>
        <input type="text" id="room" name="room" placeholder="e.g. 412" autocomplete="off">
        <label for="lastname">Last name</label>
        <input type="text" id="lastname" name="lastname" placeholder="e.g. Smith" autocomplete="off">
        <label for="access">Access code</label>
        <input type="password" id="access" name="access" placeholder="Shown on the room key card">
        <div class="terms">
          <input type="checkbox" id="terms" name="terms" value="1">
          <label for="terms" style="text-transform:none;letter-spacing:0;font-weight:400;margin:0">I accept the <a href="#" style="color:#1f3a5f">Terms of Service</a> and the <a href="#" style="color:#1f3a5f">Privacy Policy</a>.</label>
        </div>
        <button type="submit">Connect to Wi-Fi</button>
      </form>
    </div>
    <footer>Session duration: 24 h · 1 device · Free of charge</footer>
  </main>
</body>
</html>`;