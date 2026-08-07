package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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
		`INSERT INTO telemetry (time, device_id, temperature, current_amps, voltage, humidity, flow_rate, pressure, vibration, rpm, output_power)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		t.Time, t.DeviceID, t.Temperature, t.Current, t.Voltage, t.Humidity,
		t.FlowRate, t.Pressure, t.Vibration, t.RPM, t.OutputPower,
	)
	if err != nil {
		return fmt.Errorf("insert telemetry: %w", err)
	}
	return nil
}

func (r *Repository) BatchInsert(ctx context.Context, points []Telemetry) error {
	if len(points) == 0 {
		return nil
	}

	rows := make([][]any, len(points))
	for i, p := range points {
		rows[i] = []any{
			p.Time, p.DeviceID, p.Temperature, p.Current, p.Voltage, p.Humidity,
			p.FlowRate, p.Pressure, p.Vibration, p.RPM, p.OutputPower,
		}
	}

	_, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"telemetry"},
		[]string{"time", "device_id", "temperature", "current_amps", "voltage", "humidity", "flow_rate", "pressure", "vibration", "rpm", "output_power"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("batch insert telemetry (%d points): %w", len(points), err)
	}
	return nil
}

func (r *Repository) Query(ctx context.Context, deviceID string, from, to time.Time, limit int) ([]Telemetry, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	rows, err := r.pool.Query(ctx,
		`SELECT time, device_id, temperature, current_amps, voltage, humidity, flow_rate, pressure, vibration, rpm, output_power
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
		if err := rows.Scan(&t.Time, &t.DeviceID, &t.Temperature, &t.Current, &t.Voltage, &t.Humidity, &t.FlowRate, &t.Pressure, &t.Vibration, &t.RPM, &t.OutputPower); err != nil {
			return nil, fmt.Errorf("scan telemetry: %w", err)
		}
		results = append(results, t)
	}
	return results, nil
}

func (r *Repository) GetLatest(ctx context.Context, deviceID string) (*Telemetry, error) {
	t := &Telemetry{}
	err := r.pool.QueryRow(ctx,
		`SELECT time, device_id, temperature, current_amps, voltage, humidity, flow_rate, pressure, vibration, rpm, output_power
		 FROM telemetry
		 WHERE device_id = $1
		 ORDER BY time DESC
		 LIMIT 1`, deviceID,
	).Scan(&t.Time, &t.DeviceID, &t.Temperature, &t.Current, &t.Voltage, &t.Humidity, &t.FlowRate, &t.Pressure, &t.Vibration, &t.RPM, &t.OutputPower)
	if err != nil {
		return nil, fmt.Errorf("get latest telemetry %s: %w", deviceID, err)
	}
	return t, nil
}

// DistinctRecentDeviceIDs returns device IDs with telemetry in the last 15 minutes.
func (r *Repository) DistinctRecentDeviceIDs(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT device_id FROM telemetry
		 WHERE time > NOW() - INTERVAL '15 minutes'
		 ORDER BY device_id`)
	if err != nil {
		return nil, fmt.Errorf("distinct recent device ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan device id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *Repository) QueryLatest(ctx context.Context, deviceID string, n int) ([]Telemetry, error) {
	if n <= 0 || n > 100 {
		n = 10
	}
	rows, err := r.pool.Query(ctx,
		`SELECT time, device_id, temperature, current_amps, voltage, humidity, flow_rate, pressure, vibration, rpm, output_power
		 FROM telemetry
		 WHERE device_id = $1
		 ORDER BY time DESC
		 LIMIT $2`, deviceID, n)
	if err != nil {
		return nil, fmt.Errorf("query latest telemetry: %w", err)
	}
	defer rows.Close()

	results := make([]Telemetry, 0, n)
	for rows.Next() {
		var t Telemetry
		if err := rows.Scan(&t.Time, &t.DeviceID, &t.Temperature, &t.Current, &t.Voltage, &t.Humidity, &t.FlowRate, &t.Pressure, &t.Vibration, &t.RPM, &t.OutputPower); err != nil {
			return nil, fmt.Errorf("scan telemetry: %w", err)
		}
		results = append(results, t)
	}
	return results, nil
}

type AggregatedPoint struct {
	Bucket     time.Time `json:"bucket"`
	AvgTemp    *float64  `json:"avgTemp"`
	MaxTemp    *float64  `json:"maxTemp"`
	MinTemp    *float64  `json:"minTemp"`
	AvgCurrent *float64  `json:"avgCurrent"`
	MaxCurrent *float64  `json:"maxCurrent"`
	AvgVoltage *float64  `json:"avgVoltage"`
}

func (r *Repository) Aggregate(ctx context.Context, deviceID string, from, to time.Time, window string) ([]AggregatedPoint, error) {
	query := fmt.Sprintf(`
		SELECT
			time_bucket('%s', time) AS bucket,
			AVG(temperature), MAX(temperature), MIN(temperature),
			AVG(current_amps), MAX(current_amps),
			AVG(voltage)
		FROM telemetry
		WHERE device_id = $1 AND time >= $2 AND time <= $3
		GROUP BY bucket
		ORDER BY bucket ASC`, window)

	rows, err := r.pool.Query(ctx, query, deviceID, from, to)
	if err != nil {
		return nil, fmt.Errorf("aggregate telemetry: %w", err)
	}
	defer rows.Close()

	var results []AggregatedPoint
	for rows.Next() {
		var p AggregatedPoint
		if err := rows.Scan(&p.Bucket, &p.AvgTemp, &p.MaxTemp, &p.MinTemp, &p.AvgCurrent, &p.MaxCurrent, &p.AvgVoltage); err != nil {
			return nil, fmt.Errorf("scan aggregate: %w", err)
		}
		results = append(results, p)
	}
	if results == nil {
		results = []AggregatedPoint{}
	}
	return results, nil
}
