package alert

import (
	"context"
	"log/slog"
	"time"

	"github.com/inframind/backend/internal/eventbus"
	"github.com/inframind/backend/internal/telemetry"
)

type Engine struct {
	alertSvc *Service
	bus      *eventbus.Bus
	notifier Notifier
	telemetryRepo *telemetry.Repository
	deviceIDs []string
	interval  time.Duration
}

func NewEngine(alertSvc *Service, bus *eventbus.Bus, notifier Notifier, telemetryRepo *telemetry.Repository) *Engine {
	return &Engine{
		alertSvc:      alertSvc,
		bus:           bus,
		notifier:      notifier,
		telemetryRepo: telemetryRepo,
		interval:      10 * time.Second,
	}
}

func (e *Engine) Start(ctx context.Context) {
	slog.Info("alert engine started", "interval", e.interval)
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.evaluate(ctx)
		case <-ctx.Done():
			slog.Info("alert engine stopped")
			return
		}
	}
}

func (e *Engine) evaluate(ctx context.Context) {
	ids := e.deviceIDs
	if len(ids) == 0 {
		ids = []string{"tx-001"}
	}

	for _, deviceID := range ids {
		t, err := e.telemetryRepo.GetLatest(ctx, deviceID)
		if err != nil {
			continue
		}

		e.evaluateDevice(ctx, deviceID, t)
	}
}

func (e *Engine) evaluateDevice(ctx context.Context, deviceID string, t *telemetry.Telemetry) {
	type ruleCheck struct {
		rule     string
		severity string
		message  string
		check    func() bool
	}

	rules := []ruleCheck{
		{
			rule: "temperature_critical", severity: "critical",
			message: "Temperature critically high",
			check:   func() bool { return t.Temperature > 90 },
		},
		{
			rule: "temperature_warning", severity: "warning",
			message: "Temperature elevated above threshold",
			check:   func() bool { return t.Temperature > 80 },
		},
		{
			rule: "current_overload", severity: "warning",
			message: "Current draw exceeds normal range",
			check:   func() bool { return t.Current > 180 },
		},
		{
			rule: "sensor_failure", severity: "critical",
			message: "Sensor reading zero while device is online",
			check:   func() bool { return t.Current == 0 && t.Temperature < 30 },
		},
		{
			rule: "voltage_sag", severity: "warning",
			message: "Voltage dropped below safe operating range",
			check:   func() bool { return t.Voltage > 0 && t.Voltage < 10000 },
		},
	}

	for _, rc := range rules {
		if !rc.check() {
			continue
		}

		open, err := e.alertSvc.HasOpenAlertForRule(ctx, deviceID, rc.rule)
		if err != nil {
			slog.Warn("alert engine check failed", "error", err, "rule", rc.rule)
			continue
		}
		if open {
			continue
		}

		a, err := e.alertSvc.Create(ctx, deviceID, rc.severity, rc.rule, rc.message)
		if err != nil {
			slog.Error("alert engine create failed", "error", err, "rule", rc.rule)
			continue
		}

		slog.Warn("alert raised", "deviceId", deviceID, "rule", rc.rule, "severity", rc.severity)
		e.bus.Publish(eventbus.NewEvent("alert.raised", "alert_engine", a))
		e.notifier.Send(a)
	}
}
