package nm

import (
	"strings"

	"github.com/godbus/dbus/v5"
)

// PrimaryConnection returns the active connection providing the default route.
func (c *Client) PrimaryConnection() (dbus.ObjectPath, error) {
	return c.propObjectPath(c.nm, NmIfName, "PrimaryConnection")
}

// PrimaryIPv4Gateway resolves the default gateway from the primary connection.
func (c *Client) PrimaryIPv4Gateway() (string, error) {
	p, err := c.PrimaryConnection()
	if err != nil || p == "/" {
		return "", ErrNoActiveConn
	}
	obj := c.conn.Object(DBusService, p)
	ip4Path, err := c.propObjectPath(obj, ActiveConnIf, "Ip4Config")
	if err != nil || ip4Path == "/" {
		return "", ErrNoActiveConn
	}
	cfgObj := c.conn.Object(DBusService, ip4Path)
	return c.propString(cfgObj, IP4ConfigIf, "Gateway")
}

// WiFiStatus is a live snapshot of the currently connected wifi network.
type WiFiStatus struct {
	SSID      string `json:"ssid,omitempty"`
	Signal    int    `json:"signal,omitempty"`
	Frequency uint32 `json:"frequency,omitempty"`
	Band      string `json:"band,omitempty"`
	Channel   uint32 `json:"channel,omitempty"`
	Security  string `json:"security,omitempty"`
	BSSID     string `json:"bssid,omitempty"`
	Connected bool   `json:"connected"`
}

// CurrentWiFi returns details of the network the device is connected to.
func (c *Client) WiFiStatus(dev *Device) (*WiFiStatus, error) {
	acPath, err := dev.ActiveConnection()
	if err != nil {
		return nil, err
	}
	if acPath == "/" {
		return &WiFiStatus{Connected: false}, nil
	}
	ac := &ActiveConnection{c: c, path: acPath, obj: c.conn.Object(DBusService, acPath)}

	status := &WiFiStatus{Connected: true}
	if sobj, err := ac.SpecificObject(); err == nil && sobj != "/" && sobj != "" &&
		strings.HasPrefix(string(sobj), "/org/freedesktop/NetworkManager/AccessPoint/") {
		ap := c.newAP(sobj)
		if info, err := ap.Info(); err == nil {
			status.SSID = info.SSID
			status.Signal = info.Signal
			status.Frequency = info.Frequency
			status.Band = info.Band
			status.Channel = info.Channel
			status.Security = info.Security
			status.BSSID = info.BSSID
		}
	}
	if status.SSID == "" {
		if v, err := ac.obj.GetProperty(ActiveConnIf + ".Connection"); err == nil {
			if conPath, ok := v.Value().(dbus.ObjectPath); ok && conPath != "/" {
				if con := c.connAt(conPath); con != nil {
					if s, err := con.GetSettings(); err == nil {
						if w := s["802-11-wireless"]; w != nil {
							if v, ok := w["ssid"].Value().([]byte); ok {
								status.SSID = decodeSSID(v)
							}
						}
					}
				}
			}
		}
	}
	return status, nil
}

// ReasonName returns a human label for a device/active-connection state reason.
func ReasonName(reason uint32) string {
	switch reason {
	case 0:
		return "none"
	case 1:
		return "unknown"
	case 2:
		return "user-disconnected"
	case 3:
		return "device-disconnected"
	case 4:
		return "service-stopped"
	case 5:
		return "ip-config-invalid"
	case 6:
		return "connect-timeout"
	case 7:
		return "service-start-timeout"
	case 8:
		return "authentication-failed"
	case 9:
		return "no-secrets"
	case 10:
		return "login-failed"
	case 11:
		return "connection-removed"
	case 12:
		return "dependency-failed"
	case 13:
		return "device-realize-failed"
	case 14:
		return "device-removed"
	default:
		return "unknown"
	}
}