package twin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Upsert(ctx context.Context, t *DigitalTwin) error {
	meta, _ := json.Marshal(t.Metadata)
	state, _ := json.Marshal(t.LiveState)
	history, _ := json.Marshal(t.MaintenanceHistory)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO digital_twins (asset_id, device_id, metadata, live_state, maintenance_history, ai_summary, health_score, health_level, synced_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		 ON CONFLICT (asset_id) DO UPDATE SET
		     device_id = EXCLUDED.device_id,
		     metadata = EXCLUDED.metadata,
		     live_state = EXCLUDED.live_state,
		     maintenance_history = EXCLUDED.maintenance_history,
		     ai_summary = EXCLUDED.ai_summary,
		     health_score = EXCLUDED.health_score,
		     health_level = EXCLUDED.health_level,
		     synced_at = EXCLUDED.synced_at,
		     updated_at = EXCLUDED.updated_at`,
		t.AssetID, t.DeviceID, meta, state, history, t.AISummary, t.HealthScore, t.HealthLevel, t.SyncedAt, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("upsert twin: %w", err)
	}
	return nil
}

func (r *Repository) GetByAssetID(ctx context.Context, assetID string) (*DigitalTwin, error) {
	t := &DigitalTwin{}
	var meta, state, history []byte

	err := r.pool.QueryRow(ctx,
		`SELECT asset_id, device_id, metadata, live_state, maintenance_history, ai_summary, health_score, health_level, synced_at, created_at, updated_at
		 FROM digital_twins WHERE asset_id = $1`, assetID,
	).Scan(&t.AssetID, &t.DeviceID, &meta, &state, &history, &t.AISummary, &t.HealthScore, &t.HealthLevel, &t.SyncedAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get twin %s: %w", assetID, err)
	}

	json.Unmarshal(meta, &t.Metadata)
	json.Unmarshal(state, &t.LiveState)
	json.Unmarshal(history, &t.MaintenanceHistory)
	if t.Metadata == nil {
		t.Metadata = map[string]any{}
	}
	if t.LiveState == nil {
		t.LiveState = map[string]any{}
	}
	if t.MaintenanceHistory == nil {
		t.MaintenanceHistory = []TwinEvent{}
	}

	return t, nil
}

func (r *Repository) List(ctx context.Context) ([]DigitalTwin, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT asset_id, device_id, metadata, live_state, maintenance_history, ai_summary, health_score, health_level, synced_at, created_at, updated_at
		 FROM digital_twins ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list twins: %w", err)
	}
	defer rows.Close()

	var twins []DigitalTwin
	for rows.Next() {
		var t DigitalTwin
		var meta, state, history []byte
		if err := rows.Scan(&t.AssetID, &t.DeviceID, &meta, &state, &history, &t.AISummary, &t.HealthScore, &t.HealthLevel, &t.SyncedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan twin: %w", err)
		}
		json.Unmarshal(meta, &t.Metadata)
		json.Unmarshal(state, &t.LiveState)
		json.Unmarshal(history, &t.MaintenanceHistory)
		if t.Metadata == nil {
			t.Metadata = map[string]any{}
		}
		if t.LiveState == nil {
			t.LiveState = map[string]any{}
		}
		if t.MaintenanceHistory == nil {
			t.MaintenanceHistory = []TwinEvent{}
		}
		twins = append(twins, t)
	}
	if twins == nil {
		twins = []DigitalTwin{}
	}
	return twins, nil
}
