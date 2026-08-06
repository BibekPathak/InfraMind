package integration

import (
	"io"
	"net/http"
	"testing"

	"github.com/inframind/inframind/tests/internal/harness"
	"github.com/inframind/inframind/tests/internal/seed"
)

func TestMain(m *testing.M) {
	m.Run()
	harness.CloseGlobal()
}

func TestHarnessSmoke(t *testing.T) {
	h, err := harness.Global(t)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}

	// DB reachable
	if err := h.Pool().Ping(t.Context()); err != nil {
		t.Fatalf("db ping: %v", err)
	}

	// AI reachable
	if !harness.WaitFor(10_000_000_000, 250_000_000, func() bool {
		return httpOK(h.Config().AIURL + "/health")
	}) {
		t.Fatalf("ai not ready")
	}

	// Seed via API
	api := harness.NewAPIClient(h.Config().APIURL)
	env, err := seed.Seed(h, api)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if env.DeviceID == "" || env.AssetID == "" {
		t.Fatalf("seed produced empty ids: %+v", env)
	}
	t.Logf("seeded asset=%s device=%s org=%s", env.AssetID, env.DeviceID, env.OrgID)
}

func httpOK(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == 200
}
