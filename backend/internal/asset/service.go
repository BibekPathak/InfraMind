package asset

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

func (s *Service) Create(ctx context.Context, req CreateAssetRequest) (*Asset, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("asset name is required")
	}
	if req.Type == "" {
		req.Type = "transformer"
	}

	a := &Asset{
		ID:       uuidv7.New(),
		Name:     req.Name,
		Type:     req.Type,
		Location: req.Location,
		Metadata: req.Metadata,
	}

	if err := s.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create asset: %w", err)
	}
	return a, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Asset, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Asset, error) {
	return s.repo.List(ctx)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
