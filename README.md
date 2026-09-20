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
- **Network Interfaces**: View all devices, IP addresses, DNS, gateway, link status
- **IP Configuration**: Switch between DHCP, Static IP, or Disabled per interface
- **Real-time Updates**: Server-Sent Events for live status changes
- **Authentication**: HTTP Basic Auth with rate limiting
- **HTTPS**: Auto-generated self-signed certificates or custom certs
- **Mobile-First UI**: Optimized for phone screens (travel router use case)
- **Single Binary**: Go + embedded Svelte frontend, ~10MB static binary

---

## Screenshots

| Dashboard | Wi-Fi Networks | Devices | Profiles |
|-----------|----------------|---------|----------|
| ![Dashboard](docs/dashboard.png) | ![WiFi](docs/wifi.png) | ![Devices](docs/devices.png) | ![Profiles](docs/profiles.png) |

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
make build        # Requires Go 1.22+ and Node.js 20+
sudo make install
```

---

## Configuration

Create `/etc/nm-webui/config.yaml`:

```yaml
listen: "0.0.0.0:8080"
auth-pass: "your-secure-password"  # Empty = no auth
interface-filter: "^(eth|en|wlan|wifi|wwan|wl|ra|usb)[0-9A-Za-z.@_-]*$"
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
| GET | `/events` | SSE event stream |

### SSE Events

- `connectivity_changed` - Internet connectivity status
- `device_state_changed` - Device link/connection state
- `scan_done` - Wi-Fi scan completed
- `connections_changed` - Profile added/removed
- `wifi_connected` / `wifi_failed` - Connection result

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

- **Backend**: Go 1.22+, `github.com/godbus/dbus/v5`, `github.com/go-chi/chi/v5`
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
- **Сетевые интерфейсы**: Просмотр всех устройств, IP-адресов, DNS, шлюза, состояния линка
- **IP-конфигурация**: Переключение между DHCP, статическим IP или отключением на интерфейс
- **Real-time**: Server-Sent Events для мгновенных обновлений статуса
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
make build        # Требует Go 1.22+ и Node.js 20+
sudo make install
```

## Конфигурация

Создайте `/etc/nm-webui/config.yaml`:

```yaml
listen: "0.0.0.0:8080"
auth-pass: "ваш-надежный-пароль"  # Пусто = без авторизации
interface-filter: "^(eth|en|wlan|wifi|wwan|wl|ra|usb)[0-9A-Za-z.@_-]*$"
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

## Systemd

```bash
sudo cp deploy/nm-webui.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now nm-webui
sudo journalctl -u nm-webui -f
```

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