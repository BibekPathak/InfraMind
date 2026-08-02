package telemetry

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/inframind/backend/internal/device"
	"github.com/inframind/backend/internal/eventbus"
)

const (
	batchSize    = 1000
	batchTimeout = 5 * time.Second
)

type IngesterMetrics interface {
	IncTelemetryIngested()
	ObserveBatchSize(size int)
	IncMQTTMessage()
}

type Ingester struct {
	repo      *Repository
	devSvc    *device.Service
	bus       *eventbus.Bus
	hub       *WSHub
	validator *Validator
	metrics   IngesterMetrics

	mu          sync.Mutex
	batch       []Telemetry
	batchTicker *time.Ticker
	flushCh     chan struct{}
	doneCh      chan struct{}
}

func NewIngester(repo *Repository, devSvc *device.Service, bus *eventbus.Bus, hub *WSHub, metrics IngesterMetrics) *Ingester {
	ing := &Ingester{
		repo:      repo,
		devSvc:    devSvc,
		bus:       bus,
		hub:       hub,
		validator: NewValidator(),
		metrics:   metrics,
		batch:     make([]Telemetry, 0, batchSize),
		batchTicker: time.NewTicker(batchTimeout),
		flushCh:   make(chan struct{}),
		doneCh:    make(chan struct{}),
	}

	go ing.batchLoop()
	return ing
}

func (ing *Ingester) batchLoop() {
	for {
		select {
		case <-ing.batchTicker.C:
			ing.flush()
		case <-ing.flushCh:
			ing.flush()
		case <-ing.doneCh:
			ing.flush()
			return
		}
	}
}

func (ing *Ingester) flush() {
	ing.mu.Lock()
	if len(ing.batch) == 0 {
		ing.mu.Unlock()
		return
	}
	batch := ing.batch
	ing.batch = make([]Telemetry, 0, batchSize)
	ing.mu.Unlock()

	if err := ing.repo.BatchInsert(nil, batch); err != nil {
		slog.Error("batch insert failed", "error", err, "count", len(batch))
		for _, t := range batch {
			if err := ing.repo.Insert(nil, &t); err != nil {
				slog.Error("fallback insert failed", "error", err, "deviceId", t.DeviceID)
			}
		}
	} else {
		slog.Debug("batch inserted", "count", len(batch))
		if ing.metrics != nil {
			ing.metrics.ObserveBatchSize(len(batch))
		}
	}
}

func (ing *Ingester) Stop() {
	close(ing.doneCh)
}

func (ing *Ingester) HandleMQTTMessage(topic string, payload []byte) {
	if !strings.HasPrefix(topic, "telemetry/") {
		return
	}

	if ing.metrics != nil {
		ing.metrics.IncMQTTMessage()
	}

	result := ing.validator.Validate(topic, payload)
	if !result.Valid {
		slog.Warn("telemetry validation failed", "topic", topic, "reason", result.Message)
		return
	}

	p := result.Payload

	ts, err := time.Parse(time.RFC3339, p.Timestamp)
	if err != nil {
		ts = time.Now().UTC()
	}

	t := Telemetry{
		Time:        ts,
		DeviceID:    p.DeviceID,
		Temperature: p.Temperature,
		Current:     p.Current,
		Voltage:     p.Voltage,
		Humidity:    p.Humidity,
		FlowRate:    p.FlowRate,
		Pressure:    p.Pressure,
		Vibration:   p.Vibration,
		RPM:         p.RPM,
		OutputPower: p.OutputPower,
	}

	ing.mu.Lock()
	ing.batch = append(ing.batch, t)
	shouldFlush := len(ing.batch) >= batchSize
	ing.mu.Unlock()

	if shouldFlush {
		ing.flush()
	}

	if ing.metrics != nil {
		ing.metrics.IncTelemetryIngested()
	}

	if err := ing.devSvc.HandleHeartbeat(nil, p.DeviceID); err != nil {
		slog.Warn("failed to update device heartbeat", "error", err, "deviceId", p.DeviceID)
	}

	ing.bus.Publish(eventbus.NewEvent("telemetry.ingested", "backend", map[string]any{
		"deviceId": p.DeviceID,
		"time":     ts,
		"scenario": p.Scenario,
	}))

	slog.Debug("telemetry ingested",
		"deviceId", p.DeviceID,
		"temperature", p.Temperature,
		"scenario", p.Scenario,
	)

	ing.hub.Broadcast(p.DeviceID, WSEvent{
		Type:      "telemetry.updated",
		Timestamp: ts.Format(time.RFC3339),
		AssetID:   p.DeviceID,
		Payload:   t,
	})
}

func (ing *Ingester) IngestTelemetry(t *Telemetry) error {
	if err := ing.repo.Insert(nil, t); err != nil {
		return fmt.Errorf("ingest telemetry: %w", err)
	}
	ing.bus.Publish(eventbus.NewEvent("telemetry.ingested", "backend", t))
	return nil
}
