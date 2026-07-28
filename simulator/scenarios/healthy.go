package scenarios

import (
	"github.com/inframind/simulator/telemetry"
)

func Healthy(deviceID string) telemetry.ScenarioFunc {
	return func(tick int) telemetry.Reading {
		temp := 65.0 + float64(tick)*0.1
		if temp > 75 {
			temp = 75
		}
		current := 100.0 + float64(tick)*0.3
		if current > 120 {
			current = 120
		}
		voltage := 11400.0
		humidity := 45.0

		return telemetry.MakeReading(deviceID, "healthy", temp, current, voltage, humidity)
	}
}
