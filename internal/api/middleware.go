package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ru-ace/nm-webui/internal/config"
)

// authGuard is a tiny sliding-window rate limiter for failed logins.
type authGuard struct {
	mu      sync.Mutex
	marks   []time.Time
	window  time.Duration
	limit   int
	blocked time.Time
}

func newAuthGuard() *authGuard {
	return &authGuard{window: time.Minute, limit: 8}
}

func (g *authGuard) blockedUntil() time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.blocked
}

func (g *authGuard) fail() {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	g.marks = append(g.marks, now)
	cut := now.Add(-g.window)
	n := 0
	for _, t := range g.marks {
		if t.After(cut) {
			g.marks[n] = t
			n++
		}
	}
	g.marks = g.marks[:n]
	if n >= g.limit {
		g.blocked = now.Add(30 * time.Second)
		g.marks = nil
	}
}

func (g *authGuard) clear() {
	g.mu.Lock()
	g.marks = nil
	g.blocked = time.Time{}
	g.mu.Unlock()
}

var authRealm = "nm-webui"

// BasicAuth protects handlers when a password is configured. Credentials are
// compared in constant time. Failed attempts are rate limited.
func BasicAuth(cfg *config.Config) func(http.Handler) http.Handler {
	guard := newAuthGuard()
	expected := sha256.Sum256([]byte(cfg.AuthPass))
	sum := func(token []byte) [32]byte { return sha256.Sum256(token) }

	return func(next http.Handler) http.Handler {
		if !cfg.AuthEnabled() {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if until := guard.blockedUntil(); time.Now().Before(until) {
				w.Header().Set("Retry-After", "30")
				writeErr(w, http.StatusTooManyRequests, "too many failed attempts, try again later")
				return
			}
			user, pass, ok := parseBasicAuth(r.Header.Get("Authorization"))
			_ = user
			if ok {
				got := sum([]byte(pass))
				if subtle.ConstantTimeCompare(expected[:], got[:]) == 1 {
					guard.clear()
					next.ServeHTTP(w, r)
					return
				}
			}
			guard.fail()
			slog.Warn("unauthorized request", "remote", r.RemoteAddr, "path", r.URL.Path)
			w.Header().Set("WWW-Authenticate", `Basic realm="`+authRealm+`"`)
			writeErr(w, http.StatusUnauthorized, "authentication required")
		})
	}
}

// parseBasicAuth extracts (username, password) from a Basic authorization header.
func parseBasicAuth(header string) (string, string, bool) {
	const prefix = "Basic "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", "", false
	}
	raw, err := base64.StdEncoding.DecodeString(header[len(prefix):])
	if err != nil {
		return "", "", false
	}
	i := strings.IndexByte(string(raw), ':')
	if i < 0 {
		return "", "", false
	}
	return string(raw[:i]), string(raw[i+1:]), true
}