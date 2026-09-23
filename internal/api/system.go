package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/ru-ace/nm-webui/internal/nm"
	"github.com/ru-ace/nm-webui/internal/system"
)

func (s *Server) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	st, err := s.nm.Status()
	if err != nil {
		httpError(w, err)
		return
	}
	gateway := ""
	if g, err := s.nm.PrimaryIPv4Gateway(); err == nil {
		gateway = g
	}

	// The advertised verdict re-verifies NM's "portal" with our probe, so after
	// a successful sign-in the status flips to online as soon as the session is
	// seen — NM's periodic check may lag behind.
	state := s.resolvedState(r.Context(), statusText(st.Connectivity))

	external := system.ExternalIP{Status: "unavailable", CheckedAt: time.Now()}
	if state == "online" {
		// External IP is only meaningful while we consider connectivity online.
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		external, _ = s.resolver.Get(ctx, 2*time.Second)
		cancel()
	} else {
		s.resolver.Invalidate()
	}
	var externalIP interface{}
	if external.IP != "" {
		externalIP = external.IP
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"state":                  st.State,
		"state_name":             nm.NmStateMachineName(st.State),
		"hostname":               st.Hostname,
		"networkmanager_version": st.NMVersion,
		"connectivity":           state,
		"connectivity_code":      codeForState(state),
		"networking_enabled":     st.Enable,
		"primary_gateway":        gateway,
		"external_ip":            externalIP,
		"external_ip_status":     external.Status,
		"external_ip_checked_at": external.CheckedAt,
		"external_ip_country":    external.Country,
		"external_ip_city":       external.City,
		"external_ip_region":     external.Region,
		"external_ip_isp":        external.ISP,
		"external_ip_org":        external.Org,
		"external_ip_asn":        external.ASN,
		"external_ip_timezone":   external.Timezone,
		"time":                   time.Now().UTC(),
	})
}

func (s *Server) handleExternalIPRefresh(w http.ResponseWriter, r *http.Request) {
	st, err := s.nm.Status()
	if err != nil {
		httpError(w, err)
		return
	}
	// Gate on our effective verdict (probe-confirmed online), not just NM's raw
	// state: right after a successful portal sign-in we may already be online
	// while NM's periodic check is still catching up.
	if s.resolvedState(r.Context(), statusText(st.Connectivity)) != "online" {
		writeErr(w, http.StatusConflict, "internet connectivity is not online")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	external, err := s.resolver.Refresh(ctx, 2*time.Second)
	if err != nil {
		slog.Error("external IP refresh failed", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]interface{}{
			"external_ip":            nil,
			"external_ip_status":     external.Status,
			"external_ip_checked_at": external.CheckedAt,
			"error":                  "external IP refresh failed",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"external_ip":            external.IP,
		"external_ip_status":     external.Status,
		"external_ip_checked_at": external.CheckedAt,
		"external_ip_country":    external.Country,
		"external_ip_city":       external.City,
		"external_ip_region":     external.Region,
		"external_ip_isp":        external.ISP,
		"external_ip_org":        external.Org,
		"external_ip_asn":        external.ASN,
		"external_ip_timezone":   external.Timezone,
	})
}
