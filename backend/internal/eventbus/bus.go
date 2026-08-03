package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Event struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Source        string          `json:"source"`
	Timestamp     time.Time       `json:"timestamp"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	Data          json.RawMessage `json:"data"`
}

type Handler func(Event)

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	metrics  interface{ IncEventBusPublish() }
	redis    *RedisBackend
}

func New() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

func (b *Bus) SetMetrics(m interface{ IncEventBusPublish() }) {
	b.metrics = m
}

// SetRedisBackend enables durable, cross-instance delivery via Redis Streams.
// Events published afterwards are written to the stream (async) and dispatched
// locally; the backend consumer delivers events from other instances.
func (b *Bus) SetRedisBackend(rb *RedisBackend) {
	b.redis = rb
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

	if b.metrics != nil {
		b.metrics.IncEventBusPublish()
	}

	if b.redis != nil {
		b.redis.Publish(context.Background(), event)
	}

	b.dispatchLocal(handlers, event)
}

// DispatchLocal invokes local handlers for an event that originated on
// another instance (received via Redis Streams). Does not re-publish.
func (b *Bus) DispatchLocal(event Event) {
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()
	b.dispatchLocal(handlers, event)
}

// dispatchLocal invokes handlers synchronously without re-publishing to Redis.
// Used by the Redis consumer to deliver events from other instances.
func (b *Bus) dispatchLocal(handlers []Handler, event Event) {
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

			_, span := otel.Tracer("eventbus").Start(context.Background(), "event."+event.Type,
				trace.WithAttributes(attribute.String("event.type", event.Type)),
			)
			defer span.End()

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
