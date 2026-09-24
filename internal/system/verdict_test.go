package system

import (
	"errors"
	"testing"
)

// TestResolveEffective pinpoints the probe-first contract:
//   - the probe is the only source of truth for "portal"/"online" (it is the
//     side that actually walks the check endpoints);
//   - NM survives only as a link-level fallback ("limited"/"none"/"offline"/
//     "unknown"/online) when the probe proved nothing;
//   - NM's own "portal" verdict is never forwarded — without probe agreement
//     it degrades to "unknown".
func TestResolveEffective(t *testing.T) {
	tests := []struct {
		name     string
		nm       string
		pc       PortalState
		probeErr error
		want     string
	}{
		// Probe is authoritative for portal and online, no matter what NM says.
		{"nm portal + probe portal", "portal", PortalState{State: "portal"}, nil, "portal"},
		{"nm portal + probe online", "portal", PortalState{State: "online"}, nil, "online"},
		{"nm online + probe portal", "online", PortalState{State: "portal"}, nil, "portal"},
		{"nm full + probe portal", "online", PortalState{State: "portal"}, nil, "portal"},
		{"nm full + probe online", "online", PortalState{State: "online"}, nil, "online"},
		{"nm limited + probe online", "limited", PortalState{State: "online"}, nil, "online"},
		{"nm none + probe online", "none", PortalState{State: "online"}, nil, "online"},
		{"nm offline + probe portal", "offline", PortalState{State: "portal"}, nil, "portal"},

		// Probe proved nothing (no portal found, no online marker): fall back
		// to NM's link-level state. NM "portal" degrades to unknown.
		{"nm limited + probe none", "limited", PortalState{State: "none"}, nil, "limited"},
		{"nm online + probe none", "online", PortalState{State: "none"}, nil, "online"},
		{"nm offline + probe none", "offline", PortalState{State: "none"}, nil, "offline"},
		{"nm unknown + probe none", "unknown", PortalState{State: "none"}, nil, "unknown"},
		{"nm portal + probe none", "portal", PortalState{State: "none"}, nil, "unknown"},
		{"nm portal + probe unknown", "portal", PortalState{State: "unknown"}, nil, "unknown"},
		{"nm portal + probe error", "portal", PortalState{}, errors.New("boom"), "unknown"},
		{"nm unknown + probe error", "unknown", PortalState{}, errors.New("boom"), "unknown"},
		{"nm offline + probe error", "offline", PortalState{}, errors.New("boom"), "offline"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveEffective(tc.nm, tc.pc, tc.probeErr); got != tc.want {
				t.Fatalf("ResolveEffective(%q, %+v, %v) = %q, want %q",
					tc.nm, tc.pc, tc.probeErr, got, tc.want)
			}
		})
	}
}
