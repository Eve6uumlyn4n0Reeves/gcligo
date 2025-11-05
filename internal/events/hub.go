package events

import (
	"context"
	"sync"
	"time"
)

// Topic names for core domain events.
const (
	TopicConfigUpdated     = "config.updated"
	TopicCredentialsSynced = "credentials.synced"
	TopicCredentialChanged = "credentials.changed"
)

// Event represents a published message on the event bus.
type Event struct {
	Topic     string            `json:"topic"`
	Timestamp time.Time         `json:"timestamp"`
	Payload   any               `json:"payload,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Handler processes an incoming event.
type Handler func(context.Context, Event)

// Publisher exposes the ability to publish events to the hub.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload any, metadata map[string]string)
}

// Subscriber exposes subscription capabilities.
type Subscriber interface {
	Subscribe(topic string, handler Handler) func()
}

// Hub is a lightweight in-process pub/sub event bus.
type Hub struct {
	mu     sync.RWMutex
	subs   map[string]map[int64]Handler
	nextID int64
}

// NewHub constructs a new empty hub.
func NewHub() *Hub {
	return &Hub{
		subs: make(map[string]map[int64]Handler),
	}
}

// Subscribe registers a handler for the given topic.
// It returns a function that, when invoked, unsubscribes the handler.
func (h *Hub) Subscribe(topic string, handler Handler) func() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	id := h.nextID

	if _, ok := h.subs[topic]; !ok {
		h.subs[topic] = make(map[int64]Handler)
	}
	h.subs[topic][id] = handler

	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if listeners, ok := h.subs[topic]; ok {
			delete(listeners, id)
			if len(listeners) == 0 {
				delete(h.subs, topic)
			}
		}
	}
}

// Publish dispatches an event to all subscribers of the topic synchronously.
func (h *Hub) Publish(ctx context.Context, topic string, payload any, metadata map[string]string) {
	event := Event{
		Topic:     topic,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
		Metadata:  metadata,
	}

	handlers := h.snapshotHandlers(topic)
	for _, handler := range handlers {
		handler(ctx, event)
	}
}

func (h *Hub) snapshotHandlers(topic string) []Handler {
	h.mu.RLock()
	defer h.mu.RUnlock()

	listeners := h.subs[topic]
	if len(listeners) == 0 {
		return nil
	}

	out := make([]Handler, 0, len(listeners))
	for _, handler := range listeners {
		out = append(out, handler)
	}
	return out
}
