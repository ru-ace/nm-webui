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
		{dbus.WithMatchInterface(nm.ActiveConnIf), dbus.WithMatchMember("StateChanged")},
		{dbus.WithMatchInterface(nm.WirelessIf), dbus.WithMatchMember("ScanDone")},
		{dbus.WithMatchInterface(nm.WirelessIf), dbus.WithMatchMember("AccessPointAdded")},
		{dbus.WithMatchInterface(nm.WirelessIf), dbus.WithMatchMember("AccessPointRemoved")},
		{dbus.WithMatchInterface(nm.SettingsIfName)},
		{dbus.WithMatchInterface(nm.SettingsConnIf), dbus.WithMatchMember("Updated")},
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
		// NM's connectivity verdict is no longer used for portal detection at
		// all (see system.ResolveEffective): it only exists as a link-level
		// fallback inside the monitor. The signal is still valuable as a
		// trigger that forces an immediate probe from our own check.
		s.portalTrigger()
	case nm.NmIfName + ".DeviceAdded", nm.NmIfName + ".DeviceRemoved":
		s.hub.Publish("devices_changed", map[string]interface{}{
			"event": sig.Name,
		})
		s.portalTrigger()
	case nm.NmIfName + ".StateChanged":
		if len(sig.Body) > 0 {
			if v, ok := sig.Body[0].(uint32); ok {
				s.hub.Publish("manager_state_changed", map[string]interface{}{
					"state": v,
					"name":  nm.NmStateMachineName(v),
				})
			}
		}
		// Manager transitions (connecting → connected, sleep/wake) may change
		// the uplink: probe again.
		s.portalTrigger()

	case nm.ActiveConnIf + ".StateChanged":
		// Every active connection emits StateChanged on each transition,
		// regardless of what triggered the activation/deactivation (our UI,
		// KDE, nmcli, autoconnect, ...). Only the terminal states matter.
		if len(sig.Body) > 0 {
			state, ok := sig.Body[0].(uint32)
			if !ok {
				return
			}
			var active bool
			switch state {
			case nm.ACStateActivated:
				active = true
			case nm.ACStateDeactivated:
				active = false
			default:
				return // transient states (activating, deactivating)
			}
			// A connection just came up or went down: the portal state on the
			// new uplink must be re-checked immediately ("just connected").
			s.portalTrigger()
			if uuid, err := s.nm.ActiveConnectionByPath(sig.Path).Uuid(); err == nil && uuid != "" {
				s.hub.Publish("connection_state_changed", map[string]interface{}{
					"uuid":   uuid,
					"active": active,
				})
			} else {
				// The active connection object may already be gone; fall back
				// to a full profile reload.
				s.hub.Publish("connections_changed", map[string]interface{}{
					"event": "connection-state-fallback",
					"path":  string(sig.Path),
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
		if isTerminalDeviceState(state) {
			// A link settled (up/down/unavailable): the uplink may have
			// changed, so re-probe right away.
			s.portalTrigger()
		}
		payload := map[string]interface{}{
			"path":        string(sig.Path),
			"iface":       s.ifaceOfSignal(sig.Path),
			"state":       state,
			"state_name":  nm.DeviceStateName(state),
			"reason":      reason,
			"reason_name": nm.ReasonName(reason),
		}
		// IP configuration is only settled once the device reaches a terminal
		// state (addresses assigned on Activated, gone on Disconnected/
		// Unavailable/Unmanaged/Failed). Attach the full device snapshot there
		// so the UI can refresh or clear IPs without an extra round-trip that
		// would race the async activation.
		if isTerminalDeviceState(state) {
			dev, dErr := s.nm.DeviceFromPath(sig.Path)
			if dErr == nil {
				if info, iErr := dev.Info(); iErr == nil {
					payload["device"] = info
				}
			}
		}
		s.hub.Publish("device_state_changed", payload)

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
	case nm.SettingsConnIf + ".Updated":
		// A profile was modified (e.g. edited via KDE or nmcli): re-read the
		// whole list so cards reflect the new name/settings.
		s.hub.Publish("connections_changed", map[string]interface{}{
			"event": "Updated",
			"path":  sig.Path,
		})
	}
}

// portalTrigger asks the background monitor for an out-of-band probe. Safe to
// call in any signal path: the trigger is a coalesced, non-blocking channel.
func (s *Server) portalTrigger() {
	if s.monitor != nil {
		s.monitor.Trigger()
	}
}

// isTerminalDeviceState reports whether the device settled into a state whose
// IP configuration is final: addresses are assigned (Activated) or gone
// (Disconnected, Unavailable, Unmanaged, Failed). Transient states (Prepare
// through Secondaries, Deactivating) are skipped.
func isTerminalDeviceState(state uint32) bool {
	switch state {
	case nm.DeviceStateActivated, nm.DeviceStateDisconnected,
		nm.DeviceStateUnavailable, nm.DeviceStateUnmanaged, nm.DeviceStateFailed:
		return true
	}
	return false
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
