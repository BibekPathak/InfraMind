package alert

import (
	"context"
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

func (r *Repository) Create(ctx context.Context, a *Alert) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO alerts (id, device_id, severity, rule, message, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		a.ID, a.DeviceID, a.Severity, a.Rule, a.Message, a.Status, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("create alert: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Alert, error) {
	a := &Alert{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, device_id, severity, rule, message, status, created_at, updated_at
		 FROM alerts WHERE id = $1`, id,
	).Scan(&a.ID, &a.DeviceID, &a.Severity, &a.Rule, &a.Message, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get alert %s: %w", id, err)
	}
	return a, nil
}

func (r *Repository) List(ctx context.Context, filter AlertFilter) ([]Alert, error) {
	query := `SELECT id, device_id, severity, rule, message, status, created_at, updated_at
		 FROM alerts WHERE 1=1`
	args := []any{}
	argIdx := 1

	if filter.DeviceID != "" {
		query += fmt.Sprintf(" AND device_id = $%d", argIdx)
		args = append(args, filter.DeviceID)
		argIdx++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.Severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", argIdx)
		args = append(args, filter.Severity)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	offset := (filter.Page - 1) * filter.Limit
	if offset < 0 {
		offset = 0
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filter.Limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.DeviceID, &a.Severity, &a.Rule, &a.Message, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		alerts = append(alerts, a)
	}
	if alerts == nil {
		alerts = []Alert{}
	}
	return alerts, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE alerts SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("update alert status %s: %w", id, err)
	}
	return nil
}

func (r *Repository) HasOpenAlertForRule(ctx context.Context, deviceID, rule string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM alerts
		 WHERE device_id = $1 AND rule = $2 AND status IN ('open', 'acknowledged')`,
		deviceID, rule,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check open alert: %w", err)
	}
	return count > 0, nil
}
