package system

import (
	"context"
)

// ResolveEffective decides the state advertised to clients from the probe
// result and NetworkManager's connectivity verdict.
//
// The probe is the only source of truth for "portal" and "online": it walks
// well-known check endpoints from the host itself, so it is the side that
// encounters a captive portal on the uplink and that proves internet again
// once the portal session is open (the shared cookie jar makes both flows
// coherent). NetworkManager's verdict is used only as a fallback for
// link-level states ("limited", "none"/"offline", "unknown") when the probe
// could not reach a check endpoint at all and therefore proved nothing.
//
// NM's own "portal" and "full" verdicts are deliberately not trusted here:
// the uplink may live outside NetworkManager (e.g. a 4G modem on a travel
// router), so NM's connectivity check may never run against it — or it may
// check an HTTPS endpoint that a captive portal lets through while every
// plain-HTTP request is intercepted. The probe claims "online" only on a
// success marker from a check endpoint, and "portal" only on a real
// intercept/login page served to the host.
func ResolveEffective(nmState string, pc PortalState, probeErr error) string {
	if probeErr == nil {
		switch pc.State {
		case "portal":
			return "portal"
		case "online":
			return "online"
		}
	}
	// NM's portal verdict is not evidence of a portal without the probe
	// agreeing, so it degrades to "unknown" instead of being forwarded.
	if nmState == "portal" {
		return "unknown"
	}
	return nmState
}

// ProbeFn refreshes the captive-portal check using the shared session jar.
type ProbeFn func(ctx context.Context) (PortalState, error)
