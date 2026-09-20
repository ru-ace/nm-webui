package api

import (
	"context"
	"net/http"
	"time"

	"github.com/ru-ace/nm-webui/internal/nm"
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

	// External IP is best-effort and non-blocking (cached after first success).
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	externalIP, _ := s.resolver.Get(ctx, 1*time.Second)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"state":                 st.State,
		"state_name":            nm.NmStateMachineName(st.State),
		"hostname":              st.Hostname,
		"networkmanager_version": st.NMVersion,
		"connectivity":          statusText(st.Connectivity),
		"connectivity_code":     st.Connectivity,
		"networking_enabled":    st.Enable,
		"primary_gateway":       gateway,
		"external_ip":           externalIP,
		"time":                  time.Now().UTC(),
	})
}