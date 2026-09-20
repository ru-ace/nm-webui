# Руководство по установке

Это руководство описывает установку nm-webui на Linux-системы с NetworkManager.

## Требования

- **Linux** с **NetworkManager ≥ 1.30**
- Доступ к **системной шине D-Bus**
- **Root** или пользователь в группе `netdev` с D-Bus политикой
- **Ядро ≥ 5.10** (рекомендуется)

### Проверка NetworkManager

```bash
systemctl status NetworkManager
nmcli --version
```

---

## Способ 1: Готовый бинарник (Рекомендуется)

### Скачивание

```bash
# ARM64 (RK3568, Pi 4/5, Orange Pi и др.)
wget https://github.com/ru-ace/nm-webui/releases/latest/download/nm-webui-linux-arm64
chmod +x nm-webui-linux-arm64
sudo mv nm-webui-linux-arm64 /usr/local/bin/nm-webui

# AMD64 (x86_64)
wget https://github.com/ru-ace/nm-webui/releases/latest/download/nm-webui-linux-amd64
chmod +x nm-webui-linux-amd64
sudo mv nm-webui-linux-amd64 /usr/local/bin/nm-webui
```

### Проверка

```bash
nm-webui --version
```

---

## Способ 2: Сборка из исходников

### Установка зависимостей

**Debian/Ubuntu/Armbian:**
```bash
sudo apt update
sudo apt install -y golang-go nodejs npm git make
```

**Fedora/RHEL:**
```bash
sudo dnf install -y golang nodejs npm git make
```

**Arch:**
```bash
sudo pacman -S go nodejs npm git make
```

### Сборка

```bash
git clone https://github.com/ru-ace/nm-webui.git
cd nm-webui
make build
```

### Установка

```bash
sudo make install
```

Это установит:
- Бинарник → `/usr/local/bin/nm-webui`
- Systemd юнит → `/etc/systemd/system/nm-webui.service`
- Пример конфига → `/etc/nm-webui/config.yaml`

---

## Способ 3: Debian-пакет

```bash
# Скачайте .deb с releases
wget https://github.com/ru-ace/nm-webui/releases/latest/download/nm-webui_1.0.0_arm64.deb
sudo dpkg -i nm-webui_1.0.0_arm64.deb
sudo apt-get install -f  # Исправить зависимости при необходимости
```

---

## Способ 4: Установочный скрипт

```bash
cd nm-webui
sudo ./deploy/install.sh
```

Параметры:
```bash
sudo ./deploy/install.sh [путь-бинарника] [путь-конфига] [пользователь] [no-systemd]
# Пример: кастомный пользователь без systemd
sudo ./deploy/install.sh /usr/local/bin/nm-webui /etc/nm-webui/config.yaml nm-webui true
```

---

## Конфигурация

Отредактируйте `/etc/nm-webui/config.yaml`:

```yaml
listen: "0.0.0.0:8080"           # Адрес прослушивания
auth-pass: "ваш-надежный-пароль"  # Задайте пароль!
interface-filter: "^(eth|en|wlan|wifi|wwan|wl|ra|usb)[0-9A-Za-z.@_-]*$"
tls: false                        # Включить HTTPS
tls-cert: ""                      # Путь к сертификату
tls-key: ""                       # Путь к ключу
log-level: "info"                 # debug, info, warn, error
connect-timeout: 45               # Таймаут подключения WiFi (сек)
```

### Генерация самоподписанного сертификата (если TLS включён)

```bash
# Автоматически при первом запуске в /var/lib/nm-webui/
# Или укажите свои:
nm-webui --tls --tls-cert /путь/cert.pem --tls-key /путь/key.pem
```

---

## Запуск

### Как Systemd-сервис (Рекомендуется)

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now nm-webui
sudo systemctl status nm-webui
journalctl -u nm-webui -f
```

### Ручной запуск

```bash
# На переднем плане
nm-webui --config /etc/nm-webui/config.yaml

# С флагами (переопределяют конфиг)
nm-webui --listen 0.0.0.0:8080 --auth-pass "secret" --tls
```

---

## Запуск не от root (Безопаснее)

Для повышения безопасности запускайте от отдельного пользователя:

```bash
# Создать пользователя
sudo useradd -r -s /bin/false -d /var/lib/nm-webui nm-webui
sudo usermod -a -G netdev nm-webui

# D-Bus политика (создайте /etc/dbus-1/system.d/nm-webui.conf)
cat << 'EOF' | sudo tee /etc/dbus-1/system.d/nm-webui.conf
<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-BUS Bus Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
  <policy user="nm-webui">
    <allow send_destination="org.freedesktop.NetworkManager"/>
    <allow receive_sender="org.freedesktop.NetworkManager"/>
  </policy>
</busconfig>
EOF

sudo systemctl reload dbus

# Обновите systemd сервис: User=nm-webui
sudo systemctl daemon-reload
sudo systemctl restart nm-webui
```

---

## Фаервол

```bash
# UFW
sudo ufw allow 8080/tcp

# firewalld
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload

# iptables
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
```

---

## Доступ к веб-интерфейсу

Откройте в браузере: `http://<ip-устройства>:8080`

- По умолчанию: без пароля (если `auth-pass` пустой)
- С авторизацией: логин `admin`, пароль из конфига

---

## Решение проблем

### "Cannot connect to NetworkManager over D-Bus"

```bash
# Проверьте NM
systemctl status NetworkManager

# Проверьте доступ к D-Bus
busctl --system list | grep NetworkManager

# Тест от текущего пользователя
gdbus call --system --dest org.freedesktop.NetworkManager \
  --object-path /org/freedesktop/NetworkManager \
  --method org.freedesktop.DBus.Peer.Ping
```

### Permission Denied

```bash
# Добавьте пользователя в netdev
sudo usermod -a -G netdev $USER
# Перелогиньтесь

# Или запустите от root (менее безопасно)
```

### Порт уже занят

```bash
# Проверьте, что на 8080
sudo ss -tlnp | grep 8080

# Смените порт в конфиге
listen: "0.0.0.0:8081"
```

### WiFi не отображается

```bash
# Проверьте интерфейсы под управлением NM
nmcli device status

# Проверьте rfkill
rfkill list

# Регуляторный домен
iw reg get
```

### Проблемы с TLS-сертификатом

```bash
# Перегенерировать самоподписанный
sudo rm -rf /var/lib/nm-webui
sudo systemctl restart nm-webui

# Или используйте настоящий (Let's Encrypt)
nm-webui --tls --tls-cert /etc/letsencrypt/live/domain/fullchain.pem \
  --tls-key /etc/letsencrypt/live/domain/privkey.pem
```

---

## Удаление

```bash
# Systemd
sudo systemctl stop nm-webui
sudo systemctl disable nm-webui
sudo rm /etc/systemd/system/nm-webui.service
sudo systemctl daemon-reload

# Бинарник и конфиг
sudo rm /usr/local/bin/nm-webui
sudo rm -rf /etc/nm-webui
sudo rm -rf /var/lib/nm-webui

# Пользователь (если создавали)
sudo userdel nm-webui
sudo rm /etc/dbus-1/system.d/nm-webui.conf
sudo systemctl reload dbus
```

---

## Обновление

```bash
# Бинарник
wget -O /usr/local/bin/nm-webui https://github.com/ru-ace/nm-webui/releases/latest/download/nm-webui-linux-arm64
chmod +x /usr/local/bin/nm-webui
sudo systemctl restart nm-webui

# Или пересборка
cd nm-webui
git pull
make build
sudo make install
sudo systemctl restart nm-webui
```