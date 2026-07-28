package alert

import "time"

type Alert struct {
	ID        string    `json:"id"`
	DeviceID  string    `json:"deviceId"`
	Severity  string    `json:"severity"`
	Rule      string    `json:"rule"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
