package health

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/inframind/backend/internal/telemetry"
)

type Service struct {
	aiURL         string
	telemetryRepo *telemetry.Repository
	assetTypeResolver AssetTypeResolver
	eventPublisher    EventPublisher
}

type EventPublisher interface {
	PublishRecommendation(deviceID string, recs []RecommendationResult)
}

func (s *Service) SetEventPublisher(p EventPublisher) {
	s.eventPublisher = p
}

type AssetTypeResolver interface {
	ResolveAssetType(ctx context.Context, deviceID string) (string, error)
}

func NewService(aiURL string, telemetryRepo *telemetry.Repository) *Service {
	return &Service{aiURL: aiURL, telemetryRepo: telemetryRepo}
}

func (s *Service) SetAssetTypeResolver(r AssetTypeResolver) {
	s.assetTypeResolver = r
}

func (s *Service) resolveAssetType(ctx context.Context, deviceID string) string {
	if s.assetTypeResolver == nil {
		return "transformer"
	}
	assetType, err := s.assetTypeResolver.ResolveAssetType(ctx, deviceID)
	if err != nil || assetType == "" {
		return "transformer"
	}
	return assetType
}

type aiRequest struct {
	Temperature float64          `json:"temperature"`
	Current     float64          `json:"current"`
	Voltage     float64          `json:"voltage"`
	Humidity    float64          `json:"humidity"`
	AssetType   string           `json:"asset_type,omitempty"`
	History     []telemetryPoint `json:"history,omitempty"`
}

type telemetryPoint struct {
	Temperature float64 `json:"temperature"`
	Current     float64 `json:"current"`
	Voltage     float64 `json:"voltage"`
	Humidity    float64 `json:"humidity"`
}

type aiHealthResponse struct {
	Score   float64        `json:"score"`
	Level   string         `json:"level"`
	Factors []aiFactor    `json:"factors"`
}

type aiFactor struct {
	Name    string  `json:"name"`
	Impact  float64 `json:"impact"`
	Details string  `json:"details"`
}

type aiAnalysisResponse struct {
	HealthScore   float64            `json:"health_score"`
	HealthLevel   string             `json:"health_level"`
	HealthFactors []aiFactor         `json:"health_factors"`
	Anomalies     []aiAnomaly        `json:"anomalies"`
	FailurePrediction *aiPrediction  `json:"failure_prediction"`
	Recommendations  []aiRecommendation `json:"recommendations"`
}

type aiAnomaly struct {
	Metric      string  `json:"metric"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Value       float64 `json:"value"`
	Expected    float64 `json:"expected"`
}

type aiPrediction struct {
	TimeToWarningHours  *float64 `json:"time_to_warning_hours"`
	TimeToCriticalHours *float64 `json:"time_to_critical_hours"`
	Confidence          float64  `json:"confidence"`
	TrendDirection      string   `json:"trend_direction"`
}

type aiRecommendation struct {
	Priority      string         `json:"priority"`
	Action        string         `json:"action"`
	Reason        string         `json:"reason"`
	EstimatedCost string         `json:"estimated_cost"`
	ActionType    string         `json:"action_type"`
	ActionPayload map[string]any `json:"action_payload"`
}

func (s *Service) fetchHistory(ctx context.Context, deviceID string) []telemetryPoint {
	if s.telemetryRepo == nil {
		return nil
	}
	points, err := s.telemetryRepo.QueryLatest(ctx, deviceID, 10)
	if err != nil || len(points) == 0 {
		return nil
	}
	history := make([]telemetryPoint, len(points))
	for i, p := range points {
		history[i] = telemetryPoint{
			Temperature: p.Temperature,
			Current:     p.Current,
			Voltage:     p.Voltage,
			Humidity:    p.Humidity,
		}
	}
	return history
}

func (s *Service) Calculate(ctx context.Context, deviceID string, temp, current, voltage, humidity float64) (*HealthResponse, error) {
	req := aiRequest{
		Temperature: temp,
		Current:     current,
		Voltage:     voltage,
		Humidity:    humidity,
		AssetType:   s.resolveAssetType(ctx, deviceID),
		History:     s.fetchHistory(ctx, deviceID),
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(s.aiURL+"/health-score", "application/json", bytes.NewReader(body))
	if err != nil {
		return s.deterministicFallback(temp, current, humidity), nil
	}
	defer resp.Body.Close()

	var aiResp aiHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return s.deterministicFallback(temp, current, humidity), nil
	}

	factors := make([]HealthFactor, len(aiResp.Factors))
	for i, f := range aiResp.Factors {
		factors[i] = HealthFactor{Name: f.Name, Impact: f.Impact, Details: f.Details}
	}

	return &HealthResponse{
		Score:   aiResp.Score,
		Level:   aiResp.Level,
		Factors: factors,
	}, nil
}

func (s *Service) Analyze(ctx context.Context, deviceID string, temp, current, voltage, humidity float64) (*AnalysisResponse, error) {
	req := aiRequest{
		Temperature: temp,
		Current:     current,
		Voltage:     voltage,
		Humidity:    humidity,
		AssetType:   s.resolveAssetType(ctx, deviceID),
		History:     s.fetchHistory(ctx, deviceID),
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(s.aiURL+"/analyze", "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Warn("ai analyze request failed", "error", err)
		return s.analysisFallback(temp, current, humidity), nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var aiResp aiAnalysisResponse
	if err := json.Unmarshal(respBody, &aiResp); err != nil {
		return s.analysisFallback(temp, current, humidity), nil
	}

	analysis := s.toAnalysisResponse(&aiResp)
	if s.eventPublisher != nil {
		var actionable []RecommendationResult
		for _, rec := range analysis.Recommendations {
			if rec.ActionType != "" {
				actionable = append(actionable, rec)
			}
		}
		if len(actionable) > 0 {
			s.eventPublisher.PublishRecommendation(deviceID, actionable)
		}
	}

	return analysis, nil
}

func (s *Service) toAnalysisResponse(ai *aiAnalysisResponse) *AnalysisResponse {
	resp := &AnalysisResponse{
		HealthScore: ai.HealthScore,
		HealthLevel: ai.HealthLevel,
	}

	for _, f := range ai.HealthFactors {
		resp.HealthFactors = append(resp.HealthFactors, HealthFactor{
			Name: f.Name, Impact: f.Impact, Details: f.Details,
		})
	}
	for _, a := range ai.Anomalies {
		resp.Anomalies = append(resp.Anomalies, AnomalyResult{
			Metric: a.Metric, Severity: a.Severity,
			Description: a.Description, Value: a.Value, Expected: a.Expected,
		})
	}
	if ai.FailurePrediction != nil {
		resp.FailurePrediction = &FailurePredictionResult{
			TimeToWarningHours:  ai.FailurePrediction.TimeToWarningHours,
			TimeToCriticalHours: ai.FailurePrediction.TimeToCriticalHours,
			Confidence:          ai.FailurePrediction.Confidence,
			TrendDirection:      ai.FailurePrediction.TrendDirection,
		}
	}
	for _, r := range ai.Recommendations {
		resp.Recommendations = append(resp.Recommendations, RecommendationResult{
			Priority: r.Priority, Action: r.Action,
			Reason: r.Reason, EstimatedCost: r.EstimatedCost,
			ActionType: r.ActionType, ActionPayload: r.ActionPayload,
		})
	}
	if resp.Anomalies == nil {
		resp.Anomalies = []AnomalyResult{}
	}
	if resp.Recommendations == nil {
		resp.Recommendations = []RecommendationResult{}
	}
	return resp
}

func (s *Service) deterministicFallback(temp, current, humidity float64) *HealthResponse {
	score := 100.0
	if temp > 75 {
		penalty := (temp - 75) * 1.5
		if penalty > 40 {
			penalty = 40
		}
		score -= penalty
	}
	if current > 120 {
		penalty := (current - 120) * 0.3
		if penalty > 30 {
			penalty = 30
		}
		score -= penalty
	}
	humidityPenalty := humidity * 0.2
	if humidityPenalty > 10 {
		humidityPenalty = 10
	}
	score -= humidityPenalty
	if score < 0 {
		score = 0
	}
	var level string
	switch {
	case score > 80:
		level = "healthy"
	case score > 50:
		level = "warning"
	default:
		level = "critical"
	}
	return &HealthResponse{
		Score: score,
		Level: level,
		Factors: []HealthFactor{
			{Name: "temperature", Impact: -(max(0, temp-75) * 1.5)},
			{Name: "current", Impact: -(max(0, current-120) * 0.3)},
			{Name: "humidity", Impact: -(humidity * 0.2)},
		},
	}
}

func (s *Service) analysisFallback(temp, current, humidity float64) *AnalysisResponse {
	hr := s.deterministicFallback(temp, current, humidity)
	return &AnalysisResponse{
		HealthScore:     hr.Score,
		HealthLevel:     hr.Level,
		HealthFactors:   hr.Factors,
		Anomalies:       []AnomalyResult{},
		Recommendations: []RecommendationResult{},
	}
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
