package device

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/inframind/backend/internal/mqtt"
	"github.com/inframind/backend/pkg/uuidv7"
)

type Service struct {
	repo       *Repository
	emqxClient *mqtt.EMQXClient
}

func NewService(repo *Repository, emqxClient *mqtt.EMQXClient) *Service {
	return &Service{repo: repo, emqxClient: emqxClient}
}

func (s *Service) Register(ctx context.Context, req RegisterDeviceRequest) (*Device, string, string, error) {
	if req.AssetID == "" {
		return nil, "", "", fmt.Errorf("assetId is required")
	}

	id := uuidv7.New()
	mqttUser := fmt.Sprintf("device-%s", id)
	mqttPass := generatePassword(16)

	d := &Device{
		ID:              id,
		AssetID:         req.AssetID,
		FirmwareVersion: req.FirmwareVersion,
		Status:          "offline",
		Location:        req.Location,
		MQTTUsername:    mqttUser,
		MQTTPassword:    mqttPass,
	}

	if err := s.repo.Create(ctx, d); err != nil {
		return nil, "", "", fmt.Errorf("register device: %w", err)
	}

	if err := s.emqxClient.CreateUser(mqttUser, mqttPass); err != nil {
		slog.Warn("failed to create mqtt user, device registered without credentials",
			"deviceId", id, "error", err)
		return d, "", "", nil
	}

	topic := fmt.Sprintf("telemetry/%s/#", id)
	if err := s.emqxClient.CreateACLRule(mqttUser, topic, "publish"); err != nil {
		slog.Warn("failed to create mqtt acl rule", "deviceId", id, "error", err)
	}

	return d, mqttUser, mqttPass, nil
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

func (s *Service) GetConfig(ctx context.Context, id string) (map[string]any, error) {
	return s.repo.GetConfig(ctx, id)
}

func (s *Service) UpdateConfig(ctx context.Context, id string, config map[string]any) error {
	return s.repo.UpdateConfig(ctx, id, config)
}

func generatePassword(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Errorf("generate password: %w", err))
	}
	return hex.EncodeToString(b)[:length]
}
