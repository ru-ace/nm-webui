package system

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// PortalState reports captive-portal detection results.
type PortalState struct {
	State     string    `json:"state"`      // portal|online|none|unknown
	PortalURL string    `json:"portal_url"` // sign-in page when State == "portal"
	Origin    string    `json:"origin"`     // scheme://host of PortalURL
	ProbeURL  string    `json:"probe_url"`  // the probe URL that saw the portal
	Status    string    `json:"status"`     // fresh|cached|unavailable
	CheckedAt time.Time `json:"checked_at"`
}

// DefaultPortalCheckURLs are probe endpoints reachable without Google (which
// is unreliable from RU networks). Each must answer with a tiny "success"
// marker (204/plain text/Apple's HTML Success page) when the network is open
// and with an intercepted login page when a captive portal is present.
var DefaultPortalCheckURLs = []string{
	"http://captive.apple.com/hotspot-detect.html",
	"http://detectportal.firefox.com/canonical.html",
}

// Detector probes well-known captive-portal check endpoints from the host
// side — the host is the Wi-Fi client, so it is the one that "sees" the
// portal — and caches the result for a short TTL. The shared *http.Client
// carries a cookie jar so any portal session established while probing is
// reused by the portal proxy (internal/system/portalproxy.go).
type Detector struct {
	mu      sync.Mutex
	cached  PortalState
	ttl     time.Duration
	timeout time.Duration
	maxBody int64
	client  *http.Client
	urls    []string
	allowJS bool
}

// NewDetector creates a detector. urls is the list of probe URLs tried in
// order: the first URL that proves open internet wins; otherwise the first
// URL that proves a portal provides the PortalURL. When urls is empty the
// defaults are used.
//
// TLS certificate verification is disabled on the shared client: captive
// portals routinely serve self-signed, expired or otherwise invalid
// certificates and must still be reachable both for probing and proxying.
//
// allowJS controls whether proxied documents may keep executable script
// content (<script>, on* handlers). When false, scripts and dangerous
// attribute handlers are stripped from rewritten pages; when true they are
// kept for the sandboxed-iframe renderer.
func NewDetector(urls []string, ttl, timeout time.Duration, jar http.CookieJar, allowJS bool) *Detector {
	if len(urls) == 0 {
		urls = DefaultPortalCheckURLs
	}
	return &Detector{
		ttl:     ttl,
		timeout: timeout,
		maxBody: 64 * 1024,
		allowJS: allowJS,
		// #nosec G402 -- captive portals are untrusted endpoints; skipping
		// certificate verification is the whole point of a portal proxy.
		client: &http.Client{
			Jar: jar,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		},
		urls: urls,
	}
}

// Client returns the shared HTTP client (cookie jar included).
func (d *Detector) Client() *http.Client { return d.client }

// Probe returns a cached or fresh detection result.
func (d *Detector) Probe(ctx context.Context) (PortalState, error) {
	return d.lookup(ctx, false)
}

// Refresh forces a fresh detection and replaces the cache on success.
func (d *Detector) Refresh(ctx context.Context) (PortalState, error) {
	return d.lookup(ctx, true)
}

// Invalidate clears the cached result.
func (d *Detector) Invalidate() {
	d.mu.Lock()
	d.cached = PortalState{}
	d.mu.Unlock()
}

func (d *Detector) lookup(ctx context.Context, force bool) (PortalState, error) {
	d.mu.Lock()
	if !force && d.cached.Status != "" && time.Since(d.cached.CheckedAt) < d.ttl {
		c := d.cached
		c.Status = "cached"
		d.mu.Unlock()
		return c, nil
	}
	d.mu.Unlock()

	res, err := d.probe(ctx)
	res.CheckedAt = time.Now()
	if err != nil {
		res.Status = "unavailable"
		res.State = "unknown"
	} else {
		res.Status = "fresh"
	}
	d.mu.Lock()
	d.cached = res
	d.mu.Unlock()
	return res, err
}

// probe walks the configured probe URLs and classifies each response. The
// first "online" proof wins immediately; otherwise the first portal sighting
// is kept. When nothing answers, the state is "none".
func (d *Detector) probe(ctx context.Context) (PortalState, error) {
	var hit PortalState
	hit.State = "unknown"
	for _, u := range d.urls {
		reqCtx, cancel := context.WithTimeout(ctx, d.timeout)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, u, nil)
		if err != nil {
			cancel()
			continue
		}
		req.Header.Set("Accept", "*/*")
		req.Header.Set("User-Agent", "nm-webui captive-portal-check/1.0")
		resp, err := d.client.Do(req)
		cancel()
		if err != nil {
			continue
		}
		final := resp.Request.URL.String()
		ct := resp.Header.Get("Content-Type")
		body, err := io.ReadAll(io.LimitReader(resp.Body, d.maxBody+1))
		resp.Body.Close()
		if err != nil {
			continue
		}
		switch classifyProbe(resp.StatusCode, ct, body) {
		case "online":
			hit = PortalState{State: "online", ProbeURL: u}
			return hit, nil
		case "portal":
			if hit.State != "portal" {
				hit = PortalState{State: "portal", PortalURL: final, ProbeURL: u}
			}
		}
	}
	if hit.State == "portal" {
		hit.Origin = OriginOf(hit.PortalURL)
		return hit, nil
	}
	hit.State = "none"
	return hit, nil
}

// classifyProbe decides "online" vs "portal" for a single probe response.
// The well-known check endpoints answer 204, a plain "success" marker or
// Apple's tiny HTML success page when the network is open, and are replaced
// by an intercepted login page when a captive portal stands in the way.
func classifyProbe(status int, contentType string, body []byte) string {
	if status == http.StatusNoContent {
		return "online"
	}
	ct := strings.ToLower(contentType)
	bodyLow := strings.ToLower(string(body))
	if strings.Contains(ct, "text/html") ||
		strings.HasPrefix(bodyLow, "<!doctype html") ||
		strings.HasPrefix(bodyLow, "<!doctype") ||
		strings.Contains(bodyLow, "<html") {
		if strings.Contains(bodyLow, "<title>success</title>") {
			return "online"
		}
		return "portal"
	}
	if status >= 200 && status < 300 {
		// Non-HTML 2xx (plain "success", JSON, binary, ...) means open internet.
		return "online"
	}
	return "portal"
}

// OriginOf extracts scheme://host (with port when present) from a URL.
func OriginOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// validateURL restricts proxying to http/https targets.
func validateURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported url scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("url has no host")
	}
	return u, nil
}
