package api

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ru-ace/nm-webui/internal/events"
	"github.com/ru-ace/nm-webui/internal/nm"
	"github.com/ru-ace/nm-webui/internal/system"
)

// PortalMonitor runs the captive-portal probe in the background and pushes
// the result to SSE subscribers. It is the only producer of portal verdicts:
// NetworkManager is not consulted for portal detection at all (see
// system.ResolveEffective) because on travel routers the uplink (a 4G modem)
// often lives outside NM — its verdicts are link-level fallbacks only.
//
// The monitor keeps the last payload ("remembered for newcomers"): SSE clients
// that connect later receive it immediately, and HTTP status endpoints serve
// it from memory without touching the network.
type PortalMonitor struct {
	probe    system.ProbeFn // returns a fresh probe result on every call
	nmState  func() string  // current NM connectivity text
	gateway  func() string  // primary IPv4 gateway (portal_url fallback)
	hub      *events.Hub
	interval time.Duration
	timeout  time.Duration

	mu        sync.Mutex
	pc        system.PortalState
	probeErr  error
	nmText    string
	nmConn    uint32
	gatewayV  string
	state     string
	hasResult bool
	lastKey   string // fingerprint of the last published payload

	running bool
	runDone chan struct{}
	trigger chan struct{}
}

// NewPortalMonitor creates a monitor. probe must force a fresh check on each
// call (Detector.Refresh fits); nmState and gateway are read lazily on every
// run so link changes are picked up without extra plumbing.
func NewPortalMonitor(probe system.ProbeFn, nmState, gateway func() string, hub *events.Hub, interval, timeout time.Duration) *PortalMonitor {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &PortalMonitor{
		probe:    probe,
		nmState:  nmState,
		gateway:  gateway,
		hub:      hub,
		interval: interval,
		timeout:  timeout,
		trigger:  make(chan struct{}, 1),
	}
}

// Start launches the background loop: an immediate check, then a check every
// interval and on every Trigger. The goroutine exits when ctx is cancelled.
func (m *PortalMonitor) Start(ctx context.Context) {
	go func() {
		m.run(ctx)
		t := time.NewTicker(m.interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				m.run(ctx)
			case <-m.trigger:
				m.run(ctx)
			}
		}
	}()
}

// Trigger requests an out-of-band check (link up/down, connection changes).
// Non-blocking: a pending trigger is coalesced into the next run.
func (m *PortalMonitor) Trigger() {
	select {
	case m.trigger <- struct{}{}:
	default:
	}
}

// Payload returns the latest portal payload, or nil before the first check.
func (m *PortalMonitor) Payload() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.hasResult {
		return nil
	}
	return portalPayloadFrom(m.pc, m.probeErr, m.nmConn, m.nmText, m.gatewayV)
}

// Force runs a check synchronously (manual "Recheck" button) and publishes
// the result if it changed.
func (m *PortalMonitor) Force(ctx context.Context) {
	m.run(ctx)
}

// run serializes checks so concurrent triggers and ticker ticks share a single
// probe instead of stampeding the network.
func (m *PortalMonitor) run(ctx context.Context) {
	m.mu.Lock()
	if m.running {
		done := m.runDone
		m.mu.Unlock()
		select {
		case <-done:
		case <-ctx.Done():
		}
		return
	}
	m.running = true
	m.runDone = make(chan struct{})
	m.mu.Unlock()

	m.runOnce(ctx)

	m.mu.Lock()
	m.running = false
	close(m.runDone)
	m.mu.Unlock()
}

func (m *PortalMonitor) runOnce(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, m.timeout)
	pc, probeErr := m.probe(checkCtx)
	cancel()

	nmText := "unknown"
	nmConn := uint32(nm.ConnectivityUnknown)
	if m.nmState != nil {
		if t := m.nmState(); t != "" {
			nmText = t
			nmConn = codeForState(t)
		}
	}
	gw := ""
	if m.gateway != nil {
		gw = m.gateway()
	}

	m.mu.Lock()
	m.pc = pc
	m.probeErr = probeErr
	m.nmText = nmText
	m.nmConn = nmConn
	m.gatewayV = gw
	state := system.ResolveEffective(nmText, pc, probeErr)
	stateChanged := !m.hasResult || state != m.state
	m.state = state
	m.hasResult = true
	payload := portalPayloadFrom(pc, probeErr, nmConn, nmText, gw)
	key := payloadKey(payload)
	payloadChanged := key != m.lastKey
	m.lastKey = key
	m.mu.Unlock()

	if stateChanged {
		slog.Debug("portal monitor", "state", state)
		m.hub.Publish("connectivity_changed", map[string]interface{}{
			"connectivity": codeForState(state),
			"status":       state,
		})
	}
	if payloadChanged {
		m.hub.Publish("captive_portal_changed", payload)
	}
}

// payloadKey fingerprints the user-visible parts of a payload so unchanged
// results are not re-pushed over SSE (checked_at alone is not interesting).
func payloadKey(p map[string]interface{}) string {
	return fmt.Sprintf("%v|%v|%v", p["state"], p["portal_url"], p["probe_url"])
}
