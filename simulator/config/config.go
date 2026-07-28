package config

import (
	"fmt"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	MQTTURL    string
	DeviceID   string
	IntervalMs int
}

func Load() (*Config, error) {
	k := koanf.New("__")

	if err := k.Load(env.Provider("INFRA_", "__", func(s string) string {
		return s
	}), nil); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	cfg := &Config{
		MQTTURL:    k.String("mqtt__url"),
		DeviceID:   k.String("device__id"),
		IntervalMs: k.Int("interval__ms"),
	}

	if cfg.MQTTURL == "" {
		cfg.MQTTURL = "mqtt://localhost:1883"
	}
	if cfg.DeviceID == "" {
		cfg.DeviceID = "tx-001"
	}
	if cfg.IntervalMs <= 0 {
		cfg.IntervalMs = 2000
	}

	return cfg, nil
}
