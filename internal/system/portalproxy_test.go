package system

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"
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
  <a href="/login?x=1" target="_blank">Login</a>
  <a href="#section">Section</a>
  <a href="javascript:void(0)">JS</a>
  <img src="logo.png" alt="logo" onerror="alert(1)">
  <img srcset="small.png 480w, big.png 1080w" alt="set">
  <button onclick="steal()">click</button>
  <iframe src="frame.html"></iframe>
  <object data="plugin.swf"></object>
  <embed src="movie.swf">
  <form action="/submit" method="get" target="_blank"><input name="user" value="a"></form>
  <form action="https://other.example/x" method="post">
    <input name="p" value="1">
    <button type="submit" formaction="/evil" formmethod="post" formtarget="_blank">go</button>
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

	for _, gone := range []string{"window.bad", "javascript:void(0)", "onerror", "onclick",
		"<iframe", "<object", "<embed", "formaction", "formmethod", "formtarget", "srcdoc", "target=\"_blank\"",
		"__nmPortalShim"} {
		if strings.Contains(s, gone) {
			t.Errorf("rewritten page still contains %q", gone)
		}
	}
	if bases := canonicalBases(s); len(bases) == 0 || bases[0]["href"] != "/" {
		t.Errorf("canonical <base href=\"/\"> not kept (got %v):\n%s", bases, s)
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

// test proves JS mode (sandboxed iframe): script
// elements, inline handlers and javascript: hrefs survive so JS-dependent
// portals work, while <base> is canonicalised and <iframe>/<object>/<embed>
// and per-control form overrides are still stripped and a telemetry snippet
// is injected.
func TestRewritePortalHTMLAllowJS(t *testing.T) {
	out, err := RewritePortalHTML([]byte(portalHTML), portalOrigin+"/portal/index.html", true)
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	s := string(out)

	for _, gone := range []string{"<iframe", "<object", "<embed",
		"formaction", "formmethod", "formtarget", "srcdoc", "target=\"_blank\""} {
		if strings.Contains(s, gone) {
			t.Errorf("rewritten page (js) still contains %q", gone)
		}
	}
	if bases := canonicalBases(s); len(bases) == 0 || bases[0]["href"] != "/" {
		t.Errorf("canonical <base href=\"/\"> not kept (js) (got %v):\n%s", bases, s)
	}
	for _, kept := range []string{"window.bad", "onerror", "onclick", "javascript:void(0)"} {
		if !strings.Contains(s, kept) {
			t.Errorf("rewritten page (js) dropped %q", kept)
		}
	}
	if !strings.Contains(s, "__nmPortal") || !strings.Contains(s, "postMessage") {
		t.Errorf("telemetry script not injected:\n%s", s)
	}
	if !strings.Contains(s, "__nmPortalShim") {
		t.Errorf("fetch/xhr shim not injected:\n%s", s)
	}
	if !strings.Contains(s, "finalUrl") {
		t.Errorf("telemetry does not report finalUrl:\n%s", s)
	}
	if !strings.Contains(s, `name="nm-final-url"`) ||
		!strings.Contains(s, `content="`+portalOrigin+`/portal/index.html"`) {
		t.Errorf("final-url meta marker not injected:\n%s", s)
	}
	wantLink := `href="` + ProxyBase + `?url=http%3A%2F%2F192.168.1.1%2Flogin%3Fx%3D1"`
	if !strings.Contains(s, wantLink) {
		t.Errorf("absolute link not rewritten (js):\n%s", s)
	}
	if !strings.Contains(s, `name="url" value="http://192.168.1.1/submit"`) {
		t.Errorf("GET form hidden fields missing (js):\n%s", s)
	}
}

// TestRewritePortalBaseCanonicalised proves <base> is canonicalised, not
// dropped: framework SPAs need an APP_BASE_HREF to bootstrap. The first
// declared target hint survives, the raw href (which would point back at the
// portal host) is replaced with the proxy-root-relative "/", and stray base
// tags after the first are rewritten to the same canonical form.
func TestRewritePortalBaseCanonicalised(t *testing.T) {
	in := `<html><head>` +
		`<base href="https://portal.example/login/" target="_top">` +
		`<base href="/secondary/">` +
		`<a href="index.html">link</a>` +
		`</head></html>`
	out, err := RewritePortalHTML([]byte(in), "http://192.168.1.1/portal/", true)
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	s := string(out)
	bases := canonicalBases(s)
	if len(bases) != 2 || bases[0]["href"] != "/" || bases[0]["target"] != "_top" || bases[1]["href"] != "/" {
		t.Errorf("unexpected canonical bases %v:\n%s", bases, s)
	}
	if strings.Contains(s, "portal.example") || strings.Contains(s, "secondary") {
		t.Errorf("original raw base href survived:\n%s", s)
	}
	wantLink := `href="` + ProxyBase + `?url=http%3A%2F%2F192.168.1.1%2Fportal%2Findex.html"`
	if !strings.Contains(s, wantLink) {
		t.Errorf("link not rewritten against portal base:\n%s", s)
	}
}

// TestPortalShimRunsBeforeAppScripts proves the fetch/XHR shim is injected as
// the first child of <body>, ahead of the app's module scripts, and that the
// rewrite stays idempotent: the shim comes before every script the app ships.
func TestPortalShimRunsBeforeAppScripts(t *testing.T) {
	in := `<html><head><base href="/"></head><body><app-root></app-root>` +
		`<script type="module" src="runtime/main.js"></script>` +
		`<script type="module" src="polyfills/main.js"></script>` +
		`<script type="module" src="main/main.js"></script>` +
		`</body></html>`
	out, err := RewritePortalHTML([]byte(in), portalOrigin+"/portal/index.html", true)
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	s := string(out)

	body := s[strings.Index(s, "<body"):]
	iShim := strings.Index(body, "__nmPortalShim")
	// Script srcs are rewritten to proxy URLs, so the app chunk paths arrive
	// with encoded slashes (runtime%2Fmain.js).
	iRuntime := strings.Index(body, "runtime%2Fmain.js")
	iPoly := strings.Index(body, "polyfills%2Fmain.js")
	iMain := strings.Index(body, "%2Fmain%2Fmain.js")
	if iShim < 0 {
		t.Fatalf("shim missing from body:\n%s", s)
	}
	if iShim > iRuntime || iShim > iPoly || iShim > iMain {
		t.Errorf("shim must precede app module scripts (shim at %d, runtime %d, polyfills %d, main %d):\n%s",
			iShim, iRuntime, iPoly, iMain, s)
	}
	if strings.Count(s, "if (window.__nmPortalShim) return;") != 1 {
		t.Errorf("shim injected more than once:\n%s", s)
	}

	// The shim must only fire once even if the script somehow runs again.
	if !strings.Contains(s, "if (window.__nmPortalShim) return;") {
		t.Errorf("shim lacks idempotency guard:\n%s", s)
	}
	// Lazy module chunks (dynamic import()) are fetched by the browser's
	// module loader past fetch/XHR, so the shim also patches the
	// HTMLScriptElement.src setter to reroute them through the proxy; relative
	// URLs resolve against the canonical <base href="/"> document base.
	for _, marker := range []string{"HTMLScriptElement.prototype", "docBaseURL", "Object.defineProperty(ScriptProto, \"src\""} {
		if !strings.Contains(s, marker) {
			t.Errorf("shim lacks chunk-loading patch (%q):\n%s", marker, s)
		}
	}
	if !strings.Contains(s, `name="nm-final-url"`) {
		t.Errorf("final-url meta marker missing for shim rebasing:\n%s", s)
	}
}

// TestFetchPortalForwardsRequestHeaders proves the client's request context
// reaches the upstream service: Accept overrides the proxy default so the API
// content-negotiates the format the SPA asked for, and the SPA's own
// authentication/correlation headers (Authorization, X-CorrelationId) ride
// along. Cookie must not be forwarded -- the proxy session is the host's own.
func TestFetchPortalForwardsRequestHeaders(t *testing.T) {
	var gotAccept, gotAuth, gotCorr, gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		gotAuth = r.Header.Get("Authorization")
		gotCorr = r.Header.Get("X-CorrelationId")
		gotCookie = r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	d := newTestDetector()
	hdr := http.Header{}
	hdr.Set("Accept", "application/json, text/plain, */*")
	hdr.Set("Authorization", "Bearer tok123")
	hdr.Set("X-CorrelationId", "corr-42")
	hdr.Set("Cookie", "sid=secret")
	_, _, _, _, err := d.FetchPortal(context.Background(), http.MethodGet, srv.URL+"/api", nil, &PortalForward{Headers: hdr})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if gotAccept != "application/json, text/plain, */*" {
		t.Errorf("Accept = %q, want SPA Accept forwarded", gotAccept)
	}
	if gotAuth != "Bearer tok123" {
		t.Errorf("Authorization = %q, want forwarded", gotAuth)
	}
	if gotCorr != "corr-42" {
		t.Errorf("X-CorrelationId = %q, want forwarded", gotCorr)
	}
	if gotCookie != "" {
		t.Errorf("Cookie = %q, must not be forwarded", gotCookie)
	}
}

// TestFetchPortalForwardsRawJSONBody proves non-form POST payloads (JSON API
// calls made by the SPA) pass through to the upstream verbatim with their
// original Content-Type preserved.
func TestFetchPortalForwardsRawJSONBody(t *testing.T) {
	var gotBody, gotCT, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotCT = r.Header.Get("Content-Type")
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	d := newTestDetector()
	hdr := http.Header{}
	hdr.Set("Content-Type", "application/json;charset=UTF-8")
	fwd := &PortalForward{Headers: hdr, Body: []byte(`{"product":"free-internet"}`)}
	if _, _, _, status, err := d.FetchPortal(context.Background(), http.MethodPost, srv.URL+"/api", nil, fwd); err != nil {
		t.Fatalf("fetch: %v", err)
	} else if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotCT != "application/json;charset=UTF-8" {
		t.Errorf("Content-Type = %q, want preserved", gotCT)
	}
	if gotBody != `{"product":"free-internet"}` {
		t.Errorf("body = %q, want passed through verbatim", gotBody)
	}
}

func TestFetchPortalRewritesHTMLAndHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><a href="/next">next</a></html>`))
	}))
	defer srv.Close()

	d := newTestDetector()
	body, ct, finalURL, status, err := d.FetchPortal(context.Background(), http.MethodGet, srv.URL, nil, nil)
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

// TestFetchPortalFollowsRedirects proves upstream HTTP redirects are absorbed
// by the Go client on the host: the proxy returns the final document (200),
// the final URL becomes the rewrite base, and the JS-mode meta marker exposes
// the destination to the telemetry in the sandboxed iframe.
func TestFetchPortalFollowsRedirects(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/landing", http.StatusFound)
	})
	mux.HandleFunc("/landing", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><a href="/next">next</a></html>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	d := newTestDetectorJS()
	body, _, finalURL, status, err := d.FetchPortal(context.Background(), http.MethodGet, srv.URL+"/", nil, nil)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 after redirect", status)
	}
	if finalURL != srv.URL+"/landing" {
		t.Fatalf("final url = %q, want %q", finalURL, srv.URL+"/landing")
	}
	s := string(body)
	want := ProxyBase + "?url=" + url.QueryEscape("http://"+hostOf(srv)+"/next")
	if !strings.Contains(s, want) {
		t.Fatalf("link not rewritten against final base:\n%s", s)
	}
	if !strings.Contains(s, `content="`+srv.URL+`/landing"`) {
		t.Fatalf("final-url meta marker missing:\n%s", s)
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
	body, ct, _, _, err := d.FetchPortal(context.Background(), http.MethodGet, srv.URL, nil, nil)
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
	if _, _, _, _, err := d.FetchPortal(ctx, http.MethodPost, srv.URL+"/login", url.Values{"user": {"bob"}}, nil); err != nil {
		t.Fatalf("post: %v", err)
	}
	body, _, _, _, err := d.FetchPortal(ctx, http.MethodGet, srv.URL+"/welcome", nil, nil)
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
	body, _, _, status, err := d.FetchPortal(context.Background(), http.MethodGet, srv.URL+"/page", nil, nil)
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
	if _, _, _, _, err := d.FetchPortal(context.Background(), http.MethodGet, "file:///etc/passwd", nil, nil); err == nil {
		t.Fatal("expected error for file:// target")
	}
	if _, _, _, _, err := d.FetchPortal(context.Background(), http.MethodGet, "javascript:alert(1)", nil, nil); err == nil {
		t.Fatal("expected error for javascript: target")
	}
}

func hostOf(srv *httptest.Server) string {
	return strings.TrimPrefix(srv.URL, "http://")
}

// canonicalBases parses rewritten HTML and returns the attributes of every
// <base> element in document order.
func canonicalBases(s string) []map[string]string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return nil
	}
	var out []map[string]string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "base") {
			m := map[string]string{}
			for _, a := range n.Attr {
				m[strings.ToLower(a.Key)] = a.Val
			}
			out = append(out, m)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}
