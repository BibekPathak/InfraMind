package chaos

import (
	"testing"
	"time"

	"github.com/inframind/inframind/tests/internal/harness"
	"github.com/inframind/inframind/tests/internal/seed"
)

// TestAIRestart: kill the AI service, verify the backend falls back to its
// deterministic health scoring (no crash), then restart AI and verify the
// health endpoint works with the real engine again.
func TestAIRestart(t *testing.T) {
	h, err := harness.Global(t)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	api := harness.NewAPIClient(h.Config().APIURL)
	env, err := seed.Seed(h, api)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Baseline: health works with real AI.
	if !harness.WaitFor(15_000_000_000, 500_000_000, func() bool {
		return healthOK(api, env.DeviceID)
	}) {
		t.Fatalf("baseline health failed")
	}

	if err := h.StopAI(); err != nil {
		t.Fatalf("stop ai: %v", err)
	}

	// Wait a moment for the AI process to actually die.
	time.Sleep(2 * time.Second)

	// With AI down, backend must still answer health (deterministic fallback).
	if !harness.WaitFor(15_000_000_000, 500_000_000, func() bool {
		return healthOK(api, env.DeviceID)
	}) {
		t.Fatalf("health endpoint failed with AI down (should fall back deterministically)")
	}

	if err := h.RestartAI(); err != nil {
		t.Fatalf("restart ai: %v", err)
	}

	// After AI returns, health must still work (now with real engine).
	if !harness.WaitFor(30_000_000_000, 1_000_000_000, func() bool {
		return healthOK(api, env.DeviceID)
	}) {
		t.Fatalf("health endpoint failed after AI restart")
	}
	t.Logf("AI restart recovered: deterministic fallback worked, real engine resumed")
}

func healthOK(api *harness.APIClient, deviceID string) bool {
	code, _ := api.Do("GET", "/api/v1/health/"+deviceID+"?temperature=70&current=100&voltage=11400&humidity=45", nil, nil)
	return code == 200
}
