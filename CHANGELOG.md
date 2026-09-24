# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Hidden Wi-Fi networks: the profile form gains a "hidden" option (SSID is
  not broadcast); hidden saved networks get a "Hidden" badge in the Wi-Fi
  table, the Connect profile picker and the Profiles list, and are treated as
  available — NetworkManager probes their SSID during scans
- Profiles: "Forget" now asks for confirmation (reusable ConfirmModal used in
  desktop and mobile card layouts)
- **Connect** on Ethernet and Mobile Broadband cards (dashboard + Devices
  page) now activates a saved profile instead of blindly bringing the
  interface up: NetworkManager is asked which saved profiles fit the
  interface — the only one activates immediately, several open a profile
  picker, none falls back to **Manage** (Profiles section). Devices-page
  cards share the dashboard's button set, so both pages behave identically

### Changed

- Wi-Fi **Connect** (dashboard, Wi-Fi-section and Devices cards) opens the
  profile picker too — always, without single-profile auto-connect — because
  for wireless devices the applicable-profile list is only correct after a
  fresh scan; the picker opens immediately with a loading spinner while the
  scan runs
- Devices cards: the "Wi-Fi Networks" shortcut was replaced by a unified
  **Manage** button (Wi-Fi section for wireless cards, Profiles for the rest)
- Profile cards: the compact card is expand-only (its Disconnect/Connect
  shortcuts were unreachable); the expanded card shows "Disconnect" for
  active profiles and "Forget" (with confirmation) for inactive ones, and
  tracks NetworkManager's active-state changes so cards update regardless of
  what triggered the change
- Captive portal: a successful sign-in is confirmed by an active connectivity
  probe, so the UI reports `online` right after sign-in instead of waiting for
  NetworkManager's periodic check; a manual recheck forces both NetworkManager's
  check and a fresh probe
- Wi-Fi table badges (Saved / Hidden / AP count) are now icon-based on mobile,
  with text labels retained on desktop widths

### Fixed

- The Wi-Fi Connect picker listed a random/partial subset of saved profiles:
  NetworkManager builds the list for wireless devices from the *last scan*.
  The picker now re-triggers the scan before reading it (in-modal spinner;
  falls back to the cached list if the scan fails)
- **Disconnect** could silently reconnect: it now calls `Device.Disconnect`
  (like `nmcli device disconnect`), pinning the device to Disconnected until
  a manual Connect
- Profiles stayed stale when changed outside the UI (nmcli, KDE, another
  tab): active-state changes come from `Connection.Active` `StateChanged`
  signals, profile edits from `Settings.Connection.Updated`, both bridged to
  the SSE refresh
- Default HTTP listen port changed from 8080 to 8090 (config default, dev
  proxy target, deploy package, docs)
- The Wi-Fi page could ignore `?iface=` on in-app navigation and open the
  first Wi-Fi interface instead
- Devices/Dashboard cards kept a stale IP address, gateway and DNS after
  connect/disconnect until reload: terminal device-state SSE events now carry
  a full device snapshot (with a targeted refresh as fallback)

## [1.1.1] - 2026-09-23

### Changed

- Mobile header: section title (h1) and its primary action button now live in
  the sticky navbar; action buttons are icon-only and highlighted on phones,
  the nav menu opens from the section title (the hamburger button was removed)
  with its items centered, and the "nm-webui" brand text is hidden (the logo
  remains as the home button)
- Wi-Fi table: on desktop it now shows dedicated Security and Band columns
  (previously hidden in small overlays on the signal icon); the Join button is
  disabled for the network the selected interface is already connected to
- Portal: the Recheck button moved to the mobile navbar header (icon-only) and
  was removed from the toolbar on desktop
- Portal: the proxy HTML rewriter now strips `target`/`formtarget` attributes
  so proxied pages can never open their own top-level tab against the admin
  origin; "Open in new tab" is icon-only on mobile and requires an explicit
  risk acknowledgement (a checkbox confirmation dialog, like the power one)
  before the page leaves the sandbox
- Portal: the mini-browser Back/Forward buttons were removed — browsing
  inside a portal site is covered by the site's own navigation (and, for
  simple cases, the browser's back button), and the buttons never became
  enabled because their disabled state depended on a history stack Svelte
  could not make reactive; the address bar now also follows in-frame
  navigation (link clicks, redirects, history API) reported by telemetry,
  not just toolbar navigation
- Portal: the toolbar Home button was removed — the navbar (desktop) and the
  section menu (mobile) already provide one-click access to the admin
  dashboard, and a "home" icon inside a mini-browser is ambiguous
- Section headings shortened to match the navigation labels (Wi-Fi, Devices,
  Profiles)
- Desktop navbar: brand text now shows the hostname from system status instead
  of "nm-webui"
- Browser tab title now shows the device hostname (e.g. "ace-wb") once system
  status loads; "nm-webui" remains only as the pre-boot/unavailable fallback
- Desktop layout: every section except Portal uses one unified, centered
  content width (896px); the Devices grid now fits two columns
- Documentation and screenshot tooling updated: a new screenshot state for the
  Portal "Open in new tab" confirmation dialog, the mock Wi-Fi dataset now
  shows a multi-AP network (2 BSSIDs) and a 6 GHz band network, and the
  README/`docs/SCREENSHOTS.md` text is aligned with the portal changes
  (no back/forward buttons, `target`/`formtarget` stripped by the proxy, sandbox
  confirmation dialog)

### Fixed

- Desktop layout: page containers could collapse to the width of their content
  instead of honoring `max-w-4xl` (flex-column `mx-auto` prevented stretching),
  so sections rendered at inconsistent widths; containers now always fill the
  unified width

## [1.1.0] - 2026-09-21

### Added

- Reboot / power off the host from the web UI (opt-in, separate password,
  explicit confirmation step, rate limited)

### Changed

- Frontend toolchain refreshed: `lucide-svelte` → `@lucide/svelte` (package
  renamed upstream, old one deprecated), Vite 5 → 6 and
  `@sveltejs/vite-plugin-svelte` 4 → 5; `npm audit` is clean (0 vulnerabilities)
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

[Unreleased]: https://github.com/ru-ace/nm-webui/compare/v1.1.1...HEAD
[1.1.1]: https://github.com/ru-ace/nm-webui/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/ru-ace/nm-webui/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/ru-ace/nm-webui/releases/tag/v1.0.0