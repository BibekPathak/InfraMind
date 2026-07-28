package telemetry

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

func (r *Repository) Insert(ctx context.Context, t *Telemetry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO telemetry (time, device_id, temperature, current_amps, voltage, humidity)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		t.Time, t.DeviceID, t.Temperature, t.Current, t.Voltage, t.Humidity,
	)
	if err != nil {
		return fmt.Errorf("insert telemetry: %w", err)
	}
	return nil
}

func (r *Repository) Query(ctx context.Context, deviceID string, from, to time.Time, limit int) ([]Telemetry, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	rows, err := r.pool.Query(ctx,
		`SELECT time, device_id, temperature, current_amps, voltage, humidity
		 FROM telemetry
		 WHERE device_id = $1 AND time >= $2 AND time <= $3
		 ORDER BY time DESC
		 LIMIT $4`, deviceID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("query telemetry: %w", err)
	}
	defer rows.Close()

	var results []Telemetry
	for rows.Next() {
		var t Telemetry
		if err := rows.Scan(&t.Time, &t.DeviceID, &t.Temperature, &t.Current, &t.Voltage, &t.Humidity); err != nil {
			return nil, fmt.Errorf("scan telemetry: %w", err)
		}
		results = append(results, t)
	}
	return results, nil
}

func (r *Repository) GetLatest(ctx context.Context, deviceID string) (*Telemetry, error) {
	t := &Telemetry{}
	err := r.pool.QueryRow(ctx,
		`SELECT time, device_id, temperature, current_amps, voltage, humidity
		 FROM telemetry
		 WHERE device_id = $1
		 ORDER BY time DESC
		 LIMIT 1`, deviceID,
	).Scan(&t.Time, &t.DeviceID, &t.Temperature, &t.Current, &t.Voltage, &t.Humidity)
	if err != nil {
		return nil, fmt.Errorf("get latest telemetry %s: %w", deviceID, err)
	}
	return t, nil
}
