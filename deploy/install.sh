#!/bin/bash
set -euo pipefail

# nm-webui installation script
# Usage: sudo ./install.sh [--binary PATH] [--config PATH] [--user USER] [--no-systemd]

BINARY_PATH="${1:-/usr/local/bin/nm-webui}"
CONFIG_PATH="${2:-/etc/nm-webui/config.yaml}"
SERVICE_USER="${3:-root}"
INSTALL_SYSTEMD="${4:-true}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== nm-webui Installation ==="
echo "Binary: $BINARY_PATH"
echo "Config: $CONFIG_PATH"
echo "Service User: $SERVICE_USER"
echo "Install systemd: $INSTALL_SYSTEMD"

# Check root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root (use sudo)"
   exit 1
fi

# Check NetworkManager
if ! systemctl is-active --quiet NetworkManager; then
    echo "Warning: NetworkManager is not running"
    read -p "Continue anyway? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Build if binary doesn't exist
if [[ ! -f "$PROJECT_ROOT/nm-webui" ]]; then
    echo "Building nm-webui..."
    cd "$PROJECT_ROOT"
    make build
fi

# Install binary
echo "Installing binary to $BINARY_PATH..."
install -Dm755 "$PROJECT_ROOT/nm-webui" "$BINARY_PATH"

# Install config
echo "Installing config to $CONFIG_PATH..."
mkdir -p "$(dirname "$CONFIG_PATH")"
if [[ ! -f "$CONFIG_PATH" ]]; then
    install -Dm600 "$SCRIPT_DIR/config.yaml" "$CONFIG_PATH"
    echo "Config installed. Please edit $CONFIG_PATH to set auth-pass and other options."
else
    echo "Config already exists, skipping."
fi

# Create service user if not root
if [[ "$SERVICE_USER" != "root" ]]; then
    if ! id "$SERVICE_USER" &>/dev/null; then
        echo "Creating user $SERVICE_USER..."
        useradd -r -s /bin/false -d /var/lib/nm-webui "$SERVICE_USER"
    fi
    usermod -a -G netdev "$SERVICE_USER" 2>/dev/null || true
    mkdir -p /var/lib/nm-webui
    chown "$SERVICE_USER:netdev" /var/lib/nm-webui
    chmod 750 /var/lib/nm-webui
fi

# Install scoped sudo rules for the Power section (reboot/poweroff). Only
# relevant for non-root service users — root executes shutdown directly.
if [[ "$SERVICE_USER" != "root" ]]; then
    SUDOERS_SRC="$SCRIPT_DIR/sudoers/nm-webui-power"
    if [[ -f "$SUDOERS_SRC" ]]; then
        echo "Installing scoped sudoers rule for Power section..."
        SUDOERS_TMP="$(mktemp)"
        sed "s/^nm-webui /$SERVICE_USER /" "$SUDOERS_SRC" > "$SUDOERS_TMP"
        install -o root -g root -m 0440 "$SUDOERS_TMP" /etc/sudoers.d/nm-webui-power
        rm -f "$SUDOERS_TMP"
        visudo -c
    else
        echo "Warning: $SUDOERS_SRC not found; skipping sudoers install."
    fi
fi

# Install systemd service
if [[ "$INSTALL_SYSTEMD" == "true" ]]; then
    echo "Installing systemd service..."
    SERVICE_FILE="/etc/systemd/system/nm-webui.service"
    sed "s|ExecStart=.*|ExecStart=$BINARY_PATH --config $CONFIG_PATH|" "$SCRIPT_DIR/nm-webui.service" > "$SERVICE_FILE"
    sed -i "s/^User=.*/User=$SERVICE_USER/" "$SERVICE_FILE"
    # sudo is setuid; NoNewPrivileges prevents it from gaining root, which
    # would break the Power section for non-root installs.
    if [[ "$SERVICE_USER" != "root" ]]; then
        sed -i "s/^NoNewPrivileges=.*/NoNewPrivileges=false/" "$SERVICE_FILE"
    fi
    systemctl daemon-reload
    systemctl enable nm-webui
    echo "Service installed. Start with: systemctl start nm-webui"
fi

echo ""
echo "=== Installation Complete ==="
echo ""
echo "Next steps:"
echo "1. Edit $CONFIG_PATH to configure auth-pass and other settings"
echo "   (Power section: set power-action-password to enable reboot/poweroff)"
echo "2. Start the service: systemctl start nm-webui"
echo "3. Check status: systemctl status nm-webui"
echo "4. View logs: journalctl -u nm-webui -f"
echo ""
echo "Web UI will be available at http://<your-ip>:8080"