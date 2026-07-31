package workorder

import "github.com/inframind/backend/internal/eventbus"

func RegisterEvents(bus *eventbus.Bus, svc *Service) {
	// Future: subscribe to alert.raised for auto work order creation
}
