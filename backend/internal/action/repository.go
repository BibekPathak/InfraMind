package action

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

func (r *Repository) Create(ctx context.Context, a *Action) error {
	payload, _ := json.Marshal(a.Payload)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO actions (id, asset_id, device_id, type, payload, source, status, approval_required, auto_approved, reason, proposed_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11, $11)`,
		a.ID, a.AssetID, a.DeviceID, a.Type, payload, a.Source, a.Status, a.ApprovalRequired, a.AutoApproved, a.Reason, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("create action: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Action, error) {
	a := &Action{}
	var payload []byte
	var deviceID *string
	var result *string

	err := r.pool.QueryRow(ctx,
		`SELECT id, asset_id, device_id, type, payload, source, status, approval_required, auto_approved, reason, result, proposed_at, executed_at, created_at, updated_at
		 FROM actions WHERE id = $1`, id,
	).Scan(&a.ID, &a.AssetID, &deviceID, &a.Type, &payload, &a.Source, &a.Status, &a.ApprovalRequired, &a.AutoApproved, &a.Reason, &result, &a.ProposedAt, &a.ExecutedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get action %s: %w", id, err)
	}
	a.DeviceID = deviceID
	a.Result = result
	if payload != nil {
		json.Unmarshal(payload, &a.Payload)
	}
	if a.Payload == nil {
		a.Payload = map[string]any{}
	}
	return a, nil
}

func (r *Repository) List(ctx context.Context, f Filter) ([]Action, error) {
	query := `SELECT id, asset_id, device_id, type, payload, source, status, approval_required, auto_approved, reason, result, proposed_at, executed_at, created_at, updated_at
		 FROM actions WHERE 1=1`
	args := []any{}
	argIdx := 1

	if f.AssetID != "" {
		query += fmt.Sprintf(" AND asset_id = $%d", argIdx)
		args = append(args, f.AssetID)
		argIdx++
	}
	if f.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, f.Status)
		argIdx++
	}
	if f.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, f.Type)
		argIdx++
	}

	query += " ORDER BY proposed_at DESC"

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
		return nil, fmt.Errorf("list actions: %w", err)
	}
	defer rows.Close()

	var actions []Action
	for rows.Next() {
		var a Action
		var payload []byte
		var deviceID *string
		var result *string
		if err := rows.Scan(&a.ID, &a.AssetID, &deviceID, &a.Type, &payload, &a.Source, &a.Status, &a.ApprovalRequired, &a.AutoApproved, &a.Reason, &result, &a.ProposedAt, &a.ExecutedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan action: %w", err)
		}
		a.DeviceID = deviceID
		a.Result = result
		if payload != nil {
			json.Unmarshal(payload, &a.Payload)
		}
		if a.Payload == nil {
			a.Payload = map[string]any{}
		}
		actions = append(actions, a)
	}
	if actions == nil {
		actions = []Action{}
	}
	return actions, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE actions SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("update action status %s: %w", id, err)
	}
	return nil
}

func (r *Repository) MarkExecuted(ctx context.Context, id string, result string) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE actions SET status = 'executed', result = $2, executed_at = $3, updated_at = $3 WHERE id = $1`,
		id, result, now)
	if err != nil {
		return fmt.Errorf("mark action executed %s: %w", id, err)
	}
	return nil
}

func (r *Repository) MarkFailed(ctx context.Context, id string, result string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE actions SET status = 'failed', result = $2, updated_at = NOW() WHERE id = $1`,
		id, result)
	if err != nil {
		return fmt.Errorf("mark action failed %s: %w", id, err)
	}
	return nil
}
