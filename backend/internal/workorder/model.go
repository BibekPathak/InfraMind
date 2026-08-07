package workorder

import "time"

type WorkOrder struct {
	ID            string          `json:"id"`
	AssetID       string          `json:"assetId"`
	AlertID       *string         `json:"alertId,omitempty"`
	Type          string          `json:"type"`
	Priority      string          `json:"priority"`
	Status        string          `json:"status"`
	AssignedTo    *string         `json:"assignedTo,omitempty"`
	EstimatedCost *float64        `json:"estimatedCost,omitempty"`
	Description   string          `json:"description"`
	Timeline      []TimelineEvent `json:"timeline"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

type TimelineEvent struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Actor     string `json:"actor"`
	Note      string `json:"note,omitempty"`
}

type CreateWorkOrderRequest struct {
	AssetID       string   `json:"assetId"`
	AlertID       *string  `json:"alertId,omitempty"`
	Type          string   `json:"type"`
	Priority      string   `json:"priority"`
	Description   string   `json:"description"`
	EstimatedCost *float64 `json:"estimatedCost,omitempty"`
}

type AssignRequest struct {
	AssignedTo string `json:"assignedTo"`
}

type StatusUpdateRequest struct {
	Status string `json:"status"`
}

type Filter struct {
	AssetID  string
	Status   string
	Priority string
	Page     int
	Limit    int
}
