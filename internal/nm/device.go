package nm

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/godbus/dbus/v5"
)

// Device is a thin wrapper over a org.freedesktop.NetworkManager.Device object.
type Device struct {
	c    *Client
	path dbus.ObjectPath
	obj  dbus.BusObject
}

func (d *Device) Path() dbus.ObjectPath { return d.path }

// GetDevices returns devices reachable by the D-Bus client.
func (c *Client) GetDevices() ([]*Device, error) {
	var paths []dbus.ObjectPath
	if err := c.nm.Call(NmIfName+".GetDevices", 0).Store(&paths); err != nil {
		return nil, err
	}
	return c.devicesFromPaths(paths)
}

// GetAllDevices returns *all* devices including unmanaged/loopback etc.
func (c *Client) GetAllDevices() ([]*Device, error) {
	var paths []dbus.ObjectPath
	if err := c.nm.Call(NmIfName+".GetAllDevices", 0).Store(&paths); err != nil {
		return nil, err
	}
	return c.devicesFromPaths(paths)
}

func (c *Client) devicesFromPaths(paths []dbus.ObjectPath) ([]*Device, error) {
	devs := make([]*Device, 0, len(paths))
	for _, p := range paths {
		devs = append(devs, &Device{c: c, path: p, obj: c.conn.Object(DBusService, p)})
	}
	return devs, nil
}

// DeviceByInterface finds a device by its interface name.
func (c *Client) DeviceByInterface(iface string) (*Device, error) {
	var path dbus.ObjectPath
	err := c.nm.Call(NmIfName+".GetDeviceByIpIface", 0, iface).Store(&path)
	if err != nil {
		return nil, err
	}
	return &Device{c: c, path: path, obj: c.conn.Object(DBusService, path)}, nil
}

// DeviceFromPath wraps a device object by its D-Bus path.
func (c *Client) DeviceFromPath(path dbus.ObjectPath) (*Device, error) {
	return &Device{c: c, path: path, obj: c.conn.Object(DBusService, path)}, nil
}

// WiFiDevices returns devices of type wifi.
func (c *Client) WiFiDevices() ([]*Device, error) {
	devs, err := c.GetAllDevices()
	if err != nil {
		return nil, err
	}
	out := make([]*Device, 0, len(devs))
	for _, d := range devs {
		t, err := d.Type()
		if err == nil && t == DeviceTypeWiFi {
			out = append(out, d)
		}
	}
	return out, nil
}

// Interface returns the device interface name (IpInterface).
func (d *Device) Interface() (string, error) { return d.c.propString(d.obj, DeviceIf, "Interface") }

// IpInterface returns IpInterface (may differ from Interface for VLANs).
func (d *Device) IpInterface() (string, error) { return d.c.propString(d.obj, DeviceIf, "IpInterface") }

// Type returns the device type id.
func (d *Device) Type() (uint32, error) { return d.c.propUint32(d.obj, DeviceIf, "DeviceType") }

// Driver returns the kernel driver name.
func (d *Device) Driver() (string, error) { return d.c.propString(d.obj, DeviceIf, "Driver") }

// State returns the device state.
func (d *Device) State() (uint32, error) { return d.c.propUint32(d.obj, DeviceIf, "State") }

// Reason returns the state reason.
func (d *Device) StateReason() (uint32, uint32, error) {
	v, err := d.obj.GetProperty(DeviceIf + ".StateReason")
	if err != nil {
		return 0, 0, nil
	}
	t, ok := v.Value().([]uint32)
	if !ok {
		return 0, 0, fmt.Errorf("StateReason: unexpected type %T", v.Value())
	}
	if len(t) != 2 {
		return 0, 0, nil
	}
	return t[0], t[1], nil
}

// MAC returns the hardware address.
func (d *Device) MAC() (string, error) { return d.c.propString(d.obj, DeviceIf, "HwAddress") }

// MTU returns the interface MTU.
func (d *Device) MTU() (uint32, error) { return d.c.propUint32(d.obj, DeviceIf, "Mtu") }

// Managed reports whether NM manages this device.
func (d *Device) Managed() (bool, error) { return d.c.propBool(d.obj, DeviceIf, "Managed") }

// ActiveConnection returns the path of the active connection, "/" if none.
func (d *Device) ActiveConnection() (dbus.ObjectPath, error) {
	return d.c.propObjectPath(d.obj, DeviceIf, "ActiveConnection")
}

// Disconnect deactivates the active connection on the device.
func (d *Device) Disconnect() error {
	acPath, err := d.ActiveConnection()
	if err != nil {
		return err
	}
	if acPath == "/" {
		return nil
	}
	if err := d.c.nm.Call(NmIfName+".DeactivateConnection", 0, acPath).Err; err != nil {
		return err
	}
	return nil
}

// Address is a parsed IP address.
type Address struct {
	Address string `json:"address"`
	Prefix  uint32 `json:"prefix"`
	Gateway string `json:"gateway,omitempty"`
}

// IPConfig holds parsed IPv4/IPv6 configuration.
type IPConfig struct {
	Addresses  []Address `json:"addresses,omitempty"`
	Gateway    string    `json:"gateway,omitempty"`
	Nameservers []string `json:"nameservers,omitempty"`
	Domains    []string  `json:"domains,omitempty"`
}

// IP4Config reads and parses the device's IPv4 configuration.
func (d *Device) IP4Config() (IPConfig, error) {
	var cfg IPConfig
	p, err := d.c.propObjectPath(d.obj, DeviceIf, "Ip4Config")
	if err != nil || p == "/" {
		return cfg, err
	}
	obj := d.c.conn.Object(DBusService, p)

	// Try AddressData (aa{sv}) first
	if v, err := obj.GetProperty(IP4ConfigIf + ".AddressData"); err == nil {
		if arr, ok := v.Value().([]map[string]dbus.Variant); ok {
			for _, m := range arr {
				var addr Address
				if s, ok := m["address"].Value().(string); ok {
					addr.Address = s
				}
				if pref, ok := m["prefix"].Value().(uint32); ok {
					addr.Prefix = pref
				}
				if addr.Address != "" {
					cfg.Addresses = append(cfg.Addresses, addr)
				}
			}
		}
	}

	// Fall back to legacy Addresses (aau) if AddressData was empty
	if len(cfg.Addresses) == 0 {
		if v, err := obj.GetProperty(IP4ConfigIf + ".Addresses"); err == nil {
			if arr, ok := v.Value().([][]uint32); ok {
				for _, a := range arr {
					var addr Address
					if len(a) >= 1 {
						addr.Address = u32ToIP(a[0])
					}
					if len(a) >= 2 {
						addr.Prefix = a[1]
					}
					if len(a) >= 3 && a[2] != 0 {
						addr.Gateway = u32ToIP(a[2])
					}
					if addr.Address != "" {
						cfg.Addresses = append(cfg.Addresses, addr)
					}
				}
			}
		}
	}

	if v, err := obj.GetProperty(IP4ConfigIf + ".Gateway"); err == nil {
		if s, ok := v.Value().(string); ok {
			cfg.Gateway = s
		}
	}
	if v, err := obj.GetProperty(IP4ConfigIf + ".Nameservers"); err == nil {
		if arr, ok := v.Value().([]uint32); ok {
			for _, u := range arr {
				cfg.Nameservers = append(cfg.Nameservers, u32ToIP(u))
			}
		}
	}
	if v, err := obj.GetProperty(IP4ConfigIf + ".Domains"); err == nil {
		if arr, ok := v.Value().([]string); ok {
			cfg.Domains = arr
		}
	}
	return cfg, nil
}

// IP6Config reads and parses the device's IPv6 configuration.
func (d *Device) IP6Config() (IPConfig, error) {
	var cfg IPConfig
	p, err := d.c.propObjectPath(d.obj, DeviceIf, "Ip6Config")
	if err != nil || p == "/" {
		return cfg, err
	}
	obj := d.c.conn.Object(DBusService, p)

	// Try AddressData (aa{sv}) first
	if v, err := obj.GetProperty(IP6ConfigIf + ".AddressData"); err == nil {
		if arr, ok := v.Value().([]map[string]dbus.Variant); ok {
			for _, m := range arr {
				var addr Address
				if s, ok := m["address"].Value().(string); ok {
					addr.Address = s
				}
				if pref, ok := m["prefix"].Value().(uint32); ok {
					addr.Prefix = pref
				}
				if addr.Address != "" {
					cfg.Addresses = append(cfg.Addresses, addr)
				}
			}
		}
	}

	// Fall back to legacy Addresses (a(ayuay)) if AddressData was empty
	if len(cfg.Addresses) == 0 {
		if v, err := obj.GetProperty(IP6ConfigIf + ".Addresses"); err == nil {
			if arr, ok := v.Value().([]interface{}); ok {
				for _, item := range arr {
					inner, ok2 := item.([]interface{})
					if !ok2 {
						continue
					}
					var addr Address
					for i, field := range inner {
						switch i {
						case 0:
							if b, ok := field.([]byte); ok && len(b) == 16 {
								addr.Address = net.IP(b).String()
							}
						case 1:
							switch p := field.(type) {
							case uint32:
								addr.Prefix = p
							case []byte:
								if len(p) > 0 {
									addr.Prefix = uint32(p[0])
								}
							}
						case 2:
							if b, ok := field.([]byte); ok && len(b) == 16 {
								addr.Gateway = net.IP(b).String()
							}
						}
					}
					if addr.Address != "" {
						cfg.Addresses = append(cfg.Addresses, addr)
					}
				}
			}
		}
	}
	if v, err := obj.GetProperty(IP6ConfigIf + ".Gateway"); err == nil {
		if s, ok := v.Value().(string); ok {
			cfg.Gateway = s
		}
	}
	if v, err := obj.GetProperty(IP6ConfigIf + ".Nameservers"); err == nil {
		switch arr := v.Value().(type) {
		case [][]byte:
			for _, b := range arr {
				if len(b) == 16 {
					cfg.Nameservers = append(cfg.Nameservers, net.IP(b).String())
				}
			}
		case []interface{}:
			for _, item := range arr {
				if b, ok2 := item.([]byte); ok2 && len(b) == 16 {
					cfg.Nameservers = append(cfg.Nameservers, net.IP(b).String())
				}
			}
		}
	}
	if v, err := obj.GetProperty(IP6ConfigIf + ".Domains"); err == nil {
		if arr, ok := v.Value().([]string); ok {
			cfg.Domains = arr
		}
	}
	return cfg, nil
}

// DeviceInfo is the JSON-friendly view of a device for the API.
type DeviceInfo struct {
	Path             string   `json:"path"`
	Interface        string   `json:"interface"`
	Kind             string   `json:"kind"`
	TypeName         string   `json:"type_name"`
	Driver           string   `json:"driver,omitempty"`
	MAC              string   `json:"mac,omitempty"`
	MTU              uint32   `json:"mtu"`
	State            uint32   `json:"state"`
	StateName        string   `json:"state_name"`
	Managed          bool     `json:"managed"`
	IPv4             IPConfig `json:"ipv4,omitempty"`
	IPv6             IPConfig `json:"ipv6,omitempty"`
	ActiveConnection string   `json:"active_connection,omitempty"`
	Wireless         bool     `json:"wireless"`
	AutoConnect      bool     `json:"autoconnect,omitempty"`
}

// Info collects the full device information.
func (d *Device) Info() (DeviceInfo, error) {
	info := DeviceInfo{
		Path:     string(d.Path()),
		Wireless: false,
	}
	if s, err := d.IpInterface(); err == nil {
		info.Interface = s
	} else if s, err := d.Interface(); err == nil {
		info.Interface = s
	}
	if t, err := d.Type(); err == nil {
		info.TypeName = DeviceTypeName(t)
		info.Kind = info.TypeName
		if t == DeviceTypeWiFi {
			info.Wireless = true
		}
	}
	info.Driver, _ = d.Driver()
	info.MAC, _ = d.MAC()
	info.MTU, _ = d.MTU()
	info.State, _ = d.State()
	info.StateName = DeviceStateName(info.State)
	info.Managed, _ = d.Managed()
	if s, err := d.ActiveConnection(); err == nil && s != "/" {
		info.ActiveConnection = string(s)
	}
	info.IPv4, _ = d.IP4Config()
	info.IPv6, _ = d.IP6Config()
	return info, nil
}

// PropertiesChangedResult is an opaque event payload (kept for tests).
type PropertiesChangedResult = map[string]dbus.Variant

var _ = binary.BigEndian