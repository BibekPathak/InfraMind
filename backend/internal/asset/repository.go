package asset

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

func (r *Repository) Create(ctx context.Context, a *Asset) error {
	loc, _ := json.Marshal(a.Location)
	meta, _ := json.Marshal(a.Metadata)
	if a.AutonomyMode == "" {
		a.AutonomyMode = "manual"
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO assets (id, name, type, autonomy_mode, location, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		a.ID, a.Name, a.Type, a.AutonomyMode, loc, meta, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert asset: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Asset, error) {
	a := &Asset{}
	var loc, meta []byte
	var deletedAt *time.Time

	err := r.pool.QueryRow(ctx,
		`SELECT id, name, type, autonomy_mode, location, metadata, created_at, updated_at, deleted_at
		 FROM assets WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&a.ID, &a.Name, &a.Type, &a.AutonomyMode, &loc, &meta, &a.CreatedAt, &a.UpdatedAt, &deletedAt)
	if err != nil {
		return nil, fmt.Errorf("get asset %s: %w", id, err)
	}
	if loc != nil {
		json.Unmarshal(loc, &a.Location)
	}
	if meta != nil {
		json.Unmarshal(meta, &a.Metadata)
	}
	a.DeletedAt = deletedAt
	return a, nil
}

func (r *Repository) List(ctx context.Context) ([]Asset, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, type, autonomy_mode, location, metadata, created_at, updated_at
		 FROM assets WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var a Asset
		var loc, meta []byte
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.AutonomyMode, &loc, &meta, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan asset: %w", err)
		}
		if loc != nil {
			json.Unmarshal(loc, &a.Location)
		}
		if meta != nil {
			json.Unmarshal(meta, &a.Metadata)
		}
		assets = append(assets, a)
	}
	return assets, nil
}

func (r *Repository) UpdateAutonomyMode(ctx context.Context, id, mode string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE assets SET autonomy_mode = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id, mode)
	if err != nil {
		return fmt.Errorf("update autonomy mode %s: %w", id, err)
	}
	return nil
}

func (r *Repository) GetAutonomyMode(ctx context.Context, id string) (string, error) {
	var mode string
	err := r.pool.QueryRow(ctx,
		`SELECT autonomy_mode FROM assets WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&mode)
	if err != nil {
		return "manual", fmt.Errorf("get autonomy mode %s: %w", id, err)
	}
	return mode, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE assets SET deleted_at = $2, updated_at = $2 WHERE id = $1 AND deleted_at IS NULL`,
		id, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("delete asset %s: %w", id, err)
	}
	return nil
}
