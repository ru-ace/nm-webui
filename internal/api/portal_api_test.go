package api

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ru-ace/nm-webui/internal/config"
	"github.com/ru-ace/nm-webui/internal/events"
	"github.com/ru-ace/nm-webui/internal/system"
)

// newPortalOnlyTestServer builds an API router with only the captive-portal
// proxy routes registered, so the tests run without NetworkManager/D-Bus.
func newPortalOnlyTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	d := system.NewDetector(nil, 30*time.Second, 5*time.Second, jar, true)
	s := &Server{portal: d}
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/captive-portal/proxy", s.handlePortalProxyGet)
		r.Post("/captive-portal/proxy", s.handlePortalProxyPost)
		r.Options("/captive-portal/proxy", s.handlePortalProxyOptions)
	})
	return httptest.NewServer(r)
}

func doProxyGet(t *testing.T, base, target string) (int, http.Header, string) {
	t.Helper()
	resp, err := http.Get(base + "/api/v1/captive-portal/proxy?url=" + url.QueryEscape(target))
	if err != nil {
		t.Fatalf("proxy get: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, resp.Header, string(body)
}

func TestPortalProxyGetRewritesHTML(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><a href="/next">next</a></html>`))
	}))
	defer up.Close()

	srv := newPortalOnlyTestServer(t)
	defer srv.Close()

	status, hdr, body := doProxyGet(t, srv.URL, up.URL+"/")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if ct := hdr.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type = %q", ct)
	}
	if final := hdr.Get("X-Final-URL"); final == "" {
		t.Fatal("missing X-Final-URL header")
	}
	want := system.ProxyBase + "?url=" + url.QueryEscape(up.URL+"/next")
	if !strings.Contains(body, want) {
		t.Fatalf("rewritten link missing (want %s):\n%s", want, body)
	}
}

func TestPortalProxyPostFormAndSession(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		user := r.PostFormValue("user")
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: "abc-" + user})
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><a href="/welcome">go</a></html>`))
	})
	mux.HandleFunc("/welcome", func(w http.ResponseWriter, r *http.Request) {
		sid := "none"
		if c, err := r.Cookie("sid"); err == nil {
			sid = c.Value
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><p>welcome ` + sid + `</p></html>`))
	})
	up := httptest.NewServer(mux)
	defer up.Close()

	srv := newPortalOnlyTestServer(t)
	defer srv.Close()

	// POST /login through the proxy with form fields + _method marker.
	form := url.Values{
		"url":     {up.URL + "/login"},
		"_method": {"post"},
		"user":    {"bob"},
	}
	resp, err := http.PostForm(srv.URL+"/api/v1/captive-portal/proxy", form)
	if err != nil {
		t.Fatalf("proxy post: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("post status = %d, want 200", resp.StatusCode)
	}

	// The portal cookie set during login must survive in the shared jar.
	status, _, body := doProxyGet(t, srv.URL, up.URL+"/welcome")
	if status != http.StatusOK {
		t.Fatalf("welcome status = %d, want 200", status)
	}
	if !strings.Contains(body, "welcome abc-bob") {
		t.Fatalf("cookie session not preserved across proxy calls:\n%s", body)
	}
}

func TestPortalProxyMissingURL(t *testing.T) {
	srv := newPortalOnlyTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/captive-portal/proxy")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %s)", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "missing url parameter") {
		t.Fatalf("unexpected error body: %s", body)
	}
}

func TestPortalProxyPostMissingURL(t *testing.T) {
	srv := newPortalOnlyTestServer(t)
	defer srv.Close()

	resp, err := http.PostForm(srv.URL+"/api/v1/captive-portal/proxy", url.Values{"_method": {"post"}})
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %s)", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "missing url parameter") {
		t.Fatalf("unexpected error body: %s", body)
	}
}

// The SPA's fetch/XHR shim sends POSTs to the proxy with the target in the
// query (proxyURL builds "/api/v1/captive-portal/proxy?url=..."); JSON bodies
// and the SPA's own Authentication/API headers must reach the upstream.
func TestPortalProxyPostJSONFromQuery(t *testing.T) {
	var gotBody, gotCT, gotAuth string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotCT = r.Header.Get("Content-Type")
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer up.Close()

	srv := newPortalOnlyTestServer(t)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/captive-portal/proxy?url="+url.QueryEscape(up.URL+"/api"),
		strings.NewReader(`{"product":"free-internet"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer tok999")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("json post: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body: %s", resp.StatusCode, body)
	}
	if gotBody != `{"product":"free-internet"}` {
		t.Errorf("upstream body = %q, want passthrough", gotBody)
	}
	if gotCT != "application/json;charset=UTF-8" {
		t.Errorf("upstream content-type = %q", gotCT)
	}
	if gotAuth != "Bearer tok999" {
		t.Errorf("upstream Authorization = %q, want forwarded", gotAuth)
	}
}

func portalUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body>ok</body></html>`))
	}))
	t.Cleanup(up.Close)
	return up
}

// doProxyRequest sends a request with arbitrary headers to the portal proxy.
func doProxyRequest(t *testing.T, method, base, target string, hdr map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, base+"/api/v1/captive-portal/proxy?url="+url.QueryEscape(target), nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s proxy: %v", method, err)
	}
	return resp
}

// The sandboxed portal frame runs under an opaque origin ("null"). Proxied
// module scripts and subresources must be readable there, so the proxy grants
// CORS to "null" origins and exposes the final-URL header.
func TestPortalProxyCORSGrantsOpaqueOrigin(t *testing.T) {
	up := portalUpstream(t)
	srv := newPortalOnlyTestServer(t)
	defer srv.Close()

	resp := doProxyRequest(t, http.MethodGet, srv.URL, up.URL+"/", map[string]string{"Origin": "null"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "null" {
		t.Fatalf("ACAO = %q, want null", got)
	}
	if got := resp.Header.Get("Vary"); !strings.Contains(got, "Origin") {
		t.Fatalf("Vary = %q, want it to contain Origin", got)
	}
	if got := resp.Header.Get("Access-Control-Expose-Headers"); !strings.Contains(got, "X-Final-Url") {
		t.Fatalf("Expose-Headers = %q, want X-Final-Url", got)
	}
}

// An arbitrary website must not be able to read proxied responses through the
// router, so origins other than "null" get no CORS grant.
func TestPortalProxyCORSRefusesOtherOrigins(t *testing.T) {
	up := portalUpstream(t)
	srv := newPortalOnlyTestServer(t)
	defer srv.Close()

	for name, origin := range map[string]string{
		"absent":   "",
		"website":  "https://evil.example",
		"samesite": "https://portal.example",
	} {
		hdr := map[string]string{}
		if origin != "" {
			hdr["Origin"] = origin
		}
		resp := doProxyRequest(t, http.MethodGet, srv.URL, up.URL+"/", hdr)
		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("%s: ACAO = %q, want none", name, got)
		}
		resp.Body.Close()
	}
}

func TestPortalProxyOptionsPreflight(t *testing.T) {
	srv := newPortalOnlyTestServer(t)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodOptions, srv.URL+"/api/v1/captive-portal/proxy", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "null")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("options: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "null" {
		t.Fatalf("preflight ACAO = %q, want null", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(got, "GET") ||
		!strings.Contains(got, "POST") || !strings.Contains(got, "OPTIONS") {
		t.Fatalf("Allow-Methods = %q, want GET/POST/OPTIONS", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Content-Type") {
		t.Fatalf("Allow-Headers = %q, want Content-Type", got)
	}

	// Non-null origins get no grant even in preflight.
	req2, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/v1/captive-portal/proxy", nil)
	req2.Header.Set("Origin", "https://evil.example")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if got := resp2.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("preflight ACAO = %q, want none for non-null origin", got)
	}
}

// The dedicated portal-proxy listener derives its client-visible origin from
// the admin request, mirroring scheme+host and applying the port offset.
func TestPortalProxyBaseDerivation(t *testing.T) {
	cfg := &config.Config{Listen: "0.0.0.0:8090", PortalProxyListen: "0.0.0.0:8091"}
	s := &Server{cfg: cfg}

	cases := []struct {
		name string
		host string
		tls  bool
		want string
	}{
		{"direct", "192.168.1.5:8090", false, "http://192.168.1.5:8091"},
		{"forwarded", "127.0.0.1:18090", false, "http://127.0.0.1:18091"},
		{"no port", "router.lan", false, "http://router.lan:8091"},
		{"ipv6", "[::1]:8090", false, "http://[::1]:8091"},
		{"tls", "router.lan:8090", true, "https://router.lan:8091"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "http://"+tc.host+"/api/v1/system/features", nil)
			r.Host = tc.host
			if tc.tls {
				r.TLS = &tls.ConnectionState{}
			}
			if got := s.PortalProxyBase(r); got != tc.want {
				t.Fatalf("PortalProxyBase = %q, want %q", got, tc.want)
			}
		})
	}

	// Disabled listener → empty base (SPA falls back to the admin origin).
	cfg2 := &config.Config{Listen: "0.0.0.0:8090"}
	s2 := &Server{cfg: cfg2}
	r := httptest.NewRequest(http.MethodGet, "http://192.168.1.5:8090/", nil)
	r.Host = "192.168.1.5:8090"
	if got := s2.PortalProxyBase(r); got != "" {
		t.Fatalf("PortalProxyBase with disabled listener = %q, want empty", got)
	}
}

// The dedicated handler tree must expose the proxy and nothing else: no admin
// API, no auth challenges — only the proxy endpoints live on that origin.
func TestPortalProxyHandlerServesOnlyProxy(t *testing.T) {
	up := portalUpstream(t)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	d := system.NewDetector(nil, 30*time.Second, 5*time.Second, jar, true)
	s := &Server{
		portal: d,
		cfg:    &config.Config{Listen: "0.0.0.0:8090", PortalProxyListen: "0.0.0.0:8091"},
	}
	handler := httptest.NewServer(s.PortalProxyHandler())
	defer handler.Close()

	// Proxy works without credentials of any kind.
	resp := doProxyRequest(t, http.MethodGet, handler.URL, up.URL+"/", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("proxy status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// Admin routes must not exist on this origin.
	for _, path := range []string{"/api/v1/system/status", "/api/v1/devices", "/", "/index.html"} {
		r2, err := http.Get(handler.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		r2.Body.Close()
		if r2.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404 (admin API must not leak onto the portal origin)", path, r2.StatusCode)
		}
	}
}

// newMonitorTestServer builds a server with a PortalMonitor and only the
// captive-portal status routes, so the endpoints can be exercised over HTTP
// without NetworkManager/D-Bus.
func newMonitorTestServer(t *testing.T, probe system.ProbeFn, nmState func() string) *httptest.Server {
	t.Helper()
	hub := events.NewHub(50)
	mon := NewPortalMonitor(probe, nmState, nil, hub, time.Hour, time.Second)
	s := &Server{monitor: mon}
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/system/captive-portal", s.handleCaptivePortalStatus)
		r.Post("/system/captive-portal/check", s.handleCaptivePortalCheck)
	})
	return httptest.NewServer(r)
}

func TestCaptivePortalStatusServesMonitorPayload(t *testing.T) {
	var calls int
	srv := newMonitorTestServer(t, func(ctx context.Context) (system.PortalState, error) {
		calls++
		return system.PortalState{State: "portal", PortalURL: "http://10.0.0.1/login"}, nil
	}, func() string { return "online" })
	defer srv.Close()

	// Prime the monitor (as a real boot would via Start or the Recheck
	// button), then the GET must serve the remembered payload.
	if _, err := http.Post(srv.URL+"/api/v1/system/captive-portal/check", "application/json", nil); err != nil {
		t.Fatalf("check post: %v", err)
	}

	resp, err := http.Get(srv.URL + "/api/v1/system/captive-portal")
	if err != nil {
		t.Fatalf("status get: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body: %s", resp.StatusCode, body)
	}
	// Probe-first: NM would say "online", but the payload must expose portal.
	if !strings.Contains(string(body), `"state":"portal"`) {
		t.Fatalf("status does not carry probe-first portal state: %s", body)
	}
	if !strings.Contains(string(body), `"nm_connectivity_text":"online"`) {
		t.Fatalf("status lost the NM informational field: %s", body)
	}
	if calls == 0 {
		t.Fatal("probe was never forced")
	}
}

func TestCaptivePortalCheckForcesProbe(t *testing.T) {
	var calls int
	srv := newMonitorTestServer(t, func(ctx context.Context) (system.PortalState, error) {
		calls++
		return system.PortalState{State: "online"}, nil
	}, func() string { return "limited" })
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/system/captive-portal/check", "application/json", nil)
	if err != nil {
		t.Fatalf("check post: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("check status = %d, body: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), `"state":"online"`) {
		t.Fatalf("check response does not expose probe-first online: %s", body)
	}
	if calls == 0 {
		t.Fatal("check did not force a probe")
	}
}
