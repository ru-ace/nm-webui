package nm

const (
	DBusService    = "org.freedesktop.NetworkManager"
	NmPath         = "/org/freedesktop/NetworkManager"
	NmIfName       = "org.freedesktop.NetworkManager"
	SettingsPath   = "/org/freedesktop/NetworkManager/Settings"
	SettingsIfName = "org.freedesktop.NetworkManager.Settings"

	DeviceIf        = "org.freedesktop.NetworkManager.Device"
	WirelessIf      = "org.freedesktop.NetworkManager.Device.Wireless"
	APIf            = "org.freedesktop.NetworkManager.AccessPoint"
	ActiveConnIf    = "org.freedesktop.NetworkManager.Connection.Active"
	SettingsConnIf  = "org.freedesktop.NetworkManager.Settings.Connection"
	IP4ConfigIf     = "org.freedesktop.NetworkManager.IP4Config"
	IP6ConfigIf     = "org.freedesktop.NetworkManager.IP6Config"
	ConnIf          = "org.freedesktop.NetworkManager.Connection"
	DnsIf           = "org.freedesktop.NetworkManager.DnsManager"
)

// Device types.
const (
	DeviceTypeEthernet  = 1
	DeviceTypeWiFi      = 2
	DeviceTypeBt        = 5
	DeviceTypeBond      = 10
	DeviceTypeVlan      = 11
	DeviceTypeBridge    = 13
	DeviceTypeGeneric   = 14
	DeviceTypeTeam      = 15
	DeviceTypeTun       = 16
	DeviceTypeIpTunnel  = 17
	DeviceTypeMacvlan   = 18
	DeviceTypeVxlan     = 19
	DeviceTypeVeth      = 20
	DeviceTypeWireGuard = 21
	DeviceTypeWwan      = 22
	DeviceTypeOvsBridge = 25
	DeviceType6LOWPAN   = 26
	DeviceTypeWpan      = 27
)

// Device states.
const (
	DeviceStateUnknown       = 0
	DeviceStateUnmanaged     = 10
	DeviceStateUnavailable   = 20
	DeviceStateDisconnected  = 30
	DeviceStatePrepare       = 40
	DeviceStateConfig        = 50
	DeviceStateNeedAuth      = 60
	DeviceStateIPConfig      = 70
	DeviceStateIPCheck       = 80
	DeviceStateSecondaries   = 90
	DeviceStateActivated     = 100
	DeviceStateDeactivating  = 110
	DeviceStateFailed        = 120
)

// Active connection states.
const (
	ACStateUnknown      = 0
	ACStateActivating   = 1
	ACStateActivated    = 2
	ACStateDeactivating = 3
	ACStateDeactivated  = 4
)

// Connectivity states.
const (
	ConnectivityUnknown  = 0
	ConnectivityNone     = 1
	ConnectivityPortal   = 2
	ConnectivityLimited  = 3
	ConnectivityFull     = 4
)

// DeviceTypeName maps NM device types to friendly names.
func DeviceTypeName(t uint32) string {
	switch t {
	case DeviceTypeEthernet:
		return "ethernet"
	case DeviceTypeWiFi:
		return "wifi"
	case DeviceTypeBt:
		return "bluetooth"
	case DeviceTypeBond:
		return "bond"
	case DeviceTypeVlan:
		return "vlan"
	case DeviceTypeBridge:
		return "bridge"
	case DeviceTypeGeneric:
		return "generic"
	case DeviceTypeTeam:
		return "team"
	case DeviceTypeTun:
		return "tun"
	case DeviceTypeIpTunnel:
		return "ip-tunnel"
	case DeviceTypeMacvlan:
		return "macvlan"
	case DeviceTypeVxlan:
		return "vxlan"
	case DeviceTypeVeth:
		return "veth"
	case DeviceTypeWireGuard:
		return "wireguard"
	case DeviceTypeWwan:
		return "wwan"
	case DeviceTypeOvsBridge:
		return "ovs-bridge"
	case DeviceType6LOWPAN:
		return "6lowpan"
	case DeviceTypeWpan:
		return "wpan"
	default:
		return "unknown"
	}
}

// DeviceStateName maps NM device states to friendly names.
func DeviceStateName(s uint32) string {
	switch s {
	case DeviceStateUnknown:
		return "unknown"
	case DeviceStateUnmanaged:
		return "unmanaged"
	case DeviceStateUnavailable:
		return "unavailable"
	case DeviceStateDisconnected:
		return "disconnected"
	case DeviceStatePrepare:
		return "preparing"
	case DeviceStateConfig:
		return "configuring"
	case DeviceStateNeedAuth:
		return "need-auth"
	case DeviceStateIPConfig:
		return "ip-configuring"
	case DeviceStateIPCheck:
		return "ip-checking"
	case DeviceStateSecondaries:
		return "secondaries"
	case DeviceStateActivated:
		return "connected"
	case DeviceStateDeactivating:
		return "deactivating"
	case DeviceStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ConnectivityName maps NM connectivity states to dashboard labels.
func ConnectivityName(c uint32) string {
	switch c {
	case ConnectivityFull:
		return "online"
	case ConnectivityLimited, ConnectivityPortal:
		return "limited"
	case ConnectivityNone:
		return "offline"
	default:
		return "unknown"
	}
}

// Network carrying the overall NM state.
type NmState struct {
	State        uint32
	Connectivity uint32
	Hostname     string
	NMVersion    string
	Enable       bool
}