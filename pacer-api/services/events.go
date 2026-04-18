package services

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type UserEvent struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	Timestamp time.Time      `json:"timestamp"`
}

type EventHub struct {
	mu   sync.RWMutex
	subs map[uuid.UUID]map[string]chan UserEvent
}

func NewEventHub() *EventHub {
	return &EventHub{subs: make(map[uuid.UUID]map[string]chan UserEvent)}
}

func (h *EventHub) Subscribe(userID uuid.UUID) (string, <-chan UserEvent, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subID := uuid.New().String()
	ch := make(chan UserEvent, 32)

	if h.subs[userID] == nil {
		h.subs[userID] = make(map[string]chan UserEvent)
	}
	h.subs[userID][subID] = ch

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		userSubs, ok := h.subs[userID]
		if !ok {
			return
		}

		if existing, ok := userSubs[subID]; ok {
			delete(userSubs, subID)
			close(existing)
		}

		if len(userSubs) == 0 {
			delete(h.subs, userID)
		}
	}

	return subID, ch, unsubscribe
}

func (h *EventHub) Publish(userID uuid.UUID, eventType string, payload map[string]any) {
	h.mu.RLock()
	userSubs := h.subs[userID]
	h.mu.RUnlock()

	if len(userSubs) == 0 {
		return
	}

	evt := UserEvent{
		ID:        uuid.New().String(),
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}

	for _, ch := range userSubs {
		select {
		case ch <- evt:
		default:
		}
	}
}

func (h *EventHub) Push(ctx context.Context, userID uuid.UUID, eventType string, payload map[string]any) error {
	h.Publish(userID, eventType, payload)
	return nil
}
