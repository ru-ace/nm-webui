package system

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestDetector(urls ...string) *Detector {
	jar, _ := cookiejar.New(nil)
	return NewDetector(urls, 30*time.Second, 2*time.Second, jar, false)
}

// newTestDetectorJS is the JS-allowed variant (sandboxed-iframe mode).
func newTestDetectorJS(urls ...string) *Detector {
	jar, _ := cookiejar.New(nil)
	return NewDetector(urls, 30*time.Second, 2*time.Second, jar, true)
}

func TestProbeOnline204(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	st, err := newTestDetector(srv.URL).Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if st.State != "online" {
		t.Fatalf("state = %q, want online", st.State)
	}
}

func TestProbeOnlinePlainText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("success"))
	}))
	defer srv.Close()

	st, err := newTestDetector(srv.URL).Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if st.State != "online" {
		t.Fatalf("state = %q, want online", st.State)
	}
}

func TestProbeOnlineAppleSuccess(t *testing.T) {
	body := `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">
<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	st, err := newTestDetector(srv.URL).Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if st.State != "online" {
		t.Fatalf("state = %q, want online (apple success page)", st.State)
	}
}

func TestProbePortalInterceptedHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><form action="/login"><input name="user"></form></body></html>`))
	}))
	defer srv.Close()

	st, err := newTestDetector(srv.URL).Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if st.State != "portal" {
		t.Fatalf("state = %q, want portal", st.State)
	}
	if st.PortalURL != srv.URL {
		t.Fatalf("portal_url = %q, want %q", st.PortalURL, srv.URL)
	}
	if st.Origin != "http://"+srv.Listener.Addr().String() {
		t.Fatalf("origin = %q", st.Origin)
	}
}

func TestProbeNoneOnUnreachable(t *testing.T) {
	st, err := newTestDetector("http://127.0.0.1:1/probe", "http://127.0.0.1:2/probe").Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if st.State != "none" {
		t.Fatalf("state = %q, want none", st.State)
	}
}

func TestProbePortalFallbackAfterFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><form name="login"></html>`))
	}))
	defer srv.Close()

	st, err := newTestDetector("http://127.0.0.1:1/probe", srv.URL).Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if st.State != "portal" {
		t.Fatalf("state = %q, want portal (found on second probe)", st.State)
	}
}

func TestProbePortalMiddlebox307(t *testing.T) {
	// The realistic captive flow (e.g. the Yota middlebox): every plain-HTTP
	// request answers 307 with "Location: https://portal/..." and the portal
	// page is served on the redirect target. The follow must be classified as
	// a portal with the detour URL as the sign-in page.
	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>operator sign-in</body></html>`))
	}))
	defer portal.Close()

	check := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Via", "1.0 middlebox")
		http.Redirect(w, r, portal.URL+"/light?redirurl=x", http.StatusTemporaryRedirect)
	}))
	defer check.Close()

	st, err := newTestDetector(check.URL).Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if st.State != "portal" {
		t.Fatalf("state = %q, want portal (middlebox 307 → sign-in page)", st.State)
	}
	if st.PortalURL != portal.URL+"/light?redirurl=x" {
		t.Fatalf("portal_url = %q, want portal URL", st.PortalURL)
	}
}

func TestProbePortalRedirectLoop(t *testing.T) {
	// A checker that bounces the probe between two URLs forever: after the
	// hop limit the last 3xx is classified as portal, never "none".
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	base := "http://" + ln.Addr().String()
	go func() {
		for {
			c, aErr := ln.Accept()
			if aErr != nil {
				return
			}
			_, _ = fmt.Fprintf(c,
				"HTTP/1.1 307 Temporary Redirect\r\nLocation: %s/a\r\nContent-Length: 0\r\n\r\n",
				base)
			_ = c.Close()
		}
	}()

	st, err := newTestDetector(base + "/a").Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if st.State != "portal" {
		t.Fatalf("state = %q, want portal (redirect loop)", st.State)
	}
	if st.PortalURL == "" {
		t.Fatalf("portal_url must be set for a looped portal")
	}
}

func TestProbePortalRedirectFollowFails(t *testing.T) {
	// The middlebox redirected the check to a portal that then refuses the
	// connection (TLS RST, dead host). The redirect itself is portal evidence.
	check := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:1/light", http.StatusFound)
	}))
	defer check.Close()

	st, err := newTestDetector(check.URL).Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if st.State != "portal" {
		t.Fatalf("state = %q, want portal (redirect target unreachable)", st.State)
	}
	if want := "http://127.0.0.1:1/light"; st.PortalURL != want {
		t.Fatalf("portal_url = %q, want %q", st.PortalURL, want)
	}
}

func TestProbeNoneOnConnectionReset(t *testing.T) {
	// The very first request dies without any response (RST/EOF). No redirect
	// was ever pointed at a portal, so this must not be classified as portal.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, aErr := ln.Accept()
			if aErr != nil {
				return
			}
			_ = c.Close() // EOF; no response ever arrives
		}
	}()

	st, err := newTestDetector("http://" + ln.Addr().String() + "/probe").Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if st.State != "none" {
		t.Fatalf("state = %q, want none (RST/EOF is no portal evidence)", st.State)
	}
}

func TestDetectorCaches(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	d := newTestDetector(srv.URL)
	d.ttl = time.Hour
	ctx := context.Background()
	if _, err := d.Probe(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Probe(ctx); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("probe hits = %d, want 1 (second call cached)", hits)
	}
	if _, err := d.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if hits != 2 {
		t.Fatalf("probe hits after Refresh = %d, want 2", hits)
	}
}
