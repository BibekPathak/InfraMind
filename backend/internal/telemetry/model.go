package telemetry

import "time"

type Telemetry struct {
	Time        time.Time `json:"time"`
	DeviceID    string    `json:"deviceId"`
	Temperature float64   `json:"temperature"`
	Current     float64   `json:"current"`
	Voltage     float64   `json:"voltage"`
	Humidity    float64   `json:"humidity"`
}

type TelemetryPayload struct {
	DeviceID    string  `json:"device_id"`
	Timestamp   string  `json:"timestamp"`
	Temperature float64 `json:"temperature"`
	Current     float64 `json:"current"`
	Voltage     float64 `json:"voltage"`
	Humidity    float64 `json:"humidity"`
	Scenario    string  `json:"scenario,omitempty"`
}
