package integration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/inframind/inframind/tests/internal/harness"
	"github.com/inframind/inframind/tests/internal/seed"
)

func runScenario(t *testing.T, scenarioName string, timeout time.Duration) {
	h, err := harness.Global(t)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}

	api := harness.NewAPIClient(h.Config().APIURL)
	env, err := seed.Seed(h, api)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	sc, err := harness.LoadScenario(filepath.Join("..", "scenarios", scenarioName+".yaml"))
	if err != nil {
		t.Fatalf("load scenario %s: %v", scenarioName, err)
	}

	if err := sc.Play(h, env.DeviceID); err != nil {
		t.Fatalf("play scenario %s: %v", scenarioName, err)
	}

	// Let the alert engine + twin sync + action executor process.
	time.Sleep(3 * time.Second)

	if err := sc.VerifyHealth(h, api, env.DeviceID, timeout); err != nil {
		t.Errorf("scenario %s health: %v", scenarioName, err)
	}
	if err := sc.VerifyCounts(h, api, timeout); err != nil {
		t.Errorf("scenario %s counts: %v", scenarioName, err)
	}
}

// TestHealthyScenario: steady normal readings -> healthy, no alerts, no work orders.
func TestHealthyScenario(t *testing.T) {
	runScenario(t, "healthy", 20*time.Second)
}

// TestOverloadScenario: ramp current/temp -> alerts + work orders raised.
func TestOverloadScenario(t *testing.T) {
	runScenario(t, "overload", 30*time.Second)
}

// TestCoolingFailureScenario: temp spikes with normal current -> critical alert + work order.
func TestCoolingFailureScenario(t *testing.T) {
	runScenario(t, "cooling_failure", 30*time.Second)
}

// TestOfflineScenario: device stops publishing -> heartbeat timeout -> offline alert + work order.
func TestOfflineScenario(t *testing.T) {
	h, err := harness.Global(t)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	api := harness.NewAPIClient(h.Config().APIURL)
	env, err := seed.Seed(h, api)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	sc, err := harness.LoadScenario(filepath.Join("..", "scenarios", "offline.yaml"))
	if err != nil {
		t.Fatalf("load scenario offline: %v", err)
	}

	if err := sc.Play(h, env.DeviceID); err != nil {
		t.Fatalf("play scenario offline: %v", err)
	}

	// Stop publishing. Heartbeat timeout in the harness is 6s; the monitor
	// polls every 2s. Wait long enough for the device to be marked offline
	// and the offline alert + auto work order to be raised.
	if err := sc.VerifyCounts(h, api, 60*time.Second); err != nil {
		t.Errorf("scenario offline counts: %v", err)
	}
}
