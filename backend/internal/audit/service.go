package audit

import (
	"context"
	"log/slog"

	"github.com/inframind/backend/internal/tenant"
	"github.com/inframind/backend/pkg/uuidv7"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Record(ctx context.Context, action, resourceType, resourceID, userID string, details map[string]any) {
	if userID == "" {
		userID = "system"
	}

	e := &LogEntry{
		ID:             uuidv7.New(),
		OrganizationID: tenant.EffectiveOrgID(ctx),
		UserID:         userID,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		Details:        details,
	}

	if err := s.repo.Insert(ctx, e); err != nil {
		slog.Warn("failed to record audit log", "error", err, "action", action, "resourceType", resourceType)
		return
	}
	slog.Debug("audit log recorded", "action", action, "resourceType", resourceType, "resourceId", resourceID)
}

func (s *Service) List(ctx context.Context, f Filter) ([]LogEntry, error) {
	return s.repo.List(ctx, f)
}
