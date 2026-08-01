package action

import "github.com/inframind/backend/internal/eventbus"

func RegisterEvents(bus *eventbus.Bus, svc *Service) {
	// Future: subscribe to ai.recommendation.generated for auto-proposing actions
}
