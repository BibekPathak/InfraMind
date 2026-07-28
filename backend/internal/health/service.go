package health

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Service struct {
	aiURL string
}

func NewService(aiURL string) *Service {
	return &Service{aiURL: aiURL}
}

type aiRequest struct {
	Temperature float64 `json:"temperature"`
	Current     float64 `json:"current"`
	Voltage     float64 `json:"voltage"`
	Humidity    float64 `json:"humidity"`
}

type aiResponse struct {
	Score   float64        `json:"score"`
	Level   string         `json:"level"`
	Factors []HealthFactor `json:"factors"`
}

func (s *Service) Calculate(ctx context.Context, deviceID string, temp, current, voltage, humidity float64) (*HealthResponse, error) {
	req := aiRequest{
		Temperature: temp,
		Current:     current,
		Voltage:     voltage,
		Humidity:    humidity,
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(s.aiURL+"/health-score", "application/json", bytes.NewReader(body))
	if err != nil {
		return s.deterministicFallback(temp, current, humidity), nil
	}
	defer resp.Body.Close()

	var aiResp aiResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return s.deterministicFallback(temp, current, humidity), nil
	}

	return &HealthResponse{
		Score:   aiResp.Score,
		Level:   aiResp.Level,
		Factors: aiResp.Factors,
	}, nil
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

	tempImpact := 0.0
	if temp > 75 {
		tempImpact = -(temp - 75) * 1.5
	}
	currentImpact := 0.0
	if current > 120 {
		currentImpact = -(current - 120) * 0.3
	}

	return &HealthResponse{
		Score: score,
		Level: level,
		Factors: []HealthFactor{
			{Name: fmt.Sprintf("%.0f", time.Now().Sub(time.Now())), Impact: tempImpact},
			{Name: "temperature", Impact: tempImpact},
			{Name: "current", Impact: currentImpact},
			{Name: "humidity", Impact: -(humidity * 0.2)},
		},
	}
}
