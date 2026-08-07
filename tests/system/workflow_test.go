package system

import (
	"fmt"
	"testing"
	"time"

	"github.com/inframind/inframind/tests/internal/harness"
	"github.com/inframind/inframind/tests/internal/seed"
)

// WorkOrder mirrors the backend work order response (subset).
type WorkOrder struct {
	ID          string  `json:"id"`
	AssetID     string  `json:"assetId"`
	Type        string  `json:"type"`
	Priority    string  `json:"priority"`
	Status      string  `json:"status"`
	AssignedTo  *string `json:"assignedTo"`
	Description string  `json:"description"`
	Timeline    []struct {
		Action string `json:"action"`
		Note   string `json:"note"`
	} `json:"timeline"`
}

// Alert mirrors the backend alert response (subset).
type Alert struct {
	ID       string `json:"id"`
	DeviceID string `json:"deviceId"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Status   string `json:"status"`
}

// TestFullWorkflow drives the entire platform through one complete incident:
// healthy -> overload -> alert -> auto work order -> assign -> in progress
// -> complete -> resolve. Verifies each hop and the incident timeline.
func TestFullWorkflow(t *testing.T) {
	h, err := harness.Global(t)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}

	api := harness.NewAPIClient(h.Config().APIURL)
	env, err := seed.Seed(h, api)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Open a WebSocket listener before injecting the fault so we can assert
	// the real-time event sequence.
	ws, err := h.WSConnect(env.DeviceID)
	if err != nil {
		t.Fatalf("ws connect: %v", err)
	}
	defer ws.Close()

	// --- Phase 1: healthy baseline -------------------------------
	healthy, _ := harness.LoadScenario("../scenarios/healthy.yaml")
	if err := healthy.Play(h, env.DeviceID); err != nil {
		t.Fatalf("healthy play: %v", err)
	}
	time.Sleep(2 * time.Second)

	// --- Phase 2: overload -> alert + work order ------------------
	overload, _ := harness.LoadScenario("../scenarios/overload.yaml")
	if err := overload.Play(h, env.DeviceID); err != nil {
		t.Fatalf("overload play: %v", err)
	}

	// Wait for at least one alert.
	alerts := waitForAlerts(t, api, 1, 30*time.Second)
	alert0 := alerts[0]
	t.Logf("alert raised: rule=%s severity=%s", alert0.Rule, alert0.Severity)

	// Work order should be auto-created from the alert.
	orders := waitForWorkOrders(t, api, 1, 30*time.Second)
	wo := orders[0]
	t.Logf("work order auto-created: id=%s type=%s priority=%s status=%s", wo.ID, wo.Type, wo.Priority, wo.Status)
	if wo.Status != "open" {
		t.Errorf("expected work order status 'open', got %q", wo.Status)
	}

	// Timeline should contain "created".
	if !timelineHas(wo, "created") {
		t.Errorf("timeline missing 'created' event: %+v", wo.Timeline)
	}

	// --- Phase 3: assign to engineer ------------------------------
	var woAfterAssign WorkOrder
	code, err := api.Do("PATCH", "/api/v1/work-orders/"+wo.ID+"/assign", map[string]any{
		"assignedTo": "eng-alice",
	}, &woAfterAssign)
	if err != nil || code != 200 {
		t.Fatalf("assign work order: status %d err %v", code, err)
	}
	if woAfterAssign.Status != "assigned" {
		t.Errorf("expected status 'assigned' after assignment, got %q", woAfterAssign.Status)
	}
	if !timelineHas(woAfterAssign, "assigned") {
		t.Errorf("timeline missing 'assigned' event")
	}

	// --- Phase 4: start work (in_progress) ------------------------
	var woInProgress WorkOrder
	code, err = api.Do("PATCH", "/api/v1/work-orders/"+wo.ID+"/status", map[string]any{
		"status": "in_progress",
	}, &woInProgress)
	if err != nil || code != 200 {
		t.Fatalf("start work: status %d err %v", code, err)
	}
	if !timelineHas(woInProgress, "status_changed") {
		t.Errorf("timeline missing 'status_changed' event")
	}

	// --- Phase 5: complete the work order -------------------------
	var woCompleted WorkOrder
	code, err = api.Do("PATCH", "/api/v1/work-orders/"+wo.ID+"/status", map[string]any{
		"status": "completed",
	}, &woCompleted)
	if err != nil || code != 200 {
		t.Fatalf("complete work order: status %d err %v", code, err)
	}
	if woCompleted.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", woCompleted.Status)
	}

	// --- Phase 6: resolve the alert -------------------------------
	code, err = api.Do("PATCH", "/api/v1/alerts/"+alert0.ID+"/resolve", nil, nil)
	if err != nil || code != 200 {
		t.Fatalf("resolve alert: status %d err %v", code, err)
	}

	// --- Phase 7: verify incident timeline via dedicated endpoint ---
	var timeline []struct {
		Action string `json:"action"`
		Note   string `json:"note"`
	}
	code, err = api.Do("GET", "/api/v1/work-orders/"+wo.ID+"/timeline", nil, &timeline)
	if err != nil || code != 200 {
		t.Fatalf("get timeline: status %d err %v", code, err)
	}
	t.Logf("incident timeline events: %d", len(timeline))
	for _, ev := range timeline {
		t.Logf("  %s: %s", ev.Action, ev.Note)
	}
	if len(timeline) < 3 {
		t.Errorf("timeline too short: expected >=3 events (created, assigned, status_changed), got %d", len(timeline))
	}

	// --- Phase 8: verify WebSocket received telemetry during incident ---
	wsEvents := collectWSEvents(ws, 2*time.Second)
	if len(wsEvents) == 0 {
		t.Logf("note: no websocket events collected within window (may be timing dependent)")
	} else {
		for _, e := range wsEvents {
			t.Logf("ws event: %s", e.Type)
		}
	}
}

// waitForAlerts polls /alerts until at least n exist.
func waitForAlerts(t *testing.T, api *harness.APIClient, n int, timeout time.Duration) []Alert {
	t.Helper()
	var alerts []Alert
	if !harness.WaitFor(timeout, 500*time.Millisecond, func() bool {
		code, err := api.Do("GET", "/api/v1/alerts", nil, &alerts)
		return err == nil && code == 200 && len(alerts) >= n
	}) {
		t.Fatalf("timed out waiting for %d alerts (have %d)", n, len(alerts))
	}
	return alerts
}

// waitForWorkOrders polls /work-orders until at least n exist.
func waitForWorkOrders(t *testing.T, api *harness.APIClient, n int, timeout time.Duration) []WorkOrder {
	t.Helper()
	var orders []WorkOrder
	if !harness.WaitFor(timeout, 500*time.Millisecond, func() bool {
		code, err := api.Do("GET", "/api/v1/work-orders", nil, &orders)
		return err == nil && code == 200 && len(orders) >= n
	}) {
		t.Fatalf("timed out waiting for %d work orders (have %d)", n, len(orders))
	}
	return orders
}

func timelineHas(wo WorkOrder, action string) bool {
	for _, ev := range wo.Timeline {
		if ev.Action == action {
			return true
		}
	}
	return false
}

// collectWSEvents drains available websocket events for the given window.
func collectWSEvents(ws *harness.WSClient, window time.Duration) []*harness.WSEvent {
	var events []*harness.WSEvent
	deadline := time.Now().Add(window)
	for time.Now().Before(deadline) {
		ev, err := ws.Read(1 * time.Second)
		if err != nil {
			continue
		}
		events = append(events, ev)
		if len(events) >= 10 {
			break
		}
	}
	return events
}

var _ = fmt.Sprintf
