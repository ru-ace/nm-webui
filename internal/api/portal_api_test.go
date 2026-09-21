package api

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
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
