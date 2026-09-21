package system

import (
	"context"
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
