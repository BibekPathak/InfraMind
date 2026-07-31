package workorder

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

func (r *Repository) Create(ctx context.Context, wo *WorkOrder) error {
	timeline, _ := json.Marshal(wo.Timeline)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO work_orders (id, asset_id, alert_id, type, priority, status, assigned_to, estimated_cost, description, timeline, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)`,
		wo.ID, wo.AssetID, wo.AlertID, wo.Type, wo.Priority, wo.Status, wo.AssignedTo, wo.EstimatedCost, wo.Description, timeline, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("create work order: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*WorkOrder, error) {
	wo := &WorkOrder{}
	var timeline []byte
	var alertID *string
	var assignedTo *string

	err := r.pool.QueryRow(ctx,
		`SELECT id, asset_id, alert_id, type, priority, status, assigned_to, estimated_cost, description, timeline, created_at, updated_at
		 FROM work_orders WHERE id = $1`, id,
	).Scan(&wo.ID, &wo.AssetID, &alertID, &wo.Type, &wo.Priority, &wo.Status, &assignedTo, &wo.EstimatedCost, &wo.Description, &timeline, &wo.CreatedAt, &wo.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get work order %s: %w", id, err)
	}
	wo.AlertID = alertID
	wo.AssignedTo = assignedTo
	if timeline != nil {
		json.Unmarshal(timeline, &wo.Timeline)
	}
	if wo.Timeline == nil {
		wo.Timeline = []TimelineEvent{}
	}
	return wo, nil
}

func (r *Repository) List(ctx context.Context, f Filter) ([]WorkOrder, error) {
	query := `SELECT id, asset_id, alert_id, type, priority, status, assigned_to, estimated_cost, description, timeline, created_at, updated_at
		 FROM work_orders WHERE 1=1`
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
	if f.Priority != "" {
		query += fmt.Sprintf(" AND priority = $%d", argIdx)
		args = append(args, f.Priority)
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
		return nil, fmt.Errorf("list work orders: %w", err)
	}
	defer rows.Close()

	var orders []WorkOrder
	for rows.Next() {
		var wo WorkOrder
		var timeline []byte
		var alertID *string
		var assignedTo *string
		if err := rows.Scan(&wo.ID, &wo.AssetID, &alertID, &wo.Type, &wo.Priority, &wo.Status, &assignedTo, &wo.EstimatedCost, &wo.Description, &timeline, &wo.CreatedAt, &wo.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan work order: %w", err)
		}
		wo.AlertID = alertID
		wo.AssignedTo = assignedTo
		if timeline != nil {
			json.Unmarshal(timeline, &wo.Timeline)
		}
		if wo.Timeline == nil {
			wo.Timeline = []TimelineEvent{}
		}
		orders = append(orders, wo)
	}
	if orders == nil {
		orders = []WorkOrder{}
	}
	return orders, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE work_orders SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("update work order status %s: %w", id, err)
	}
	return nil
}

func (r *Repository) Assign(ctx context.Context, id, assignedTo string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE work_orders SET assigned_to = $2, status = 'assigned', updated_at = NOW() WHERE id = $1`, id, assignedTo)
	if err != nil {
		return fmt.Errorf("assign work order %s: %w", id, err)
	}
	return nil
}

func (r *Repository) AppendTimeline(ctx context.Context, id string, event TimelineEvent) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE work_orders SET timeline = timeline || $2::jsonb, updated_at = NOW() WHERE id = $1`,
		id, mustJSON(event))
	if err != nil {
		return fmt.Errorf("append timeline %s: %w", id, err)
	}
	return nil
}

func (r *Repository) HasOpenForAsset(ctx context.Context, assetID string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM work_orders
		 WHERE asset_id = $1 AND status IN ('open', 'assigned', 'in_progress')`,
		assetID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check open work order: %w", err)
	}
	return count > 0, nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
