package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ru-ace/nm-webui/internal/nm"
	"github.com/ru-ace/nm-webui/internal/system"
)

const (
	portalProbeWait = 5 * time.Second
)

// resolvedState returns the effective connectivity for endpoints that serve a
// status without running a probe of their own. When the background monitor is
// running its latest verdict (kept fresh by the periodic probe) is returned
// without any network access; otherwise a direct probe falls back to it. See
// system.ResolveEffective: the probe is the source of truth for "portal" and
// "online", NetworkManager only survives as a link-level fallback.
func (s *Server) resolvedState(ctx context.Context, nmState string) string {
	if s.monitor != nil {
		if p := s.monitor.Payload(); p != nil {
			if st, ok := p["state"].(string); ok && st != "" {
				return st
			}
		}
	}
	pc, err := s.portal.Probe(ctx)
	return system.ResolveEffective(nmState, pc, err)
}

// portalPayload is the captive-portal JSON payload: probe verdict plus the
// sign-in page URL, with NetworkManager reduced to an informational field.
func (s *Server) portalPayload(ctx context.Context) (map[string]interface{}, error) {
	if s.monitor != nil {
		if p := s.monitor.Payload(); p != nil {
			return p, nil
		}
	}
	return s.portalPayloadProbe(ctx)
}

// portalPayloadProbe builds a fresh payload (used when no monitor is
// available, e.g. in unit tests).
func (s *Server) portalPayloadProbe(ctx context.Context) (map[string]interface{}, error) {
	st, stErr := s.nm.Status()
	pc, probeErr := s.portal.Probe(ctx)

	nmConn := uint32(nm.ConnectivityUnknown)
	nmText := "unknown"
	if stErr == nil {
		nmConn = st.Connectivity
		nmText = statusText(st.Connectivity)
	}
	gateway := ""
	if g, err := s.nm.PrimaryIPv4Gateway(); err == nil {
		gateway = g
	}
	return portalPayloadFrom(pc, probeErr, nmConn, nmText, gateway), nil
}

// portalPayloadFrom builds the shared captive-portal payload from raw inputs,
// so the monitor and the REST endpoints stay consistent.
func portalPayloadFrom(pc system.PortalState, probeErr error, nmConn uint32, nmText, gateway string) map[string]interface{} {
	state := system.ResolveEffective(nmText, pc, probeErr)

	portalURL := ""
	if state == "portal" {
		portalURL = pc.PortalURL
		if portalURL == "" && gateway != "" {
			// The gateway usually hosts the sign-in page.
			portalURL = "http://" + gateway + "/"
		}
	}
	str := func(v string) interface{} {
		if v == "" {
			return nil
		}
		return v
	}
	return map[string]interface{}{
		"state":                state,
		"portal_url":           str(portalURL),
		"origin":               str(system.OriginOf(portalURL)),
		"probe_url":            str(pc.ProbeURL),
		"status":               pc.Status,
		"checked_at":           pc.CheckedAt,
		"nm_connectivity":      nmConn,
		"nm_connectivity_text": nmText,
	}
}

func (s *Server) handleCaptivePortalStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), portalProbeWait+3*time.Second)
	defer cancel()
	payload, err := s.portalPayload(ctx)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleCaptivePortalCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	if s.monitor != nil {
		// Force a fresh probe; changed results are pushed over SSE.
		s.monitor.Force(ctx)
	} else {
		if _, err := s.portal.Refresh(ctx); err != nil {
			slog.Warn("captive portal probe failed", "err", err)
		}
	}

	payload, err := s.portalPayload(ctx)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handlePortalProxyGet(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("url")
	if target == "" {
		writeErr(w, http.StatusBadRequest, "missing url parameter")
		return
	}
	s.servePortalFetch(w, r, http.MethodGet, target, nil)
}

func (s *Server) handlePortalProxyPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeErr(w, http.StatusBadRequest, "malformed form")
		return
	}
	target := r.PostFormValue("url")
	if target == "" {
		writeErr(w, http.StatusBadRequest, "missing url parameter")
		return
	}
	form := r.PostForm
	form.Del("url")
	method := http.MethodPost
	if strings.EqualFold(r.PostFormValue("_method"), "get") {
		method = http.MethodGet
	}
	form.Del("_method")
	s.servePortalFetch(w, r, method, target, form)
}

func (s *Server) servePortalFetch(w http.ResponseWriter, r *http.Request, method, target string, form url.Values) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	body, ct, finalURL, status, err := s.portal.FetchPortal(ctx, method, target, form)
	if err != nil {
		slog.Warn("portal fetch failed", "url", target, "err", err)
		writeErr(w, http.StatusBadGateway, "portal fetch failed: "+err.Error())
		return
	}
	h := w.Header()
	h.Set("Content-Type", ct)
	h.Set("X-Final-URL", finalURL)
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
