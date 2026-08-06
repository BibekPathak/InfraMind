package seed

import (
	"encoding/json"
	"fmt"

	"github.com/inframind/inframind/tests/internal/harness"
)

// DefaultTestUser is the admin credentials baked into the backend's auth service.
const (
	DefaultEmail    = "admin@inframind.io"
	DefaultPassword = "admin123"
)

type Asset struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	AutonomyMode string `json:"autonomyMode"`
}

type Device struct {
	ID       string `json:"id"`
	AssetID  string `json:"assetId"`
	Status   string `json:"status"`
	MQTTUser string `json:"mqttUsername"`
	MQTTPass string `json:"mqttPassword"`
}

// Environment holds references to seeded entities for a test org.
type Environment struct {
	OrgID    string
	AssetID  string
	DeviceID string
	Client   *harness.APIClient
}

// Seed creates a default org, one asset, and one device using the real API.
// Returns the environment with admin client authenticated.
func Seed(h *harness.Harness, api *harness.APIClient) (*Environment, error) {
	if _, err := api.Login(DefaultEmail, DefaultPassword); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	// Asset
	var asset Asset
	code, err := api.Do("POST", "/api/v1/assets", map[string]any{
		"name": "Test Transformer",
		"type": "transformer",
	}, &asset)
	if err != nil {
		return nil, err
	}
	if code != 201 {
		return nil, fmt.Errorf("create asset: status %d", code)
	}

	// Device (returns MQTT credentials nested under "device")
	var reg struct {
		Device       Device `json:"device"`
		MQTTUsername string `json:"mqttUsername"`
		MQTTPassword string `json:"mqttPassword"`
	}
	code, err = api.Do("POST", "/api/v1/devices", map[string]any{
		"assetId": asset.ID,
	}, &reg)
	if err != nil {
		return nil, err
	}
	if code != 201 {
		return nil, fmt.Errorf("register device: status %d", code)
	}
	dev := reg.Device
	dev.MQTTUser = reg.MQTTUsername
	dev.MQTTPass = reg.MQTTPassword

	// Org id from login response
	lr, _ := api.Login(DefaultEmail, DefaultPassword)

	return &Environment{
		OrgID:    lr.OrganizationID,
		AssetID:  asset.ID,
		DeviceID: dev.ID,
		Client:   api,
	}, nil
}

// CreateOrg creates a second, isolated org (for tenant isolation tests).
// Returns the new org's ID. Assets within the org must be created using a
// token scoped to that org.
func CreateOrg(h *harness.Harness, admin *harness.APIClient, name string) (string, error) {
	var org struct {
		ID string `json:"id"`
	}
	code, err := admin.Do("POST", "/api/v1/organizations", map[string]any{"name": name}, &org)
	if err != nil {
		return "", err
	}
	if code != 201 {
		return "", fmt.Errorf("create org: status %d", code)
	}
	return org.ID, nil
}

// MustJSON is a test helper to marshal a struct for MQTT payloads.
func MustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
