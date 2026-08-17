package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// SSEEvent represents a single Server-Sent Event payload.
type SSEEvent struct {
	ID    int64       `json:"id"`
	Type  string      `json:"type"`
	RunID string      `json:"run_id,omitempty"`
	Data  interface{} `json:"data"`
}

// AssertionProgressEventData is emitted for assertion_progress events.
type AssertionProgressEventData struct {
	RunID      string `json:"run_id"`
	Code       string `json:"code"`
	Status     string `json:"status"` // "pending" | "running" | "passed" | "failed" | "cancelled"
	Passed     bool   `json:"passed"`
	Score      int    `json:"score"`
	MinScore   int    `json:"min_score"`
	DurationMs int64  `json:"duration_ms"`
	Output     string `json:"output,omitempty"`
}

// LogEventData is emitted for log events.
type LogEventData struct {
	RunID   string `json:"run_id"`
	Message string `json:"message"`
}

// ExecutionCompletedEventData is emitted for execution_completed events.
type ExecutionCompletedEventData struct {
	RunID      string `json:"run_id"`
	Status     string `json:"status"`
	DurationMs int64  `json:"duration_ms"`
}

// ExecutionCancelledEventData is emitted for execution_cancelled events.
type ExecutionCancelledEventData struct {
	RunID string `json:"run_id"`
}

// EventBroker manages SSE subscriptions and multi-client event broadcasting.
type EventBroker struct {
	mu          sync.Mutex
	subscribers map[chan SSEEvent]struct{}
	lastEventID int64
}

// NewEventBroker creates an initialized EventBroker.
func NewEventBroker() *EventBroker {
	return &EventBroker{
		subscribers: make(map[chan SSEEvent]struct{}),
	}
}

// Subscribe creates a new buffered channel for a connected SSE client.
func (b *EventBroker) Subscribe() chan SSEEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan SSEEvent, 128)
	b.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a client channel and closes it.
func (b *EventBroker) Unsubscribe(ch chan SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscribers[ch]; exists {
		delete(b.subscribers, ch)
		close(ch)
	}
}

// Broadcast dispatches an event to all active subscribers and returns the assigned monotonic event ID.
func (b *EventBroker) Broadcast(eventType, runID string, data interface{}) int64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.lastEventID++
	ev := SSEEvent{
		ID:    b.lastEventID,
		Type:  eventType,
		RunID: runID,
		Data:  data,
	}

	for ch := range b.subscribers {
		select {
		case ch <- ev:
		default:
			// Buffer full: drop to prevent slow clients from stalling other subscribers
		}
	}

	return ev.ID
}

// LastEventID returns the highest event ID assigned so far.
func (b *EventBroker) LastEventID() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastEventID
}

// SubscriberCount returns the current number of active subscribers.
func (b *EventBroker) SubscriberCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subscribers)
}

// handleEvents serves GET /api/events as a Server-Sent Events stream.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	clientChan := s.broker.Subscribe()
	if s.lifecycle != nil {
		s.lifecycle.OnClientConnect()
	}
	defer func() {
		s.broker.Unsubscribe(clientChan)
		if s.lifecycle != nil {
			s.lifecycle.OnClientDisconnect()
		}
	}()

	// Send initial comment to establish SSE stream immediately
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.shutdownChan:
			return
		case ev, ok := <-clientChan:
			if !ok {
				return
			}
			dataBytes, err := json.Marshal(ev.Data)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", ev.ID, ev.Type, string(dataBytes))
			flusher.Flush()
		}
	}
}
