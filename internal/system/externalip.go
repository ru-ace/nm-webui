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

const externalIPURL = "https://api64.ipify.org?format=json"

// ExternalIP is the result of an external IP lookup.
type ExternalIP struct {
	IP        string
	Status    string
	CheckedAt time.Time
}

// Resolver caches the external IP resolved via an HTTPS endpoint.
type Resolver struct {
	mu          sync.Mutex
	last        string
	lastChecked time.Time
	ttl         time.Duration
	client      *http.Client
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

// Refresh forces a new HTTPS lookup and replaces the cached address on success.
func (r *Resolver) Refresh(ctx context.Context, timeout time.Duration) (ExternalIP, error) {
	return r.lookup(ctx, timeout, true)
}

// Invalidate removes the cached address after connectivity is lost.
func (r *Resolver) Invalidate() {
	r.mu.Lock()
	r.last = ""
	r.lastChecked = time.Time{}
	r.mu.Unlock()
}

func (r *Resolver) lookup(ctx context.Context, timeout time.Duration, force bool) (ExternalIP, error) {
	r.mu.Lock()
	if !force && r.last != "" && time.Since(r.lastChecked) < r.ttl {
		result := ExternalIP{IP: r.last, Status: "cached", CheckedAt: r.lastChecked}
		r.mu.Unlock()
		return result, nil
	}
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, externalIPURL, nil)
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

	var body struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4*1024)).Decode(&body); err != nil {
		return ExternalIP{Status: "unavailable", CheckedAt: time.Now()}, fmt.Errorf("decode external IP response: %w", err)
	}
	if net.ParseIP(body.IP) == nil {
		return ExternalIP{Status: "unavailable", CheckedAt: time.Now()}, fmt.Errorf("external IP endpoint returned invalid IP")
	}

	checkedAt := time.Now()
	r.mu.Lock()
	r.last = body.IP
	r.lastChecked = checkedAt
	r.mu.Unlock()
	return ExternalIP{IP: body.IP, Status: "fresh", CheckedAt: checkedAt}, nil
}
