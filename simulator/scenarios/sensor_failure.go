package scenarios

import (
	"github.com/inframind/simulator/telemetry"
)

func SensorFailure(deviceID string) telemetry.ScenarioFunc {
	return func(tick int) telemetry.Reading {
		temp := 25.0 + float64(tick)*0.2
		if temp > 35 {
			temp = 35
		}
		current := 0.0
		voltage := 0.0
		humidity := 0.0

		return telemetry.MakeReading(deviceID, "sensor_failure", temp, current, voltage, humidity)
	}
}
