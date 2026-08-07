package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inframind/backend/internal/tenant"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Insert(ctx context.Context, e *LogEntry) error {
	details, _ := json.Marshal(e.Details)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_logs (id, organization_id, user_id, action, resource_type, resource_id, details, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		e.ID, e.OrganizationID, e.UserID, e.Action, e.ResourceType, e.ResourceID, details, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (r *Repository) List(ctx context.Context, f Filter) ([]LogEntry, error) {
	query := `SELECT id, organization_id, user_id, action, resource_type, resource_id, details, created_at
		 FROM audit_logs WHERE organization_id = $1`
	args := []any{tenant.EffectiveOrgID(ctx)}
	argIdx := 2

	if f.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, f.ResourceType)
		argIdx++
	}
	if f.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, f.Action)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 50
	}
	offset := (f.Page - 1) * f.Limit
	if offset < 0 {
		offset = 0
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, f.Limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var e LogEntry
		var details []byte
		var orgID *string
		if err := rows.Scan(&e.ID, &orgID, &e.UserID, &e.Action, &e.ResourceType, &e.ResourceID, &details, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		if orgID != nil {
			e.OrganizationID = *orgID
		}
		if details != nil {
			json.Unmarshal(details, &e.Details)
		}
		if e.Details == nil {
			e.Details = map[string]any{}
		}
		logs = append(logs, e)
	}
	if logs == nil {
		logs = []LogEntry{}
	}
	return logs, nil
}
