package eventbus

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type Event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

type Handler func(Event)

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func New() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

func (b *Bus) Subscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	slog.Debug("event subscription registered", "type", eventType)
}

func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	for _, h := range handlers {
		func(h Handler) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("event handler panic", "type", event.Type, "recover", r)
				}
			}()
			h(event)
		}(h)
	}
}

func NewEvent(eventType, source string, data any) Event {
	raw, err := json.Marshal(data)
	if err != nil {
		panic(fmt.Errorf("marshal event data: %w", err))
	}
	return Event{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().UTC(),
		Data:      raw,
	}
}
