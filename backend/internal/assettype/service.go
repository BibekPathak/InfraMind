package assettype

import (
	"context"
	"fmt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateAssetTypeRequest) (*AssetType, error) {
	if req.Type == "" {
		return nil, fmt.Errorf("type is required")
	}
	if req.DisplayName == "" {
		return nil, fmt.Errorf("displayName is required")
	}

	at := &AssetType{
		Type:          req.Type,
		DisplayName:   req.DisplayName,
		Metrics:       req.Metrics,
		Thresholds:    req.Thresholds,
		HealthWeights: req.HealthWeights,
		Active:        true,
	}

	if err := s.repo.Create(ctx, at); err != nil {
		return nil, fmt.Errorf("create asset type: %w", err)
	}
	return at, nil
}

func (s *Service) GetByType(ctx context.Context, typeName string) (*AssetType, error) {
	return s.repo.GetByType(ctx, typeName)
}

func (s *Service) List(ctx context.Context) ([]AssetType, error) {
	return s.repo.List(ctx)
}

func (s *Service) Update(ctx context.Context, typeName string, req UpdateAssetTypeRequest) (*AssetType, error) {
	existing, err := s.repo.GetByType(ctx, typeName)
	if err != nil {
		return nil, err
	}

	existing.DisplayName = req.DisplayName
	existing.Metrics = req.Metrics
	existing.Thresholds = req.Thresholds
	existing.HealthWeights = req.HealthWeights
	if req.Active != nil {
		existing.Active = *req.Active
	}

	if err := s.repo.Update(ctx, typeName, existing); err != nil {
		return nil, fmt.Errorf("update asset type: %w", err)
	}
	return existing, nil
}

func (s *Service) Exists(ctx context.Context, typeName string) (bool, error) {
	_, err := s.repo.GetByType(ctx, typeName)
	if err != nil {
		return false, nil
	}
	return true, nil
}
