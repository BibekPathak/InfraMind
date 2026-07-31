package scenarios

import (
	"github.com/inframind/simulator/telemetry"
)

func PumpHealthy(deviceID string) telemetry.ScenarioFunc {
	return func(tick int) telemetry.Reading {
		r := telemetry.MakeReading(deviceID, "healthy", 60.0, 85.0, 480.0, 50.0)
		flow := 65.0 + float64(tick)*0.1
		if flow > 75 {
			flow = 75
		}
		pressure := 6.0 + float64(tick)*0.01
		if pressure > 7 {
			pressure = 7
		}
		vibration := 2.0 + float64(tick)*0.01
		if vibration > 3 {
			vibration = 3
		}
		r.FlowRate = fptr(flow)
		r.Pressure = fptr(pressure)
		r.Vibration = fptr(vibration)
		return r
	}
}

func PumpOverloaded(deviceID string) telemetry.ScenarioFunc {
	return func(tick int) telemetry.Reading {
		r := telemetry.MakeReading(deviceID, "overloaded", 70.0+float64(tick)*0.5, 120.0, 480.0, 50.0)
		flow := 55.0 - float64(tick)*0.5
		if flow < 40 {
			flow = 40
		}
		pressure := 7.0 + float64(tick)*0.1
		if pressure > 9 {
			pressure = 9
		}
		vibration := 3.0 + float64(tick)*0.08
		if vibration > 5 {
			vibration = 5
		}
		r.FlowRate = fptr(flow)
		r.Pressure = fptr(pressure)
		r.Vibration = fptr(vibration)
		return r
	}
}

func PumpCavitation(deviceID string) telemetry.ScenarioFunc {
	return func(tick int) telemetry.Reading {
		r := telemetry.MakeReading(deviceID, "cavitation", 65.0+float64(tick)*0.3, 100.0, 480.0, 50.0)
		flow := 60.0 - float64(tick)*1.0
		if flow < 30 {
			flow = 30
		}
		pressure := 4.0 - float64(tick)*0.1
		if pressure < 2 {
			pressure = 2
		}
		vibration := 3.5 + float64(tick)*0.15
		if vibration > 7 {
			vibration = 7
		}
		r.FlowRate = fptr(flow)
		r.Pressure = fptr(pressure)
		r.Vibration = fptr(vibration)
		return r
	}
}
