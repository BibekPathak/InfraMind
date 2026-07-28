package health

import "time"

type HealthScore struct {
	DeviceID  string         `json:"deviceId"`
	Score     float64        `json:"score"`
	Level     string         `json:"level"`
	Factors   []HealthFactor `json:"factors"`
	CreatedAt time.Time      `json:"createdAt"`
}

type HealthFactor struct {
	Name   string  `json:"name"`
	Impact float64 `json:"impact"`
}

type HealthResponse struct {
	Score     float64        `json:"score"`
	Level     string         `json:"level"`
	Factors   []HealthFactor `json:"factors"`
}
