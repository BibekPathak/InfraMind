package assettype

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

func (r *Repository) Create(ctx context.Context, at *AssetType) error {
	metrics, _ := json.Marshal(at.Metrics)
	thresholds, _ := json.Marshal(at.Thresholds)
	weights, _ := json.Marshal(at.HealthWeights)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO asset_types (type, display_name, metrics, thresholds, health_weights, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		at.Type, at.DisplayName, metrics, thresholds, weights, at.Active, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("create asset type: %w", err)
	}
	return nil
}

func (r *Repository) GetByType(ctx context.Context, typeName string) (*AssetType, error) {
	at := &AssetType{}
	var metrics, thresholds, weights []byte

	err := r.pool.QueryRow(ctx,
		`SELECT type, display_name, metrics, thresholds, health_weights, active, created_at, updated_at
		 FROM asset_types WHERE type = $1`, typeName,
	).Scan(&at.Type, &at.DisplayName, &metrics, &thresholds, &weights, &at.Active, &at.CreatedAt, &at.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get asset type %s: %w", typeName, err)
	}

	json.Unmarshal(metrics, &at.Metrics)
	json.Unmarshal(thresholds, &at.Thresholds)
	json.Unmarshal(weights, &at.HealthWeights)
	if at.Metrics == nil {
		at.Metrics = []Metric{}
	}
	return at, nil
}

func (r *Repository) List(ctx context.Context) ([]AssetType, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT type, display_name, metrics, thresholds, health_weights, active, created_at, updated_at
		 FROM asset_types ORDER BY type ASC`)
	if err != nil {
		return nil, fmt.Errorf("list asset types: %w", err)
	}
	defer rows.Close()

	var types []AssetType
	for rows.Next() {
		var at AssetType
		var metrics, thresholds, weights []byte
		if err := rows.Scan(&at.Type, &at.DisplayName, &metrics, &thresholds, &weights, &at.Active, &at.CreatedAt, &at.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan asset type: %w", err)
		}
		json.Unmarshal(metrics, &at.Metrics)
		json.Unmarshal(thresholds, &at.Thresholds)
		json.Unmarshal(weights, &at.HealthWeights)
		if at.Metrics == nil {
			at.Metrics = []Metric{}
		}
		types = append(types, at)
	}
	if types == nil {
		types = []AssetType{}
	}
	return types, nil
}

func (r *Repository) Update(ctx context.Context, typeName string, at *AssetType) error {
	metrics, _ := json.Marshal(at.Metrics)
	thresholds, _ := json.Marshal(at.Thresholds)
	weights, _ := json.Marshal(at.HealthWeights)

	_, err := r.pool.Exec(ctx,
		`UPDATE asset_types SET display_name = $2, metrics = $3, thresholds = $4, health_weights = $5, active = $6, updated_at = NOW()
		 WHERE type = $1`,
		typeName, at.DisplayName, metrics, thresholds, weights, at.Active,
	)
	if err != nil {
		return fmt.Errorf("update asset type %s: %w", typeName, err)
	}
	return nil
}
