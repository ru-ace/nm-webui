package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ru-ace/nm-webui/internal/events"
)

// handleEvents streams Server-Sent Events. The last received event id is
// honoured via the Last-Event-ID header for graceful reconnects.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")

	since := parseLastEventID(r.Header.Get("Last-Event-ID"))
	ch, replay, cleanup := s.hub.SubscribeReplay(since)
	defer cleanup()

	// Replay missed events on reconnect.
	if len(replay) > 0 {
		for _, ev := range replay {
			writeEvent(w, ev)
		}
		flusher.Flush()
	}

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-ch:
			writeEvent(w, ev)
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func parseLastEventID(v string) int64 {
	var id int64
	_, _ = fmt.Sscanf(v, "%d", &id)
	return id
}

func writeEvent(w http.ResponseWriter, ev events.Event) {
	data, err := json.Marshal(ev.Data)
	if err != nil {
		data = []byte("{}")
	}
	_, _ = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", ev.ID, ev.Type, data)
}
