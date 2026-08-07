package chaos

import (
	"testing"

	"github.com/inframind/inframind/tests/internal/harness"
	"github.com/inframind/inframind/tests/internal/seed"
)

// TestDBRestart: stop TimescaleDB, verify the backend survives (still answers
// HTTP, batches locally without crashing), then restart DB and verify
// telemetry ingestion resumes after the backend reconnects.
func TestDBRestart(t *testing.T) {
	h, err := harness.Global(t)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	api := harness.NewAPIClient(h.Config().APIURL)
	env, err := seed.Seed(h, api)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := h.StopContainer("timescaledb"); err != nil {
		t.Fatalf("stop timescaledb: %v", err)
	}

	// Backend must remain up (HTTP still answers) even with DB down.
	if !harness.WaitFor(15_000_000_000, 500_000_000, func() bool {
		return httpOK(api, "/api/v1/health") == 200
	}) {
		t.Fatalf("backend died or became unresponsive while DB was down")
	}

	if err := h.StartContainer("timescaledb"); err != nil {
		t.Fatalf("start timescaledb: %v", err)
	}

	// Testcontainers reassigns the mapped host port on restart. Refresh the
	// config and restart the backend so it connects to the new port.
	if err := h.RefreshDBPort(); err != nil {
		t.Fatalf("refresh db port: %v", err)
	}
	if err := h.RestartBackend(); err != nil {
		t.Fatalf("restart backend after db: %v", err)
	}

	// After DB returns, telemetry ingestion must resume.
	if !harness.WaitFor(60_000_000_000, 1_000_000_000, func() bool {
		return mqttFlowWorks(h, env.DeviceID)
	}) {
		t.Fatalf("telemetry not ingested after DB restart")
	}
	t.Logf("DB restart recovered: backend survived, ingestion resumed")
}
