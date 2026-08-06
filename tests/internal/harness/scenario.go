package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Scenario is a golden end-to-end test definition.
type Scenario struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	DeviceID    string         `yaml:"deviceId"`
	Telemetry   TelemetrySpec  `yaml:"telemetry"`
	Expect      ExpectSpec     `yaml:"expect"`
}

type TelemetrySpec struct {
	IntervalMs int            `yaml:"intervalMs"`
	Steps      []TelemetryStep `yaml:"steps"`
}

type TelemetryStep struct {
	Temperature float64 `yaml:"temperature"`
	Current     float64 `yaml:"current"`
	Voltage     float64 `yaml:"voltage"`
	Humidity    float64 `yaml:"humidity"`
	Repeats     int     `yaml:"repeats"`
}

type ExpectSpec struct {
	HealthScore *HealthExpect `yaml:"healthScore"`
	Alerts      *CountExpect  `yaml:"alerts"`
	WorkOrders  *CountExpect  `yaml:"workOrders"`
	Actions     *CountExpect  `yaml:"actions"`
}

type HealthExpect struct {
	Min   float64 `yaml:"min"`
	Level string  `yaml:"level"`
}

type CountExpect struct {
	Min *int `yaml:"min"`
	Max *int `yaml:"max"`
}

// LoadScenario reads a YAML scenario file.
func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse scenario %s: %w", path, err)
	}
	return &s, nil
}

type telemetryPayload struct {
	DeviceID    string  `json:"device_id"`
	Timestamp   string  `json:"timestamp"`
	Temperature float64 `json:"temperature"`
	Current     float64 `json:"current"`
	Voltage     float64 `json:"voltage"`
	Humidity    float64 `json:"humidity"`
	Scenario    string  `json:"scenario"`
}

// Play publishes the scenario's telemetry script to MQTT for the device.
func (s *Scenario) Play(h *Harness, deviceID string) error {
	interval := s.Telemetry.IntervalMs
	if interval <= 0 {
		interval = 200
	}

	topic := fmt.Sprintf("telemetry/%s/data", deviceID)
	for _, step := range s.Telemetry.Steps {
		repeats := step.Repeats
		if repeats <= 0 {
			repeats = 1
		}
		for i := 0; i < repeats; i++ {
			payload := telemetryPayload{
				DeviceID:    deviceID,
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
				Temperature: step.Temperature,
				Current:     step.Current,
				Voltage:     step.Voltage,
				Humidity:    step.Humidity,
				Scenario:    s.Name,
			}
			data, _ := json.Marshal(payload)
			if err := h.MQTTPub(topic, data); err != nil {
				return fmt.Errorf("scenario %s publish: %w", s.Name, err)
			}
			time.Sleep(time.Duration(interval) * time.Millisecond)
		}
	}
	return nil
}

// VerifyHealth polls the health endpoint until expectations hold.
func (s *Scenario) VerifyHealth(h *Harness, api *APIClient, deviceID string, timeout time.Duration) error {
	if s.Expect.HealthScore == nil {
		return nil
	}

	var last *HealthResult
	ok := WaitFor(timeout, 500*time.Millisecond, func() bool {
		var hr HealthResult
		code, err := api.Do("GET", fmt.Sprintf("/api/v1/health/%s?temperature=70&current=100&voltage=11400&humidity=45", deviceID), nil, &hr)
		if err != nil || code != 200 {
			return false
		}
		last = &hr
		return hr.Score >= s.Expect.HealthScore.Min && (s.Expect.HealthScore.Level == "" || hr.Level == s.Expect.HealthScore.Level)
	})
	if !ok {
		if last != nil {
			return fmt.Errorf("health not met: want min=%.0f level=%q got score=%.1f level=%q",
				s.Expect.HealthScore.Min, s.Expect.HealthScore.Level, last.Score, last.Level)
		}
		return fmt.Errorf("health endpoint never returned valid data")
	}
	return nil
}

// VerifyCounts asserts alert/workorder/action counts against expectations.
func (s *Scenario) VerifyCounts(h *Harness, api *APIClient, timeout time.Duration) error {
	if s.Expect.Alerts != nil {
		if err := waitForCount(api, "/api/v1/alerts", s.Expect.Alerts, timeout, "alerts"); err != nil {
			return err
		}
	}
	if s.Expect.WorkOrders != nil {
		if err := waitForCount(api, "/api/v1/work-orders", s.Expect.WorkOrders, timeout, "work orders"); err != nil {
			return err
		}
	}
	if s.Expect.Actions != nil {
		if err := waitForCount(api, "/api/v1/actions", s.Expect.Actions, timeout, "actions"); err != nil {
			return err
		}
	}
	return nil
}

type HealthResult struct {
	Score float64 `json:"score"`
	Level string  `json:"level"`
}

type countResult struct {
	Count int `json:"count"`
}

func waitForCount(api *APIClient, path string, expect *CountExpect, timeout time.Duration, label string) error {
	ok := WaitFor(timeout, 500*time.Millisecond, func() bool {
		var items []json.RawMessage
		code, err := api.Do("GET", path, nil, &items)
		if err != nil || code != 200 {
			return false
		}
		n := len(items)
		if expect.Min != nil && n < *expect.Min {
			return false
		}
		if expect.Max != nil && n > *expect.Max {
			return false
		}
		return true
	})
	if !ok {
		var items []json.RawMessage
		code, _ := api.Do("GET", path, nil, &items)
		_ = code
		return fmt.Errorf("%s count not met: got %d (min=%v max=%v)", label, len(items), derefInt(expect.Min), derefInt(expect.Max))
	}
	return nil
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
