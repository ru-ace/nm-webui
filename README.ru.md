# nm-webui

[![Версия](https://img.shields.io/github/v/release/ru-ace/nm-webui?sort=semver&label=версия)](https://github.com/ru-ace/nm-webui/releases/latest)
[![Лицензия](https://img.shields.io/github/license/ru-ace/nm-webui)](LICENSE)
[![Версия Go](https://img.shields.io/github/go-mod/go-version/ru-ace/nm-webui)](go.mod)

Лёгкий веб-интерфейс для управления NetworkManager. Создан для тревел-роутеров и безголовых Linux-устройств. Один статический бинарник, никаких внешних зависимостей.

**[English version](README.md) · [Руководство по установке](INSTALL.ru.md) · [Скриншоты](docs/SCREENSHOTS.md)**

---

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
- **Power**: Перезагрузка или выключение хоста из веб-интерфейса с отдельным паролем и явным подтверждением (опционально)
- **HTTPS**: Авто-генерируемые самоподписанные сертификаты или свои
- **Mobile-First UI**: Оптимизировано для экранов телефонов (сценарий тревел-роутера)
- **Single Binary**: Go + встроенный Svelte-фронтенд, ~10 МБ статический бинарник

---

## Установка

Готовые бинарники, сборка из исходников, Debian-пакет, установочный скрипт,
конфигурация, systemd-сервис, фаервол и решение проблем — всё это описано в
**[руководстве по установке](INSTALL.ru.md)**.

Требования: Linux с NetworkManager 1.30+, доступ к системной шине D-Bus
(root или группа `netdev`).

---

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
  синхронизирует адресную строку мини-браузера с навигацией внутри iframe
  (переходы по ссылкам, редиректы, history API) через `postMessage`.
- `<base>`, `<iframe>`, `<object>`, `<embed>` и пер-элементные атрибуты
  (`formaction`/`formmethod` и `target`/`formtarget`) прокси вырезает всегда —
  проксированный контент не может перенаправить окно админки или открыть
  собственную вкладку поверх origin админки.
- **Open in new tab** открывает страницу через прокси уже *вне* песочницы,
  поэтому сначала требуется явное согласие (диалог с чекбоксом
  подтверждения), прежде чем страница покинет iframe.
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

Чтобы проксировать страницы **без** JavaScript, задайте `--portal-allow-js=false`
(`NM_WEBUI_PORTAL_ALLOW_JS=false`) — тогда `<script>` и обработчики `on*`
вырезаются, а не исполняются в песочнице iframe.

---

## Power

Опционально: перезагрузка или выключение хоста из веб-интерфейса. Раздел
скрыт по умолчанию и появляется, только когда задан отдельный пароль:

```yaml
# config.yaml
power-action-password: "secret"
```

```bash
# CLI / env
nm-webui --power-action-password "secret"
export NM_WEBUI_POWER_ACTION_PASSWORD="secret"
```

Страница **Power** запрашивает этот пароль и предлагает две кнопки:

- **Reboot** — перезапуск хоста;
- **Power off** — выключение хоста.

Обе открывают диалог подтверждения с предупреждением, что сервер станет
недоступен; кнопка действия заблокирована, пока не отмечен чекбокс
подтверждения. Пароль проверяется на сервере (constant-time сравнение,
rate limit как у админского логина) до выполнения действия. Неверный пароль
отвечает `401`, повторные неудачи троттлятся.

### Как выполняются команды

- От **root** (дефолтный systemd-юнит): `shutdown -r now` / `shutdown -h now`
  вызываются напрямую.
- От **не-root сервис-пользователя**: команды уходят через `sudo -n`.
  Установите приложенное узкое правило sudoers, чтобы сервис-пользователь
  мог запускать ровно эти две команды и ничего больше:

  ```bash
  sudo install -o root -g root -m 0440 deploy/sudoers/nm-webui-power /etc/sudoers.d/nm-webui-power
  sudo visudo -c
  ```

  `deploy/install.sh` делает это автоматически при выборе не-root
  пользователя и выставляет `NoNewPrivileges=false` в юните — иначе
  setuid-бинарник sudo не сможет поднять права (комментарии в
  `deploy/nm-webui.service`). На системах без бинарника `shutdown` те же
  правила можно записать под `systemctl reboot` / `systemctl poweroff`.

---

## Справочник API

Базовый путь: `/api/v1`

| Метод | Эндпоинт | Описание |
|-------|----------|----------|
| GET | `/system/status` | Статус системы, connectivity, внешний IP |
| GET | `/system/features` | Какие опциональные разделы включены (напр. `power`) |
| POST | `/system/power` | Перезагрузка/выключение — тело `{"action":"reboot"\|"poweroff","password":"…"}` (нужен power-пароль) |
| POST | `/system/external-ip/refresh` | Принудительно обновить внешний IP через HTTPS |
| GET | `/devices` | Список всех сетевых устройств |
| GET | `/devices/{iface}` | Детали устройства |
| POST | `/devices/{iface}/disconnect` | Отключить устройство |
| POST | `/devices/{iface}/up` | Подключить устройство |
| POST | `/wifi/{iface}/scan` | Запустить сканирование Wi-Fi |
| GET | `/wifi/{iface}/networks` | Список найденных сетей |
| POST | `/wifi/{iface}/connect` | Подключиться к Wi-Fi |
| GET | `/wifi/{iface}/status` | Текущее Wi-Fi-подключение |
| GET | `/connections` | Список сохранённых профилей |
| POST | `/connections` | Создать профиль |
| DELETE | `/connections/{uuid}` | Удалить профиль |
| PUT | `/connections/{uuid}` | Обновить профиль |
| POST | `/connections/{uuid}/up` | Активировать профиль |
| POST | `/connections/{uuid}/down` | Деактивировать профиль |
| GET | `/system/captive-portal` | Статус обнаружения captive portal |
| POST | `/system/captive-portal/check` | Принудительная перепроверка connectivity/портала |
| GET | `/captive-portal/proxy?url=…` | Загрузка (GET) URL через прокси портала |
| POST | `/captive-portal/proxy` | Отправка формы портала через прокси (`url`, `_method`, поля) |
| GET | `/events` | Поток Server-Sent Events |

### SSE-события

- `connectivity_changed` — статус доступа в интернет
- `device_state_changed` — состояние линка/подключения устройства
- `devices_changed` — сетевое устройство добавлено или удалено
- `scan_done` — сканирование Wi-Fi завершено
- `wifi_networks_changed` — список точек доступа Wi-Fi изменился
- `connections_changed` — профиль добавлен/удалён
- `wifi_connected` / `wifi_failed` — результат подключения
- `manager_state_changed` — состояние NetworkManager

### Поведение внешнего IP

Публичный IP определяется HTTPS-запросом к `api64.ipify.org` и кэшируется на
пять минут. Кэш используется только пока NetworkManager сообщает
`online`-connectivity. Если connectivity не `online`, API возвращает
`external_ip: null` и инвалидирует кэш.

`GET /system/status` содержит:

- `external_ip` — текущий публичный IPv4/IPv6 адрес или `null`
- `external_ip_status` — `fresh`, `cached` или `unavailable`
- `external_ip_checked_at` — время последнего результата

---

## Архитектура

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

- **Бэкенд**: Go 1.27+, `github.com/godbus/dbus/v5`, `github.com/go-chi/chi/v5`
- **Фронтенд**: Svelte 5, Vite, Tailwind CSS, DaisyUI, Lucide Icons
- **Транспорт**: REST API + Server-Sent Events
- **D-Bus**: Прямая работа с `org.freedesktop.NetworkManager` (без `nmcli`)

---

## Целевое железо

- **ARM64**: RK3568, Raspberry Pi 4/5, Orange Pi, Radxa Rock
- **x86_64**: Mini PC, ВМ, ноутбуки
- **ОС**: Armbian, Debian, Ubuntu, Fedora, Arch, OpenWrt (с NM)

---

## Ресурсы

| Метрика | Цель |
|---------|------|
| RAM | 25-35 МБ |
| CPU (простой) | 0% |
| CPU (нагрузка) | 3-5% |
| Размер бинарника | ~10 МБ |
| Зависимости | Нет (статический) |

---

## Разработка

```bash
# Бэкенд (dev-сервер)
go run ./cmd/nm-webui --log-level debug

# Фронтенд (с hot reload)
cd webui && npm run dev

# Кросс-компиляция
make cross-arm64
make cross-amd64

# Пакеты
make deb
```

Сборка и установка бинарника на устройстве описаны в
[руководстве по установке](INSTALL.ru.md) (Способ 2: сборка из исходников).
Полная проверка: `make test && make lint`.

---

## Участие в разработке

1. Сделайте форк репозитория
2. Создайте ветку для изменений
3. Внесите изменения
4. Запустите тесты: `make test && make lint`
5. Отправьте PR

---

## Лицензия

MIT — см. [LICENSE](LICENSE).