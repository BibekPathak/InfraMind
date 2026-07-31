package health

import (
	"context"
	"fmt"

	"github.com/inframind/backend/internal/asset"
	"github.com/inframind/backend/internal/device"
)

type DeviceAssetResolver struct {
	deviceSvc *device.Service
	assetSvc  *asset.Service
}

func NewDeviceAssetResolver(deviceSvc *device.Service, assetSvc *asset.Service) *DeviceAssetResolver {
	return &DeviceAssetResolver{deviceSvc: deviceSvc, assetSvc: assetSvc}
}

func (r *DeviceAssetResolver) ResolveAssetType(ctx context.Context, deviceID string) (string, error) {
	d, err := r.deviceSvc.GetByID(ctx, deviceID)
	if err != nil {
		return "", fmt.Errorf("resolve asset type: get device: %w", err)
	}

	a, err := r.assetSvc.GetByID(ctx, d.AssetID)
	if err != nil {
		return "", fmt.Errorf("resolve asset type: get asset: %w", err)
	}

	return a.Type, nil
}
