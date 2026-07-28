package device

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

func (r *Repository) Create(ctx context.Context, d *Device) error {
	loc, _ := json.Marshal(d.Location)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO devices (id, asset_id, firmware_version, status, location, last_heartbeat, mqtt_username, mqtt_password, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)`,
		d.ID, d.AssetID, d.FirmwareVersion, d.Status, loc, d.LastHeartbeat, d.MQTTUsername, d.MQTTPassword, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert device: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Device, error) {
	d := &Device{}
	var loc []byte
	var deletedAt *time.Time

	err := r.pool.QueryRow(ctx,
		`SELECT id, asset_id, firmware_version, status, location, last_heartbeat, mqtt_username, mqtt_password, created_at, updated_at, deleted_at
		 FROM devices WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&d.ID, &d.AssetID, &d.FirmwareVersion, &d.Status, &loc, &d.LastHeartbeat, &d.MQTTUsername, &d.MQTTPassword, &d.CreatedAt, &d.UpdatedAt, &deletedAt)
	if err != nil {
		return nil, fmt.Errorf("get device %s: %w", id, err)
	}
	if loc != nil {
		json.Unmarshal(loc, &d.Location)
	}
	d.DeletedAt = deletedAt
	return d, nil
}

func (r *Repository) ListByAsset(ctx context.Context, assetID string) ([]Device, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, asset_id, firmware_version, status, location, last_heartbeat, mqtt_username, created_at, updated_at
		 FROM devices WHERE asset_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`, assetID)
	if err != nil {
		return nil, fmt.Errorf("list devices by asset: %w", err)
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		var loc []byte
		if err := rows.Scan(&d.ID, &d.AssetID, &d.FirmwareVersion, &d.Status, &loc, &d.LastHeartbeat, &d.MQTTUsername, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		if loc != nil {
			json.Unmarshal(loc, &d.Location)
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func (r *Repository) UpdateHeartbeat(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE devices SET last_heartbeat = $2, status = 'online', updated_at = $2
		 WHERE id = $1 AND deleted_at IS NULL`, id, now)
	if err != nil {
		return fmt.Errorf("update heartbeat %s: %w", id, err)
	}
	return nil
}

func (r *Repository) GetConfig(ctx context.Context, id string) (map[string]any, error) {
	var configBytes []byte
	err := r.pool.QueryRow(ctx,
		`SELECT config FROM devices WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&configBytes)
	if err != nil {
		return nil, fmt.Errorf("get config %s: %w", id, err)
	}
	var cfg map[string]any
	if configBytes != nil {
		if err := json.Unmarshal(configBytes, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshal config: %w", err)
		}
	}
	return cfg, nil
}

func (r *Repository) UpdateConfig(ctx context.Context, id string, config map[string]any) error {
	cfgBytes, _ := json.Marshal(config)
	_, err := r.pool.Exec(ctx,
		`UPDATE devices SET config = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id, cfgBytes)
	if err != nil {
		return fmt.Errorf("update config %s: %w", id, err)
	}
	return nil
}
