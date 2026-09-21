# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Reboot / power off the host from the web UI (opt-in, separate password,
  explicit confirmation step, rate limited)

### Changed

- Documentation updated (README, INSTALL, Russian mirrors)

### Fixed

- Debian package: `control` file now ends with a newline so it passes `dpkg` validation
- Dashboard: city/country under the external IP are separated by ", " (Svelte
  stripped the space from the template text node at compile time)

## [1.0.0] - 2026-09-21

### Added

- Wi-Fi management: scan, connect to open/WPA2/WPA3 networks, manage saved
  profiles (existing NetworkManager profiles are reused without asking for the
  password again)
- 4G/5G modems: operator, signal strength, access technology (LTE/5G NR), SIM
  and IMEI info, APN-based connect/disconnect
- Network interfaces view: all devices, IP addresses, DNS, gateway, link status
- IP configuration: switch between DHCP, static IP, or disabled per interface
- Real-time updates via Server-Sent Events (SSE)
- External IP: HTTPS lookup (`api64.ipify.org`), synchronized with
  NetworkManager connectivity, manual cache refresh
- Captive portal bypass: hotel/airport sign-in detection and in-UI sign-in
  through a built-in proxy (invalid portal TLS certificates are ignored)
- Themes: light, dark, or automatic device-system theme with a header toggle
- Authentication: HTTP Basic Auth with rate limiting
- HTTPS: auto-generated self-signed certificates or custom certs
- Mobile-first UI optimized for phone screens (travel router use case)
- Dashboard shows Ethernet and modem interfaces
- Debian packages (`arm64`, `amd64`) and a GitHub Actions release workflow
- Single static binary (~10 MB) with the Svelte frontend embedded

### Fixed

- Dashboard bugs
- Wi-Fi scanning
- Edit profile form
- Buttons and CSS layout issues
- Captive portal behavior

### Changed

- External IP provider switched to `api64.ipify.org`
- Documentation overhaul

[Unreleased]: https://github.com/ru-ace/nm-webui/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/ru-ace/nm-webui/releases/tag/v1.0.0