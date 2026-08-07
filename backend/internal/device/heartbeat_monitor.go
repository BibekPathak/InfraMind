package device

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/inframind/backend/internal/eventbus"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HeartbeatMonitor struct {
	pool     *pgxpool.Pool
	bus      *eventbus.Bus
	interval time.Duration
	timeout  time.Duration
}

func NewHeartbeatMonitor(pool *pgxpool.Pool, bus *eventbus.Bus) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		pool:     pool,
		bus:      bus,
		interval: 60 * time.Second,
		timeout:  2 * time.Minute,
	}
}

func NewHeartbeatMonitorWithInterval(pool *pgxpool.Pool, bus *eventbus.Bus, interval, timeout time.Duration) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		pool:     pool,
		bus:      bus,
		interval: interval,
		timeout:  timeout,
	}
}

func (m *HeartbeatMonitor) Start(ctx context.Context) {
	slog.Info("heartbeat monitor started", "interval", m.interval, "timeout", m.timeout)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.check(ctx)
		case <-ctx.Done():
			slog.Info("heartbeat monitor stopped")
			return
		}
	}
}

func (m *HeartbeatMonitor) check(ctx context.Context) {
	cutoff := time.Now().UTC().Add(-m.timeout)

	rows, err := m.pool.Query(ctx,
		`UPDATE devices
		 SET status = 'offline', updated_at = NOW()
		 WHERE status = 'online' AND last_heartbeat < $1 AND deleted_at IS NULL
		 RETURNING id`,
		cutoff,
	)
	if err != nil {
		slog.Error("heartbeat monitor query failed", "error", err)
		return
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var deviceID string
		if err := rows.Scan(&deviceID); err != nil {
			slog.Error("heartbeat monitor scan failed", "error", err)
			continue
		}
		m.bus.Publish(eventbus.NewEvent("device.status_changed", "heartbeat_monitor", map[string]string{
			"deviceId": deviceID,
			"from":     "online",
			"to":       "offline",
		}))
		count++
	}

	if count > 0 {
		slog.Info("devices marked offline due to heartbeat timeout", "count", count, "timeout", m.timeout)
	}
}

func (m *HeartbeatMonitor) String() string {
	return fmt.Sprintf("HeartbeatMonitor(interval=%s, timeout=%s)", m.interval, m.timeout)
}
