package nm

import (
	"net"

	"github.com/godbus/dbus/v5"
)

// ActivateProfile activates a stored profile. If the profile declares an
// interface name it is used, otherwise NM picks a compatible device.
func (c *Client) ActivateProfile(uuid string) (*ActiveConnection, error) {
	conn, err := c.ConnectionByUUID(uuid)
	if err != nil {
		return nil, err
	}
	settings, err := conn.GetSettings()
	if err != nil {
		return nil, err
	}
	var dev *Device
	if iface := variantString(settings["connection"], "interface-name"); iface != "" {
		dev, err = c.DeviceByInterface(iface)
		if err != nil {
			dev = nil
		}
	}
	return c.ActivateConnection(conn, dev, "/")
}

// ActiveRef describes which device a profile is active on.
type ActiveRef struct {
	ACP   dbus.ObjectPath
	Iface string
}

// ActiveUUIDs returns the active connection reference for each profile uuid.
func (c *Client) ActiveUUIDs() (map[string]ActiveRef, error) {
	actives := map[string]ActiveRef{}
	acPaths, err := c.activeConnectionPaths()
	if err != nil {
		return actives, err
	}
	for _, p := range acPaths {
		obj := c.conn.Object(DBusService, p)
		v, err := obj.GetProperty(ActiveConnIf + ".Connection")
		if err != nil {
			continue
		}
		conPath, ok := v.Value().(dbus.ObjectPath)
		if !ok || conPath == "/" {
			continue
		}
		settings, err := c.connAt(conPath).GetSettings()
		if err != nil {
			continue
		}
		uuid := variantString(settings["connection"], "uuid")
		if uuid == "" {
			continue
		}
		ref := ActiveRef{ACP: p}
		if devs, err := obj.GetProperty(ActiveConnIf + ".Devices"); err == nil {
			if paths, ok := devs.Value().([]dbus.ObjectPath); ok && len(paths) > 0 {
				if d, err := c.DeviceFromPath(paths[0]); err == nil {
					ref.Iface, _ = d.IpInterface()
				}
			}
		}
		actives[uuid] = ref
	}
	return actives, nil
}

func (c *Client) activeConnectionPaths() ([]dbus.ObjectPath, error) {
	v, err := c.nm.GetProperty(NmIfName + ".ActiveConnections")
	if err != nil {
		return nil, err
	}
	paths, ok := v.Value().([]dbus.ObjectPath)
	if !ok {
		return nil, nil
	}
	return paths, nil
}

// ActiveForUUID returns the active connection instance of a profile, if any.
func (c *Client) ActiveForUUID(uuid string) (*ActiveConnection, error) {
	devs, err := c.GetAllDevices()
	if err != nil {
		return nil, err
	}
	for _, d := range devs {
		acPath, err := d.ActiveConnection()
		if err != nil || acPath == "/" {
			continue
		}
		obj := c.conn.Object(DBusService, acPath)
		v, err := obj.GetProperty(ActiveConnIf + ".Connection")
		if err != nil {
			continue
		}
		conPath, ok := v.Value().(dbus.ObjectPath)
		if !ok || conPath == "/" {
			continue
		}
		settings, err := c.connAt(conPath).GetSettings()
		if err != nil {
			continue
		}
		if variantString(settings["connection"], "uuid") == uuid {
			return &ActiveConnection{c: c, path: acPath, obj: obj}, nil
		}
	}
	return nil, ErrNoActiveConn
}

// DeactivateProfile deactivates the active connection of a profile, if any.
func (c *Client) DeactivateProfile(uuid string) error {
	ac, err := c.ActiveForUUID(uuid)
	if err != nil {
		return err
	}
	return ac.Deactivate()
}

// DeleteProfile removes a connection profile.
func (c *Client) DeleteProfile(uuid string) error {
	conn, err := c.ConnectionByUUID(uuid)
	if err != nil {
		return err
	}
	return conn.Delete()
}

// UpdateProfileIPv4 applies an IPv4 configuration (auto/manual/disabled) to a
// profile keeping all other settings untouched.
func (c *Client) UpdateProfileIPv4(uuid string, block map[string]dbus.Variant) error {
	return c.updateProfileSettings(uuid, func(s map[string]map[string]dbus.Variant) {
		mergeIPSettings(s, "ipv4", block)
	})
}

// UpdateProfileIPv6 applies an IPv6 configuration to a profile.
func (c *Client) UpdateProfileIPv6(uuid string, block map[string]dbus.Variant) error {
	return c.updateProfileSettings(uuid, func(s map[string]map[string]dbus.Variant) {
		mergeIPSettings(s, "ipv6", block)
	})
}

func mergeIPSettings(settings map[string]map[string]dbus.Variant, family string, block map[string]dbus.Variant) {
	current := settings[family]
	if current == nil {
		current = map[string]dbus.Variant{}
		settings[family] = current
	}
	for _, key := range []string{"method", "dhcp-timeout", "address-data", "addresses", "gateway", "dns", "ignore-auto-dns"} {
		delete(current, key)
	}
	for key, value := range block {
		current[key] = value
	}
}

// UpdateProfile autoconnect flips the autoconnect flag.
func (c *Client) SetProfileAutoconnect(uuid string, enabled bool) error {
	return c.updateProfileSettings(uuid, func(s map[string]map[string]dbus.Variant) {
		if m := s["connection"]; m != nil {
			m["autoconnect"] = dbus.MakeVariant(enabled)
		}
	})
}

func (c *Client) updateProfileSettings(uuid string, mutate func(s map[string]map[string]dbus.Variant)) error {
	conn, err := c.ConnectionByUUID(uuid)
	if err != nil {
		return err
	}
	settings, err := conn.GetSettings()
	if err != nil {
		return err
	}
	mutate(settings)
	return conn.Update(settings)
}

// IPv4Block builds an ipv4 settings block from method and optional static values.
func IPv4Block(method string, addr StaticConfig) map[string]dbus.Variant {
	block := map[string]dbus.Variant{
		"method": dbus.MakeVariant(method),
	}
	switch method {
	case "auto":
		block["dhcp-timeout"] = dbus.MakeVariant(int32(30))
		// make sure stale static values are cleared
		block["address-data"] = dbus.MakeVariant([]map[string]dbus.Variant{})
		block["gateway"] = dbus.MakeVariant("")
		block["dns"] = dbus.MakeVariant([]uint32{})
	case "manual":
		if addr.Address != "" {
			block["address-data"] = dbus.MakeVariant([]map[string]dbus.Variant{
				{
					"address": dbus.MakeVariant(addr.Address),
					"prefix":  dbus.MakeVariant(addr.Prefix),
				},
			})
			if addr.Gateway != "" {
				block["gateway"] = dbus.MakeVariant(addr.Gateway)
			}
		}
		if len(addr.DNS) > 0 {
			block["dns"] = dbus.MakeVariant(parseDNS4(addr.DNS))
			block["ignore-auto-dns"] = dbus.MakeVariant(true)
		}
	case "disabled":
		block["method"] = dbus.MakeVariant("disabled")
		block["address-data"] = dbus.MakeVariant([]map[string]dbus.Variant{})
		block["gateway"] = dbus.MakeVariant("")
		block["dns"] = dbus.MakeVariant([]uint32{})
	}
	return block
}

// IPv6Block builds an ipv6 settings block.
func IPv6Block(method string, addr StaticConfig) map[string]dbus.Variant {
	block := map[string]dbus.Variant{
		"method": dbus.MakeVariant(method),
	}
	switch method {
	case "auto":
		block["dhcp-timeout"] = dbus.MakeVariant(int32(30))
		block["address-data"] = dbus.MakeVariant([]map[string]dbus.Variant{})
		block["gateway"] = dbus.MakeVariant("")
		block["dns"] = dbus.MakeVariant([][]byte{})
	case "manual":
		if addr.Address != "" {
			block["address-data"] = dbus.MakeVariant([]map[string]dbus.Variant{
				{
					"address": dbus.MakeVariant(addr.Address),
					"prefix":  dbus.MakeVariant(addr.Prefix),
				},
			})
			if addr.Gateway != "" {
				block["gateway"] = dbus.MakeVariant(addr.Gateway)
			}
		}
		if len(addr.DNS) > 0 {
			block["dns"] = dbus.MakeVariant(ipv6DNS(addr.DNS))
			block["ignore-auto-dns"] = dbus.MakeVariant(true)
		}
	case "disabled", "ignore":
		block["method"] = dbus.MakeVariant("disabled")
		block["address-data"] = dbus.MakeVariant([]map[string]dbus.Variant{})
		block["gateway"] = dbus.MakeVariant("")
		block["dns"] = dbus.MakeVariant([][]byte{})
	}
	return block
}

func ipv6DNS(list []string) [][]byte {
	out := make([][]byte, 0, len(list))
	for _, s := range list {
		ip := parseIPv6(s)
		if len(ip) == 16 {
			out = append(out, ip)
		}
	}
	return out
}

func parseIPv6(s string) []byte {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil
	}
	return ip.To16()
}
