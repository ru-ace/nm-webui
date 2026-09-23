package api

import (
	"context"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/ru-ace/nm-webui/internal/nm"
)

// startBridge subscribes to the relevant NetworkManager D-Bus signals and
// transposes them into SSE events for the web UI. Subscription failures are
// returned synchronously so the caller can abort startup.
func (s *Server) startBridge(ctx context.Context) error {
	// NOTE: godbus broadcasts every received signal to each registered channel,
	// so we add several match rules but dispatch from a single channel by name.
	matchRules := [][]dbus.MatchOption{
		{dbus.WithMatchInterface(nm.NmIfName)},
		{dbus.WithMatchInterface(nm.DeviceIf), dbus.WithMatchMember("StateChanged")},
		{dbus.WithMatchInterface(nm.WirelessIf), dbus.WithMatchMember("ScanDone")},
		{dbus.WithMatchInterface(nm.WirelessIf), dbus.WithMatchMember("AccessPointAdded")},
		{dbus.WithMatchInterface(nm.WirelessIf), dbus.WithMatchMember("AccessPointRemoved")},
		{dbus.WithMatchInterface(nm.SettingsIfName)},
	}
	ch, cleanup, err := s.nm.WatchMany(matchRules...)
	if err != nil {
		return err
	}

	go func() {
		defer cleanup()
		for {
			select {
			case <-ctx.Done():
				return
			case sig := <-ch:
				if sig == nil {
					continue
				}
				s.dispatch(ctx, sig)
			}
		}
	}()
	return nil
}

// dispatch routes a single D-Bus signal to the appropriate SSE event.
func (s *Server) dispatch(ctx context.Context, sig *dbus.Signal) {
	switch sig.Name {
	case nm.NmIfName + ".ConnectivityChanged":
		if len(sig.Body) > 0 {
			if v, ok := sig.Body[0].(uint32); ok {
				// NM is the trigger, our probe is the judge: a "portal" verdict
				// is re-verified, so a stale NM signal right after a successful
				// sign-in does not flip the advertised state back.
				state, changed := s.verdict.HandleNM(ctx, statusText(v))
				if changed {
					s.hub.Publish("connectivity_changed", map[string]interface{}{
						"connectivity": codeForState(state),
						"status":       state,
					})
				}
			}
		}
	case nm.NmIfName + ".DeviceAdded", nm.NmIfName + ".DeviceRemoved":
		s.hub.Publish("devices_changed", map[string]interface{}{
			"event": sig.Name,
		})
	case nm.NmIfName + ".StateChanged":
		if len(sig.Body) > 0 {
			if v, ok := sig.Body[0].(uint32); ok {
				s.hub.Publish("manager_state_changed", map[string]interface{}{
					"state": v,
					"name":  nm.NmStateMachineName(v),
				})
			}
		}

	case nm.DeviceIf + ".StateChanged":
		var state, reason uint32
		if len(sig.Body) > 0 {
			state, _ = sig.Body[0].(uint32)
		}
		if len(sig.Body) > 1 {
			reason, _ = sig.Body[1].(uint32)
		}
		s.hub.Publish("device_state_changed", map[string]interface{}{
			"path":        string(sig.Path),
			"iface":       s.ifaceOfSignal(sig.Path),
			"state":       state,
			"state_name":  nm.DeviceStateName(state),
			"reason":      reason,
			"reason_name": nm.ReasonName(reason),
		})

	case nm.WirelessIf + ".ScanDone":
		s.hub.Publish("scan_done", map[string]interface{}{
			"iface": s.ifaceOfSignal(sig.Path),
		})
	case nm.WirelessIf + ".AccessPointAdded", nm.WirelessIf + ".AccessPointRemoved":
		s.hub.Publish("wifi_networks_changed", map[string]interface{}{
			"iface": s.ifaceOfSignal(sig.Path),
		})

	case nm.SettingsIfName + ".NewConnection", nm.SettingsIfName + ".ConnectionRemoved":
		s.hub.Publish("connections_changed", map[string]interface{}{
			"event": strings.TrimPrefix(sig.Name, nm.SettingsIfName+"."),
			"path":  sig.Path,
		})
	}
}

// ifaceOfSignal best-effort resolves the interface name for a device path.
func (s *Server) ifaceOfSignal(path dbus.ObjectPath) string {
	dev, err := s.nm.DeviceFromPath(path)
	if err != nil {
		return ""
	}
	name, err := dev.InterfaceName()
	if err != nil {
		return ""
	}
	return name
}
