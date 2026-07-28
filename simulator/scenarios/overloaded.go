package scenarios

import (
	"github.com/inframind/simulator/telemetry"
)

func Overloaded(deviceID string) telemetry.ScenarioFunc {
	return func(tick int) telemetry.Reading {
		temp := 75.0 + float64(tick)*0.6
		if temp > 95 {
			temp = 95
		}
		current := 145.0 + float64(tick)*1.2
		if current > 200 {
			current = 200
		}
		voltage := 11200.0 - float64(tick)*3.0
		if voltage < 11000 {
			voltage = 11000
		}
		humidity := 45.0

		return telemetry.MakeReading(deviceID, "overloaded", temp, current, voltage, humidity)
	}
}
