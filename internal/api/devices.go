package api

import (
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ru-ace/nm-webui/internal/nm"
)

// Timeouts/waits for the WiFi profile-picker refresh: NetworkManager builds
// Device.AvailableConnections for wireless devices from the *last scan*, so
// the picker re-triggers a scan and waits for ScanDone before reading the
// list. If the scan fails or stalls we fall back to the cached list instead
// of failing the request.
const (
	wifiScanRefreshTimeout = 8 * time.Second
	wifiScanSettleDelay    = 500 * time.Millisecond
)

// visibleDevices returns devices allowed by the interface filter.
func (s *Server) visibleDevices() ([]*nm.Device, error) {
	devs, err := s.nm.GetAllDevices()
	if err != nil {
		return nil, err
	}
	out := make([]*nm.Device, 0, len(devs))
	for _, d := range devs {
		name, err := d.InterfaceName()
		if err != nil {
			continue
		}
		if s.cfg.InterfaceAllowed(name) {
			out = append(out, d)
		}
	}
	return out, nil
}

// deviceFromRequest resolves a device by interface name honouring the filter.
func (s *Server) deviceFromRequest(r *http.Request) (*nm.Device, error) {
	iface := chi.URLParam(r, "iface")
	if !s.cfg.InterfaceAllowed(iface) {
		return nil, errors.New("interface is hidden by filter")
	}
	return s.nm.DeviceByInterface(iface)
}

func (s *Server) handleDeviceList(w http.ResponseWriter, r *http.Request) {
	devs, err := s.visibleDevices()
	if err != nil {
		httpError(w, err)
		return
	}
	infos := make([]nm.DeviceInfo, 0, len(devs))
	for _, d := range devs {
		info, err := d.Info()
		if err != nil {
			continue
		}
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Kind != infos[j].Kind {
			return infos[i].Kind < infos[j].Kind
		}
		return infos[i].Interface < infos[j].Interface
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{"devices": infos})
}

func (s *Server) handleDeviceGet(w http.ResponseWriter, r *http.Request) {
	dev, err := s.deviceFromRequest(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "device not found")
		return
	}
	info, err := dev.Info()
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleDeviceDisconnect(w http.ResponseWriter, r *http.Request) {
	dev, err := s.deviceFromRequest(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "device not found")
		return
	}
	if err := dev.Disconnect(); err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnecting"})
}

func (s *Server) handleDeviceUp(w http.ResponseWriter, r *http.Request) {
	dev, err := s.deviceFromRequest(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "device not found")
		return
	}
	name, _ := dev.InterfaceName()
	if err := s.upIgnoredDevice(name, dev); err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "connecting"})
}

// handleDeviceConnections lists connection profiles that NetworkManager
// reports as applicable to the given device (Device.AvailableConnections).
// This is the source of truth for the dashboard "connect to a profile"
// picker: NM decides which stored profiles can activate on this interface.
// For wireless devices NM derives the list from the *last scan*, so before
// answering we refresh the scan (best effort) to make the list reflect the
// networks actually in range instead of whatever the previous scan saw.
func (s *Server) handleDeviceConnections(w http.ResponseWriter, r *http.Request) {
	dev, err := s.deviceFromRequest(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "device not found")
		return
	}
	if t, terr := dev.Type(); terr == nil && t == nm.DeviceTypeWiFi {
		if wl, werr := dev.Wireless(); werr == nil {
			if serr := wl.ScanAndWait(wifiScanRefreshTimeout); serr != nil {
				slog.Debug("device connections: scan refresh failed, using cached results",
					"iface", chi.URLParam(r, "iface"), "err", serr)
			} else {
				// Give NM a moment to reconcile the fresh scan into the AP list
				// and re-run its available-connections check before we read it.
				time.Sleep(wifiScanSettleDelay)
			}
		}
	}
	conns, err := dev.AvailableConnections()
	if err != nil {
		httpError(w, err)
		return
	}
	actives, err := s.nm.ActiveUUIDs()
	if err != nil {
		httpError(w, err)
		return
	}
	infos := make([]nm.ConnectionInfo, 0, len(conns))
	for _, c := range conns {
		info, err := c.Info()
		if err != nil {
			continue
		}
		if ref, ok := actives[info.UUID]; ok {
			info.Active = true
			info.Device = ref.Iface
		}
		infos = append(infos, info)
	}
	// Active profile first, then by name.
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Active != infos[j].Active {
			return infos[i].Active
		}
		return infos[i].ID < infos[j].ID
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{"connections": infos})
}

// upIgnoredDevice re-activates the last used connection of a device by looking
// up its saved profile (used by the dashboard quick toggle).
func (s *Server) upIgnoredDevice(iface string, dev *nm.Device) error {
	conns, err := dev.AvailableConnections()
	if err != nil {
		return err
	}
	if len(conns) == 0 {
		return errors.New("no connection profile for device")
	}
	_, err = s.nm.ActivateConnection(conns[0], dev, "/")
	return err
}
