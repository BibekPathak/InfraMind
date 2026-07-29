package twin

import "time"

type DigitalTwin struct {
	AssetID            string         `json:"assetId"`
	DeviceID           *string        `json:"deviceId,omitempty"`
	Metadata           map[string]any `json:"metadata"`
	LiveState          map[string]any `json:"liveState"`
	MaintenanceHistory []TwinEvent    `json:"maintenanceHistory"`
	AISummary          string         `json:"aiSummary"`
	HealthScore        *float64       `json:"healthScore,omitempty"`
	HealthLevel        *string        `json:"healthLevel,omitempty"`
	SyncedAt           *time.Time     `json:"syncedAt,omitempty"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

type TwinEvent struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Summary   string `json:"summary"`
	Details   string `json:"details,omitempty"`
}

type AddEventRequest struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
	Details string `json:"details,omitempty"`
}
