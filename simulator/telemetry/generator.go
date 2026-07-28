package telemetry

import (
	"math"
	"time"
)

type Reading struct {
	DeviceID    string  `json:"device_id"`
	Timestamp   string  `json:"timestamp"`
	Temperature float64 `json:"temperature"`
	Current     float64 `json:"current"`
	Voltage     float64 `json:"voltage"`
	Humidity    float64 `json:"humidity"`
	Scenario    string  `json:"scenario"`
}

type ScenarioFunc func(tick int) Reading

type Generator struct {
	DeviceID  string
	scenarios []ScenarioFunc
	names     []string
}

func NewGenerator(deviceID string) *Generator {
	return &Generator{
		DeviceID:  deviceID,
		scenarios: make([]ScenarioFunc, 0),
		names:     make([]string, 0),
	}
}

func (g *Generator) AddScenario(name string, fn ScenarioFunc) {
	g.scenarios = append(g.scenarios, fn)
	g.names = append(g.names, name)
}

func (g *Generator) Generate(tick int) Reading {
	if len(g.scenarios) == 0 {
		return Reading{
			DeviceID:    g.DeviceID,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			Temperature: 25,
			Current:     0,
			Voltage:     0,
			Humidity:    0,
			Scenario:    "idle",
		}
	}

	scenarioIndex := (tick / 30) % len(g.scenarios)
	localTick := tick % 30

	return g.scenarios[scenarioIndex](localTick)
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func round(v float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}

func MakeReading(deviceID, scenario string, temp, current, voltage, humidity float64) Reading {
	ts := time.Now().UTC()
	return Reading{
		DeviceID:    deviceID,
		Timestamp:   ts.Format(time.RFC3339),
		Temperature: round(clamp(temp, 0, 200), 1),
		Current:     round(clamp(current, 0, 500), 1),
		Voltage:     round(clamp(voltage, 0, 20000), 1),
		Humidity:    round(clamp(humidity, 0, 100), 1),
		Scenario:    scenario,
	}
}
