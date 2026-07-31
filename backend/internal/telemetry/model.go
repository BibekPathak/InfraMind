package telemetry

import "time"

type Telemetry struct {
	Time        time.Time `json:"time"`
	DeviceID    string    `json:"deviceId"`
	Temperature float64   `json:"temperature"`
	Current     float64   `json:"current"`
	Voltage     float64   `json:"voltage"`
	Humidity    float64   `json:"humidity"`

	FlowRate    *float64 `json:"flowRate,omitempty"`
	Pressure    *float64 `json:"pressure,omitempty"`
	Vibration   *float64 `json:"vibration,omitempty"`
	RPM         *float64 `json:"rpm,omitempty"`
	OutputPower *float64 `json:"outputPower,omitempty"`
}

type TelemetryPayload struct {
	DeviceID    string  `json:"device_id"`
	Timestamp   string  `json:"timestamp"`
	Temperature float64 `json:"temperature"`
	Current     float64 `json:"current"`
	Voltage     float64 `json:"voltage"`
	Humidity    float64 `json:"humidity"`
	Scenario    string  `json:"scenario,omitempty"`

	FlowRate    *float64 `json:"flow_rate,omitempty"`
	Pressure    *float64 `json:"pressure,omitempty"`
	Vibration   *float64 `json:"vibration,omitempty"`
	RPM         *float64 `json:"rpm,omitempty"`
	OutputPower *float64 `json:"output_power,omitempty"`
}
