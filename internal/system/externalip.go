package system

import (
	"context"
	"net"
	"sync"
	"time"
)

// TXT queries for external IP use Google's myaddr over DNS (UDP only, no HTTP
// dependency): the answer is a single IPv4/IPv6 literal.
const externalIPTXT = "o-o.myaddr.l.google.com"

// Resolver caches the external IP resolved via DNS TXT records.
type Resolver struct {
	mu    sync.Mutex
	last  string
	stale time.Time
	ttl   time.Duration
}

// NewResolver creates a resolver with a 5 minute cache TTL.
func NewResolver() *Resolver {
	return &Resolver{ttl: 5 * time.Minute}
}

// Get returns the cached external IP if fresh, otherwise performs a TXT DNS
// lookup. It never blocks the caller for longer than timeout.
func (r *Resolver) Get(ctx context.Context, timeout time.Duration) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.last != "" && time.Since(r.stale) < r.ttl {
		return r.last, nil
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	records, err := net.DefaultResolver.LookupTXT(ctx, externalIPTXT)
	if err != nil {
		if r.last != "" {
			return r.last, nil // serve stale data offline
		}
		return "", err
	}
	for _, rec := range records {
		if ip := net.ParseIP(rec); ip != nil {
			r.last = rec
			r.stale = time.Now()
			return rec, nil
		}
	}
	if r.last != "" {
		return r.last, nil
	}
	return "", errNoExternalIP
}

type errExternal string

func (e errExternal) Error() string { return string(e) }

const errNoExternalIP = errExternal("no external IP resolved")