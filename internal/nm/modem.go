package nm

import (
	"strings"

	"github.com/godbus/dbus/v5"
)

// Modem capabilities (NM_MODEM_CAPABILITY / MM_MODEM_CAPABILITY bitmask).
const (
	ModemCapabilityPots     = 1 << 0
	ModemCapabilityCdmaEvdo = 1 << 1
	ModemCapabilityGSMUMTS  = 1 << 2
	ModemCapabilityLTE      = 1 << 3
	ModemCapability5GNR     = 1 << 4
)

// Modem states (MM_MODEM_STATE).
const (
	ModemStateUnknown       = 0
	ModemStateInitializing  = 1
	ModemStateLocked        = 2
	ModemStateDisabled      = 3
	ModemStateDisabling     = 4
	ModemStateEnabling      = 5
	ModemStateEnabled       = 6
	ModemStateSearching     = 7
	ModemStateRegistered    = 8
	ModemStateDisconnecting = 9
	ModemStateConnecting    = 10
	ModemStateConnected     = 11
)

// Modem access technologies (MM_MODEM_ACCESS_TECHNOLOGY bitmask).
const (
	AccessTechUnknown      = 0
	AccessTechPOTS         = 1 << 0
	AccessTechGSM          = 1 << 1
	AccessTechGSMCompact   = 1 << 2
	AccessTechGPRS         = 1 << 3
	AccessTechEDGE         = 1 << 4
	AccessTechUMTS         = 1 << 5
	AccessTechHSPA         = 1 << 6
	AccessTechHSPAPlus     = 1 << 7
	AccessTech1xRTT        = 1 << 8
	AccessTechEVDO0        = 1 << 9
	AccessTechEVDOA        = 1 << 10
	AccessTechEVDOB        = 1 << 11
	AccessTechLTE          = 1 << 12
	AccessTech5GNR         = 1 << 13
	AccessTechLTEAdvanced  = 1 << 14
	AccessTech5GNRNSA      = 1 << 15
	AccessTech5GNRSANR     = 1 << 16
	AccessTech5GNRSAUplink = 1 << 17
)

// ModemSignal is the radio signal quality reported by ModemManager.
type ModemSignal struct {
	Percent uint32 `json:"percent"`
	Recent  bool   `json:"recent,omitempty"`
}

// ModemSimInfo describes the currently active SIM card.
type ModemSimInfo struct {
	OperatorName string `json:"operator_name,omitempty"`
	OperatorCode string `json:"operator_code,omitempty"`
	ICCID        string `json:"iccid,omitempty"`
	IMSI         string `json:"imsi,omitempty"`
	Active       bool   `json:"active,omitempty"`
}

// ModemInfo is the JSON-friendly view of a mobile broadband (4G/5G) modem.
type ModemInfo struct {
	Apn             string        `json:"apn,omitempty"`
	OperatorCode    string        `json:"operator_code,omitempty"`
	OperatorName    string        `json:"operator_name,omitempty"`
	Capabilities    uint32        `json:"capabilities"`
	CapabilitiesStr string        `json:"capabilities_text,omitempty"`
	Signal          *ModemSignal  `json:"signal,omitempty"`
	State           int32         `json:"state"`
	StateName       string        `json:"state_name,omitempty"`
	AccessTech      uint32        `json:"access_tech"`
	AccessTechStr   string        `json:"access_tech_name,omitempty"`
	Manufacturer    string        `json:"manufacturer,omitempty"`
	Model           string        `json:"model,omitempty"`
	IMEI            string        `json:"imei,omitempty"`
	Firmware        string        `json:"firmware,omitempty"`
	Sim             *ModemSimInfo `json:"sim,omitempty"`
}

// Udi returns the device's UDI (ModemManager object path for modems).
func (d *Device) Udi() (string, error) { return d.c.propString(d.obj, DeviceIf, "Udi") }

// Modem reports whether the device is a mobile broadband modem.
func (d *Device) IsModem() (bool, error) {
	t, err := d.Type()
	if err != nil {
		return false, err
	}
	return t == DeviceTypeModem || t == DeviceTypeWwan, nil
}

// Modem reads the mobile broadband information for a modem device. Network
// Manager properties are always available; ModemManager is consulted for
// signal strength, access technology and SIM details when the daemon is up.
func (d *Device) Modem() (*ModemInfo, error) {
	m := &ModemInfo{}

	// NetworkManager Device.Modem properties.
	if v, err := d.c.propString(d.obj, DeviceModemIf, "Apn"); err == nil {
		m.Apn = v
	}
	if v, err := d.c.propString(d.obj, DeviceModemIf, "OperatorCode"); err == nil {
		m.OperatorCode = v
	}
	if v, err := d.c.propUint32(d.obj, DeviceModemIf, "CurrentCapabilities"); err == nil {
		m.Capabilities = v
	}
	m.CapabilitiesStr = ModemCapabilityNames(m.Capabilities)

	// ModemManager enrichment via the Device.Udi object path.
	udi, err := d.Udi()
	if err == nil && strings.HasPrefix(udi, "/org/freedesktop/ModemManager1/") {
		d.c.fillFromModemManager(m, dbus.ObjectPath(udi))
	}

	if m.OperatorName == "" && m.Sim != nil && m.Sim.OperatorName != "" {
		m.OperatorName = m.Sim.OperatorName
	}
	if m.State == 0 && m.OperatorCode == "" && m.Apn == "" {
		// Bare NM-only modem: keep StateName consistent with the device state.
	}
	return m, nil
}

// fillFromModemManager enriches m with ModemManager facts. All failures are
// ignored: a missing ModemManager service is not an error for the API.
func (c *Client) fillFromModemManager(m *ModemInfo, path dbus.ObjectPath) {
	obj := c.conn.Object(MMService, path)

	if v, err := obj.GetProperty(MMModemIf + ".SignalQuality"); err == nil {
		if sig := parseSignalQuality(v.Value()); sig != nil {
			m.Signal = sig
		}
	}
	if v, err := obj.GetProperty(MMModemIf + ".State"); err == nil {
		if st, ok := v.Value().(int32); ok {
			m.State = st
		}
	}
	m.StateName = ModemStateName(m.State)

	if v, err := obj.GetProperty(MMModemIf + ".AccessTechnologies"); err == nil {
		if t, ok := v.Value().(uint32); ok {
			m.AccessTech = t
		}
	}
	m.AccessTechStr = AccessTechName(m.AccessTech)

	if str, err := c.propsString(obj, MMModemIf, "Manufacturer"); err == nil {
		m.Manufacturer = str
	}
	if str, err := c.propsString(obj, MMModemIf, "Model"); err == nil {
		m.Model = str
	}
	if str, err := c.propsString(obj, MMModemIf, "EquipmentIdentifier"); err == nil {
		m.IMEI = str
	}
	if str, err := c.propsString(obj, MMModemIf, "Revision"); err == nil {
		m.Firmware = str
	}

	simPath, err := c.propObjectPath(obj, MMModemIf, "Sim")
	if err != nil || simPath == "/" || simPath == "" {
		return
	}
	sim := c.conn.Object(MMService, simPath)
	m.Sim = &ModemSimInfo{}
	m.Sim.OperatorName, _ = c.propsString(sim, MMSimIf, "OperatorName")
	m.Sim.OperatorCode, _ = c.propsString(sim, MMSimIf, "OperatorIdentifier")
	m.Sim.ICCID, _ = c.propsString(sim, MMSimIf, "SimIdentifier")
	m.Sim.IMSI, _ = c.propsString(sim, MMSimIf, "Imsi")
	if v, err := sim.GetProperty(MMSimIf + ".Active"); err == nil {
		if b, ok := v.Value().(bool); ok {
			m.Sim.Active = b
		}
	}
}

// propsString reads a string property and returns an error when it cannot.
func (c *Client) propsString(obj dbus.BusObject, iface, name string) (string, error) {
	v, err := obj.GetProperty(iface + "." + name)
	if err != nil {
		return "", err
	}
	s, ok := v.Value().(string)
	if !ok {
		return "", nil
	}
	return s, nil
}

// parseSignalQuality decodes the ModemManager (ub) SignalQuality struct.
func parseSignalQuality(v interface{}) *ModemSignal {
	sig := &ModemSignal{}
	switch t := v.(type) {
	case []interface{}:
		if len(t) > 0 {
			if u, ok := t[0].(uint32); ok {
				sig.Percent = u
			}
		}
		if len(t) > 1 {
			if b, ok := t[1].(bool); ok {
				sig.Recent = b
			}
		}
	case []uint32:
		if len(t) > 0 {
			sig.Percent = t[0]
		}
		sig.Recent = true
	default:
		return nil
	}
	return sig
}

// ModemCapabilityNames converts a capability bitmask to human labels.
func ModemCapabilityNames(u uint32) string {
	var out []string
	if u&ModemCapabilityPots != 0 {
		out = append(out, "POTS")
	}
	if u&ModemCapabilityCdmaEvdo != 0 {
		out = append(out, "CDMA/EV-DO")
	}
	if u&ModemCapabilityGSMUMTS != 0 {
		out = append(out, "GSM/UMTS")
	}
	if u&ModemCapabilityLTE != 0 {
		out = append(out, "LTE")
	}
	if u&ModemCapability5GNR != 0 {
		out = append(out, "5G NR")
	}
	return strings.Join(out, ", ")
}

// AccessTechName converts an access technology bitmask to human labels.
func AccessTechName(u uint32) string {
	var out []string
	for bit, name := range map[uint32]string{
		AccessTechPOTS:         "POTS",
		AccessTechGSM:          "GSM",
		AccessTechGSMCompact:   "GSM Compact",
		AccessTechGPRS:         "GPRS",
		AccessTechEDGE:         "EDGE",
		AccessTechUMTS:         "UMTS",
		AccessTechHSPA:         "HSPA",
		AccessTechHSPAPlus:     "HSPA+",
		AccessTech1xRTT:        "1xRTT",
		AccessTechEVDO0:        "EV-DO 0",
		AccessTechEVDOA:        "EV-DO A",
		AccessTechEVDOB:        "EV-DO B",
		AccessTechLTE:          "LTE",
		AccessTechLTEAdvanced:  "LTE Advanced",
		AccessTech5GNR:         "5G NR",
		AccessTech5GNRNSA:      "5G NR NSA",
		AccessTech5GNRSANR:     "5G NR SA",
		AccessTech5GNRSAUplink: "5G NR SA (UL)",
	} {
		if u&bit != 0 {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, ", ")
}

// ModemStateName maps ModemManager modem states to friendly names.
func ModemStateName(s int32) string {
	switch s {
	case ModemStateUnknown:
		return "unknown"
	case ModemStateInitializing:
		return "initializing"
	case ModemStateLocked:
		return "locked"
	case ModemStateDisabled:
		return "disabled"
	case ModemStateDisabling:
		return "disabling"
	case ModemStateEnabling:
		return "enabling"
	case ModemStateEnabled:
		return "enabled"
	case ModemStateSearching:
		return "searching"
	case ModemStateRegistered:
		return "registered"
	case ModemStateDisconnecting:
		return "disconnecting"
	case ModemStateConnecting:
		return "connecting"
	case ModemStateConnected:
		return "connected"
	default:
		return "unknown"
	}
}
