package asset

import "time"

type Asset struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	AutonomyMode string         `json:"autonomyMode"`
	Location     map[string]any `json:"location,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    *time.Time     `json:"deletedAt,omitempty"`
}

type CreateAssetRequest struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Location map[string]any `json:"location,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type UpdateAutonomyRequest struct {
	AutonomyMode string `json:"autonomyMode"`
}
