package integration

import (
	"fmt"
	"testing"

	"github.com/inframind/inframind/tests/internal/harness"
	"github.com/inframind/inframind/tests/internal/seed"
)

// TestTenantIsolationScenario: org A must never read org B's data.
//
// Org scoping is enforced server-side from the JWT's organization_id claim.
// We create org B via the admin API, mint org-scoped JWTs directly (matching
// the backend's signing scheme), and verify cross-org reads are rejected.
func TestTenantIsolationScenario(t *testing.T) {
	h, err := harness.Global(t)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}

	admin := harness.NewAPIClient(h.Config().APIURL)

	// Org A (default) with one asset.
	envA, err := seed.Seed(h, admin)
	if err != nil {
		t.Fatalf("seed org A: %v", err)
	}

	// Org B, created by admin.
	orgBID, err := seed.CreateOrg(h, admin, "Isolation Tenant B")
	if err != nil {
		t.Fatalf("create org B: %v", err)
	}
	envBOrgID := orgBID

	// Mint an org-B-scoped token (no such user exists via login; tests mint it).
	orgBToken, err := harness.MintToken("user-b", "admin", envBOrgID)
	if err != nil {
		t.Fatalf("mint org B token: %v", err)
	}
	clientB := harness.NewAPIClient(h.Config().APIURL)
	clientB.SetToken(orgBToken)

	// Org B creates an asset scoped to org B.
	var assetB struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	code, err := clientB.Do("POST", "/api/v1/assets", map[string]any{
		"name": "Org B Transformer",
		"type": "transformer",
	}, &assetB)
	if err != nil {
		t.Fatalf("org B create asset: %v", err)
	}
	if code != 201 {
		t.Fatalf("org B create asset: status %d", code)
	}

	// Admin (org A) must NOT read org B's asset.
	var readBack struct {
		ID string `json:"id"`
	}
	code, err = admin.Do("GET", "/api/v1/assets/"+assetB.ID, nil, &readBack)
	if err != nil {
		t.Fatalf("admin read org B asset: %v", err)
	}
	if code == 200 {
		t.Errorf("tenant isolation violated: admin (org A) read org B asset %s (status %d)", assetB.ID, code)
	}

	// Org B must NOT read org A's asset.
	code, err = clientB.Do("GET", "/api/v1/assets/"+envA.AssetID, nil, &readBack)
	if err != nil {
		t.Fatalf("org B read org A asset: %v", err)
	}
	if code == 200 {
		t.Errorf("tenant isolation violated: org B read org A asset %s (status %d)", envA.AssetID, code)
	}

	// Each client can still read its own asset.
	code, err = clientB.Do("GET", "/api/v1/assets/"+assetB.ID, nil, &readBack)
	if err != nil || code != 200 {
		t.Errorf("org B should read its own asset, got %d err %v", code, err)
	}
	code, err = admin.Do("GET", "/api/v1/assets/"+envA.AssetID, nil, &readBack)
	if err != nil || code != 200 {
		t.Errorf("org A should read its own asset, got %d err %v", code, err)
	}

	t.Logf("tenant isolation verified: orgA=%s orgB=%s assetB=%s (cross-org reads blocked)",
		envA.OrgID, envBOrgID, fmt.Sprintf("%.8s", assetB.ID))
}
