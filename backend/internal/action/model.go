package action

import "time"

type Action struct {
	ID               string         `json:"id"`
	AssetID          string         `json:"assetId"`
	DeviceID         *string        `json:"deviceId,omitempty"`
	Type             string         `json:"type"`
	Payload          map[string]any `json:"payload"`
	Source           string         `json:"source"`
	Status           string         `json:"status"`
	ApprovalRequired bool           `json:"approvalRequired"`
	AutoApproved     bool           `json:"autoApproved"`
	Reason           string         `json:"reason"`
	Result           *string        `json:"result,omitempty"`
	ProposedAt       time.Time      `json:"proposedAt"`
	ExecutedAt       *time.Time     `json:"executedAt,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type ActionTemplate struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	ActionType  string         `json:"actionType"`
	Payload     map[string]any `json:"payload"`
	AutoApprove bool           `json:"autoApprove"`
	Condition   string         `json:"condition"`
	Description string         `json:"description"`
	Active      bool           `json:"active"`
}

type ProposeActionRequest struct {
	AssetID          string         `json:"assetId"`
	DeviceID         *string        `json:"deviceId,omitempty"`
	Type             string         `json:"type"`
	Payload          map[string]any `json:"payload"`
	Source           string         `json:"source"`
	Reason           string         `json:"reason"`
	ApprovalRequired *bool          `json:"approvalRequired,omitempty"`
}

type Filter struct {
	AssetID  string
	Status   string
	Type     string
	Page     int
	Limit    int
}
