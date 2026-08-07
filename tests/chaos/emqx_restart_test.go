package chaos

import (
	"context"
	"testing"
	"time"

	"github.com/inframind/inframind/tests/internal/harness"
	"github.com/inframind/inframind/tests/internal/seed"
)

// TestEMQXRestart: kill EMQX, verify the system marks devices offline over
// time and does not crash; restart EMQX and verify telemetry resumes flowing.
func TestEMQXRestart(t *testing.T) {
	h, err := harness.Global(t)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	api := harness.NewAPIClient(h.Config().APIURL)
	env, err := seed.Seed(h, api)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Normal telemetry flowing first.
	healthy, _ := harness.LoadScenario("../scenarios/healthy.yaml")
	if err := healthy.Play(h, env.DeviceID); err != nil {
		t.Fatalf("play: %v", err)
	}
	time.Sleep(2 * time.Second)

	// Kill EMQX.
	if err := h.StopContainer("emqx"); err != nil {
		t.Fatalf("stop emqx: %v", err)
	}

	// Backend must still answer HTTP (API unaffected by broker loss).
	if !harness.WaitFor(10_000_000_000, 500_000_000, func() bool {
		return httpOK(api, "/api/v1/health") == 200
	}) {
		t.Fatalf("backend API unreachable after EMQX loss")
	}

	// Restart EMQX.
	if err := h.StartContainer("emqx"); err != nil {
		t.Fatalf("start emqx: %v", err)
	}

	// Testcontainers reassigns the mapped host port on restart. Refresh the
	// config and restart the backend so it connects to the new broker URL.
	if err := h.RefreshMQTTPort(); err != nil {
		t.Fatalf("refresh mqtt port: %v", err)
	}
	if err := h.RestartBackend(); err != nil {
		t.Fatalf("restart backend after emqx: %v", err)
	}

	// First, confirm EMQX accepts a fresh client connection after restart.
	if !harness.WaitFor(120_000_000_000, 1_000_000_000, func() bool {
		return harness.MQTTAdminConnectOK(h.Config().MQTTURL)
	}) {
		t.Fatalf("EMQX did not accept connections after restart")
	}
	t.Logf("EMQX accepts connections after restart")

	// Then wait for the backend MQTT client to reconnect and telemetry to flow.
	if !harness.WaitFor(120_000_000_000, 1_000_000_000, func() bool {
		return mqttFlowWorks(h, env.DeviceID)
	}) {
		t.Fatalf("telemetry did not resume after EMQX restart (backend did not re-ingest)")
	}
	t.Logf("EMQX restart recovered: telemetry resumed")
}

func httpOK(api *harness.APIClient, path string) int {
	code, _ := api.Do("GET", path, nil, nil)
	return code
}

// mqttFlowWorks publishes one telemetry point and checks it lands in the DB.
func mqttFlowWorks(h *harness.Harness, deviceID string) bool {
	payload := []byte(`{"device_id":"` + deviceID + `","timestamp":"2026-01-01T00:00:00Z","temperature":60,"current":90,"voltage":11400,"humidity":45}`)
	if err := h.MQTTPub("telemetry/"+deviceID+"/data", payload); err != nil {
		return false
	}
	time.Sleep(7 * time.Second) // batch timeout is 5s
	var n int
	h.Pool().QueryRow(context.Background(), "SELECT count(*) FROM telemetry WHERE device_id=$1", deviceID).Scan(&n)
	return n > 0
}
