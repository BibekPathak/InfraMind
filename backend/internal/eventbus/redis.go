package eventbus

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultStream = "infra:events"
	defaultGroup  = "infra-backend"
)

// RedisBackend adds durable delivery to the in-process Bus via Redis Streams.
// Events published to the bus are written to the stream (async, non-blocking)
// and dispatched locally. A background consumer reads the stream and dispatches
// events originating from OTHER instances, enabling multi-replica fan-out.
type RedisBackend struct {
	client     *redis.Client
	stream     string
	group      string
	instanceID string
	done       chan struct{}
}

func NewRedisBackend(url, instanceID string) (*RedisBackend, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	rb := &RedisBackend{
		client:     redis.NewClient(opts),
		stream:     defaultStream,
		group:      defaultGroup,
		instanceID: instanceID,
		done:       make(chan struct{}),
	}

	if err := rb.client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return rb, nil
}

// Publish writes the event to the Redis stream with the source instance ID.
// Non-blocking from the caller's perspective.
func (rb *RedisBackend) Publish(ctx context.Context, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("eventbus: marshal event for redis", "error", err, "type", event.Type)
		return
	}

	values := map[string]any{
		"type":   event.Type,
		"source": event.Source,
		"ts":     event.Timestamp.Format(time.RFC3339Nano),
		"data":   string(data),
		"origin": rb.instanceID,
	}

	// Fire-and-forget write; the consumer loop handles retry/ack.
	go func() {
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := rb.client.XAdd(ctxTimeout, &redis.XAddArgs{
			Stream: rb.stream,
			Values: values,
		}).Err(); err != nil {
			slog.Warn("eventbus: redis xadd failed", "error", err, "type", event.Type)
		}
	}()
}

// Start runs the consumer loop. It reads new stream entries via a consumer
// group and dispatches events originating from other instances to the bus.
// Returns when ctx is cancelled.
func (rb *RedisBackend) Start(ctx context.Context, dispatch func(Event)) {
	if err := rb.ensureGroup(ctx); err != nil {
		slog.Warn("eventbus: redis group setup failed, consumer disabled", "error", err)
		return
	}

	slog.Info("eventbus redis consumer started", "stream", rb.stream, "group", rb.group, "instance", rb.instanceID)

	for {
		select {
		case <-ctx.Done():
			slog.Info("eventbus redis consumer stopped")
			return
		case <-rb.done:
			return
		default:
		}

		streams, err := rb.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    rb.group,
			Consumer: rb.instanceID,
			Streams:  []string{rb.stream, ">"},
			Count:    10,
			Block:    2 * time.Second,
		}).Result()
		if err != nil {
			if err == context.Canceled {
				return
			}
			if err != redis.Nil {
				slog.Debug("eventbus: redis xreadgroup", "error", err)
			}
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				rb.handleMessage(ctx, msg, dispatch)
				rb.client.XAck(ctx, rb.stream, rb.group, msg.ID)
			}
		}
	}
}

func (rb *RedisBackend) Close() {
	close(rb.done)
	rb.client.Close()
}

func (rb *RedisBackend) handleMessage(ctx context.Context, msg redis.XMessage, dispatch func(Event)) {
	origin, _ := msg.Values["origin"].(string)
	if origin == rb.instanceID {
		// Event was published locally and already dispatched synchronously.
		return
	}

	dataStr, _ := msg.Values["data"].(string)
	var event Event
	if err := json.Unmarshal([]byte(dataStr), &event); err != nil {
		slog.Warn("eventbus: failed to unmarshal redis event", "error", err)
		return
	}

	dispatch(event)
}

func (rb *RedisBackend) ensureGroup(ctx context.Context) error {
	err := rb.client.XGroupCreateMkStream(ctx, rb.stream, rb.group, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}
