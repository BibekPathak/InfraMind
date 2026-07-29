package alert

import (
	"context"
	"fmt"

	"github.com/inframind/backend/pkg/uuidv7"
)

type AlertFilter struct {
	DeviceID string
	Status   string
	Severity string
	Page     int
	Limit    int
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, deviceID, severity, rule, message string) (*Alert, error) {
	a := &Alert{
		ID:       uuidv7.New(),
		DeviceID: deviceID,
		Severity: severity,
		Rule:     rule,
		Message:  message,
		Status:   "open",
	}

	if err := s.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create alert: %w", err)
	}
	return a, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Alert, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, filter AlertFilter) ([]Alert, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) Acknowledge(ctx context.Context, id string) (*Alert, error) {
	if err := s.repo.UpdateStatus(ctx, id, "acknowledged"); err != nil {
		return nil, fmt.Errorf("acknowledge alert: %w", err)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Resolve(ctx context.Context, id string) (*Alert, error) {
	if err := s.repo.UpdateStatus(ctx, id, "resolved"); err != nil {
		return nil, fmt.Errorf("resolve alert: %w", err)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) HasOpenAlertForRule(ctx context.Context, deviceID, rule string) (bool, error) {
	return s.repo.HasOpenAlertForRule(ctx, deviceID, rule)
}
