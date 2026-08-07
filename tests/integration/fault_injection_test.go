package integration

import (
	"testing"
	"time"

	"github.com/inframind/inframind/tests/internal/harness"
	"github.com/inframind/inframind/tests/internal/seed"
)

// TestFaultInjectionAPI verifies the internal testing endpoint publishes a
// fault-control command to the broker for the simulator to consume.
func TestFaultInjectionAPI(t *testing.T) {
	h, err := harness.Global(t)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	api := harness.NewAPIClient(h.Config().APIURL)
	env, err := seed.Seed(h, api)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Subscribe to the fault control topic so we can observe the injected command.
	observed := make(chan []byte, 4)
	if err := h.MQTTSubscribe(t, "simulator/"+env.DeviceID+"/fault", func(payload []byte) {
		observed <- payload
	}); err != nil {
		t.Fatalf("subscribe fault topic: %v", err)
	}

	// Inject a fault via the internal endpoint.
	var resp map[string]string
	code, err := api.Do("POST", "/internal/testing/fault", map[string]any{
		"deviceId": env.DeviceID,
		"fault":    "cooling_failure",
	}, &resp)
	if err != nil {
		t.Fatalf("inject fault: %v", err)
	}
	if code != 202 {
		t.Fatalf("inject fault: status %d (want 202)", code)
	}

	// The fault command must arrive on the broker topic.
	select {
	case payload := <-observed:
		t.Logf("fault command observed: %s", payload)
	case <-time.After(10 * time.Second):
		t.Fatalf("fault command was not published to broker")
	}
}
