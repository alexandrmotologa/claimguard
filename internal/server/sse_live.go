package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alexandrmotologa/claimguard/internal/telemetry"
)

// LiveEvent is broadcast to all active verification studio subscribers.
type LiveEvent struct {
	Type      string                `json:"type"` // AUDIT_INTERCEPTION, CLAIM_VERIFIED, METRIC_TICK
	Timestamp time.Time             `json:"timestamp"`
	Data      interface{}           `json:"data"`
}

// Broadcaster manages active SSE subscriptions for the live React studio dashboard.
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]bool
}

// NewBroadcaster initializes a thread-safe SSE broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan []byte]bool),
	}
}

// Subscribe registers a new subscriber channel.
func (b *Broadcaster) Subscribe() chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan []byte, 64)
	b.subscribers[ch] = true
	return ch
}

// Unsubscribe removes an active subscriber channel.
func (b *Broadcaster) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.subscribers[ch]; ok {
		delete(b.subscribers, ch)
		close(ch)
	}
}

// Publish broadcasts a LiveEvent to all active subscribers.
func (b *Broadcaster) Publish(eventType string, data interface{}) {
	evt := LiveEvent{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		return
	}

	formatted := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(payload)))

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- formatted:
		default:
			// Non-blocking drop if client buffer is congested
		}
	}
}

// PublishAudit broadcasts a new audit event.
func (b *Broadcaster) PublishAudit(event telemetry.AuditEvent) {
	b.Publish("AUDIT_INTERCEPTION", event)
}

// ServeHTTP handles incoming `/api/events/live` SSE connections.
func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// Send initial connected ping
	w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			w.Write(msg)
			flusher.Flush()
		}
	}
}
