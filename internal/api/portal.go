package api

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

// handlePortalProxyOptions answers CORS preflights for the portal proxy
// (module scripts and fetch/XHR from sandboxed frames trigger them). Access
// is granted only to opaque ("null") origins — the proxy responses thereby
// stay unreadable to arbitrary websites, so neither listener becomes an open
// CORS relay, while framework SPAs sandboxed without allow-same-origin can
// still load their script bundles.
func (s *Server) handlePortalProxyOptions(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "Content-Type")
	h.Set("Access-Control-Max-Age", "600")
	h.Set("Access-Control-Expose-Headers", "X-Final-Url")
	h.Set("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
	if r.Header.Get("Origin") == "null" {
		h.Set("Access-Control-Allow-Origin", "null")
	}
	w.WriteHeader(http.StatusNoContent)
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
	// CORS for sandboxed portal frames: opaque-origin documents (sandbox
	// without allow-same-origin, data: URLs) must be able to read proxied
	// module scripts and subresources. Only "null" origins are granted;
	// Vary: Origin keeps caches (browsers, intermediaries) honest.
	h.Set("Vary", "Origin")
	if r.Header.Get("Origin") == "null" {
		h.Set("Access-Control-Allow-Origin", "null")
		h.Set("Access-Control-Expose-Headers", "X-Final-Url")
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// PortalProxyHandler returns the handler tree of the dedicated portal-proxy
// listener. It exposes nothing but the proxy endpoints: with the sandboxed
// portal iframe served from this origin (allow-same-origin), portal content
// becomes same-origin with this origin only — the router admin API must not
// be reachable here, otherwise framable portal pages could drive it.
func (s *Server) PortalProxyHandler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(requestLogger)
	r.Options("/api/v1/captive-portal/proxy", s.handlePortalProxyOptions)
	r.Get("/api/v1/captive-portal/proxy", s.handlePortalProxyGet)
	r.Post("/api/v1/captive-portal/proxy", s.handlePortalProxyPost)
	return r
}

// PortalProxyBase returns the client-visible origin of the dedicated
// portal-proxy listener: it mirrors the scheme and host the client used for
// the admin UI and substitutes the proxy port. When the client reaches the
// admin through a port-mapped forward (ssh -L, docker -p publishing different
// host ports), the same numeric offset is applied to the proxy port so both
// listeners stay on the same reachable interface. Returns "" when the
// dedicated listener is disabled (the SPA then falls back to the admin's own
// origin and keeps the fully opaque iframe sandbox).
func (s *Server) PortalProxyBase(r *http.Request) string {
	port := s.cfg.PortalProxyPort()
	if port == "" {
		return ""
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if h, hp, err := net.SplitHostPort(r.Host); err == nil {
		host = h
		if clientPort, cerr := strconv.Atoi(hp); cerr == nil {
			if basePort, berr := strconv.Atoi(port); berr == nil {
				if adminPort, aerr := strconv.Atoi(s.cfg.ListenPort()); aerr == nil {
					port = strconv.Itoa(basePort + (clientPort - adminPort))
				}
			}
		}
	}
	return scheme + "://" + net.JoinHostPort(host, port)
}
