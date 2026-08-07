package chaos

import (
	"context"
	"testing"

	"github.com/inframind/inframind/tests/internal/harness"
	"github.com/inframind/inframind/tests/internal/seed"
)

// TestBackendRestart: SIGKILL the backend, restart it, verify it comes back
// healthy and resumes ingesting telemetry. No manual intervention.
func TestBackendRestart(t *testing.T) {
	h, err := harness.Global(t)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	api := harness.NewAPIClient(h.Config().APIURL)
	env, err := seed.Seed(h, api)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Restart the backend (kills + restarts the subprocess, new port).
	if err := h.RestartBackend(); err != nil {
		t.Fatalf("restart backend: %v", err)
	}

	// Re-login on the new port (token still valid, but client baseURL changed).
	api = harness.NewAPIClient(h.Config().APIURL)
	if _, err := api.Login(seed.DefaultEmail, seed.DefaultPassword); err != nil {
		t.Fatalf("login after restart: %v", err)
	}

	// Backend must ingest telemetry again.
	if !harness.WaitFor(30_000_000_000, 1_000_000_000, func() bool {
		return mqttFlowWorks(h, env.DeviceID)
	}) {
		t.Fatalf("telemetry not ingested after backend restart")
	}

	// Seeded device still present.
	var devices []map[string]any
	code, err := api.Do("GET", "/api/v1/assets/"+env.AssetID+"/devices", nil, &devices)
	if err != nil || code != 200 {
		t.Fatalf("list devices after restart: status %d err %v", code, err)
	}
	if len(devices) == 0 {
		t.Fatalf("no devices returned after backend restart")
	}
	t.Logf("backend restart recovered: healthy, seeded data intact, telemetry flowing")
}

var _ = context.Background
