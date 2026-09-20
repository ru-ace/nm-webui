package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ru-ace/nm-webui/internal/nm"
)

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// httpError converts an error into a JSON error response with a sane status.
func httpError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, nm.ErrNoActiveConn):
		writeErr(w, http.StatusNotFound, "no active connection")
	case errors.Is(err, nm.ErrConnectTimeout):
		writeErr(w, http.StatusGatewayTimeout, "connection timed out")
	case errors.Is(err, nm.ErrAuthFailed):
		writeErr(w, http.StatusBadGateway, "authentication failed: invalid password")
	case errors.Is(err, nm.ErrDeviceFailed):
		writeErr(w, http.StatusBadGateway, "device failed while connecting")
	default:
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
}

func statusText(c uint32) string {
	switch c {
	case 4:
		return "online"
	case 3, 2:
		return "limited"
	case 1:
		return "offline"
	default:
		return "unknown"
	}
}