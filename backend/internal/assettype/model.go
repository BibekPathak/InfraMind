package assettype

import "time"

type Metric struct {
	Name string `json:"name"`
	Unit string `json:"unit"`
}

type Thresholds map[string]map[string]float64

type AssetType struct {
	Type          string         `json:"type"`
	DisplayName   string         `json:"displayName"`
	Metrics       []Metric       `json:"metrics"`
	Thresholds    map[string]any `json:"thresholds"`
	HealthWeights map[string]any `json:"healthWeights"`
	Active        bool           `json:"active"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

type CreateAssetTypeRequest struct {
	Type          string         `json:"type"`
	DisplayName   string         `json:"displayName"`
	Metrics       []Metric       `json:"metrics"`
	Thresholds    map[string]any `json:"thresholds"`
	HealthWeights map[string]any `json:"healthWeights"`
}

type UpdateAssetTypeRequest struct {
	DisplayName   string         `json:"displayName"`
	Metrics       []Metric       `json:"metrics"`
	Thresholds    map[string]any `json:"thresholds"`
	HealthWeights map[string]any `json:"healthWeights"`
	Active        *bool          `json:"active,omitempty"`
}
