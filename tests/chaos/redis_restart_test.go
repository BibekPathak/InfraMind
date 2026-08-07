package chaos

import (
	"testing"

	"github.com/inframind/inframind/tests/internal/harness"
	"github.com/inframind/inframind/tests/internal/seed"
)

// TestRedisRestart: Redis is only used for the durable event bus (disabled in
// tests) and rate limiting (falls back to in-memory). Verify that killing
// Redis does not break ingestion or API, and that everything recovers after
// Redis comes back.
func TestRedisRestart(t *testing.T) {
	h, err := harness.Global(t)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	api := harness.NewAPIClient(h.Config().APIURL)
	env, err := seed.Seed(h, api)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := h.StopContainer("redis"); err != nil {
		t.Fatalf("stop redis: %v", err)
	}

	// Ingest must keep working with Redis down (events bus is disabled in the
	// harness; rate limiter falls back to in-memory).
	if !harness.WaitFor(30_000_000_000, 1_000_000_000, func() bool {
		return mqttFlowWorks(h, env.DeviceID)
	}) {
		t.Fatalf("telemetry ingestion stopped while Redis was down")
	}

	if err := h.StartContainer("redis"); err != nil {
		t.Fatalf("start redis: %v", err)
	}

	// Testcontainers reassigns the mapped host port on restart.
	if err := h.RefreshRedisPort(); err != nil {
		t.Fatalf("refresh redis port: %v", err)
	}

	// API still healthy after Redis returns.
	if !harness.WaitFor(15_000_000_000, 500_000_000, func() bool {
		return httpOK(api, "/api/v1/health") == 200
	}) {
		t.Fatalf("API unhealthy after Redis restart")
	}
	t.Logf("Redis restart recovered: ingestion + API unaffected")
}
