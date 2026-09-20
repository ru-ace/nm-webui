package nm

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
)

// Connect errors surfaced to the API layer.
var (
	ErrConnectTimeout   = errors.New("connection timed out")
	ErrAuthFailed       = errors.New("authentication failed: invalid password")
	ErrDeviceFailed     = errors.New("device failed while connecting")
	ErrNoActiveConn     = errors.New("no active connection")
	ErrNotConnectedSSID = errors.New("already connected to another network")
)

func newUUID() (string, error) { return NewUUID() }

// NewUUID generates a random RFC 4122 version 4 UUID string.
func NewUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate UUID: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// BuildWiFiSettings constructs the D-Bus settings block for a wifi profile.
func BuildWiFiSettings(ssid, password string) (map[string]map[string]dbus.Variant, error) {
	uuid, err := newUUID()
	if err != nil {
		return nil, err
	}
	settings := map[string]map[string]dbus.Variant{
		"connection": {
			"type":        dbus.MakeVariant("802-11-wireless"),
			"id":          dbus.MakeVariant(ssid),
			"uuid":        dbus.MakeVariant(uuid),
			"autoconnect": dbus.MakeVariant(true),
		},
		"802-11-wireless": {
			"ssid": dbus.MakeVariant([]byte(ssid)),
			"mode": dbus.MakeVariant("infrastructure"),
		},
		"ipv4": {
			"method": dbus.MakeVariant("auto"),
		},
		"ipv6": {
			"method": dbus.MakeVariant("auto"),
		},
	}
	if password != "" {
		settings["802-11-wireless"]["security"] = dbus.MakeVariant("802-11-wireless-security")
		settings["802-11-wireless-security"] = map[string]dbus.Variant{
			"key-mgmt": dbus.MakeVariant("wpa-psk"),
			"psk":      dbus.MakeVariant(password),
		}
	}
	return settings, nil
}

// FindWiFiConnection locates a saved profile with the given SSID.
func (c *Client) FindWiFiConnection(ssid string) (*Connection, error) {
	conns, err := c.ListConnections()
	if err != nil {
		return nil, err
	}
	for _, co := range conns {
		s, err := co.GetSettings()
		if err != nil {
			continue
		}
		if variantString(s["connection"], "type") != "802-11-wireless" {
			continue
		}
		if w := s["802-11-wireless"]; w != nil {
			if v, ok := w["ssid"].Value().([]byte); ok {
				if decodeSSID(v) == ssid {
					return co, nil
				}
			}
		}
	}
	return nil, nil
}

// SetWiFiPassword updates the PSK of an existing wifi profile.
func (c *Client) SetWiFiPassword(conn *Connection, password string) error {
	settings, err := conn.GetSettings()
	if err != nil {
		return err
	}
	wireless, ok := settings["802-11-wireless"]
	if !ok {
		return errors.New("not a wireless connection")
	}
	if password != "" {
		wireless["security"] = dbus.MakeVariant("802-11-wireless-security")
		if sec := settings["802-11-wireless-security"]; sec == nil {
			settings["802-11-wireless-security"] = map[string]dbus.Variant{
				"key-mgmt": dbus.MakeVariant("wpa-psk"),
			}
		}
		settings["802-11-wireless-security"]["psk"] = dbus.MakeVariant(password)
	}
	return conn.Update(settings)
}

// deviceConnectedSSID reports whether the device is already on the network.
func (c *Client) deviceConnectedSSID(dev *Device, ssid string) (bool, error) {
	acPath, err := dev.ActiveConnection()
	if err != nil || acPath == "/" {
		return false, nil
	}
	v, err := c.conn.Object(DBusService, acPath).GetProperty(ActiveConnIf + ".Connection")
	if err != nil {
		return false, nil
	}
	conPath, ok := v.Value().(dbus.ObjectPath)
	if !ok || conPath == "/" {
		return false, nil
	}
	con := c.connAt(conPath)
	s, err := con.GetSettings()
	if err != nil {
		return false, nil
	}
	if w := s["802-11-wireless"]; w != nil {
		if v, ok := w["ssid"].Value().([]byte); ok {
			return decodeSSID(v) == ssid, nil
		}
	}
	return false, nil
}

// ConnectWiFi connects a wifi device to a network, choosing between an
// existing profile (with optional PSK update) and creating a new one.
// It blocks until DHCP/IP config completes, the attempt fails, or timeout.
func (c *Client) ConnectWiFi(dev *Device, ssid, password string, timeout time.Duration) (*Connection, *ActiveConnection, error) {
	ok, err := c.deviceConnectedSSID(dev, ssid)
	if err != nil {
		return nil, nil, err
	}
	if ok {
		return nil, nil, nil // already connected
	}

	existing, err := c.FindWiFiConnection(ssid)
	if err != nil {
		return nil, nil, err
	}

	var conn *Connection
	var ac *ActiveConnection
	if existing != nil {
		if password != "" {
			if err := c.SetWiFiPassword(existing, password); err != nil {
				return nil, nil, err
			}
		}
		conn = existing
		ac, err = c.ActivateConnection(conn, dev, "/")
	} else {
		settings, err := BuildWiFiSettings(ssid, password)
		if err != nil {
			return nil, nil, err
		}
		conn, ac, err = c.AddAndActivateConnection(settings, dev, "/")
	}
	if err != nil {
		return nil, nil, err
	}

	state, reason, err := c.waitActivation(ac, dev, timeout)
	if err != nil {
		return conn, ac, classifyFailure(state, reason)
	}
	return conn, ac, nil
}

// waitActivation watches the active connection and device until activation
// succeeds or fails. Returns the final (state, reason).
func (c *Client) waitActivation(ac *ActiveConnection, dev *Device, timeout time.Duration) (uint32, uint32, error) {
	acCh, acDone, err := c.Watch(
		dbus.WithMatchInterface(ActiveConnIf),
		dbus.WithMatchMember("StateChanged"),
		dbus.WithMatchObjectPath(ac.path),
	)
	if err != nil {
		return 0, 0, err
	}
	defer acDone()

	devCh, devDone, err := c.Watch(
		dbus.WithMatchInterface(DeviceIf),
		dbus.WithMatchMember("StateChanged"),
		dbus.WithMatchObjectPath(dev.path),
	)
	if err != nil {
		return 0, 0, err
	}
	defer devDone()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	// Poll AC state too: activation may start off with a short delay and a
	// state query avoids missing the signal race window.
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case s := <-acCh:
			if s.Path != ac.path {
				continue
			}
			var state, reason uint32
			if len(s.Body) > 0 {
				state, _ = s.Body[0].(uint32)
			}
			if len(s.Body) > 1 {
				reason, _ = s.Body[1].(uint32)
			}
			switch state {
			case ACStateActivated:
				return ACStateActivated, reason, nil
			case ACStateDeactivated:
				return ACStateDeactivated, reason, nil
			case ACStateDeactivating:
				// wait for terminal state
			}
		case s := <-devCh:
			if s.Path != dev.path {
				continue
			}
			var state, reason uint32
			if len(s.Body) > 0 {
				state, _ = s.Body[0].(uint32)
			}
			if len(s.Body) > 1 {
				reason, _ = s.Body[1].(uint32)
			}
			switch state {
			case DeviceStateActivated:
				return ACStateActivated, reason, nil
			case DeviceStateFailed:
				return ACStateDeactivated, reason, nil
			}
		case <-ticker.C:
			st, err := ac.State()
			if err == nil {
				switch st {
				case ACStateActivated:
					return ACStateActivated, 0, nil
				case ACStateDeactivated:
					reason, _, _ := ac.StateReason()
					return ACStateDeactivated, reason, nil
				}
			}
		case <-timeoutTimer.C:
			return 0, 0, ErrConnectTimeout
		}
	}
}

// classifyFailure converts raw state/reason codes into an actionable error.
func classifyFailure(state, reason uint32) error {
	switch reason {
	case 9, 10: // NO_SECRETS, LOGIN_FAILED
		return ErrAuthFailed
	case 8: // 8021X_FAILED
		return ErrAuthFailed
	case 6: // CONNECT_TIMEOUT
		return ErrConnectTimeout
	}
	switch state {
	case ACStateDeactivated:
		if reason == 0 {
			return ErrConnectTimeout
		}
		return fmt.Errorf("connection failed (reason %d)", reason)
	default:
		return ErrDeviceFailed
	}
}

// DisconnectWiFi deactivates the device's active connection.
func (c *Client) DisconnectWiFi(dev *Device) error {
	return dev.Disconnect()
}
