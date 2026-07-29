package twin

import (
	"context"
	"fmt"
	"time"

	"github.com/inframind/backend/internal/asset"
	"github.com/inframind/backend/internal/device"
	"github.com/inframind/backend/internal/health"
	"github.com/inframind/backend/internal/telemetry"
)

type Service struct {
	repo          *Repository
	assetSvc      *asset.Service
	deviceSvc     *device.Service
	telemetryRepo *telemetry.Repository
	healthSvc     *health.Service
}

func NewService(repo *Repository, assetSvc *asset.Service, deviceSvc *device.Service, telemetryRepo *telemetry.Repository, healthSvc *health.Service) *Service {
	return &Service{
		repo:          repo,
		assetSvc:      assetSvc,
		deviceSvc:     deviceSvc,
		telemetryRepo: telemetryRepo,
		healthSvc:     healthSvc,
	}
}

func (s *Service) GetByAssetID(ctx context.Context, assetID string) (*DigitalTwin, error) {
	return s.repo.GetByAssetID(ctx, assetID)
}

func (s *Service) List(ctx context.Context) ([]DigitalTwin, error) {
	return s.repo.List(ctx)
}

func (s *Service) Sync(ctx context.Context, assetID string) (*DigitalTwin, error) {
	a, err := s.assetSvc.GetByID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("sync twin: get asset: %w", err)
	}

	devices, err := s.deviceSvc.ListByAsset(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("sync twin: list devices: %w", err)
	}

	var liveDeviceID *string
	var liveState map[string]any = map[string]any{
		"deviceCount": len(devices),
		"assetName":   a.Name,
		"assetType":   a.Type,
	}

	if len(devices) > 0 {
		d := devices[0]
		liveDeviceID = &d.ID
		liveState["deviceStatus"] = d.Status
		liveState["firmwareVersion"] = d.FirmwareVersion

		t, err := s.telemetryRepo.GetLatest(ctx, d.ID)
		if err == nil {
			liveState["temperature"] = t.Temperature
			liveState["current"] = t.Current
			liveState["voltage"] = t.Voltage
			liveState["humidity"] = t.Humidity
		}

		healthResp, err := s.healthSvc.Calculate(ctx, d.ID,
			getFloat(liveState, "temperature"),
			getFloat(liveState, "current"),
			getFloat(liveState, "voltage"),
			getFloat(liveState, "humidity"),
		)
		if err == nil {
			liveState["healthScore"] = healthResp.Score
			liveState["healthLevel"] = healthResp.Level
		}
	}

	twin := &DigitalTwin{
		AssetID:   assetID,
		DeviceID:  liveDeviceID,
		Metadata:  a.Metadata,
		LiveState: liveState,
		SyncedAt:  timePtr(time.Now().UTC()),
	}

	if score, ok := liveState["healthScore"].(float64); ok {
		twin.HealthScore = &score
	}
	if level, ok := liveState["healthLevel"].(string); ok {
		twin.HealthLevel = &level
	}

	if err := s.repo.Upsert(ctx, twin); err != nil {
		return nil, fmt.Errorf("sync twin: upsert: %w", err)
	}

	return twin, nil
}

func (s *Service) AddEvent(ctx context.Context, assetID string, req AddEventRequest) (*DigitalTwin, error) {
	twin, err := s.repo.GetByAssetID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("get twin for event: %w", err)
	}

	event := TwinEvent{
		ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Type:      req.Type,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Summary:   req.Summary,
		Details:   req.Details,
	}

	twin.MaintenanceHistory = append(twin.MaintenanceHistory, event)

	if err := s.repo.Upsert(ctx, twin); err != nil {
		return nil, fmt.Errorf("add event: %w", err)
	}

	return twin, nil
}

func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func timePtr(t time.Time) *time.Time {
	return &t
}
