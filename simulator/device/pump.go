package device

type Pump struct {
	DeviceID string
	AssetID  string
	Firmware string
	Location string
}

func NewPump(deviceID string) *Pump {
	return &Pump{
		DeviceID: deviceID,
		AssetID:  "asset-water-plant-002",
		Firmware: "fw-v1.0.0",
		Location: "water-plant-beta",
	}
}
