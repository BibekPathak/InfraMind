package action

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/inframind/backend/internal/eventbus"
)

type Publisher interface {
	Publish(topic string, qos byte, payload []byte) error
}

type ExecutorMetrics interface {
	IncActionsExecuted()
}

type Executor struct {
	svc     *Service
	bus     *eventbus.Bus
	pub     Publisher
	policy  *PolicyEvaluator
	metrics ExecutorMetrics
}

func NewExecutor(svc *Service, bus *eventbus.Bus, pub Publisher, policy *PolicyEvaluator, metrics ExecutorMetrics) *Executor {
	return &Executor{svc: svc, bus: bus, pub: pub, policy: policy, metrics: metrics}
}

func (e *Executor) Run(ctx context.Context) {
	slog.Info("action executor started")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.processPending(ctx)
		case <-ctx.Done():
			slog.Info("action executor stopped")
			return
		}
	}
}

func (e *Executor) processPending(ctx context.Context) {
	approved, err := e.svc.List(ctx, Filter{Status: "approved", Limit: 50})
	if err != nil {
		slog.Error("action executor: list approved failed", "error", err)
		return
	}
	for _, a := range approved {
		e.Execute(ctx, a)
	}

	if e.policy != nil {
		e.autoApproveProposed(ctx)
	}
}

func (e *Executor) autoApproveProposed(ctx context.Context) {
	proposed, err := e.svc.List(ctx, Filter{Status: "proposed", Limit: 50})
	if err != nil {
		slog.Error("action executor: list proposed failed", "error", err)
		return
	}

	for _, a := range proposed {
		autoOK, reason := e.policy.Evaluate(ctx, a)
		if !autoOK {
			continue
		}

		if _, err := e.svc.Approve(ctx, a.ID); err != nil {
			slog.Error("action executor: auto-approve failed", "error", err, "actionId", a.ID)
			continue
		}
		slog.Info("action auto-approved by policy", "actionId", a.ID, "type", a.Type, "reason", reason)
		e.bus.Publish(eventbus.NewEvent("action.approved", "policy", map[string]any{
			"actionId": a.ID,
			"type":     a.Type,
			"reason":   reason,
		}))

		e.Execute(ctx, a)
	}
}

func (e *Executor) Execute(ctx context.Context, a Action) {
	if e.pub == nil {
		e.markFailed(a, "no MQTT publisher configured")
		return
	}

	topic := "device/command"
	if a.DeviceID != nil {
		topic = fmt.Sprintf("device/%s/command", *a.DeviceID)
	}

	cmd := map[string]any{
		"action_id": a.ID,
		"type":      a.Type,
		"payload":   a.Payload,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	payload, err := json.Marshal(cmd)
	if err != nil {
		e.markFailed(a, "failed to marshal command")
		return
	}

	if err := e.pub.Publish(topic, 1, payload); err != nil {
		e.markFailed(a, err.Error())
		return
	}

	slog.Info("action executed",
		"actionId", a.ID, "type", a.Type, "deviceId", deref(a.DeviceID), "topic", topic)

	if e.metrics != nil {
		e.metrics.IncActionsExecuted()
	}

	if err := e.svc.MarkExecuted(ctx, a.ID, "command published to "+topic); err != nil {
		slog.Error("action executor: mark executed failed", "error", err, "actionId", a.ID)
		return
	}

	e.bus.Publish(eventbus.NewEvent("action.executed", "executor", map[string]any{
		"actionId": a.ID,
		"type":     a.Type,
		"result":   "executed",
	}))
}

func (e *Executor) markFailed(a Action, reason string) {
	slog.Error("action execution failed", "actionId", a.ID, "reason", reason)
	if err := e.svc.MarkFailed(context.Background(), a.ID, reason); err != nil {
		slog.Error("action executor: mark failed failed", "error", err, "actionId", a.ID)
	}
	e.bus.Publish(eventbus.NewEvent("action.failed", "executor", map[string]any{
		"actionId": a.ID,
		"reason":   reason,
	}))
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
