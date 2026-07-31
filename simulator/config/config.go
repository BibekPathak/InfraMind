package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

type DeviceSpec struct {
	ID   string
	Type string
}

type Config struct {
	MQTTURL    string
	Devices    []DeviceSpec
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
		IntervalMs:   k.Int("interval__ms"),
		MQTTUsername: k.String("mqtt__username"),
		MQTTPassword: k.String("mqtt__password"),
		BackendURL:   k.String("backend__url"),
	}

	if cfg.MQTTURL == "" {
		cfg.MQTTURL = "mqtt://localhost:1883"
	}
	if cfg.IntervalMs <= 0 {
		cfg.IntervalMs = 2000
	}
	if cfg.BackendURL == "" {
		cfg.BackendURL = "http://localhost:8080"
	}

	cfg.Devices = parseDevices(k.String("devices"))

	return cfg, nil
}

func parseDevices(raw string) []DeviceSpec {
	if raw == "" {
		return []DeviceSpec{{ID: "tx-001", Type: "transformer"}}
	}

	parts := strings.Split(raw, ",")
	specs := make([]DeviceSpec, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		spec := DeviceSpec{ID: p, Type: "transformer"}
		if strings.Contains(p, ":") {
			seg := strings.SplitN(p, ":", 2)
			spec.ID = strings.TrimSpace(seg[0])
			spec.Type = strings.TrimSpace(seg[1])
		}
		specs = append(specs, spec)
	}
	if len(specs) == 0 {
		specs = []DeviceSpec{{ID: "tx-001", Type: "transformer"}}
	}
	return specs
}
