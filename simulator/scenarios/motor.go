package scenarios

import (
	"github.com/inframind/simulator/telemetry"
)

func MotorHealthy(deviceID string) telemetry.ScenarioFunc {
	return func(tick int) telemetry.Reading {
		r := telemetry.MakeReading(deviceID, "healthy", 70.0, 90.0, 480.0, 45.0)
		rpm := 1450.0 + float64(tick)*0.1
		if rpm > 1460 {
			rpm = 1460
		}
		vibration := 1.5 + float64(tick)*0.01
		if vibration > 2 {
			vibration = 2
		}
		r.RPM = fptr(rpm)
		r.Vibration = fptr(vibration)
		return r
	}
}

func MotorBearingWear(deviceID string) telemetry.ScenarioFunc {
	return func(tick int) telemetry.Reading {
		r := telemetry.MakeReading(deviceID, "bearing_wear", 80.0+float64(tick)*0.4, 95.0, 480.0, 45.0)
		rpm := 1440.0 - float64(tick)*0.5
		if rpm < 1400 {
			rpm = 1400
		}
		vibration := 2.5 + float64(tick)*0.1
		if vibration > 5 {
			vibration = 5
		}
		r.RPM = fptr(rpm)
		r.Vibration = fptr(vibration)
		return r
	}
}

func MotorOverload(deviceID string) telemetry.ScenarioFunc {
	return func(tick int) telemetry.Reading {
		r := telemetry.MakeReading(deviceID, "overloaded", 85.0+float64(tick)*0.5, 120.0, 480.0, 45.0)
		rpm := 1450.0 - float64(tick)*1.0
		if rpm < 1350 {
			rpm = 1350
		}
		vibration := 2.0 + float64(tick)*0.05
		if vibration > 3.5 {
			vibration = 3.5
		}
		r.RPM = fptr(rpm)
		r.Vibration = fptr(vibration)
		return r
	}
}
