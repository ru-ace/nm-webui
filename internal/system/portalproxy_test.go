package system

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const portalHTML = `<!DOCTYPE html>
<html>
<head>
  <base href="/">
  <script>window.bad = 1;</script>
  <style>body { background: url(img/bg.png); }</style>
  <meta http-equiv="refresh" content="5; url=/landing">
</head>
<body>
  <a href="/login?x=1">Login</a>
  <a href="#section">Section</a>
  <a href="javascript:void(0)">JS</a>
  <img src="logo.png" alt="logo" onerror="alert(1)">
  <img srcset="small.png 480w, big.png 1080w" alt="set">
  <button onclick="steal()">click</button>
  <iframe src="frame.html"></iframe>
  <object data="plugin.swf"></object>
  <embed src="movie.swf">
  <form action="/submit" method="get"><input name="user" value="a"></form>
  <form action="https://other.example/x" method="post">
    <input name="p" value="1">
    <button type="submit" formaction="/evil" formmethod="post">go</button>
  </form>
</body>
</html>`

var portalOrigin = "http://192.168.1.1"

func TestRewritePortalHTML(t *testing.T) {
	out, err := RewritePortalHTML([]byte(portalHTML), portalOrigin+"/portal/index.html", false)
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	s := string(out)

	for _, gone := range []string{"window.bad", "<base", "javascript:void(0)", "onerror", "onclick",
		"<iframe", "<object", "<embed", "formaction", "formmethod", "srcdoc"} {
		if strings.Contains(s, gone) {
			t.Errorf("rewritten page still contains %q", gone)
		}
	}

	wantLink := `href="` + ProxyBase + `?url=http%3A%2F%2F192.168.1.1%2Flogin%3Fx%3D1"`
	if !strings.Contains(s, wantLink) {
		t.Errorf("absolute link not rewritten (want %s):\n%s", wantLink, s)
	}
	wantImg := `src="` + ProxyBase + `?url=http%3A%2F%2F192.168.1.1%2Fportal%2Flogo.png"`
	if !strings.Contains(s, wantImg) {
		t.Errorf("relative img src not resolved/rewritten:\n%s", s)
	}
	wantCSS := `url('` + ProxyBase + `?url=http%3A%2F%2F192.168.1.1%2Fportal%2Fimg%2Fbg.png')`
	if !strings.Contains(s, wantCSS) {
		t.Errorf("style url() not rewritten:\n%s", s)
	}
	wantRefresh := `content="5; url=` + ProxyBase + `?url=http%3A%2F%2F192.168.1.1%2Flanding"`
	if !strings.Contains(s, wantRefresh) {
		t.Errorf("meta refresh not rewritten (want %s):\n%s", wantRefresh, s)
	}
	wantSrcset := `srcset="` + ProxyBase + `?url=http%3A%2F%2F192.168.1.1%2Fportal%2Fsmall.png 480w, ` + ProxyBase + `?url=http%3A%2F%2F192.168.1.1%2Fportal%2Fbig.png 1080w"`
	if !strings.Contains(s, wantSrcset) {
		t.Errorf("srcset not rewritten:\n%s", s)
	}

	if !strings.Contains(s, `name="url" value="http://192.168.1.1/submit"`) ||
		!strings.Contains(s, `name="_method" value="get"`) {
		t.Errorf("GET form hidden fields missing:\n%s", s)
	}
	if !strings.Contains(s, `name="url" value="https://other.example/x"`) {
		t.Errorf("cross-origin form hidden url missing:\n%s", s)
	}
	if !strings.Contains(s, `#section`) {
		t.Errorf("anchor fragment lost:\n%s", s)
	}
}

// TestRewritePortalHTMLAllowJS proves JS mode (sandboxed iframe): script
// elements, inline handlers and javascript: hrefs survive so JS-dependent
// portals work, while <base>/<iframe>/<object>/<embed> and per-control form
// overrides are still stripped and a telemetry snippet is injected.
func TestRewritePortalHTMLAllowJS(t *testing.T) {
	out, err := RewritePortalHTML([]byte(portalHTML), portalOrigin+"/portal/index.html", true)
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	s := string(out)

	for _, gone := range []string{"<base", "<iframe", "<object", "<embed",
		"formaction", "formmethod", "srcdoc"} {
		if strings.Contains(s, gone) {
			t.Errorf("rewritten page (js) still contains %q", gone)
		}
	}
	for _, kept := range []string{"window.bad", "onerror", "onclick", "javascript:void(0)"} {
		if !strings.Contains(s, kept) {
			t.Errorf("rewritten page (js) dropped %q", kept)
		}
	}
	if !strings.Contains(s, "__nmPortal") || !strings.Contains(s, "postMessage") {
		t.Errorf("telemetry script not injected:\n%s", s)
	}
	wantLink := `href="` + ProxyBase + `?url=http%3A%2F%2F192.168.1.1%2Flogin%3Fx%3D1"`
	if !strings.Contains(s, wantLink) {
		t.Errorf("absolute link not rewritten (js):\n%s", s)
	}
	if !strings.Contains(s, `name="url" value="http://192.168.1.1/submit"`) {
		t.Errorf("GET form hidden fields missing (js):\n%s", s)
	}
}

func TestFetchPortalRewritesHTMLAndHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><a href="/next">next</a></html>`))
	}))
	defer srv.Close()

	d := newTestDetector()
	body, ct, finalURL, status, err := d.FetchPortal(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if ct != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	if finalURL != srv.URL {
		t.Fatalf("final url = %q", finalURL)
	}
	want := ProxyBase + "?url=" + url.QueryEscape("http://"+hostOf(srv)+"/next")
	if !strings.Contains(string(body), want) {
		t.Fatalf("link not rewritten (want %s):\n%s", want, body)
	}
}

func TestFetchPortalPassThroughNonHTML(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\nprobe")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	}))
	defer srv.Close()

	d := newTestDetector()
	body, ct, _, _, err := d.FetchPortal(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if ct != "image/png" {
		t.Fatalf("content-type = %q", ct)
	}
	if string(body) != string(png) {
		t.Fatal("binary passed through altered")
	}
}

func TestFetchPortalSessionCookieSurvives(t *testing.T) {
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
	srv := httptest.NewServer(mux)
	defer srv.Close()

	d := newTestDetector()
	ctx := context.Background()
	if _, _, _, _, err := d.FetchPortal(ctx, http.MethodPost, srv.URL+"/login", url.Values{"user": {"bob"}}); err != nil {
		t.Fatalf("post: %v", err)
	}
	body, _, _, _, err := d.FetchPortal(ctx, http.MethodGet, srv.URL+"/welcome", nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(string(body), "welcome abc-bob") {
		t.Fatalf("cookie session not preserved across fetches:\n%s", body)
	}
}

// TestFetchPortalIgnoresSelfSignedTLS proves the proxy client skips TLS
// certificate verification: httptest.NewTLSServer uses a self-signed
// certificate that a default client would reject.
func TestFetchPortalIgnoresSelfSignedTLS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><a href="/ok">ok</a></html>`))
	}))
	defer srv.Close()

	d := newTestDetector()
	body, _, _, status, err := d.FetchPortal(context.Background(), http.MethodGet, srv.URL+"/page", nil)
	if err != nil {
		t.Fatalf("fetch over self-signed TLS: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if !strings.Contains(string(body), ProxyBase+"?url=") {
		t.Fatalf("expected rewritten page:\n%s", body)
	}
}

func TestFetchPortalRejectsBadScheme(t *testing.T) {
	d := newTestDetector()
	if _, _, _, _, err := d.FetchPortal(context.Background(), http.MethodGet, "file:///etc/passwd", nil); err == nil {
		t.Fatal("expected error for file:// target")
	}
	if _, _, _, _, err := d.FetchPortal(context.Background(), http.MethodGet, "javascript:alert(1)", nil); err == nil {
		t.Fatal("expected error for javascript: target")
	}
}

func hostOf(srv *httptest.Server) string {
	return strings.TrimPrefix(srv.URL, "http://")
}
