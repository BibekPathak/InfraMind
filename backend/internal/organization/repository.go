package organization

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

func (r *Repository) Create(ctx context.Context, o *Organization) error {
	settings, _ := json.Marshal(o.Settings)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO organizations (id, name, slug, settings, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $5)`,
		o.ID, o.Name, o.Slug, settings, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("create organization: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Organization, error) {
	o := &Organization{}
	var settings []byte
	var deletedAt *time.Time

	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, settings, created_at, updated_at, deleted_at
		 FROM organizations WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&o.ID, &o.Name, &o.Slug, &settings, &o.CreatedAt, &o.UpdatedAt, &deletedAt)
	if err != nil {
		return nil, fmt.Errorf("get organization %s: %w", id, err)
	}
	if settings != nil {
		json.Unmarshal(settings, &o.Settings)
	}
	if o.Settings == nil {
		o.Settings = map[string]any{}
	}
	o.DeletedAt = deletedAt
	return o, nil
}

func (r *Repository) GetBySlug(ctx context.Context, slug string) (*Organization, error) {
	o := &Organization{}
	var settings []byte
	var deletedAt *time.Time

	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, settings, created_at, updated_at, deleted_at
		 FROM organizations WHERE slug = $1 AND deleted_at IS NULL`, slug,
	).Scan(&o.ID, &o.Name, &o.Slug, &settings, &o.CreatedAt, &o.UpdatedAt, &deletedAt)
	if err != nil {
		return nil, fmt.Errorf("get organization by slug %s: %w", slug, err)
	}
	if settings != nil {
		json.Unmarshal(settings, &o.Settings)
	}
	if o.Settings == nil {
		o.Settings = map[string]any{}
	}
	o.DeletedAt = deletedAt
	return o, nil
}

func (r *Repository) List(ctx context.Context) ([]Organization, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, slug, settings, created_at, updated_at
		 FROM organizations WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var o Organization
		var settings []byte
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &settings, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		if settings != nil {
			json.Unmarshal(settings, &o.Settings)
		}
		if o.Settings == nil {
			o.Settings = map[string]any{}
		}
		orgs = append(orgs, o)
	}
	if orgs == nil {
		orgs = []Organization{}
	}
	return orgs, nil
}

func (r *Repository) Update(ctx context.Context, id string, o *Organization) error {
	settings, _ := json.Marshal(o.Settings)
	_, err := r.pool.Exec(ctx,
		`UPDATE organizations SET name = $2, slug = $3, settings = $4, updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`,
		id, o.Name, o.Slug, settings,
	)
	if err != nil {
		return fmt.Errorf("update organization %s: %w", id, err)
	}
	return nil
}
