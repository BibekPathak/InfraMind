package telemetry

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

type ValidationResult struct {
	Valid   bool
	Message string
	Payload *TelemetryPayload
}

type DeviceRateLimiter struct {
	mu       sync.Mutex
	counters map[string]*rateCounter
	limit    int
	window   time.Duration
}

type rateCounter struct {
	count       int
	windowStart time.Time
}

func NewDeviceRateLimiter(limit int, window time.Duration) *DeviceRateLimiter {
	return &DeviceRateLimiter{
		counters: make(map[string]*rateCounter),
		limit:    limit,
		window:   window,
	}
}

func (rl *DeviceRateLimiter) Allow(deviceID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rc, exists := rl.counters[deviceID]

	if !exists || now.Sub(rc.windowStart) > rl.window {
		rl.counters[deviceID] = &rateCounter{
			count:       1,
			windowStart: now,
		}
		return true
	}

	if rc.count >= rl.limit {
		return false
	}

	rc.count++
	return true
}

type Validator struct {
	rateLimiter *DeviceRateLimiter
}

func NewValidator() *Validator {
	return &Validator{
		rateLimiter: NewDeviceRateLimiter(10, 1*time.Second),
	}
}

func (v *Validator) Validate(topic string, payload []byte) ValidationResult {
	payloadStr := strings.TrimSpace(string(payload))
	if len(payloadStr) == 0 {
		return ValidationResult{Valid: false, Message: "empty payload"}
	}

	parts := strings.Split(topic, "/")
	var topicDeviceID string
	if len(parts) >= 2 {
		topicDeviceID = parts[1]
	}

	var p TelemetryPayload
	if err := json.Unmarshal([]byte(payloadStr), &p); err != nil {
		return ValidationResult{Valid: false, Message: fmt.Sprintf("invalid json: %s", err.Error())}
	}

	if p.DeviceID == "" {
		if topicDeviceID == "" {
			return ValidationResult{Valid: false, Message: "missing device_id in payload and topic"}
		}
		p.DeviceID = topicDeviceID
	}

	if p.Temperature < -50 || p.Temperature > 200 {
		return ValidationResult{Valid: false, Message: fmt.Sprintf("temperature out of range: %.1f", p.Temperature)}
	}
	if p.Current < 0 || p.Current > 500 {
		return ValidationResult{Valid: false, Message: fmt.Sprintf("current out of range: %.1f", p.Current)}
	}
	if p.Voltage < 0 || p.Voltage > 50000 {
		return ValidationResult{Valid: false, Message: fmt.Sprintf("voltage out of range: %.1f", p.Voltage)}
	}
	if p.Humidity < 0 || p.Humidity > 100 {
		return ValidationResult{Valid: false, Message: fmt.Sprintf("humidity out of range: %.1f", p.Humidity)}
	}

	if !v.rateLimiter.Allow(p.DeviceID) {
		slog.Warn("telemetry rate limit exceeded", "deviceId", p.DeviceID)
		return ValidationResult{Valid: false, Message: "rate limit exceeded"}
	}

	return ValidationResult{Valid: true, Payload: &p}
}
