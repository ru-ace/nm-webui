package nm

import (
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"
)

// Client is a thin, typed D-Bus client for NetworkManager built directly on
// github.com/godbus/dbus/v5. It talks to the system bus under service
// org.freedesktop.NetworkManager without shelling out to nmcli/ip.
type Client struct {
	conn  *dbus.Conn
	nm    dbus.BusObject
	mu    sync.Mutex
	watch map[chan *dbus.Signal]struct{}
	done  chan struct{}
}

// Connect opens the system D-Bus connection and prepares NetworkManager object.
func Connect() (*Client, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("system bus: %w", err)
	}
	c := &Client{
		conn:  conn,
		nm:    conn.Object(DBusService, NmPath),
		watch: map[chan *dbus.Signal]struct{}{},
		done:  make(chan struct{}),
	}
	return c, nil
}

// Close releases the D-Bus connection.
func (c *Client) Close() error {
	close(c.done)
	return c.conn.Close()
}

func (c *Client) Nm() dbus.BusObject   { return c.nm }
func (c *Client) Conn() *dbus.Conn     { return c.conn }

func (c *Client) settings() dbus.BusObject {
	return c.conn.Object(DBusService, SettingsPath)
}

// Property reads a D-Bus property from an object on a given interface.
func (c *Client) Property(obj dbus.BusObject, iface, name string) (dbus.Variant, error) {
	return obj.GetProperty(iface + "." + name)
}

func (c *Client) propUint32(obj dbus.BusObject, iface, name string) (uint32, error) {
	v, err := obj.GetProperty(iface + "." + name)
	if err != nil {
		return 0, err
	}
	n, ok := v.Value().(uint32)
	if !ok {
		return 0, fmt.Errorf("property %s.%s: unexpected type %T", iface, name, v.Value())
	}
	return n, nil
}

func (c *Client) propString(obj dbus.BusObject, iface, name string) (string, error) {
	v, err := obj.GetProperty(iface + "." + name)
	if err != nil {
		return "", err
	}
	s, ok := v.Value().(string)
	if !ok {
		return "", fmt.Errorf("property %s.%s: unexpected type %T", iface, name, v.Value())
	}
	return s, nil
}

func (c *Client) propBool(obj dbus.BusObject, iface, name string) (bool, error) {
	v, err := obj.GetProperty(iface + "." + name)
	if err != nil {
		return false, err
	}
	b, ok := v.Value().(bool)
	if !ok {
		return false, fmt.Errorf("property %s.%s: unexpected type %T", iface, name, v.Value())
	}
	return b, nil
}

func (c *Client) propObjectPath(obj dbus.BusObject, iface, name string) (dbus.ObjectPath, error) {
	v, err := obj.GetProperty(iface + "." + name)
	if err != nil {
		return "", err
	}
	p, ok := v.Value().(dbus.ObjectPath)
	if !ok {
		return "", fmt.Errorf("property %s.%s: unexpected type %T", iface, name, v.Value())
	}
	return p, nil
}

// Watch subscribes to D-Bus signals matching the given match options and
// returns a channel that receives them. Call the returned cleanup function to
// stop delivery.
func (c *Client) Watch(opts ...dbus.MatchOption) (<-chan *dbus.Signal, func(), error) {
	return c.WatchMany(opts)
}

// WatchMany subscribes to several independent match rules using a single
// receipt channel (godbus fans all deliveries out to every registered channel,
// so the caller dispatches by signal name).
func (c *Client) WatchMany(sets ...[]dbus.MatchOption) (<-chan *dbus.Signal, func(), error) {
	for _, opts := range sets {
		if err := c.conn.AddMatchSignal(opts...); err != nil {
			return nil, nil, fmt.Errorf("add match: %w", err)
		}
	}
	ch := make(chan *dbus.Signal, 64)
	c.conn.Signal(ch)
	c.mu.Lock()
	c.watch[ch] = struct{}{}
	c.mu.Unlock()

	cleanup := func() {
		for _, opts := range sets {
			_ = c.conn.RemoveMatchSignal(opts...)
		}
		c.conn.RemoveSignal(ch)
		c.mu.Lock()
		if _, ok := c.watch[ch]; ok {
			delete(c.watch, ch)
			c.mu.Unlock()
			close(ch)
		} else {
			c.mu.Unlock()
		}
	}
	return ch, cleanup, nil
}

// ListWatched returns the set of registered signal channels (for tests).
func (c *Client) ListWatched() map[chan *dbus.Signal]struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.watch
}