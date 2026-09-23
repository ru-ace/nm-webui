package system

import (
	"context"
	"sync"
	"time"
)

// ResolveEffective decides the state advertised to clients from NetworkManager's
// connectivity verdict and our own probe result.
//
// NM is authoritative for every verdict except "portal", which is re-verified
// by the probe: the probe shares the captive-portal session cookie jar, so
// right after a successful sign-in it proves "online" even while NM's periodic
// check is still catching up. Only a probe that explicitly reports "online"
// overrides NM's verdict; a portal page, no connectivity or a probe error all
// keep it.
func ResolveEffective(nmState string, pc PortalState, probeErr error) string {
	if nmState != "portal" {
		return nmState
	}
	if probeErr != nil || pc.State != "online" {
		return "portal"
	}
	return "online"
}

// ProbeFn re-verifies connectivity using the shared portal session jar.
type ProbeFn func(ctx context.Context) (PortalState, error)

// probeRecheckTimeout bounds a probe run triggered by a ConnectivityChanged
// signal so the D-Bus bridge goroutine never blocks for long.
const probeRecheckTimeout = 5 * time.Second

// VerdictLatcher records the last state pushed to clients and deduplicates SSE
// publishes. NM signals are the trigger; the probe is the judge for the
// "portal" verdict (see ResolveEffective).
type VerdictLatcher struct {
	mu       sync.Mutex
	lastPush string // last state pushed via SSE ("" = nothing pushed yet)
	probe    ProbeFn
}

// NewVerdictLatcher creates a latcher that re-verifies "portal" verdicts with
// probe.
func NewVerdictLatcher(probe ProbeFn) *VerdictLatcher {
	return &VerdictLatcher{probe: probe}
}

// HandleNM applies an NM ConnectivityChanged-derived state and returns the
// resulting advertised state plus whether it changed since the last push.
func (l *VerdictLatcher) HandleNM(ctx context.Context, nmState string) (string, bool) {
	return l.resolve(ctx, nmState)
}

// Recheck re-applies a fresh NM status-derived state, e.g. from the
// user-initiated recheck endpoint.
func (l *VerdictLatcher) Recheck(ctx context.Context, nmState string) (string, bool) {
	return l.resolve(ctx, nmState)
}

func (l *VerdictLatcher) resolve(ctx context.Context, nmState string) (string, bool) {
	var state string
	if nmState == "portal" {
		probeCtx, cancel := context.WithTimeout(ctx, probeRecheckTimeout)
		defer cancel()
		pc, err := l.probe(probeCtx)
		state = ResolveEffective(nmState, pc, err)
	} else {
		state = nmState
	}

	l.mu.Lock()
	changed := state != l.lastPush
	if changed {
		l.lastPush = state
	}
	l.mu.Unlock()
	return state, changed
}
