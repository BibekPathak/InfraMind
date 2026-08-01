package action

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/inframind/backend/internal/device"
	"github.com/inframind/backend/internal/eventbus"
)

func RegisterEvents(bus *eventbus.Bus, svc *Service, devSvc *device.Service) {
	bus.Subscribe("ai.recommendation.generated", func(evt eventbus.Event) {
		var payload RecommendationEvent
		if err := json.Unmarshal(evt.Data, &payload); err != nil {
			slog.Warn("action: failed to parse recommendation event", "error", err)
			return
		}

		assetID, err := resolveAssetID(context.Background(), devSvc, payload.DeviceID)
		if err != nil {
			slog.Warn("action: failed to resolve asset for recommendation", "deviceId", payload.DeviceID, "error", err)
			return
		}

		for _, rec := range payload.Recommendations {
			if rec.ActionType == "" {
				continue
			}

			pending, err := svc.HasPendingForDeviceType(context.Background(), payload.DeviceID, rec.ActionType)
			if err != nil {
				slog.Warn("action: dedup check failed", "error", err, "actionType", rec.ActionType)
				continue
			}
			if pending {
				slog.Debug("action: already pending, skipping proposal", "deviceId", payload.DeviceID, "actionType", rec.ActionType)
				continue
			}

			req := ProposeActionRequest{
				AssetID:  assetID,
				DeviceID: &payload.DeviceID,
				Type:     rec.ActionType,
				Payload:  rec.ActionPayload,
				Source:   "ai",
				Reason:   rec.Reason,
			}

			if _, err := svc.Propose(context.Background(), req); err != nil {
				slog.Error("action: propose from recommendation failed", "error", err, "actionType", rec.ActionType)
				continue
			}
			slog.Info("action proposed from AI recommendation",
				"assetId", assetID, "deviceId", payload.DeviceID,
				"actionType", rec.ActionType, "reason", rec.Reason)
		}
	})
}

func resolveAssetID(ctx context.Context, devSvc *device.Service, deviceID string) (string, error) {
	d, err := devSvc.GetByID(ctx, deviceID)
	if err != nil {
		return "", err
	}
	return d.AssetID, nil
}

type RecommendationEvent struct {
	DeviceID        string              `json:"deviceId"`
	Recommendations []RecommendationItem `json:"recommendations"`
}

type RecommendationItem struct {
	ActionType    string         `json:"actionType,omitempty"`
	ActionPayload map[string]any `json:"actionPayload,omitempty"`
	Reason        string         `json:"reason"`
}
