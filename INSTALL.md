# Installation Guide

This guide covers installing nm-webui on Linux systems with NetworkManager.

## Prerequisites

- **Linux** with **NetworkManager ≥ 1.30**
- **D-Bus system bus** access
- **Root** or user in `netdev` group with D-Bus policy
- **Kernel ≥ 5.10** (recommended)

### Verify NetworkManager

```bash
systemctl status NetworkManager
nmcli --version
```

---

## Method 1: Pre-built Binary (Recommended)

### Download

```bash
# ARM64 (RK3568, Pi 4/5, Orange Pi, etc.)
wget https://github.com/ru-ace/nm-webui/releases/latest/download/nm-webui-linux-arm64
chmod +x nm-webui-linux-arm64
sudo mv nm-webui-linux-arm64 /usr/local/bin/nm-webui

# AMD64 (x86_64)
wget https://github.com/ru-ace/nm-webui/releases/latest/download/nm-webui-linux-amd64
chmod +x nm-webui-linux-amd64
sudo mv nm-webui-linux-amd64 /usr/local/bin/nm-webui
```

### Verify

```bash
nm-webui --version
```

---

## Method 2: Build from Source

### Install Dependencies

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

### Build

```bash
git clone https://github.com/ru-ace/nm-webui.git
cd nm-webui
make build
```

### Install

```bash
sudo make install
```

This installs:
- Binary → `/usr/local/bin/nm-webui`
- Systemd unit → `/etc/systemd/system/nm-webui.service`
- Config template → `/etc/nm-webui/config.yaml`

---

## Method 3: Debian Package

```bash
# Download .deb from releases
wget https://github.com/ru-ace/nm-webui/releases/latest/download/nm-webui_1.0.0_arm64.deb
sudo dpkg -i nm-webui_1.0.0_arm64.deb
sudo apt-get install -f  # Fix dependencies if needed
```

---

## Method 4: Install Script

```bash
cd nm-webui
sudo ./deploy/install.sh
```

Options:
```bash
sudo ./deploy/install.sh [binary-path] [config-path] [user] [no-systemd]
# Example: custom user without systemd
sudo ./deploy/install.sh /usr/local/bin/nm-webui /etc/nm-webui/config.yaml nm-webui true
```

---

## Configuration

Edit `/etc/nm-webui/config.yaml`:

```yaml
listen: "0.0.0.0:8080"           # Listen address
auth-pass: "your-secure-password" # Set a password!
interface-filter: "^(eth|en|wlan|wifi|wwan|wwan0|cdc|wl|ra|usb)[0-9A-Za-z.@_-]*$"
tls: false                        # Enable HTTPS
tls-cert: ""                      # Custom cert path
tls-key: ""                       # Custom key path
log-level: "info"                 # debug, info, warn, error
connect-timeout: 45               # WiFi connect timeout (seconds)
```

### Generate Self-Signed Cert (if TLS enabled)

```bash
# Auto-generated on first run to /var/lib/nm-webui/
# Or provide your own:
nm-webui --tls --tls-cert /path/cert.pem --tls-key /path/key.pem
```

---

## Running

### As Systemd Service (Recommended)

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now nm-webui
sudo systemctl status nm-webui
journalctl -u nm-webui -f
```

### Manual Run

```bash
# Foreground
nm-webui --config /etc/nm-webui/config.yaml

# With flags (override config)
nm-webui --listen 0.0.0.0:8080 --auth-pass "secret" --tls
```

---

## Non-Root User Setup

For improved security, run as dedicated user:

```bash
# Create user
sudo useradd -r -s /bin/false -d /var/lib/nm-webui nm-webui
sudo usermod -a -G netdev nm-webui

# D-Bus policy (create /etc/dbus-1/system.d/nm-webui.conf)
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

# Update systemd service User=nm-webui
sudo systemctl daemon-reload
sudo systemctl restart nm-webui
```

---

## Firewall

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

## Access Web UI

Open browser: `http://<device-ip>:8080`

- Default: no password (if `auth-pass` empty)
- With auth: username `admin`, password from config

---

## Troubleshooting

### "Cannot connect to NetworkManager over D-Bus"

```bash
# Check NM running
systemctl status NetworkManager

# Check D-Bus access
busctl --system list | grep NetworkManager

# Test as current user
gdbus call --system --dest org.freedesktop.NetworkManager \
  --object-path /org/freedesktop/NetworkManager \
  --method org.freedesktop.DBus.Peer.Ping
```

### Permission Denied

```bash
# Add user to netdev
sudo usermod -a -G netdev $USER
# Log out/in

# Or run as root (less secure)
```

### Port Already in Use

```bash
# Check what's on 8080
sudo ss -tlnp | grep 8080

# Change port in config
listen: "0.0.0.0:8081"
```

### WiFi Not Showing

```bash
# Check interface managed by NM
nmcli device status

# Check rfkill
rfkill list

# Regulatory domain
iw reg get
```

### TLS Certificate Issues

```bash
# Regenerate self-signed
sudo rm -rf /var/lib/nm-webui
sudo systemctl restart nm-webui

# Or use real cert (Let's Encrypt)
nm-webui --tls --tls-cert /etc/letsencrypt/live/domain/fullchain.pem \
  --tls-key /etc/letsencrypt/live/domain/privkey.pem
```

---

## Uninstall

```bash
# Systemd
sudo systemctl stop nm-webui
sudo systemctl disable nm-webui
sudo rm /etc/systemd/system/nm-webui.service
sudo systemctl daemon-reload

# Binary & config
sudo rm /usr/local/bin/nm-webui
sudo rm -rf /etc/nm-webui
sudo rm -rf /var/lib/nm-webui

# User (if created)
sudo userdel nm-webui
sudo rm /etc/dbus-1/system.d/nm-webui.conf
sudo systemctl reload dbus
```

---

## Upgrading

```bash
# Binary
wget -O /usr/local/bin/nm-webui https://github.com/ru-ace/nm-webui/releases/latest/download/nm-webui-linux-arm64
chmod +x /usr/local/bin/nm-webui
sudo systemctl restart nm-webui

# Or rebuild
cd nm-webui
git pull
make build
sudo make install
sudo systemctl restart nm-webui
```