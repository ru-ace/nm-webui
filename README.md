# nm-webui

![Version](https://img.shields.io/github/v/tag/ru-ace/nm-webui?label=version)
![License](https://img.shields.io/github/license/ru-ace/nm-webui)
![Go Version](https://img.shields.io/github/go-mod/go-version/ru-ace/nm-webui)
![Build](https://img.shields.io/github/actions/workflow/status/ru-ace/nm-webui/build.yml?label=build)

Lightweight web interface for NetworkManager. Designed for travel routers and headless Linux devices. Single static binary, no external dependencies.

**[Русская версия](#-nm-webui-на-русском)**

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

## Quick Start

### Pre-built Binaries

Download from [Releases](https://github.com/ru-ace/nm-webui/releases):

```bash
# Linux ARM64 (RK3568, Raspberry Pi 4, etc.)
wget https://github.com/ru-ace/nm-webui/releases/latest/download/nm-webui-linux-arm64
chmod +x nm-webui-linux-arm64
sudo mv nm-webui-linux-arm64 /usr/local/bin/nm-webui

# Linux AMD64
wget https://github.com/ru-ace/nm-webui/releases/latest/download/nm-webui-linux-amd64
chmod +x nm-webui-linux-amd64
sudo mv nm-webui-linux-amd64 /usr/local/bin/nm-webui
```

### Build from Source

```bash
git clone https://github.com/ru-ace/nm-webui.git
cd nm-webui
make build        # Requires Go 1.27+ and Node.js 20+
sudo make install
```

---

## Configuration

Create `/etc/nm-webui/config.yaml`:

```yaml
listen: "0.0.0.0:8080"
auth-pass: "your-secure-password"  # Empty = no auth
interface-filter: "^(eth|en|wlan|wifi|wwan|wwan0|cdc|wl|ra|usb)[0-9A-Za-z.@_-]*$"
tls: false
tls-cert: ""
tls-key: ""
log-level: "info"
connect-timeout: 45
```

Or use environment variables:

```bash
export NM_WEBUI_LISTEN="0.0.0.0:8080"
export NM_WEBUI_AUTH_PASS="your-secure-password"
export NM_WEBUI_TLS="true"
nm-webui
```

### Command Line Flags

```bash
nm-webui --listen 0.0.0.0:8080 --auth-pass "secret" --tls --log-level debug
```

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
  `postMessage`. Disable with `--portal-allow-js=false` to fall back to
  stripping `<script>` and `on*` handlers entirely.
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

Portal JavaScript is on by default (`--portal-allow-js=true` /
`NM_WEBUI_PORTAL_ALLOW_JS=true`). Set it to `false` to strip `<script>` and
`on*` handlers from proxied pages instead of running them in the sandboxed
iframe.

---

## Systemd Service

```bash
sudo cp deploy/nm-webui.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now nm-webui
sudo journalctl -u nm-webui -f
```

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

## Requirements

- Linux with NetworkManager 1.30+
- D-Bus system bus access (root or `netdev` group)
- Kernel: 5.10+ recommended

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
# Backend
go run ./cmd/nm-webui --log-level debug

# Frontend (with hot reload)
cd webui && npm run dev

# Build everything
make build

# Cross-compile
make cross-arm64
make cross-amd64

# Package
make deb
```

---

## License

MIT License - see [LICENSE](LICENSE) for details.

---

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test && make lint`
5. Submit a PR

---

# nm-webui на русском

Лёгкий веб-интерфейс для управления NetworkManager. Создан для тревел-роутеров и безголовых Linux-устройств. Один статический бинарник, никаких внешних зависимостей.

## Возможности

- **Wi-Fi**: Сканирование, подключение к открытым/WPA2/WPA3 сетям, управление сохранёнными профилями
- **Сохранённые Wi-Fi данные**: Для известных NetworkManager сетей пароль повторно не запрашивается
- **Сетевые интерфейсы**: Просмотр всех устройств, IP-адресов, DNS, шлюза, состояния линка
- **4G/5G модемы**: Мобильные модемы с оператором, уровнем сигнала, технологией доступа (LTE/5G NR), данными SIM и IMEI; профили с APN (подключение/отключение)
- **IP-конфигурация**: Переключение между DHCP, статическим IP или отключением на интерфейс
- **Real-time**: Server-Sent Events для мгновенных обновлений статуса
- **Внешний IP**: Получение через HTTPS, синхронизация со статусом подключения и ручное обновление кэша
- **Прохождение Captive Portal**: Обнаружение страниц-«заглушек» в отелях/аэропортах и вход в сеть прямо из веб-интерфейса через встроенный прокси (невалидные TLS-сертификаты портала игнорируются)
- **Темы**: Светлая, тёмная или автоматическая тема устройства с переключателем в шапке
- **Авторизация**: HTTP Basic Auth с защитой от брутфорса
- **HTTPS**: Авто-генерируемые самоподписанные сертификаты или свои
- **Mobile-First UI**: Оптимизировано для экранов телефонов (сценарий тревел-роутера)
- **Single Binary**: Go + встроенный Svelte-фронтенд, ~10 МБ статический бинарник

## Быстрый старт

### Готовые бинарники

Скачайте с [Releases](https://github.com/ru-ace/nm-webui/releases):

```bash
# Linux ARM64 (RK3568, Raspberry Pi 4 и др.)
wget https://github.com/ru-ace/nm-webui/releases/latest/download/nm-webui-linux-arm64
chmod +x nm-webui-linux-arm64
sudo mv nm-webui-linux-arm64 /usr/local/bin/nm-webui

# Linux AMD64
wget https://github.com/ru-ace/nm-webui/releases/latest/download/nm-webui-linux-amd64
chmod +x nm-webui-linux-amd64
sudo mv nm-webui-linux-amd64 /usr/local/bin/nm-webui
```

### Сборка из исходников

```bash
git clone https://github.com/ru-ace/nm-webui.git
cd nm-webui
make build        # Требует Go 1.27+ и Node.js 20+
sudo make install
```

## Конфигурация

Создайте `/etc/nm-webui/config.yaml`:

```yaml
listen: "0.0.0.0:8080"
auth-pass: "ваш-надежный-пароль"  # Пусто = без авторизации
interface-filter: "^(eth|en|wlan|wifi|wwan|wwan0|cdc|wl|ra|usb)[0-9A-Za-z.@_-]*$"
tls: false
tls-cert: ""
tls-key: ""
log-level: "info"
connect-timeout: 45
```

Или переменные окружения:

```bash
export NM_WEBUI_LISTEN="0.0.0.0:8080"
export NM_WEBUI_AUTH_PASS="ваш-пароль"
export NM_WEBUI_TLS="true"
nm-webui
```

## Captive Portal

Когда тревел-роутер подключается к Wi-Fi отеля/аэропорта, закрытому
captive portal, веб-интерфейс обнаруживает его опросом известных эндпоинтов
(`captive.apple.com`, `detectportal.firefox.com`) и предлагает страницу
**Portal** — мини-браузер внутри SPA — для входа на стороне хоста:

- Диагностика объединяет вердикт NetworkManager и результат опроса; URL
  страницы входа берётся из опроса.
- Документ портала отрисовывается в песочнице `<iframe>`, чей `src` всегда
  указывает на прокси-эндпоинт (`/api/v1/captive-portal/proxy?url=…`), а не
  на хост портала напрямую, поэтому hostname портала резолвит только
  Go-прокси на хосте (например, DNS гостиницы), но никогда браузер клиента.
- Прокси переписывает документ на сервере (ссылки, формы, стили, `srcset`,
  meta-refresh) так, что всё резолвится обратно через прокси; формы
  отправляются через прокси, а session cookie портала хранится на стороне
  сервера, поэтому вход «прилипает».
- JavaScript портала включён по умолчанию и исполняется внутри песочницы
  iframe (`allow-forms allow-scripts allow-popups allow-modals`, **без**
  `allow-same-origin`): код портала работает в opaque origin и не может
  добраться до SPA админки, её кук или API. Маленький телеметрия-скрипт
  синхронизирует адресную строку и историю мини-браузера через
  `postMessage`. Отключить — `--portal-allow-js=false`, тогда `<script>` и
  обработчики `on*` вырезаются полностью.
- `<base>`, `<iframe>`, `<object>`, `<embed>` и пер-элементные атрибуты
  `formaction`/`formmethod` прокси вырезает всегда.
- Сертификаты портала **не проверяются** (самоподписанные и протухшие —
  обычное дело), поэтому HTTPS-порталы работают сразу.

Список проверочных URL настраивается:

```yaml
# config.yaml
portal-check-urls: "http://captive.apple.com/hotspot-detect.html,http://detectportal.firefox.com/canonical.html"
```

```bash
# CLI / env
nm-webui --portal-check-urls "http://captive.apple.com/hotspot-detect.html,http://detectportal.firefox.com/canonical.html"
export NM_WEBUI_PORTAL_URLS="http://captive.apple.com/hotspot-detect.html,http://detectportal.firefox.com/canonical.html"
```

JavaScript портала включён по умолчанию (`--portal-allow-js=true` /
`NM_WEBUI_PORTAL_ALLOW_JS=true`). Значение `false` вырезает `<script>` и
обработчики `on*` из проксируемых страниц вместо исполнения их в песочнице
iframe.

## Systemd

```bash
sudo cp deploy/nm-webui.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now nm-webui
sudo journalctl -u nm-webui -f
```

## API и SSE

Базовый путь API: `/api/v1`.

- `GET /system/status` — статус NetworkManager и connectivity.
- `POST /system/external-ip/refresh` — принудительно обновить внешний IP через HTTPS.
- `GET /system/captive-portal` — статус обнаружения captive portal.
- `POST /system/captive-portal/check` — принудительная перепроверка connectivity/портала.
- `GET/POST /captive-portal/proxy?url=…` — прокси-доступ к странице портала (GET-загрузка или POST-отправка формы с полями `url`, `_method`).
- `GET /events` — поток Server-Sent Events.

Внешний IP показывается только при статусе `online`. При отсутствии интернета значение очищается. Успешный адрес кэшируется на пять минут; на Dashboard его можно обновить кнопкой внутри иконки щита.

Основные SSE-события: `connectivity_changed`, `device_state_changed`, `devices_changed`, `scan_done`, `wifi_networks_changed`, `connections_changed`, `wifi_connected`, `wifi_failed`, `manager_state_changed`.

## Требования

- Linux с NetworkManager 1.30+
- Доступ к системной шине D-Bus (root или группа `netdev`)
- Ядро: 5.10+ рекомендуется

## Целевое железо

- **ARM64**: RK3568, Raspberry Pi 4/5, Orange Pi, Radxa Rock
- **x86_64**: Mini PC, ВМ, ноутбуки
- **ОС**: Armbian, Debian, Ubuntu, Fedora, Arch, OpenWrt (с NM)

## Ресурсы

| Метрика | Цель |
|---------|------|
| RAM | 25-35 МБ |
| CPU (простой) | 0% |
| CPU (нагрузка) | 3-5% |
| Размер бинарника | ~10 МБ |
| Зависимости | Нет (статический) |

## Лицензия

MIT — см. [LICENSE](LICENSE).
