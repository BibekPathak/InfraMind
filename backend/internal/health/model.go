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
	Name    string  `json:"name"`
	Impact  float64 `json:"impact"`
	Details string  `json:"details,omitempty"`
}

type HealthResponse struct {
	Score     float64        `json:"score"`
	Level     string         `json:"level"`
	Factors   []HealthFactor `json:"factors"`
}

type AnomalyResult struct {
	Metric      string  `json:"metric"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Value       float64 `json:"value"`
	Expected    float64 `json:"expected"`
}

type FailurePredictionResult struct {
	TimeToWarningHours  *float64 `json:"timeToWarningHours"`
	TimeToCriticalHours *float64 `json:"timeToCriticalHours"`
	Confidence          float64  `json:"confidence"`
	TrendDirection      string   `json:"trendDirection"`
}

type RecommendationResult struct {
	Priority      string         `json:"priority"`
	Action        string         `json:"action"`
	Reason        string         `json:"reason"`
	EstimatedCost string         `json:"estimatedCost"`
	ActionType    string         `json:"actionType,omitempty"`
	ActionPayload map[string]any `json:"actionPayload,omitempty"`
}

type AnalysisResponse struct {
	HealthScore     float64                   `json:"healthScore"`
	HealthLevel     string                    `json:"healthLevel"`
	HealthFactors   []HealthFactor            `json:"healthFactors"`
	Anomalies       []AnomalyResult           `json:"anomalies"`
	FailurePrediction *FailurePredictionResult `json:"failurePrediction"`
	Recommendations []RecommendationResult    `json:"recommendations"`
}
