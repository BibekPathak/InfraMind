package alert

import "log/slog"

type Notifier interface {
	Send(alert *Alert)
}

type LogNotifier struct{}

func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

func (n *LogNotifier) Send(alert *Alert) {
	slog.Warn("notification",
		"alertId", alert.ID,
		"deviceId", alert.DeviceID,
		"severity", alert.Severity,
		"rule", alert.Rule,
		"message", alert.Message,
	)
}
