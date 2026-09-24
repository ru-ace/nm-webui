package api

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ru-ace/nm-webui/internal/events"
	"github.com/ru-ace/nm-webui/internal/system"
)

func testMonitor(probe system.ProbeFn, nmState, gateway func() string, interval, timeout time.Duration) (*PortalMonitor, *events.Hub, <-chan events.Event, func()) {
	hub := events.NewHub(50)
	ch, cleanup := hub.Subscribe()
	m := NewPortalMonitor(probe, nmState, gateway, hub, interval, timeout)
	return m, hub, ch, cleanup
}

func waitEvent(t *testing.T, ch <-chan events.Event, typ string, d time.Duration) events.Event {
	t.Helper()
	deadline := time.After(d)
	for {
		select {
		case ev := <-ch:
			if ev.Type == typ {
				return ev
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %q event", typ)
		}
	}
}

// drain consumes anything already queued (used before asserting silence).
func drain(ch <-chan events.Event) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func assertSilent(t *testing.T, ch <-chan events.Event, d time.Duration) {
	t.Helper()
	select {
	case ev := <-ch:
		t.Fatalf("unexpected event %q after dedup", ev.Type)
	case <-time.After(d):
	}
}

func TestPortalMonitorNilBeforeFirstRun(t *testing.T) {
	m, _, _, cleanup := testMonitor(
		func(context.Context) (system.PortalState, error) { return system.PortalState{State: "online"}, nil },
		func() string { return "online" }, nil, time.Hour, time.Second)
	defer cleanup()
	if p := m.Payload(); p != nil {
		t.Fatalf("Payload() before first run = %v, want nil", p)
	}
}

func TestPortalMonitorPublishesProbeFirst(t *testing.T) {
	// NM keeps claiming "online" (a sham full verdict on an unmanaged uplink)
	// while the probe sees the portal: the advertised state must be portal.
	m, _, ch, cleanup := testMonitor(
		func(context.Context) (system.PortalState, error) {
			return system.PortalState{
				State:     "portal",
				PortalURL: "http://10.0.0.1/login",
				ProbeURL:  "http://captive.example/hotspot-detect.html",
			}, nil
		},
		func() string { return "online" },
		func() string { return "10.0.0.1" },
		time.Hour, time.Second)
	defer cleanup()

	m.run(context.Background())

	cev := waitEvent(t, ch, "connectivity_changed", time.Second)
	if got := cev.Data.(map[string]interface{})["status"]; got != "portal" {
		t.Fatalf("connectivity_changed status = %v, want portal", got)
	}
	pv := waitEvent(t, ch, "captive_portal_changed", time.Second)
	p := pv.Data.(map[string]interface{})
	if p["state"] != "portal" {
		t.Fatalf("payload state = %v, want portal", p["state"])
	}
	if p["portal_url"] != "http://10.0.0.1/login" {
		t.Fatalf("payload portal_url = %v", p["portal_url"])
	}
	if p["nm_connectivity_text"] != "online" {
		t.Fatalf("payload nm_connectivity_text = %v, want online", p["nm_connectivity_text"])
	}
}

func TestPortalMonitorNMFallbackWhenProbeInconclusive(t *testing.T) {
	// Probe proved nothing (no portal, no online marker): NM's link-level
	// state survives as the fallback.
	m, _, ch, cleanup := testMonitor(
		func(context.Context) (system.PortalState, error) {
			return system.PortalState{State: "none"}, nil
		},
		func() string { return "limited" }, nil, time.Hour, time.Second)
	defer cleanup()

	m.run(context.Background())

	cev := waitEvent(t, ch, "connectivity_changed", time.Second)
	if got := cev.Data.(map[string]interface{})["status"]; got != "limited" {
		t.Fatalf("connectivity_changed status = %v, want limited (NM fallback)", got)
	}
}

func TestPortalMonitorDedup(t *testing.T) {
	m, _, ch, cleanup := testMonitor(
		func(context.Context) (system.PortalState, error) {
			return system.PortalState{
				State:     "portal",
				PortalURL: "http://10.0.0.1/login",
			}, nil
		},
		func() string { return "limited" }, nil, time.Hour, time.Second)
	defer cleanup()

	m.run(context.Background())
	waitEvent(t, ch, "connectivity_changed", time.Second)
	waitEvent(t, ch, "captive_portal_changed", time.Second)
	drain(ch)

	// An identical second check must not re-push anything.
	m.run(context.Background())
	assertSilent(t, ch, 60*time.Millisecond)
}

func TestPortalMonitorRepublishesOnPortalURLChange(t *testing.T) {
	var i int
	m, _, ch, cleanup := testMonitor(
		func(context.Context) (system.PortalState, error) {
			i++
			u := "http://10.0.0.1/login"
			if i == 2 {
				u = "http://10.0.0.1/?wlan=1" // same state, new session URL
			}
			return system.PortalState{State: "portal", PortalURL: u}, nil
		},
		func() string { return "limited" }, nil, time.Hour, time.Second)
	defer cleanup()

	m.run(context.Background())
	waitEvent(t, ch, "captive_portal_changed", time.Second)
	drain(ch)

	// Same state, different portal_url: only captive_portal_changed re-pushed.
	m.run(context.Background())
	pv := waitEvent(t, ch, "captive_portal_changed", time.Second)
	if pv.Data.(map[string]interface{})["portal_url"] != "http://10.0.0.1/?wlan=1" {
		t.Fatalf("portal_url not updated on republish")
	}
	if got := cevUnseen(t, ch, 60*time.Millisecond); got {
		t.Fatalf("connectivity_changed must not be re-pushed when state is unchanged")
	}
}

func cevUnseen(t *testing.T, ch <-chan events.Event, d time.Duration) bool {
	t.Helper()
	seen := false
	select {
	case ev := <-ch:
		if ev.Type == "connectivity_changed" {
			seen = true
		}
	case <-time.After(d):
	}
	return seen
}

func TestPortalMonitorForceIsSynchronous(t *testing.T) {
	m, _, _, cleanup := testMonitor(
		func(context.Context) (system.PortalState, error) {
			return system.PortalState{State: "online"}, nil
		},
		func() string { return "online" }, nil, time.Hour, time.Second)
	defer cleanup()

	m.Force(context.Background())
	p := m.Payload()
	if p == nil || p["state"] != "online" {
		t.Fatalf("Payload after Force = %v, want online", p)
	}
}

func TestPortalMonitorRunSerializes(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	m, _, _, cleanup := testMonitor(
		func(context.Context) (system.PortalState, error) {
			close(started)
			<-release
			return system.PortalState{State: "none"}, nil
		},
		nil, nil, time.Hour, time.Second)
	defer cleanup()

	ctx := context.Background()
	done1 := make(chan struct{})
	go func() { m.run(ctx); close(done1) }()
	<-started

	done2 := make(chan struct{})
	go func() { m.run(ctx); close(done2) }()
	select {
	case <-done2:
		t.Fatal("second run completed while the first probe was still in flight")
	case <-time.After(40 * time.Millisecond):
	}
	close(release)
	<-done1
	<-done2
}

func TestPortalMonitorTriggerCausesRun(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	m, _, _, cleanup := testMonitor(
		func(context.Context) (system.PortalState, error) {
			mu.Lock()
			calls++
			mu.Unlock()
			return system.PortalState{State: "none"}, nil
		},
		nil, nil, time.Hour, time.Second)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	// The loop runs an immediate first check.
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := calls
		mu.Unlock()
		if n >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no initial run happened (calls=%d)", n)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Now a trigger must force a second probe even though the ticker is far away.
	m.Trigger()
	deadline = time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := calls
		mu.Unlock()
		if n >= 2 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("trigger did not cause a run (calls=%d)", n)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestPortalMonitorDefaultInterval(t *testing.T) {
	m, _, _, cleanup := testMonitor(
		func(context.Context) (system.PortalState, error) { return system.PortalState{State: "none"}, nil },
		nil, nil, 0, time.Second)
	defer cleanup()
	if m.interval != 30*time.Second {
		t.Fatalf("interval = %v, want 30s default", m.interval)
	}
}

func TestServerPortalServesMonitorPayloadWithoutNetwork(t *testing.T) {
	// resolvedState/portalPayload must serve the monitor's cached verdict and
	// must never touch the (nil) nm client while a monitor is running.
	mon, _, _, cleanup := testMonitor(
		func(context.Context) (system.PortalState, error) {
			return system.PortalState{State: "portal", PortalURL: "http://10.0.0.1/login"}, nil
		},
		func() string { return "online" }, nil, time.Hour, time.Second)
	defer cleanup()
	s := &Server{monitor: mon}

	mon.run(context.Background())

	if got := s.resolvedState(context.Background(), "portal"); got != "portal" {
		t.Fatalf("resolvedState = %q, want portal", got)
	}
	p, err := s.portalPayload(context.Background())
	if err != nil {
		t.Fatalf("portalPayload: %v", err)
	}
	if p["state"] != "portal" {
		t.Fatalf("payload state = %v, want portal", p["state"])
	}
}
