package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/ru-ace/nm-webui/internal/system"
)

// powerActions abstracts privileged host power commands so handlers can be
// tested without rebooting anything.
type powerActions interface {
	Reboot() error
	PowerOff() error
	Check(action string) error
}

// powerResponseGrace is how long the API waits after acknowledging a power
// action before actually executing it, so the HTTP response is flushed before
// the host (and this process) goes away. Overridable in tests.
var powerResponseGrace = 300 * time.Millisecond

type powerRequest struct {
	Action   string `json:"action"`
	Password string `json:"password"`
}

// handleSystemFeatures reports which optional sections of the UI are enabled,
// so the SPA can show/hide them (e.g. the Power page).
func (s *Server) handleSystemFeatures(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"power": s.cfg.PowerActionEnabled(),
	})
}

// handlePowerAction verifies the power password and schedules a reboot or
// poweroff. It answers before the command is executed, then dispatches the
// action after a short grace period. Wrong passwords are rate limited.
func (s *Server) handlePowerAction(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.PowerActionEnabled() {
		http.NotFound(w, r)
		return
	}
	if s.powerGuard.isBlocked() {
		w.Header().Set("Retry-After", "30")
		writeErr(w, http.StatusTooManyRequests, "too many failed attempts, try again later")
		return
	}

	var req powerRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Action != system.PowerActionReboot && req.Action != system.PowerActionPoweroff {
		writeErr(w, http.StatusBadRequest, `action must be "reboot" or "poweroff"`)
		return
	}

	expected := sha256.Sum256([]byte(s.cfg.PowerActionPass))
	got := sha256.Sum256([]byte(req.Password))
	if subtle.ConstantTimeCompare(expected[:], got[:]) != 1 {
		s.powerGuard.fail()
		slog.Warn("power action rejected", "remote", r.RemoteAddr, "reason", "wrong password")
		writeErr(w, http.StatusUnauthorized, "invalid power password")
		return
	}
	s.powerGuard.clear()

	if err := s.power.Check(req.Action); err != nil {
		slog.Error("power action not available", "action", req.Action, "err", err)
		writeErr(w, http.StatusInternalServerError, "power action not available: "+err.Error())
		return
	}

	var (
		action func() error
		status string
	)
	switch req.Action {
	case system.PowerActionReboot:
		action = s.power.Reboot
		status = "rebooting"
	case system.PowerActionPoweroff:
		action = s.power.PowerOff
		status = "powering off"
	}

	slog.Warn("power action authorized", "action", req.Action, "remote", r.RemoteAddr)
	writeJSON(w, http.StatusOK, map[string]string{"status": status})

	go func() {
		time.Sleep(powerResponseGrace)
		if err := action(); err != nil {
			slog.Error("power action failed", "action", req.Action, "err", err)
		}
	}()
}
