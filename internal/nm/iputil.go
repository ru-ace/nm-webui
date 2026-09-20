package nm

import (
	"encoding/binary"
	"net"
)

// u32ToIP converts a NetworkManager D-Bus uint32 IPv4 address to dotted-decimal string.
// NetworkManager encodes IPv4 addresses in network byte order on the wire, which
// deserializes into little-endian host integers (first octet in lowest byte).
func u32ToIP(u uint32) string {
	return net.IPv4(byte(u), byte(u>>8), byte(u>>16), byte(u>>24)).String()
}

// parseIP4 converts an IPv4 string to the uint32 expected by NetworkManager.
func parseIP4(s string) uint32 {
	ip := net.ParseIP(s)
	if ip == nil {
		return 0
	}
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(ip)
}

// encIP4 converts an IPv4 string to the canonical dotted form.
func encIP4(s string) string {
	ip := net.ParseIP(s)
	if ip == nil {
		return ""
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.String()
}

// parseDNS4 converts a list of IPv4 strings into the uint32 form NM expects.
func parseDNS4(list []string) []uint32 {
	out := make([]uint32, 0, len(list))
	for _, s := range list {
		if u := parseIP4(s); u != 0 {
			out = append(out, u)
		}
	}
	return out
}