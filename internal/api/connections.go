package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/godbus/dbus/v5"
	"github.com/ru-ace/nm-webui/internal/nm"
)

type ipSetting struct {
	Method  string   `json:"method"`
	Address string   `json:"address"`
	Prefix  uint32   `json:"prefix"`
	Gateway string   `json:"gateway"`
	DNS     []string `json:"dns"`
}

type connRequest struct {
	ID          string     `json:"id"`
	Interface   string     `json:"interface"`
	Autoconnect *bool      `json:"autoconnect"`
	Type        string     `json:"type"`
	SSID        string     `json:"ssid"`
	Password    string     `json:"password"`
	APN         string     `json:"apn"`
	Number      string     `json:"number"`
	UserName    string     `json:"username"`
	PIN         string     `json:"pin"`
	IPv4        *ipSetting `json:"ipv4"`
	IPv6        *ipSetting `json:"ipv6"`
}

func (s *Server) handleConnectionsList(w http.ResponseWriter, _ *http.Request) {
	conns, err := s.nm.ListConnections()
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
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].IsWiFi != infos[j].IsWiFi {
			return infos[i].IsWiFi
		}
		return infos[i].ID < infos[j].ID
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{"connections": infos})
}

func normalizeMethod(m string) string {
	switch m {
	case "", "auto", "dhcp":
		return "auto"
	case "manual", "static":
		return "manual"
	case "disabled", "off":
		return "disabled"
	default:
		return m
	}
}

func (s *Server) handleConnectionsCreate(w http.ResponseWriter, r *http.Request) {
	var req connRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ID == "" {
		req.ID = req.Interface
	}
	if req.ID == "" {
		writeErr(w, http.StatusBadRequest, "id or interface is required")
		return
	}
	if req.Type == "" {
		req.Type = "ethernet"
	}
	uuid, err := nm.NewUUID()
	if err != nil {
		httpError(w, err)
		return
	}

	settings := map[string]map[string]dbus.Variant{
		"connection": {
			"type":        dbus.MakeVariant(mapConType(req.Type)),
			"id":          dbus.MakeVariant(req.ID),
			"uuid":        dbus.MakeVariant(uuid),
			"autoconnect": dbus.MakeVariant(req.Autoconnect == nil || *req.Autoconnect),
		},
	}
	if req.Interface != "" {
		settings["connection"]["interface-name"] = dbus.MakeVariant(req.Interface)
	}
	if mapConType(req.Type) == "802-11-wireless" {
		if strings.TrimSpace(req.SSID) == "" {
			writeErr(w, http.StatusBadRequest, "ssid is required for wifi connections")
			return
		}
		settings["802-11-wireless"] = map[string]dbus.Variant{
			"ssid": dbus.MakeVariant([]byte(req.SSID)),
			"mode": dbus.MakeVariant("infrastructure"),
		}
		if req.Password != "" {
			settings["802-11-wireless"]["security"] = dbus.MakeVariant("802-11-wireless-security")
			settings["802-11-wireless-security"] = map[string]dbus.Variant{
				"key-mgmt": dbus.MakeVariant("wpa-psk"),
				"psk":      dbus.MakeVariant(req.Password),
			}
		}
	}
	if mapConType(req.Type) == "gsm" {
		gsm := map[string]dbus.Variant{}
		if req.APN != "" {
			gsm["apn"] = dbus.MakeVariant(req.APN)
		}
		number := req.Number
		if number == "" {
			number = "*99#"
		}
		gsm["number"] = dbus.MakeVariant(number)
		if req.UserName != "" {
			gsm["username"] = dbus.MakeVariant(req.UserName)
		}
		if req.Password != "" {
			gsm["password"] = dbus.MakeVariant(req.Password)
			gsm["password-flags"] = dbus.MakeVariant(uint32(0))
		}
		if req.PIN != "" {
			gsm["pin"] = dbus.MakeVariant(req.PIN)
			gsm["pin-flags"] = dbus.MakeVariant(uint32(0))
		}
		settings["gsm"] = gsm
	}
	if req.IPv4 != nil || req.IPv6 != nil {
		settings["ipv4"] = ipv4FromRequest(req.IPv4)
		settings["ipv6"] = ipv6FromRequest(req.IPv6)
	}

	conn, err := s.nm.AddConnection(settings)
	if err != nil {
		httpError(w, err)
		return
	}
	info, _ := conn.Info()
	writeJSON(w, http.StatusCreated, info)
}

// mapConType converts friendly type names to NM D-Bus identifiers.
func mapConType(t string) string {
	switch strings.ToLower(t) {
	case "wifi", "wireless", "802-11-wireless":
		return "802-11-wireless"
	case "gsm", "mobile-broadband", "mobile", "4g", "5g":
		return "gsm"
	case "bridge":
		return "bridge"
	default:
		return "802-3-ethernet"
	}
}

func ipv4FromRequest(p *ipSetting) map[string]dbus.Variant {
	if p == nil {
		return map[string]dbus.Variant{"method": dbus.MakeVariant("auto")}
	}
	return nm.IPv4Block(normalizeMethod(p.Method), nm.StaticConfig{
		Address: p.Address, Prefix: p.Prefix, Gateway: p.Gateway, DNS: p.DNS,
	})
}

func ipv6FromRequest(p *ipSetting) map[string]dbus.Variant {
	if p == nil {
		return map[string]dbus.Variant{"method": dbus.MakeVariant("auto")}
	}
	return nm.IPv6Block(normalizeMethod(p.Method), nm.StaticConfig{
		Address: p.Address, Prefix: p.Prefix, Gateway: p.Gateway, DNS: p.DNS,
	})
}

func (s *Server) handleConnectionsDelete(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")
	if err := s.nm.DeleteProfile(uuid); err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleConnectionsUpdate(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")
	var req connRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Autoconnect != nil {
		if err := s.nm.SetProfileAutoconnect(uuid, *req.Autoconnect); err != nil {
			httpError(w, err)
			return
		}
	}
	if req.IPv4 != nil {
		if err := s.nm.UpdateProfileIPv4(uuid, ipv4FromRequest(req.IPv4)); err != nil {
			httpError(w, err)
			return
		}
	}
	if req.IPv6 != nil {
		if err := s.nm.UpdateProfileIPv6(uuid, ipv6FromRequest(req.IPv6)); err != nil {
			httpError(w, err)
			return
		}
	}
	if req.ID != "" || req.Type != "" || req.Interface != "" {
		conn, err := s.nm.ConnectionByUUID(uuid)
		if err != nil {
			httpError(w, err)
			return
		}
		settings, err := conn.GetSettings()
		if err != nil {
			httpError(w, err)
			return
		}
		connection := settings["connection"]
		if req.ID != "" {
			connection["id"] = dbus.MakeVariant(req.ID)
		}
		if req.Type != "" {
			connection["type"] = dbus.MakeVariant(mapConType(req.Type))
		}
		if req.Interface != "" {
			connection["interface-name"] = dbus.MakeVariant(req.Interface)
		} else if req.ID != "" {
			delete(connection, "interface-name")
		}
		if err := conn.Update(settings); err != nil {
			httpError(w, err)
			return
		}
	}
	if req.APN != "" || req.Number != "" || req.UserName != "" || req.PIN != "" || req.Password != "" {
		conn, err := s.nm.ConnectionByUUID(uuid)
		if err != nil {
			httpError(w, err)
			return
		}
		settings, err := conn.GetSettings()
		if err != nil {
			httpError(w, err)
			return
		}
		gsm := settings["gsm"]
		if gsm == nil {
			gsm = map[string]dbus.Variant{}
			settings["gsm"] = gsm
		}
		if req.APN != "" {
			gsm["apn"] = dbus.MakeVariant(req.APN)
		}
		if req.Number != "" {
			gsm["number"] = dbus.MakeVariant(req.Number)
		}
		if req.UserName != "" {
			gsm["username"] = dbus.MakeVariant(req.UserName)
		}
		if req.Password != "" {
			gsm["password"] = dbus.MakeVariant(req.Password)
			gsm["password-flags"] = dbus.MakeVariant(uint32(0))
		}
		if req.PIN != "" {
			gsm["pin"] = dbus.MakeVariant(req.PIN)
			gsm["pin-flags"] = dbus.MakeVariant(uint32(0))
		}
		if err := conn.Update(settings); err != nil {
			httpError(w, err)
			return
		}
	}
	if mapConType(req.Type) == "802-11-wireless" {
		conn, err := s.nm.ConnectionByUUID(uuid)
		if err != nil {
			httpError(w, err)
			return
		}
		settings, err := conn.GetSettings()
		if err != nil {
			httpError(w, err)
			return
		}
		wl := settings["802-11-wireless"]
		if wl == nil {
			wl = map[string]dbus.Variant{}
			settings["802-11-wireless"] = wl
		}
		changed := false
		if req.SSID != "" {
			wl["ssid"] = dbus.MakeVariant([]byte(req.SSID))
			changed = true
		}
		// A non-empty password stores a new PSK; an empty one leaves the
		// existing credential untouched (the UI never sends the stored key).
		if req.Password != "" {
			wl["security"] = dbus.MakeVariant("802-11-wireless-security")
			sec := settings["802-11-wireless-security"]
			if sec == nil {
				sec = map[string]dbus.Variant{}
				settings["802-11-wireless-security"] = sec
			}
			sec["key-mgmt"] = dbus.MakeVariant("wpa-psk")
			sec["psk"] = dbus.MakeVariant(req.Password)
			changed = true
		}
		if changed {
			if err := conn.Update(settings); err != nil {
				httpError(w, err)
				return
			}
		}
	}
	conn, _ := s.nm.ConnectionByUUID(uuid)
	if conn == nil {
		writeErr(w, http.StatusNotFound, "connection not found")
		return
	}
	info, err := conn.Info()
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleConnectionsUp(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")
	ac, err := s.nm.ActivateProfile(uuid)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "connecting",
		"path":   string(ac.Path()),
	})
}

func (s *Server) handleConnectionsDown(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")
	if err := s.nm.DeactivateProfile(uuid); err != nil {
		if errors.Is(err, nm.ErrNoActiveConn) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "already-disconnected"})
			return
		}
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnecting"})
}
