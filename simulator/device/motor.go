package device

type Motor struct {
	DeviceID string
	AssetID  string
	Firmware string
	Location string
}

func NewMotor(deviceID string) *Motor {
	return &Motor{
		DeviceID: deviceID,
		AssetID:  "asset-production-line-003",
		Firmware: "fw-v1.0.0",
		Location: "production-line-gamma",
	}
}
