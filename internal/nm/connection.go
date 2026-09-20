package nm

import (
	"fmt"
	"net"

	"github.com/godbus/dbus/v5"
)

// Connection wraps a org.freedesktop.NetworkManager.Settings.Connection.
type Connection struct {
	c    *Client
	path dbus.ObjectPath
	obj  dbus.BusObject
}

// AvailableConnections returns connection profiles applicable for a device.
func (d *Device) AvailableConnections() ([]*Connection, error) {
	var paths []dbus.ObjectPath
	v, err := d.obj.GetProperty(DeviceIf + ".AvailableConnections")
	if err != nil {
		return nil, err
	}
	paths, ok := v.Value().([]dbus.ObjectPath)
	if !ok {
		return nil, fmt.Errorf("AvailableConnections: unexpected type %T", v.Value())
	}
	out := make([]*Connection, 0, len(paths))
	for _, p := range paths {
		out = append(out, d.c.connAt(p))
	}
	return out, nil
}

func (c *Client) connAt(p dbus.ObjectPath) *Connection {
	return &Connection{c: c, path: p, obj: c.conn.Object(DBusService, p)}
}

// ListConnections returns all stored connection profiles.
func (c *Client) ListConnections() ([]*Connection, error) {
	var paths []dbus.ObjectPath
	if err := c.settings().Call(SettingsIfName+".ListConnections", 0).Store(&paths); err != nil {
		return nil, err
	}
	out := make([]*Connection, 0, len(paths))
	for _, p := range paths {
		out = append(out, c.connAt(p))
	}
	return out, nil
}

// ConnectionByUUID looks up a connection profile by uuid.
func (c *Client) ConnectionByUUID(uuid string) (*Connection, error) {
	var p dbus.ObjectPath
	if err := c.settings().Call(SettingsIfName+".GetConnectionByUuid", 0, uuid).Store(&p); err != nil {
		return nil, err
	}
	return c.connAt(p), nil
}

// AddConnection creates a new connection profile.
func (c *Client) AddConnection(settings map[string]map[string]dbus.Variant) (*Connection, error) {
	var p dbus.ObjectPath
	if err := c.settings().Call(SettingsIfName+".AddConnection", 0, settings).Store(&p); err != nil {
		return nil, err
	}
	return c.connAt(p), nil
}

func (co *Connection) Path() dbus.ObjectPath { return co.path }

// GetSettings returns the full connection settings block.
func (co *Connection) GetSettings() (map[string]map[string]dbus.Variant, error) {
	var settings map[string]map[string]dbus.Variant
	if err := co.obj.Call(SettingsConnIf+".GetSettings", 0).Store(&settings); err != nil {
		return nil, err
	}
	return settings, nil
}

// Update replaces the connection settings.
func (co *Connection) Update(settings map[string]map[string]dbus.Variant) error {
	sanitizeIPBlocks(settings)
	return co.obj.Call(SettingsConnIf+".Update", 0, settings).Err
}

// sanitizeIPBlocks normalizes legacy raw address/route arrays (aau for IPv4
// and a(ayuay) for IPv6) that godbus decodes as aav back into the
// address-data/route-data form NM accepts on Update. Without this NM rejects
// the round-trip with "can't set property of type 'a(ayuayu)' from value of
// type 'aav'" when editing any saved profile.
func sanitizeIPBlocks(settings map[string]map[string]dbus.Variant) {
	for _, family := range []string{"ipv4", "ipv6"} {
		block := settings[family]
		if block == nil {
			continue
		}
		if _, hasData := block["address-data"]; !hasData {
			if v, ok := block["addresses"]; ok {
				if data := legacyToAddressData(family, v.Value()); data != nil {
					block["address-data"] = dbus.MakeVariant(data)
				}
			}
		}
		delete(block, "addresses")
		if _, hasData := block["route-data"]; !hasData {
			if v, ok := block["routes"]; ok {
				if data := legacyToRouteData(family, v.Value()); data != nil {
					block["route-data"] = dbus.MakeVariant(data)
				}
			}
		}
		delete(block, "routes")
		if g, ok := block["gateway"]; ok {
			if s, ok2 := g.Value().(string); ok2 && s == "" {
				delete(block, "gateway")
			}
		}
	}
}

// legacyToAddressData converts a legacy raw addresses array into address-data
// entries (aa{sv}), or nil when no usable data is present.
func legacyToAddressData(family string, v interface{}) []map[string]dbus.Variant {
	var out []map[string]dbus.Variant
	switch arr := v.(type) {
	case [][]uint32: // ipv4: aau
		for _, a := range arr {
			if len(a) < 2 {
				continue
			}
			entry := map[string]dbus.Variant{
				"address": dbus.MakeVariant(u32ToIP(a[0])),
				"prefix":  dbus.MakeVariant(a[1]),
			}
			if len(a) > 2 && a[2] != 0 {
				entry["gateway"] = dbus.MakeVariant(u32ToIP(a[2]))
			}
			out = append(out, entry)
		}
	case []interface{}: // ipv6: a(ayuay)
		for _, item := range arr {
			inner, ok := item.([]interface{})
			if !ok || len(inner) < 2 {
				continue
			}
			entry := map[string]dbus.Variant{}
			if b, ok := inner[0].([]byte); ok && len(b) == 16 {
				entry["address"] = dbus.MakeVariant(net.IP(b).String())
			}
			switch p := inner[1].(type) {
			case uint32:
				entry["prefix"] = dbus.MakeVariant(p)
			case byte:
				entry["prefix"] = dbus.MakeVariant(uint32(p))
			}
			if len(inner) > 2 {
				if b, ok := inner[2].([]byte); ok && len(b) == 16 {
					entry["gateway"] = dbus.MakeVariant(net.IP(b).String())
				}
			}
			if _, ok := entry["address"]; ok {
				out = append(out, entry)
			}
		}
	}
	return out
}

// legacyToRouteData converts a legacy raw routes array into route-data entries.
func legacyToRouteData(family string, v interface{}) []map[string]dbus.Variant {
	var out []map[string]dbus.Variant
	switch arr := v.(type) {
	case [][]uint32: // ipv4: aau
		for _, r := range arr {
			if len(r) < 2 {
				continue
			}
			entry := map[string]dbus.Variant{
				"dest":   dbus.MakeVariant(u32ToIP(r[0])),
				"prefix": dbus.MakeVariant(r[1]),
			}
			if len(r) > 2 && r[2] != 0 {
				entry["next-hop"] = dbus.MakeVariant(u32ToIP(r[2]))
			}
			if len(r) > 3 && r[3] != 0 {
				entry["metric"] = dbus.MakeVariant(r[3])
			}
			out = append(out, entry)
		}
	case []interface{}: // ipv6: a(ayuayu)
		for _, item := range arr {
			inner, ok := item.([]interface{})
			if !ok || len(inner) < 2 {
				continue
			}
			entry := map[string]dbus.Variant{}
			if b, ok := inner[0].([]byte); ok && len(b) == 16 {
				entry["dest"] = dbus.MakeVariant(net.IP(b).String())
			}
			switch p := inner[1].(type) {
			case uint32:
				entry["prefix"] = dbus.MakeVariant(p)
			case byte:
				entry["prefix"] = dbus.MakeVariant(uint32(p))
			}
			if len(inner) > 2 {
				if b, ok := inner[2].([]byte); ok && len(b) == 16 {
					entry["next-hop"] = dbus.MakeVariant(net.IP(b).String())
				}
			}
			if len(inner) > 3 {
				if m, ok := inner[3].(uint32); ok && m != 0 {
					entry["metric"] = dbus.MakeVariant(m)
				}
			}
			if _, ok := entry["dest"]; ok {
				out = append(out, entry)
			}
		}
	}
	return out
}

// Delete removes the connection profile.
func (co *Connection) Delete() error {
	return co.obj.Call(SettingsConnIf+".Delete", 0).Err
}

func variantString(m map[string]dbus.Variant, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.Value().(string)
	return s
}

func variantBool(m map[string]dbus.Variant, key string) bool {
	v, ok := m[key]
	if !ok {
		return true
	}
	b, _ := v.Value().(bool)
	return b
}

// StaticConfig is the parsed static (manual) address configuration.
type StaticConfig struct {
	Address string   `json:"address,omitempty"`
	Prefix  uint32   `json:"prefix,omitempty"`
	Gateway string   `json:"gateway,omitempty"`
	DNS     []string `json:"dns,omitempty"`
}

// ConnectionInfo is a JSON-friendly profile description.
type ConnectionInfo struct {
	Path        string        `json:"path"`
	UUID        string        `json:"uuid"`
	ID          string        `json:"id"`
	Type        string        `json:"type"`
	TypeName    string        `json:"type_name"`
	Interface   string        `json:"interface,omitempty"`
	SSID        string        `json:"ssid,omitempty"`
	APN         string        `json:"apn,omitempty"`
	Number      string        `json:"number,omitempty"`
	UserName    string        `json:"username,omitempty"`
	Autoconnect bool          `json:"autoconnect"`
	Active      bool          `json:"active"`
	Device      string        `json:"device,omitempty"`
	IPv4Method  string        `json:"ipv4_method"`
	IPv6Method  string        `json:"ipv6_method"`
	Static4     *StaticConfig `json:"static4,omitempty"`
	Static6     *StaticConfig `json:"static6,omitempty"`
	IsWiFi      bool          `json:"is_wifi"`
	IsModem     bool          `json:"is_modem"`
}

// Info parses the connection settings into ConnectionInfo.
func (co *Connection) Info() (ConnectionInfo, error) {
	info := ConnectionInfo{
		Path:       string(co.path),
		IPv4Method: "auto",
		IPv6Method: "auto",
	}
	settings, err := co.GetSettings()
	if err != nil {
		return info, err
	}

	conn := settings["connection"]
	info.UUID = variantString(conn, "uuid")
	info.ID = variantString(conn, "id")
	info.Type = variantString(conn, "type")
	switch info.Type {
	case "802-11-wireless":
		info.TypeName = "wifi"
		info.IsWiFi = true
	case "802-3-ethernet":
		info.TypeName = "ethernet"
	case "gsm":
		info.TypeName = "gsm"
		info.IsModem = true
	case "bridge":
		info.TypeName = "bridge"
	default:
		info.TypeName = info.Type
	}
	info.Interface = variantString(conn, "interface-name")
	info.Autoconnect = variantBool(conn, "autoconnect")

	if w := settings["802-11-wireless"]; w != nil {
		if v, ok := w["ssid"]; ok {
			if b, ok := v.Value().([]byte); ok {
				info.SSID = decodeSSID(b)
			}
		}
		if info.Interface == "" {
			// A profile without interface-name is valid for any compatible device.
			// The wireless MAC is not an interface name.
		}
	}

	if g := settings["gsm"]; g != nil {
		info.APN = variantString(g, "apn")
		info.Number = variantString(g, "number")
		info.UserName = variantString(g, "username")
	}

	if ip4 := settings["ipv4"]; ip4 != nil {
		info.IPv4Method = variantString(ip4, "method")
		info.Static4 = parseStatic4(ip4)
	}
	if ip6 := settings["ipv6"]; ip6 != nil {
		info.IPv6Method = variantString(ip6, "method")
		info.Static6 = parseStatic6(ip6)
	}
	return info, nil
}

func parseStatic4(s map[string]dbus.Variant) *StaticConfig {
	st := &StaticConfig{}
	if val, ok := s["address-data"]; ok {
		if v, ok := val.Value().([]map[string]dbus.Variant); ok && len(v) > 0 {
			entry := v[0]
			st.Address = variantString(entry, "address")
			if p, ok := entry["prefix"].Value().(uint32); ok {
				st.Prefix = p
			}
		}
	}
	if st.Address == "" {
		if val, ok := s["addresses"]; ok {
			if v, ok := val.Value().([][]uint32); ok && len(v) > 0 && len(v[0]) >= 2 {
				st.Address = u32ToIP(v[0][0])
				st.Prefix = v[0][1]
			}
		}
	}
	st.Gateway = variantString(s, "gateway")
	if val, ok := s["dns"]; ok {
		if v, ok := val.Value().([]uint32); ok {
			for _, u := range v {
				st.DNS = append(st.DNS, u32ToIP(u))
			}
		}
	}
	if st.Address == "" && st.Gateway == "" && len(st.DNS) == 0 {
		return nil
	}
	return st
}

func parseStatic6(s map[string]dbus.Variant) *StaticConfig {
	st := &StaticConfig{}
	if val, ok := s["address-data"]; ok {
		if v, ok := val.Value().([]map[string]dbus.Variant); ok && len(v) > 0 {
			entry := v[0]
			st.Address = variantString(entry, "address")
			if p, ok := entry["prefix"].Value().(uint32); ok {
				st.Prefix = p
			}
		}
	}
	if st.Address == "" {
		if val, ok := s["addresses"]; ok {
			if v, ok := val.Value().([]interface{}); ok && len(v) > 0 {
				if inner, ok := v[0].([]interface{}); ok && len(inner) >= 2 {
					if b, ok := inner[0].([]byte); ok {
						st.Address = decodeIP6(b)
					}
					switch p := inner[1].(type) {
					case uint32:
						st.Prefix = p
					case []byte:
						if len(p) > 0 {
							st.Prefix = uint32(p[0])
						}
					}
				}
			}
		}
	}
	st.Gateway = variantString(s, "gateway")
	if val, ok := s["dns"]; ok {
		switch v := val.Value().(type) {
		case [][]byte:
			for _, b := range v {
				if len(b) == 16 {
					st.DNS = append(st.DNS, decodeIP6(b))
				}
			}
		case []interface{}:
			for _, item := range v {
				if b, ok := item.([]byte); ok && len(b) == 16 {
					st.DNS = append(st.DNS, decodeIP6(b))
				}
			}
		}
	}
	if st.Address == "" && st.Gateway == "" && len(st.DNS) == 0 {
		return nil
	}
	return st
}

func decodeIP6(b []byte) string {
	return net.IP(b).String()
}

// ActiveConnection wraps org.freedesktop.NetworkManager.Connection.Active.
type ActiveConnection struct {
	c    *Client
	path dbus.ObjectPath
	obj  dbus.BusObject
}

// ActivateConnection asks NM to activate a profile on a device.
func (c *Client) ActivateConnection(conn *Connection, dev *Device, specific dbus.ObjectPath) (*ActiveConnection, error) {
	devPath := dbus.ObjectPath("/")
	if dev != nil {
		devPath = dev.Path()
	}
	var acPath dbus.ObjectPath
	if err := c.nm.Call(NmIfName+".ActivateConnection", 0, conn.path, devPath, specific).Store(&acPath); err != nil {
		return nil, err
	}
	return &ActiveConnection{c: c, path: acPath, obj: c.conn.Object(DBusService, acPath)}, nil
}

// AddAndActivateConnection creates a profile from settings and activates it.
func (c *Client) AddAndActivateConnection(settings map[string]map[string]dbus.Variant, dev *Device, specific dbus.ObjectPath) (*Connection, *ActiveConnection, error) {
	devPath := dbus.ObjectPath("/")
	if dev != nil {
		devPath = dev.Path()
	}
	var conPath, acPath dbus.ObjectPath
	if err := c.nm.Call(NmIfName+".AddAndActivateConnection", 0, settings, devPath, specific).Store(&conPath, &acPath); err != nil {
		return nil, nil, err
	}
	return c.connAt(conPath), &ActiveConnection{c: c, path: acPath, obj: c.conn.Object(DBusService, acPath)}, nil
}

func (ac *ActiveConnection) Path() dbus.ObjectPath { return ac.path }

// State returns the active connection state.
func (ac *ActiveConnection) State() (uint32, error) {
	return ac.c.propUint32(ac.obj, ActiveConnIf, "State")
}

// StateReason returns (state, reason).
func (ac *ActiveConnection) StateReason() (uint32, uint32, error) {
	v, err := ac.obj.GetProperty(ActiveConnIf + ".StateReason")
	if err != nil {
		return 0, 0, nil
	}
	t, ok := v.Value().([]uint32)
	if !ok || len(t) != 2 {
		return 0, 0, fmt.Errorf("StateReason: unexpected type %T", v.Value())
	}
	return t[0], t[1], nil
}

// SpecificObject returns the specific object (e.g. the AP path) used.
func (ac *ActiveConnection) SpecificObject() (dbus.ObjectPath, error) {
	return ac.c.propObjectPath(ac.obj, ActiveConnIf, "SpecificObject")
}

// Deactivate deactivates the active connection.
func (ac *ActiveConnection) Deactivate() error {
	return ac.c.nm.Call(NmIfName+".DeactivateConnection", 0, ac.path).Err
}
