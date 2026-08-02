package audit

import "time"

type LogEntry struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organizationId"`
	UserID         string         `json:"userId"`
	Action         string         `json:"action"`
	ResourceType   string         `json:"resourceType"`
	ResourceID     string         `json:"resourceId"`
	Details        map[string]any `json:"details"`
	CreatedAt      time.Time      `json:"createdAt"`
}

type Filter struct {
	OrganizationID string
	ResourceType   string
	Action         string
	Page           int
	Limit          int
}
