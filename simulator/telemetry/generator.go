package telemetry

import (
	"log/slog"
	"math"
	"sync"
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

	FlowRate    *float64 `json:"flow_rate,omitempty"`
	Pressure    *float64 `json:"pressure,omitempty"`
	Vibration   *float64 `json:"vibration,omitempty"`
	RPM         *float64 `json:"rpm,omitempty"`
	OutputPower *float64 `json:"output_power,omitempty"`
}

type ScenarioFunc func(tick int) Reading

type Generator struct {
	DeviceID  string
	scenarios []ScenarioFunc
	names     []string

	mu         sync.RWMutex
	forcedName string // optional: force a specific scenario name
	localTick  int
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

// ForceScenario switches the generator to a named scenario on demand. Used by
// the failure-injection control channel. Falls back to cycling if unknown.
func (g *Generator) ForceScenario(name string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if name == "" {
		g.forcedName = ""
		g.localTick = 0
		return
	}
	for _, n := range g.names {
		if n == name {
			g.forcedName = name
			g.localTick = 0
			return
		}
	}
	slog.Warn("unknown scenario, ignoring force", "scenario", name, "deviceId", g.DeviceID)
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

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.forcedName != "" {
		for i, n := range g.names {
			if n == g.forcedName {
				r := g.scenarios[i](g.localTick)
				g.localTick++
				return r
			}
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

func fptr(v float64) *float64 {
	return &v
}
