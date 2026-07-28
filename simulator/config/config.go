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

	MQTTUsername string
	MQTTPassword string
	BackendURL   string
}

func Load() (*Config, error) {
	k := koanf.New("__")

	if err := k.Load(env.Provider("INFRA_", "__", func(s string) string {
		return s
	}), nil); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	cfg := &Config{
		MQTTURL:      k.String("mqtt__url"),
		DeviceID:     k.String("device__id"),
		IntervalMs:   k.Int("interval__ms"),
		MQTTUsername: k.String("mqtt__username"),
		MQTTPassword: k.String("mqtt__password"),
		BackendURL:   k.String("backend__url"),
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
	if cfg.BackendURL == "" {
		cfg.BackendURL = "http://localhost:8080"
	}

	return cfg, nil
}
