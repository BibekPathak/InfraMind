package device

import "time"

type Device struct {
	ID              string         `json:"id"`
	AssetID         string         `json:"assetId"`
	FirmwareVersion string         `json:"firmwareVersion"`
	Status          string         `json:"status"`
	Location        map[string]any `json:"location,omitempty"`
	Config          map[string]any `json:"config,omitempty"`
	LastHeartbeat   *time.Time     `json:"lastHeartbeat,omitempty"`
	MQTTUsername    string         `json:"-"`
	MQTTPassword    string         `json:"-"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       *time.Time     `json:"deletedAt,omitempty"`
}

type RegisterDeviceRequest struct {
	AssetID         string         `json:"assetId"`
	FirmwareVersion string         `json:"firmwareVersion"`
	Location        map[string]any `json:"location,omitempty"`
}

type RegistrationResponse struct {
	Device       Device `json:"device"`
	MQTTUsername string `json:"mqttUsername"`
	MQTTPassword string `json:"mqttPassword"`
}

type ConfigUpdateRequest struct {
	Config map[string]any `json:"config"`
}
