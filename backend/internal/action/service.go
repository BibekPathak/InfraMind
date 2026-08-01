package action

import (
	"context"
	"fmt"

	"github.com/inframind/backend/pkg/uuidv7"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Propose(ctx context.Context, req ProposeActionRequest) (*Action, error) {
	if req.AssetID == "" {
		return nil, fmt.Errorf("assetId is required")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("type is required")
	}
	if req.Source == "" {
		req.Source = "manual"
	}

	approvalRequired := true
	if req.ApprovalRequired != nil {
		approvalRequired = *req.ApprovalRequired
	}

	a := &Action{
		ID:               uuidv7.New(),
		AssetID:          req.AssetID,
		DeviceID:         req.DeviceID,
		Type:             req.Type,
		Payload:          req.Payload,
		Source:           req.Source,
		Status:           "proposed",
		ApprovalRequired: approvalRequired,
		Reason:           req.Reason,
	}

	if err := s.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("propose action: %w", err)
	}
	return a, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Action, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, f Filter) ([]Action, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Approve(ctx context.Context, id string) (*Action, error) {
	if err := s.repo.UpdateStatus(ctx, id, "approved"); err != nil {
		return nil, fmt.Errorf("approve action: %w", err)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Reject(ctx context.Context, id string) (*Action, error) {
	if err := s.repo.UpdateStatus(ctx, id, "rejected"); err != nil {
		return nil, fmt.Errorf("reject action: %w", err)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) MarkExecuted(ctx context.Context, id string, result string) error {
	return s.repo.MarkExecuted(ctx, id, result)
}

func (s *Service) MarkFailed(ctx context.Context, id string, result string) error {
	return s.repo.MarkFailed(ctx, id, result)
}
