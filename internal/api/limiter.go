package api

import (
	"sync"
	"time"
)

// attemptLimiter is a sliding-window rate limiter for failed attempts (e.g.
// wrong passwords). After `limit` failures within the window the caller is
// blocked until blockedUntil(), after which the failure history is cleared.
type attemptLimiter struct {
	mu      sync.Mutex
	marks   []time.Time
	window  time.Duration
	limit   int
	blocked time.Time
}

func newAttemptLimiter(window time.Duration, limit int) *attemptLimiter {
	return &attemptLimiter{window: window, limit: limit}
}

func (l *attemptLimiter) blockedUntil() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.blocked
}

// blocked reports whether the caller is currently rate limited.
func (l *attemptLimiter) isBlocked() bool {
	return time.Now().Before(l.blockedUntil())
}

// fail records a failed attempt and blocks when the threshold is hit.
func (l *attemptLimiter) fail() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.marks = append(l.marks, now)
	cut := now.Add(-l.window)
	n := 0
	for _, t := range l.marks {
		if t.After(cut) {
			l.marks[n] = t
			n++
		}
	}
	l.marks = l.marks[:n]
	if n >= l.limit {
		l.blocked = now.Add(30 * time.Second)
		l.marks = nil
	}
}

// clear resets the failure history after a successful attempt.
func (l *attemptLimiter) clear() {
	l.mu.Lock()
	l.marks = nil
	l.blocked = time.Time{}
	l.mu.Unlock()
}
