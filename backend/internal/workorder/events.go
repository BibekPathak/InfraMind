package workorder

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/inframind/backend/internal/alert"
	"github.com/inframind/backend/internal/device"
	"github.com/inframind/backend/internal/eventbus"
)

func RegisterEvents(bus *eventbus.Bus, svc *Service, devSvc *device.Service) {
	bus.Subscribe("alert.raised", func(evt eventbus.Event) {
		var a alert.Alert
		if err := json.Unmarshal(evt.Data, &a); err != nil {
			slog.Warn("workorder: failed to parse alert event", "error", err)
			return
		}

		if a.Severity != "critical" && a.Severity != "warning" {
			return
		}

		assetID, err := resolveAssetID(context.Background(), devSvc, a.DeviceID)
		if err != nil {
			slog.Warn("workorder: failed to resolve asset for alert", "deviceId", a.DeviceID, "error", err)
			return
		}

		open, err := svc.HasOpenForAsset(context.Background(), assetID)
		if err != nil {
			slog.Warn("workorder: dedup check failed", "error", err)
			return
		}
		if open {
			slog.Debug("workorder: open order exists for asset, skipping auto-create", "assetId", assetID)
			return
		}

		alertID := a.ID
		req := CreateWorkOrderRequest{
			AssetID:     assetID,
			AlertID:     &alertID,
			Type:        ruleToType(a.Rule),
			Priority:    severityToPriority(a.Severity),
			Description: a.Message,
		}

		if _, err := svc.Create(context.Background(), req); err != nil {
			slog.Error("workorder: auto-create from alert failed", "error", err, "alertId", a.ID)
			return
		}
		slog.Info("workorder auto-created from alert",
			"alertId", a.ID, "assetId", assetID, "severity", a.Severity)
	})
}

func resolveAssetID(ctx context.Context, devSvc *device.Service, deviceID string) (string, error) {
	d, err := devSvc.GetByID(ctx, deviceID)
	if err != nil {
		return "", err
	}
	return d.AssetID, nil
}

func severityToPriority(severity string) string {
	switch severity {
	case "critical":
		return "critical"
	case "warning":
		return "high"
	default:
		return "medium"
	}
}

func ruleToType(rule string) string {
	switch rule {
	case "sensor_failure":
		return "diagnostic"
	case "current_overload", "voltage_sag":
		return "repair"
	case "temperature_critical", "temperature_warning":
		return "inspection"
	default:
		return "inspection"
	}
}
