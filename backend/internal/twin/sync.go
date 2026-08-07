package twin

import (
	"context"
	"log/slog"
	"time"

	"github.com/inframind/backend/internal/eventbus"
	"github.com/inframind/backend/internal/telemetry"
)

const defaultSyncInterval = 30 * time.Second

type SyncEngine struct {
	svc      *Service
	bus      *eventbus.Bus
	wsHub    *telemetry.WSHub
	interval time.Duration
}

func NewSyncEngine(svc *Service, bus *eventbus.Bus, wsHub *telemetry.WSHub) *SyncEngine {
	return NewSyncEngineWithInterval(svc, bus, wsHub, defaultSyncInterval)
}

func NewSyncEngineWithInterval(svc *Service, bus *eventbus.Bus, wsHub *telemetry.WSHub, interval time.Duration) *SyncEngine {
	return &SyncEngine{svc: svc, bus: bus, wsHub: wsHub, interval: interval}
}

func (e *SyncEngine) Start(ctx context.Context) {
	slog.Info("twin sync engine started", "interval", e.interval)
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.syncAll(ctx)
		case <-ctx.Done():
			slog.Info("twin sync engine stopped")
			return
		}
	}
}

func (e *SyncEngine) SyncAsset(ctx context.Context, assetID string) {
	twin, err := e.svc.Sync(ctx, assetID)
	if err != nil {
		slog.Error("twin sync failed", "assetId", assetID, "error", err)
		return
	}

	e.bus.Publish(eventbus.NewEvent("twin.updated", "sync_engine", twin))

	if e.wsHub != nil {
		e.wsHub.Broadcast(assetID, telemetry.WSEvent{
			Type:      "twin.updated",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			AssetID:   assetID,
			Payload:   twin,
		})
	}

	slog.Debug("twin synced", "assetId", assetID)
}

func (e *SyncEngine) syncAll(ctx context.Context) {
	twins, err := e.svc.List(ctx)
	if err != nil {
		slog.Error("twin sync: list failed", "error", err)
		return
	}

	for _, t := range twins {
		select {
		case <-ctx.Done():
			return
		default:
		}
		e.SyncAsset(ctx, t.AssetID)
	}
}
