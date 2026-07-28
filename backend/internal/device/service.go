package device

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

func (s *Service) Register(ctx context.Context, req RegisterDeviceRequest) (*Device, error) {
	if req.AssetID == "" {
		return nil, fmt.Errorf("assetId is required")
	}

	d := &Device{
		ID:              uuidv7.New(),
		AssetID:         req.AssetID,
		FirmwareVersion: req.FirmwareVersion,
		Status:          "offline",
		Location:        req.Location,
	}

	if err := s.repo.Create(ctx, d); err != nil {
		return nil, fmt.Errorf("register device: %w", err)
	}
	return d, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Device, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByAsset(ctx context.Context, assetID string) ([]Device, error) {
	return s.repo.ListByAsset(ctx, assetID)
}

func (s *Service) HandleHeartbeat(ctx context.Context, deviceID string) error {
	return s.repo.UpdateHeartbeat(ctx, deviceID)
}
