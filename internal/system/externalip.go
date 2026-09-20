package system

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// geoIPURL resolves the external address and its ISP/geo details in one
// request: called without an IP path, ip-api.com returns the caller's address
// ("query") together with the requested fields.
const geoIPURL = "http://ip-api.com/json?fields=status,message,query,country,city,regionName,isp,org,as,timezone"

// ExternalIP is the result of an external IP lookup plus optional geo/ISP
// details reported by ip-api.com (empty when the geo lookup fails).
type ExternalIP struct {
	IP        string
	Country   string
	City      string
	Region    string
	ISP       string
	Org       string
	ASN       string
	Timezone  string
	Status    string
	CheckedAt time.Time
}

// Resolver caches the external IP and its ip-api.com country/ISP details.
type Resolver struct {
	mu     sync.Mutex
	cached ExternalIP
	ttl    time.Duration
	client *http.Client
}

// NewResolver creates a resolver with a 5 minute cache TTL.
func NewResolver() *Resolver {
	return &Resolver{
		ttl:    5 * time.Minute,
		client: &http.Client{},
	}
}

// Get returns a fresh or cached external IP. Failed lookups never return a
// stale address as if it were current.
func (r *Resolver) Get(ctx context.Context, timeout time.Duration) (ExternalIP, error) {
	return r.lookup(ctx, timeout, false)
}

// Refresh forces a new lookup and replaces the cached address on success.
func (r *Resolver) Refresh(ctx context.Context, timeout time.Duration) (ExternalIP, error) {
	return r.lookup(ctx, timeout, true)
}

// Invalidate removes the cached address after connectivity is lost.
func (r *Resolver) Invalidate() {
	r.mu.Lock()
	r.cached = ExternalIP{}
	r.mu.Unlock()
}

// geoInfo is the ip-api.com response we consume.
type geoInfo struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	Query      string `json:"query"`
	Country    string `json:"country"`
	RegionName string `json:"regionName"`
	City       string `json:"city"`
	ISP        string `json:"isp"`
	Org        string `json:"org"`
	AS         string `json:"as"`
	Timezone   string `json:"timezone"`
}

func (r *Resolver) lookup(ctx context.Context, timeout time.Duration, force bool) (ExternalIP, error) {
	r.mu.Lock()
	if !force && r.cached.IP != "" && time.Since(r.cached.CheckedAt) < r.ttl {
		cached := r.cached
		cached.Status = "cached"
		r.mu.Unlock()
		return cached, nil
	}
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, geoIPURL, nil)
	if err != nil {
		return ExternalIP{Status: "unavailable", CheckedAt: time.Now()}, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return ExternalIP{Status: "unavailable", CheckedAt: time.Now()}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ExternalIP{Status: "unavailable", CheckedAt: time.Now()}, fmt.Errorf("external IP endpoint returned %s", resp.Status)
	}

	var info geoInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4*1024)).Decode(&info); err != nil {
		return ExternalIP{Status: "unavailable", CheckedAt: time.Now()}, fmt.Errorf("decode external IP response: %w", err)
	}
	if info.Status != "success" {
		msg := info.Message
		if msg == "" {
			msg = "unknown error"
		}
		return ExternalIP{Status: "unavailable", CheckedAt: time.Now()}, fmt.Errorf("external IP endpoint failed: %s", msg)
	}
	if net.ParseIP(info.Query) == nil {
		return ExternalIP{Status: "unavailable", CheckedAt: time.Now()}, fmt.Errorf("external IP endpoint returned invalid IP")
	}

	external := ExternalIP{
		IP:        info.Query,
		Country:   info.Country,
		City:      info.City,
		Region:    info.RegionName,
		ISP:       info.ISP,
		Org:       info.Org,
		ASN:       info.AS,
		Timezone:  info.Timezone,
		Status:    "fresh",
		CheckedAt: time.Now(),
	}

	r.mu.Lock()
	r.cached = external
	r.mu.Unlock()
	return external, nil
}