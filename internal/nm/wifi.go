package nm

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/godbus/dbus/v5"
)

// ErrScanTimeout is returned by ScanAndWait when no ScanDone signal arrives
// within the deadline (scan still allowed to finish, results may be stale).
var ErrScanTimeout = errors.New("wifi scan timed out")

// 802.11 access point capability / security bit constants.
const (
	apFlagPrivacy = 0x0001

	wpaKeyMgmtPsk   = 0x00000100
	wpaKeyMgmt8021x = 0x00000200
	wpaKeyMgmtSAE   = 0x00000800
	wpaKeyMgmtOWE   = 0x00003000
)

// AP is the D-Bus conterpart of org.freedesktop.NetworkManager.AccessPoint.
type AP struct {
	path dbus.ObjectPath
	c    *Client
	obj  dbus.BusObject
}

func (c *Client) newAP(p dbus.ObjectPath) *AP {
	return &AP{path: p, c: c, obj: c.conn.Object(DBusService, p)}
}

// Wireless is a device implementing the Wireless D-Bus interface.
type Wireless struct {
	*Device
}

// Wireless returns the device as a Wireless interface (type must be wifi).
func (d *Device) Wireless() (*Wireless, error) {
	t, err := d.Type()
	if err != nil {
		return nil, err
	}
	if t != DeviceTypeWiFi {
		return nil, fmt.Errorf("device %s is not a wifi device", d.path)
	}
	return &Wireless{Device: d}, nil
}

// RequestScan triggers an active wireless scan on the interface. A
// ScanDone signal is emitted by NetworkManager when finished (only when the
// previous scan finished as well).
func (w *Wireless) RequestScan() error {
	opts := map[string]dbus.Variant{
		"active": dbus.MakeVariant(true),
	}
	return w.obj.Call(WirelessIf+".RequestScan", 0, opts).Err
}

// ScanDoneChan watches for the ScanDone signal on this wireless device.
func (w *Wireless) ScanDoneChan() (<-chan *dbus.Signal, func(), error) {
	return w.c.Watch(
		dbus.WithMatchInterface(WirelessIf),
		dbus.WithMatchMember("ScanDone"),
		dbus.WithMatchObjectPath(w.path),
	)
}

// ScanAndWait triggers an active scan and blocks until the ScanDone signal
// fires or the timeout elapses. Callers use it to refresh scan-dependent
// state (such as AvailableConnections) right before reading it. NetworkManager
// coalesces scans, so if one is already running it completes it and emits
// ScanDone; either way the results are as fresh as the driver allows.
func (w *Wireless) ScanAndWait(timeout time.Duration) error {
	ch, done, err := w.ScanDoneChan()
	if err != nil {
		return err
	}
	defer done()
	if err := w.RequestScan(); err != nil {
		return err
	}
	select {
	case <-ch:
		return nil
	case <-time.After(timeout):
		return ErrScanTimeout
	}
}

// AccessPoints lists all currently known access points.
func (w *Wireless) AccessPoints() ([]*AP, error) {
	var paths []dbus.ObjectPath
	if err := w.obj.Call(WirelessIf+".GetAccessPoints", 0).Store(&paths); err != nil {
		return nil, err
	}
	out := make([]*AP, 0, len(paths))
	for _, p := range paths {
		out = append(out, w.c.newAP(p))
	}
	return out, nil
}

// GetSsid returns the SSID as a string, cleaning non-printable bytes.
func (a *AP) Ssid() (string, error) {
	v, err := a.obj.GetProperty(APIf + ".Ssid")
	if err != nil {
		return "", err
	}
	b, ok := v.Value().([]byte)
	if !ok {
		return "", nil
	}
	return decodeSSID(b), nil
}

func decodeSSID(b []byte) string {
	for len(b) > 0 && b[len(b)-1] == 0 {
		b = b[:len(b)-1]
	}
	if !utf8.Valid(b) {
		// replace invalid bytes with placeholder
		b = []byte(strings.Map(func(r rune) rune {
			if r == utf8.RuneError {
				return '?'
			}
			return r
		}, string(b)))
	}
	s := string(b)
	// collapse control characters
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' {
			return -1
		}
		return r
	}, s)
}

// Bssid returns the hardware address of the AP.
func (a *AP) Bssid() (string, error) {
	s, err := a.c.propString(a.obj, APIf, "HwAddress")
	if err == nil && s != "" {
		return s, nil
	}
	return a.c.propString(a.obj, APIf, "Bssid")
}

// Frequency returns the AP frequency in MHz.
func (a *AP) Frequency() (uint32, error) { return a.c.propUint32(a.obj, APIf, "Frequency") }

// Signal returns the signal strength in percent (0..100).
func (a *AP) Signal() (int, error) {
	v, err := a.obj.GetProperty(APIf + ".Strength")
	if err != nil {
		return 0, err
	}
	if t, ok := v.Value().(uint8); ok {
		return int(t), nil
	}
	return 0, fmt.Errorf("Strength: unexpected type %T", v.Value())
}

// Flags returns the AP flags bitmap.
func (a *AP) Flags() (uint32, error) { return a.c.propUint32(a.obj, APIf, "Flags") }

// WpaFlags returns WPA flags.
func (a *AP) WpaFlags() (uint32, error) { return a.c.propUint32(a.obj, APIf, "WpaFlags") }

// RsnFlags returns RSN (WPA2/3) flags.
func (a *AP) RsnFlags() (uint32, error) { return a.c.propUint32(a.obj, APIf, "RsnFlags") }

// LastSeen returns seconds since last seen.
func (a *AP) LastSeen() (int32, error) {
	v, err := a.obj.GetProperty(APIf + ".LastSeen")
	if err != nil {
		return 0, err
	}
	if i, ok := v.Value().(int32); ok {
		return i, nil
	}
	return 0, nil
}

// SecurityType classifies the AP security from flags.
func SecurityType(flags, wpa, rsn uint32) string {
	switch {
	case wpa&wpaKeyMgmt8021x != 0 || rsn&wpaKeyMgmt8021x != 0:
		return "enterprise"
	case wpa&wpaKeyMgmtSAE != 0 || rsn&wpaKeyMgmtSAE != 0:
		return "wpa3"
	case wpa&wpaKeyMgmtOWE != 0 || rsn&wpaKeyMgmtOWE != 0:
		return "owe"
	case wpa&wpaKeyMgmtPsk != 0 || rsn&wpaKeyMgmtPsk != 0:
		return "wpa2"
	case wpa != 0 || rsn != 0:
		return "wpa"
	case flags&apFlagPrivacy != 0:
		return "wep"
	default:
		return "open"
	}
}

// Band returns a human band label ("2.4", "5", "6" or "").
func Band(freq uint32) string {
	switch {
	case freq >= 2400 && freq < 3000:
		return "2.4"
	case freq >= 5000 && freq < 6000:
		return "5"
	case freq >= 6000 && freq < 7200:
		return "6"
	default:
		return ""
	}
}

// Channel converts a frequency to a wifi channel number (0 if unknown).
func Channel(freq uint32) uint32 {
	switch {
	case freq == 2484:
		return 14
	case freq >= 2412 && freq <= 2472:
		return (freq-2412)/5 + 1
	case freq >= 5180 && freq <= 5825:
		return (freq-5180)/5 + 36
	case freq >= 5955 && freq <= 7115:
		return (freq-5955)/5 + 1
	default:
		return 0
	}
}

// APInfo is a JSON-friendly representation of an access point.
type APInfo struct {
	SSID       string `json:"ssid"`
	Saved      bool   `json:"saved"`
	Hidden     bool   `json:"hidden,omitempty"`
	BSSID      string `json:"bssid"`
	Signal     int    `json:"signal"`
	SignalPct  int    `json:"signal_pct"`
	Frequency  uint32 `json:"frequency"`
	Band       string `json:"band,omitempty"`
	Channel    uint32 `json:"channel"`
	Security   string `json:"security"`
	LastSeen   int32  `json:"last_seen"`
	MaxBitrate uint32 `json:"max_bitrate,omitempty"`
}

// Info returns the access point information.
func (a *AP) Info() (APInfo, error) {
	info := APInfo{SSID: "(hidden)", Security: "unknown"}
	ssid, _ := a.Ssid()
	if ssid != "" {
		info.SSID = ssid
	}
	info.BSSID, _ = a.Bssid()
	s, err := a.Signal()
	if err == nil {
		info.Signal = s
		info.SignalPct = s
	}
	info.Frequency, _ = a.Frequency()
	info.Band = Band(info.Frequency)
	info.Channel = Channel(info.Frequency)
	if info.Channel == 0 && info.Band == "" && info.Frequency > 0 {
		info.Band = "?"
	}
	f, _ := a.Flags()
	wf, _ := a.WpaFlags()
	rf, _ := a.RsnFlags()
	info.Security = SecurityType(f, wf, rf)
	info.LastSeen, _ = a.LastSeen()
	if v, err := a.obj.GetProperty(APIf + ".MaxBitrate"); err == nil {
		if u, ok := v.Value().(uint32); ok {
			info.MaxBitrate = u
		}
	}
	return info, nil
}
