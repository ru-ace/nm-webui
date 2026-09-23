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
	portalStateTTL  = 30 * time.Second
	portalProbeWait = 5 * time.Second
)

// resolvedState applies the captive-portal re-verification to an NM-derived
// connectivity verdict (see system.ResolveEffective). Used by endpoints that
// serve connectivity without an explicit probe fetch of their own.
func (s *Server) resolvedState(ctx context.Context, nmState string) string {
	if nmState != "portal" {
		return nmState
	}
	pc, err := s.portal.Probe(ctx)
	return system.ResolveEffective(nmState, pc, err)
}

// portalPayload merges NetworkManager's connectivity verdict (authoritative)
// with the detector's probe result, which contributes the sign-in page URL.
func (s *Server) portalPayload(ctx context.Context) (map[string]interface{}, error) {
	st, stErr := s.nm.Status()
	pc, probeErr := s.portal.Probe(ctx)

	state := "unknown"
	if stErr == nil {
		switch st.Connectivity {
		case nm.ConnectivityPortal:
			state = "portal"
		case nm.ConnectivityFull:
			state = "online"
		case nm.ConnectivityLimited:
			state = "limited"
		case nm.ConnectivityNone:
			state = "none"
		}
	}
	// When NM has not run its check yet, fall back to the probe verdict.
	if probeErr == nil && state == "unknown" {
		state = pc.State
	}
	// NM's "portal" verdict is re-verified by our probe, which shares the portal
	// session cookie jar: right after a successful sign-in the probe proves
	// "online" even though NM's periodic check may still lag behind.
	state = system.ResolveEffective(state, pc, probeErr)

	portalURL := ""
	if state == "portal" {
		portalURL = pc.PortalURL
		if portalURL == "" {
			// The gateway usually hosts the sign-in page.
			if g, err := s.nm.PrimaryIPv4Gateway(); err == nil && g != "" {
				portalURL = "http://" + g + "/"
			}
		}
	}

	nmConnect := uint32(nm.ConnectivityUnknown)
	nmText := "unknown"
	if stErr == nil {
		nmConnect = st.Connectivity
		nmText = statusText(st.Connectivity)
	}
	str := func(v string) interface{} {
		if v == "" {
			return nil
		}
		return v
	}
	return map[string]interface{}{
		"state":               state,
		"portal_url":          str(portalURL),
		"origin":              str(system.OriginOf(portalURL)),
		"probe_url":           str(pc.ProbeURL),
		"status":              pc.Status,
		"checked_at":          pc.CheckedAt,
		"nm_connectivity":     nmConnect,
		"nm_connectivity_text": nmText,
	}, nil
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

	// Re-run NM's own connectivity check unless it already reports online.
	if st, err := s.nm.Status(); err == nil && st.Connectivity != nm.ConnectivityFull {
		if v, err := s.nm.CheckConnectivity(); err == nil {
			slog.Info("connectivity re-check", "state", statusText(v))
		}
	}
	// Force a fresh probe so the portal URL is up to date after sign-in.
	if _, err := s.portal.Refresh(ctx); err != nil {
		slog.Warn("captive portal probe failed", "err", err)
	}

	// Re-resolve the effective verdict and push it to every connected client
	// when it changes (NM is the trigger, the fresh probe is the judge).
	if st, err := s.nm.Status(); err == nil {
		if state, changed := s.verdict.Recheck(ctx, statusText(st.Connectivity)); changed {
			s.hub.Publish("connectivity_changed", map[string]interface{}{
				"connectivity": codeForState(state),
				"status":       state,
			})
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