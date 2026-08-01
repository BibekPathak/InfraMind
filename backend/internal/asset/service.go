package asset

import (
	"context"
	"fmt"

	"github.com/inframind/backend/pkg/uuidv7"
)

type TypeValidator interface {
	Exists(ctx context.Context, typeName string) (bool, error)
}

type Service struct {
	repo    *Repository
	typeSvc TypeValidator
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetTypeValidator(tv TypeValidator) {
	s.typeSvc = tv
}

func (s *Service) Create(ctx context.Context, req CreateAssetRequest) (*Asset, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("asset name is required")
	}
	if req.Type == "" {
		req.Type = "transformer"
	}

	if s.typeSvc != nil {
		exists, err := s.typeSvc.Exists(ctx, req.Type)
		if err != nil {
			return nil, fmt.Errorf("validate asset type: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("unknown asset type: %s", req.Type)
		}
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

func (s *Service) UpdateAutonomyMode(ctx context.Context, id, mode string) (*Asset, error) {
	valid := map[string]bool{"manual": true, "advisory": true, "autonomous": true}
	if !valid[mode] {
		return nil, fmt.Errorf("invalid autonomy mode: %s (must be manual, advisory, or autonomous)", mode)
	}
	if err := s.repo.UpdateAutonomyMode(ctx, id, mode); err != nil {
		return nil, fmt.Errorf("update autonomy mode: %w", err)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetAutonomyMode(ctx context.Context, id string) (string, error) {
	return s.repo.GetAutonomyMode(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
