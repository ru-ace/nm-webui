package nm

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
)

// State returns the global NetworkManager state.
func (c *Client) State() (uint32, error) { return c.propUint32(c.nm, NmIfName, "State") }

// NmStateMachineName maps NM state machine states to text.
func NmStateMachineName(s uint32) string {
	switch s {
	case 0:
		return "unknown"
	case 10:
		return "asleep"
	case 20:
		return "disconnected"
	case 30:
		return "disconnecting"
	case 40:
		return "connecting"
	case 50:
		return "connected-local"
	case 60:
		return "connected-site"
	case 70:
		return "connected-global"
	default:
		return "unknown"
	}
}

// Connectivity returns the current internet connectivity state.
func (c *Client) Connectivity() (uint32, error) { return c.propUint32(c.nm, NmIfName, "Connectivity") }

// Hostname returns the system hostname as NM knows it. Newer NM versions
// (>= 1.46) moved hostname handling onto a dedicated D-Bus object and removed
// the root Hostname property, so fall back to os.Hostname() when unavailable.
func (c *Client) Hostname() (string, error) {
	host, err := c.propString(c.nm, NmIfName, "Hostname")
	if err == nil && host != "" {
		return host, nil
	}
	obj := c.conn.Object(DBusService, "/org/freedesktop/NetworkManager/Hostname")
	if host, err = c.propString(obj, "org.freedesktop.NetworkManager.Hostname", "Hostname"); err == nil && host != "" {
		return host, nil
	}
	return os.Hostname()
}

// Version returns the NetworkManager daemon version.
func (c *Client) Version() (string, error) { return c.propString(c.nm, NmIfName, "Version") }

// NetworkingEnabled reports whether networking is enabled at all.
func (c *Client) NetworkingEnabled() (bool, error) {
	return c.propBool(c.nm, NmIfName, "NetworkingEnabled")
}

// SetNetworking enables or disables networking globally.
func (c *Client) SetNetworking(enabled bool) error {
	return c.nm.Call(NmIfName+".Enable", 0, enabled).Err
}

// CheckConnectivity forces NM to re-check connectivity.
func (c *Client) CheckConnectivity() (uint32, error) {
	var state uint32
	if err := c.nm.Call(NmIfName+".CheckConnectivity", 0).Store(&state); err != nil {
		return 0, err
	}
	return state, nil
}

// Status gathers a summary of the whole system.
func (c *Client) Status() (NmState, error) {
	var st NmState
	var err error
	for _, get := range []func() error{
		func() error { st.State, err = c.State(); return err },
		func() error { st.Connectivity, err = c.Connectivity(); return err },
		func() error { st.Hostname, err = c.Hostname(); return err },
		func() error { st.NMVersion, err = c.Version(); return err },
		func() error { st.Enable, err = c.NetworkingEnabled(); return err },
	} {
		if e := get(); e != nil {
			return st, fmt.Errorf("networkmanager status: %w", e)
		}
	}
	return st, nil
}

// WatchManagerEvents subscribes to root-manager signals (DeviceAdded/Removed,
// ConnectivityChanged, StateChanged).
func (c *Client) WatchManagerEvents() (<-chan *dbus.Signal, func(), error) {
	return c.Watch(dbus.WithMatchInterface(NmIfName))
}