package scenarios

import (
	"github.com/inframind/simulator/telemetry"
)

func CoolingFailure(deviceID string) telemetry.ScenarioFunc {
	return func(tick int) telemetry.Reading {
		temp := 85.0 + float64(tick)*1.2
		if temp > 115 {
			temp = 115
		}
		current := 100.0 + float64(tick)*0.2
		if current > 120 {
			current = 120
		}
		voltage := 11400.0 - float64(tick)*5.0
		if voltage < 11200 {
			voltage = 11200
		}
		humidity := 45.0

		return telemetry.MakeReading(deviceID, "cooling_failure", temp, current, voltage, humidity)
	}
}
