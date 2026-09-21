# nm-webui

![Version](https://img.shields.io/github/v/tag/ru-ace/nm-webui?label=version)
![License](https://img.shields.io/github/license/ru-ace/nm-webui)
![Go Version](https://img.shields.io/github/go-mod/go-version/ru-ace/nm-webui)

Lightweight web interface for NetworkManager. Designed for travel routers and headless Linux devices. Single static binary, no external dependencies.

**[Русская версия](README.ru.md) · [Installation Guide](install.md)**

---

## Features

- **Wi-Fi Management**: Scan, connect to open/WPA2/WPA3 networks, manage saved profiles
- **Saved Wi-Fi Credentials**: Existing NetworkManager profiles are reused without asking for the password again
- **Network Interfaces**: View all devices, IP addresses, DNS, gateway, link status
- **4G/5G Modems**: Mobile broadband devices with operator, signal strength, access technology (LTE/5G NR), SIM and IMEI info; APN-based profiles (connect/disconnect)
- **IP Configuration**: Switch between DHCP, Static IP, or Disabled per interface
- **Real-time Updates**: Server-Sent Events for live status changes
- **External IP**: HTTPS-based public IP lookup, synchronized with NetworkManager connectivity, with manual cache refresh
- **Captive Portal Bypass**: Detect hotel/airport sign-in pages, and sign in straight from the Web UI through a built-in proxy (invalid TLS certificates on portal side are ignored)
- **Themes**: Light, dark, or automatic device-system theme with a header toggle
- **Authentication**: HTTP Basic Auth with rate limiting
- **HTTPS**: Auto-generated self-signed certificates or custom certs
- **Mobile-First UI**: Optimized for phone screens (travel router use case)
- **Single Binary**: Go + embedded Svelte frontend, ~10MB static binary

---

## Installation

Pre-built binaries, building from source, Debian package, install script,
configuration, systemd service, firewall and troubleshooting are all covered
in the **[Installation Guide](install.md)**.

Requirements: Linux with NetworkManager 1.30+, D-Bus system bus access
(root or `netdev` group).

---

## Captive Portal

When the travel router joins a hotel/airport Wi-Fi that is gated by a captive
portal, the web UI detects it by probing well-known check endpoints
(`captive.apple.com`, `detectportal.firefox.com`) and offers a **Portal** page
(a mini-browser in the SPA) to sign in on behalf of the host:

- Detection combines the NetworkManager connectivity verdict with the probe
  result; the sign-in page URL is taken from the probe.
- The portal document is rendered in a sandboxed `<iframe>` whose `src` always
  points at the proxy endpoint (`/api/v1/captive-portal/proxy?url=…`) — never
  at the portal host directly — so portal hostnames are resolved only by the
  Go proxy on the host (e.g. by the hotel's DNS) and never by the client
  browser.
- The proxy rewrites the document server-side (links, forms, styles, `srcset`,
  meta-refresh) so everything resolves back through the proxy; forms are
  submitted through the proxy and any portal session cookie is kept
  server-side, so the sign-in sticks.
- Portal JavaScript is enabled by default and runs inside the sandboxed iframe
  (`allow-forms allow-scripts allow-popups allow-modals`, **without**
  `allow-same-origin`), so the portal code executes in an opaque origin and
  can never touch the admin SPA, its cookies or its API. A small telemetry
  snippet keeps the mini-browser address bar and history in sync via
  `postMessage`.
- `<base>`, `<iframe>`, `<object>`, `<embed>` and per-control `formaction`/
  `formmethod` attributes are always stripped by the proxy.
- Portal certificates are **not verified** (self-signed/expired ones are the
  norm), so HTTPS portals work out of the box.

Probe URLs are configurable:

```yaml
# config.yaml
portal-check-urls: "http://captive.apple.com/hotspot-detect.html,http://detectportal.firefox.com/canonical.html"
```

```bash
# CLI / env
nm-webui --portal-check-urls "http://captive.apple.com/hotspot-detect.html,http://detectportal.firefox.com/canonical.html"
export NM_WEBUI_PORTAL_URLS="http://captive.apple.com/hotspot-detect.html,http://detectportal.firefox.com/canonical.html"
```

To run proxied pages **without** JavaScript, set `--portal-allow-js=false`
(`NM_WEBUI_PORTAL_ALLOW_JS=false`) — `<script>` and `on*` handlers are then
stripped instead of being executed in the sandboxed iframe.

---

## API Reference

Base path: `/api/v1`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/system/status` | System status, connectivity, external IP |
| POST | `/system/external-ip/refresh` | Force an HTTPS external IP lookup and refresh the cache |
| GET | `/devices` | List all network devices |
| GET | `/devices/{iface}` | Device details |
| POST | `/devices/{iface}/disconnect` | Disconnect device |
| POST | `/devices/{iface}/up` | Connect device |
| POST | `/wifi/{iface}/scan` | Trigger Wi-Fi scan |
| GET | `/wifi/{iface}/networks` | List scanned networks |
| POST | `/wifi/{iface}/connect` | Connect to Wi-Fi |
| GET | `/wifi/{iface}/status` | Current Wi-Fi connection |
| GET | `/connections` | List saved profiles |
| POST | `/connections` | Create profile |
| DELETE | `/connections/{uuid}` | Delete profile |
| PUT | `/connections/{uuid}` | Update profile |
| POST | `/connections/{uuid}/up` | Activate profile |
| POST | `/connections/{uuid}/down` | Deactivate profile |
| GET | `/system/captive-portal` | Captive-portal detection status |
| POST | `/system/captive-portal/check` | Force a connectivity/portal re-check |
| GET | `/captive-portal/proxy?url=…` | Fetch (GET) a URL through the portal proxy |
| POST | `/captive-portal/proxy` | Submit a portal form through the proxy (`url`, `_method`, fields) |
| GET | `/events` | SSE event stream |

### SSE Events

- `connectivity_changed` - Internet connectivity status
- `device_state_changed` - Device link/connection state
- `devices_changed` - Network device added or removed
- `scan_done` - Wi-Fi scan completed
- `wifi_networks_changed` - Wi-Fi access point list changed
- `connections_changed` - Profile added/removed
- `wifi_connected` / `wifi_failed` - Connection result
- `manager_state_changed` - NetworkManager state changed

### External IP Behavior

The public IP is resolved through an HTTPS request to `api64.ipify.org` and cached for five minutes. The cache is used only while NetworkManager reports `online` connectivity. When connectivity is not `online`, the API returns `external_ip: null` and invalidates the cached value.

`GET /system/status` includes:

- `external_ip` - Current public IPv4 or IPv6 address, or `null`
- `external_ip_status` - `fresh`, `cached`, or `unavailable`
- `external_ip_checked_at` - Timestamp of the last lookup result

---

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    nm-webui (Single Binary)                  │
│                                                              │
│  ┌─────────────────────────┐      ┌───────────────────────┐  │
│  │   Embedded Web SPA      │ <──> │    Go HTTP Server     │  │
│  │   (Svelte + Tailwind)   │      │    (chi router, SSE)  │  │
│  └─────────────────────────┘      └───────────┬───────────┘  │
│                                               │              │
│                                   ┌───────────▼───────────┐  │
│                                   │   D-Bus Client        │  │
│                                   │   (godbus/dbus v5)    │  │
│                                   └───────────┬───────────┘  │
└───────────────────────────────────────────────┼──────────────┘
                                                │ System D-Bus
                                                ▼
                                   ┌─────────────────────────────┐
                                   │  NetworkManager.service     │
                                   └─────────────────────────────┘
```

- **Backend**: Go 1.27+, `github.com/godbus/dbus/v5`, `github.com/go-chi/chi/v5`
- **Frontend**: Svelte 5, Vite, Tailwind CSS, DaisyUI, Lucide Icons
- **Transport**: REST API + Server-Sent Events
- **D-Bus**: Direct `org.freedesktop.NetworkManager` communication (no `nmcli`)

---

## Hardware Targets

- **ARM64**: RK3568, Raspberry Pi 4/5, Orange Pi, Radxa Rock
- **x86_64**: Mini PCs, VMs, laptops
- **OS**: Armbian, Debian, Ubuntu, Fedora, Arch, OpenWrt (with NM)

---

## Resource Usage

| Metric | Target |
|--------|--------|
| RAM | 25-35 MB |
| CPU (idle) | 0% |
| CPU (active) | 3-5% |
| Binary Size | ~10 MB |
| Dependencies | None (static) |

---

## Development

```bash
# Backend (dev server)
go run ./cmd/nm-webui --log-level debug

# Frontend (with hot reload)
cd webui && npm run dev

# Cross-compile
make cross-arm64
make cross-amd64

# Package
make deb
```

Building and installing the binary on a device is described in the
[Installation Guide](install.md) (Method 2: Build from Source). Run the full
check suite with `make test && make lint`.

---

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test && make lint`
5. Submit a PR

---

## License

MIT License - see [LICENSE](LICENSE) for details.