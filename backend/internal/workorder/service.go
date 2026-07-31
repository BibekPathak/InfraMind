package workorder

import (
	"context"
	"fmt"
	"time"

	"github.com/inframind/backend/pkg/uuidv7"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateWorkOrderRequest) (*WorkOrder, error) {
	if req.AssetID == "" {
		return nil, fmt.Errorf("assetId is required")
	}
	if req.Type == "" {
		req.Type = "inspection"
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}

	wo := &WorkOrder{
		ID:            uuidv7.New(),
		AssetID:       req.AssetID,
		AlertID:       req.AlertID,
		Type:          req.Type,
		Priority:      req.Priority,
		Status:        "open",
		Description:   req.Description,
		EstimatedCost: req.EstimatedCost,
		Timeline: []TimelineEvent{{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Action:    "created",
			Actor:     "system",
			Note:      "Work order created",
		}},
	}

	if err := s.repo.Create(ctx, wo); err != nil {
		return nil, fmt.Errorf("create work order: %w", err)
	}
	return wo, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*WorkOrder, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, f Filter) ([]WorkOrder, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Assign(ctx context.Context, id, assignedTo string) (*WorkOrder, error) {
	if assignedTo == "" {
		return nil, fmt.Errorf("assignedTo is required")
	}
	if err := s.repo.Assign(ctx, id, assignedTo); err != nil {
		return nil, fmt.Errorf("assign: %w", err)
	}
	s.appendEvent(ctx, id, "assigned", "system", "Assigned to "+assignedTo)
	return s.repo.GetByID(ctx, id)
}

func (s *Service) UpdateStatus(ctx context.Context, id, status string) (*WorkOrder, error) {
	valid := map[string]bool{
		"open": true, "assigned": true, "in_progress": true,
		"completed": true, "cancelled": true,
	}
	if !valid[status] {
		return nil, fmt.Errorf("invalid status: %s", status)
	}
	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}
	s.appendEvent(ctx, id, "status_changed", "system", "Status changed to "+status)
	return s.repo.GetByID(ctx, id)
}

func (s *Service) appendEvent(ctx context.Context, id, action, actor, note string) {
	s.repo.AppendTimeline(ctx, id, TimelineEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    action,
		Actor:     actor,
		Note:      note,
	})
}
