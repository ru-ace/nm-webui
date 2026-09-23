package system

import (
	"context"
	"errors"
	"testing"
)

func TestResolveEffective(t *testing.T) {
	tests := []struct {
		name     string
		nm       string
		pc       PortalState
		probeErr error
		want     string
	}{
		{"portal + probe online", "portal", PortalState{State: "online"}, nil, "online"},
		{"portal + probe portal", "portal", PortalState{State: "portal"}, nil, "portal"},
		{"portal + probe none", "portal", PortalState{State: "none"}, nil, "portal"},
		{"portal + probe unknown", "portal", PortalState{State: "unknown"}, nil, "portal"},
		{"portal + probe error", "portal", PortalState{}, errors.New("boom"), "portal"},
		{"online passthrough", "online", PortalState{State: "portal"}, nil, "online"},
		{"limited passthrough", "limited", PortalState{State: "online"}, nil, "limited"},
		{"none passthrough", "none", PortalState{State: "online"}, nil, "none"},
		{"offline passthrough", "offline", PortalState{State: "portal"}, nil, "offline"},
		{"unknown passthrough", "unknown", PortalState{State: "portal"}, nil, "unknown"},
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

func TestVerdictLatcherPortalFlipsOnProbe(t *testing.T) {
	// NM says portal while the probe still sees the portal page; once the
	// session cookie appears the probe reports online and the next NM signal
	// flips the advertised verdict.
	results := []PortalState{{State: "portal"}, {State: "online"}}
	i := 0
	l := NewVerdictLatcher(func(ctx context.Context) (PortalState, error) {
		r := results[i]
		i++
		return r, nil
	})

	state, changed := l.HandleNM(context.Background(), "portal")
	if state != "portal" || !changed {
		t.Fatalf("first signal: state=%q changed=%v, want portal/true", state, changed)
	}
	state, changed = l.HandleNM(context.Background(), "portal")
	if state != "online" || !changed {
		t.Fatalf("second signal: state=%q changed=%v, want online/true", state, changed)
	}
}

func TestVerdictLatcherNMPortalIgnoredWhileProbeOnline(t *testing.T) {
	// A repeated NM "portal" signal while the probe says online must not flip
	// the advertised verdict back to portal, nor re-push.
	l := NewVerdictLatcher(func(ctx context.Context) (PortalState, error) {
		return PortalState{State: "online"}, nil
	})

	if state, changed := l.HandleNM(context.Background(), "portal"); state != "online" || !changed {
		t.Fatalf("first signal: state=%q changed=%v, want online/true", state, changed)
	}
	if state, changed := l.HandleNM(context.Background(), "portal"); state != "online" || changed {
		t.Fatalf("second signal: state=%q changed=%v, want online/false", state, changed)
	}
}

func TestVerdictLatcherDedup(t *testing.T) {
	l := NewVerdictLatcher(func(ctx context.Context) (PortalState, error) {
		return PortalState{State: "online"}, nil
	})

	if state, changed := l.HandleNM(context.Background(), "portal"); state != "online" || !changed {
		t.Fatalf("first: state=%q changed=%v, want online/true", state, changed)
	}
	if _, changed := l.Recheck(context.Background(), "portal"); changed {
		t.Fatalf("recheck with an identical verdict must not re-push (changed=%v)", changed)
	}
}

func TestVerdictLatcherPassthrough(t *testing.T) {
	l := NewVerdictLatcher(func(ctx context.Context) (PortalState, error) {
		return PortalState{State: "online"}, nil
	})

	if state, changed := l.HandleNM(context.Background(), "limited"); state != "limited" || !changed {
		t.Fatalf("limited: state=%q changed=%v, want limited/true", state, changed)
	}
	if state, changed := l.Recheck(context.Background(), "online"); state != "online" || !changed {
		t.Fatalf("online: state=%q changed=%v, want online/true", state, changed)
	}
	if state, changed := l.Recheck(context.Background(), "online"); state != "online" || changed {
		t.Fatalf("online dedup: state=%q changed=%v, want online/false", state, changed)
	}
}
