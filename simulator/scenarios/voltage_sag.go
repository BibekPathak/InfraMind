package scenarios

import (
	"github.com/inframind/simulator/telemetry"
)

func VoltageSag(deviceID string) telemetry.ScenarioFunc {
	return func(tick int) telemetry.Reading {
		temp := 60.0 + float64(tick)*0.3
		if temp > 70 {
			temp = 70
		}
		current := 100.0 + float64(tick)*2.0
		if current > 150 {
			current = 150
		}
		voltage := 11000.0 - float64(tick)*50.0
		if voltage < 9500 {
			voltage = 9500
		}
		humidity := 45.0

		return telemetry.MakeReading(deviceID, "voltage_sag", temp, current, voltage, humidity)
	}
}
