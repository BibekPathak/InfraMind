package alert

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/inframind/backend/internal/eventbus"
)

func RegisterEvents(bus *eventbus.Bus, svc *Service) {
	bus.Subscribe("device.status_changed", func(evt eventbus.Event) {
		var payload struct {
			DeviceID string `json:"deviceId"`
			From     string `json:"from"`
			To       string `json:"to"`
		}
		if err := json.Unmarshal(evt.Data, &payload); err != nil {
			slog.Warn("alert: failed to parse status_changed", "error", err)
			return
		}

		if payload.To != "offline" {
			return
		}

		open, err := svc.HasOpenAlertForRule(context.Background(), payload.DeviceID, "device_offline")
		if err != nil {
			slog.Warn("alert: dedup check failed", "error", err)
			return
		}
		if open {
			return
		}

		a, err := svc.Create(context.Background(), payload.DeviceID, "warning", "device_offline", "Device went offline (heartbeat timeout)")
		if err != nil {
			slog.Error("alert: failed to raise offline alert", "error", err, "deviceId", payload.DeviceID)
			return
		}

		slog.Warn("alert raised", "deviceId", payload.DeviceID, "rule", "device_offline", "severity", "warning")
		bus.Publish(eventbus.NewEvent("alert.raised", "alert_events", a))
	})
}
