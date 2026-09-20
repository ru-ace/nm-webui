package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/ru-ace/nm-webui/internal/nm"
)

// wifiDevice resolves the {iface} param as a wireless device.
func (s *Server) wifiDevice(r *http.Request) (*nm.Device, *nm.Wireless, error) {
	dev, err := s.deviceFromRequest(r)
	if err != nil {
		return nil, nil, err
	}
	w, err := dev.Wireless()
	if err != nil {
		return nil, nil, err
	}
	return dev, w, nil
}

func (s *Server) handleWifiScan(w http.ResponseWriter, r *http.Request) {
	_, wl, err := s.wifiDevice(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "wifi device not found")
		return
	}
	iface := chi.URLParam(r, "iface")
	if err := wl.RequestScan(); err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "scanning", "iface": iface})
}

// groupedNetwork is one entry per SSID (best BSSID), with all BSSIDs attached.
type groupedNetwork struct {
	SSID      string      `json:"ssid"`
	Saved     bool        `json:"saved"`
	Signal    int         `json:"signal"`
	BSSID     string      `json:"bssid,omitempty"`
	Security  string      `json:"security"`
	Band      string      `json:"band,omitempty"`
	Channel   uint32      `json:"channel"`
	Frequency uint32      `json:"frequency"`
	BSSIDs    []nm.APInfo `json:"bssids,omitempty"`
}

func groupBySSID(aps []nm.APInfo, group bool) []interface{} {
	if !group {
		out := make([]interface{}, len(aps))
		for i := range aps {
			out[i] = aps[i]
		}
		return out
	}
	bySSID := map[string]*groupedNetwork{}
	var order []string
	for _, ap := range aps {
		if ap.SSID == "" || ap.SSID == "(hidden)" {
			continue
		}
		g, ok := bySSID[ap.SSID]
		if !ok {
			g = &groupedNetwork{SSID: ap.SSID, Saved: ap.Saved, Security: ap.Security}
			bySSID[ap.SSID] = g
			order = append(order, ap.SSID)
		}
		// prefer stronger BSSID as representation
		if ap.Signal > g.Signal {
			g.Signal = ap.Signal
			g.Saved = ap.Saved
			g.BSSID = ap.BSSID
			g.Security = ap.Security
			g.Band = ap.Band
			g.Channel = ap.Channel
			g.Frequency = ap.Frequency
		}
		g.BSSIDs = append(g.BSSIDs, ap)
	}
	out := make([]interface{}, 0, len(order))
	for _, ssid := range order {
		g := bySSID[ssid]
		sort.Slice(g.BSSIDs, func(i, j int) bool { return g.BSSIDs[i].Signal > g.BSSIDs[j].Signal })
		out = append(out, g)
	}
	return out
}

func (s *Server) handleWifiNetworks(w http.ResponseWriter, r *http.Request) {
	_, wl, err := s.wifiDevice(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "wifi device not found")
		return
	}
	aps, err := wl.AccessPoints()
	if err != nil {
		httpError(w, err)
		return
	}
	infos := make([]nm.APInfo, 0, len(aps))
	saved, err := s.savedWiFiSSIDs()
	if err != nil {
		httpError(w, err)
		return
	}
	for _, ap := range aps {
		info, err := ap.Info()
		if err != nil {
			continue
		}
		info.Saved = saved[info.SSID]
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Signal > infos[j].Signal })

	group := r.URL.Query().Get("group") != "0"
	networks := groupBySSID(infos, group)
	writeJSON(w, http.StatusOK, map[string]interface{}{"networks": networks})
}

func (s *Server) savedWiFiSSIDs() (map[string]bool, error) {
	conns, err := s.nm.ListConnections()
	if err != nil {
		return nil, err
	}
	saved := make(map[string]bool)
	for _, conn := range conns {
		info, err := conn.Info()
		if err != nil {
			continue
		}
		if info.IsWiFi && info.SSID != "" {
			saved[info.SSID] = true
		}
	}
	return saved, nil
}

func (s *Server) handleWifiStatus(w http.ResponseWriter, r *http.Request) {
	dev, _, err := s.wifiDevice(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "wifi device not found")
		return
	}
	st, err := s.nm.WiFiStatus(dev)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) handleWifiConnect(w http.ResponseWriter, r *http.Request) {
	dev, _, err := s.wifiDevice(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "wifi device not found")
		return
	}
	var body struct {
		SSID     string `json:"ssid"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	body.SSID = strings.TrimSpace(body.SSID)
	if body.SSID == "" {
		writeErr(w, http.StatusBadRequest, "ssid is required")
		return
	}

	iface := chi.URLParam(r, "iface")

	// Start the connection in the background, push progress via SSE.
	go func() {
		s.hub.Publish("wifi_connecting", map[string]interface{}{
			"iface": iface,
			"ssid":  body.SSID,
		})
		_, _, err := s.nm.ConnectWiFi(dev, body.SSID, body.Password, s.timeout)
		switch {
		case err == nil:
			s.hub.Publish("wifi_connected", map[string]interface{}{
				"iface":   iface,
				"ssid":    body.SSID,
				"message": "connected",
			})
		case err == nm.ErrNotConnectedSSID:
			s.hub.Publish("wifi_connected", map[string]interface{}{
				"iface":   iface,
				"ssid":    body.SSID,
				"message": "already connected",
			})
		default:
			s.hub.Publish("wifi_failed", map[string]interface{}{
				"iface": iface,
				"ssid":  body.SSID,
				"error": err.Error(),
			})
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "connecting",
		"ssid":   body.SSID,
	})
}
