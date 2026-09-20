package events

import (
	"sync"
	"time"
)

// Event is pushed to all SSE subscribers.
type Event struct {
	ID   int64       `json:"-"`
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
	Time time.Time   `json:"time"`
}

// Hub fans out events to SSE subscribers and keeps a small replay buffer to
// support Last-Event-ID reconnect semantics.
type Hub struct {
	mu     sync.Mutex
	subs   map[chan Event]struct{}
	recent []Event
	max    int
	seq    int64
}

// NewHub creates a hub that retains up to `max` recent events.
func NewHub(max int) *Hub {
	if max <= 0 {
		max = 50
	}
	return &Hub{subs: map[chan Event]struct{}{}, max: max}
}

// Subscribe registers a new subscriber channel with a cleanup function.
func (h *Hub) Subscribe() (<-chan Event, func()) {
	ch, _, cleanup := h.SubscribeReplay(0)
	return ch, cleanup
}

// SubscribeReplay registers a subscriber and snapshots missed events under
// one lock, preventing a publish from appearing in both replay and live data.
func (h *Hub) SubscribeReplay(since int64) (<-chan Event, []Event, func()) {
	ch := make(chan Event, 64)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	replay := make([]Event, 0)
	for _, ev := range h.recent {
		if ev.ID > since {
			replay = append(replay, ev)
		}
	}
	h.mu.Unlock()
	return ch, replay, func() {
		h.mu.Lock()
		if _, ok := h.subs[ch]; ok {
			delete(h.subs, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
}

// Replay since supplies events strictly newer than the given ID.
func (h *Hub) Replay(since int64, emit func(Event)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ev := range h.recent {
		if ev.ID > since {
			emit(ev)
		}
	}
}

// Publish broadcasts an event to all subscribers, assigning the next ID.
func (h *Hub) Publish(typ string, data interface{}) {
	h.mu.Lock()
	h.seq++
	ev := Event{ID: h.seq, Type: typ, Data: data, Time: time.Now()}
	h.recent = append(h.recent, ev)
	if len(h.recent) > h.max {
		h.recent = h.recent[len(h.recent)-h.max:]
	}
	for ch := range h.subs {
		select {
		case ch <- ev:
		default:
			// slow consumer: drop this event; client may replay on reconnect
		}
	}
	h.mu.Unlock()
}

// ReplayAll emits every retained event.
func (h *Hub) ReplayAll(emit func(Event)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ev := range h.recent {
		emit(ev)
	}
}
