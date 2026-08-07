package performance

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/inframind/inframind/tests/internal/harness"
	"github.com/inframind/inframind/tests/internal/seed"
)

func TestMain(m *testing.M) {
	code := m.Run()
	harness.CloseGlobal()
	os.Exit(code)
}

// TestIngestionSmoke validates sustained ingestion at increasing device counts
// (1, 10, 100) and asserts the /telemetry/live query stays fast. It is a
// gate (pass/fail), not a benchmark report.
func TestIngestionSmoke(t *testing.T) {
	h, err := harness.Global(t)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	api := harness.NewAPIClient(h.Config().APIURL)
	env, err := seed.Seed(h, api)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	for _, devices := range []int{1, 10, 100} {
		t.Run(fmt.Sprintf("devices_%d", devices), func(t *testing.T) {
			runLoadPhase(t, h, api, env, devices)
		})
	}
}

// runLoadPhase registers N devices, publishes sustained telemetry for ~6s at
// ~5 msg/s each, and checks ingestion keeps up and the live query is fast.
func runLoadPhase(t *testing.T, h *harness.Harness, api *harness.APIClient, env *seed.Environment, devices int) {
	ids := make([]string, devices)
	for i := range ids {
		var reg struct {
			Device struct {
				ID string `json:"id"`
			} `json:"device"`
		}
		code, err := api.Do("POST", "/api/v1/devices", map[string]any{
			"assetId": env.AssetID,
		}, &reg)
		if err != nil || code != 201 {
			t.Fatalf("register device %d: status %d err %v", i, code, err)
		}
		ids[i] = reg.Device.ID
	}

	duration := 6 * time.Second
	interval := 200 * time.Millisecond
	start := time.Now()
	end := start.Add(duration)

	published := 0
	for time.Now().Before(end) {
		for _, id := range ids {
			payload := seed.MustJSON(map[string]any{
				"device_id":   id,
				"timestamp":   time.Now().UTC().Format(time.RFC3339),
				"temperature": 65,
				"current":     100,
				"voltage":     11400,
				"humidity":    45,
				"scenario":    "perf",
			})
			if err := h.MQTTPub(fmt.Sprintf("telemetry/%s/data", id), payload); err != nil {
				t.Fatalf("publish: %v", err)
			}
			published++
		}
		time.Sleep(interval)
	}

	// Allow batch flush (5s timeout) to drain.
	time.Sleep(8 * time.Second)

	// 1. Ingestion kept up: count rows for all load devices.
	var ingested int
	err := h.Pool().QueryRow(context.Background(),
		`SELECT count(*) FROM telemetry WHERE device_id = ANY($1)`, ids).Scan(&ingested)
	if err != nil {
		t.Fatalf("count ingested: %v", err)
	}
	minExpected := int(float64(published) * 0.7)
	if ingested < minExpected {
		t.Errorf("ingestion lagged: published=%d ingested=%d (expected >= %d)", published, ingested, minExpected)
	} else {
		t.Logf("ingestion kept up: published=%d ingested=%d (%.1f%%)", published, ingested, 100*float64(ingested)/float64(published))
	}

	// 2. Live query latency: p95 under 500ms.
	latencies := measureLiveLatency(api, ids[0], 20)
	if len(latencies) == 0 {
		t.Fatalf("no live telemetry returned")
	}
	p95 := percentile(latencies, 0.95)
	t.Logf("live query p95=%dms across %d samples", p95.Milliseconds(), len(latencies))
	if p95 > 500*time.Millisecond {
		t.Errorf("live query p95 too slow: %v (>500ms)", p95)
	}
}

// measureLiveLatency samples the live-telemetry endpoint and returns latencies.
func measureLiveLatency(api *harness.APIClient, deviceID string, samples int) []time.Duration {
	var out []time.Duration
	for i := 0; i < samples; i++ {
		start := time.Now()
		var v map[string]any
		code, err := api.Do("GET", "/api/v1/telemetry/live?device_id="+deviceID, nil, &v)
		if err == nil && code == 200 {
			out = append(out, time.Since(start))
		}
	}
	return out
}

func percentile(ds []time.Duration, p float64) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	for i := 1; i < len(ds); i++ {
		for j := i; j > 0 && ds[j] < ds[j-1]; j-- {
			ds[j], ds[j-1] = ds[j-1], ds[j]
		}
	}
	idx := int(p * float64(len(ds)-1))
	return ds[idx]
}
