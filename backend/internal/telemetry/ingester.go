package telemetry

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/inframind/backend/internal/device"
	"github.com/inframind/backend/internal/eventbus"
)

type Ingester struct {
	repo    *Repository
	devSvc  *device.Service
	bus     *eventbus.Bus
}

func NewIngester(repo *Repository, devSvc *device.Service, bus *eventbus.Bus) *Ingester {
	return &Ingester{repo: repo, devSvc: devSvc, bus: bus}
}

func (ing *Ingester) HandleMQTTMessage(topic string, payload []byte) {
	if !strings.HasPrefix(topic, "telemetry/") {
		return
	}

	parts := strings.Split(topic, "/")
	if len(parts) < 2 {
		slog.Warn("invalid telemetry topic", "topic", topic)
		return
	}

	var p TelemetryPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		slog.Warn("invalid telemetry payload", "error", err, "topic", topic)
		return
	}

	ts, err := time.Parse(time.RFC3339, p.Timestamp)
	if err != nil {
		ts = time.Now().UTC()
	}

	t := &Telemetry{
		Time:        ts,
		DeviceID:    p.DeviceID,
		Temperature: p.Temperature,
		Current:     p.Current,
		Voltage:     p.Voltage,
		Humidity:    p.Humidity,
	}

	if err := ing.repo.Insert(nil, t); err != nil {
		slog.Error("failed to persist telemetry", "error", err, "deviceId", p.DeviceID)
		return
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
}

func (ing *Ingester) IngestTelemetry(t *Telemetry) error {
	if err := ing.repo.Insert(nil, t); err != nil {
		return fmt.Errorf("ingest telemetry: %w", err)
	}

	ing.bus.Publish(eventbus.NewEvent("telemetry.ingested", "backend", t))
	return nil
}
