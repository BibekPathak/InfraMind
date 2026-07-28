package device

type Transformer struct {
	DeviceID string
	AssetID  string
	Firmware string
	Location string
}

func NewTransformer(deviceID string) *Transformer {
	return &Transformer{
		DeviceID: deviceID,
		AssetID:  "asset-substation-001",
		Firmware: "fw-v1.0.0",
		Location: "grid-substation-alpha",
	}
}
