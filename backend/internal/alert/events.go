package alert

import "github.com/inframind/backend/internal/eventbus"

func RegisterEvents(bus *eventbus.Bus, svc *Service) {
	// Future: subscribe to telemetry.ingested for real-time evaluation
}
