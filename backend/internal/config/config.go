package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	AppEnv string
	Server ServerConfig
	DB     DBConfig
	MQTT   MQTTConfig
	Redis  RedisConfig
	AI     AIConfig
	Auth   AuthConfig
	Timing TimingConfig
}

type TimingConfig struct {
	HeartbeatInterval time.Duration
	DeviceTimeout     time.Duration
	AlertInterval     time.Duration
	TwinSyncInterval  time.Duration
	ActionInterval    time.Duration
}

type AuthConfig struct {
	JWTSecret string
}

type ServerConfig struct {
	Port       int
	TLSEnabled bool
	TLSCert    string
	TLSKey     string
}

type DBConfig struct {
	URL string
}

type MQTTConfig struct {
	URL          string
	APIURL       string
	AdminUsername string
	AdminPassword string
}

type RedisConfig struct {
	URL           string
	EnableEvents  bool
}

type AIConfig struct {
	URL string
}

func Load() (*Config, error) {
	k := koanf.New("__")

	if err := k.Load(file.Provider("config.toml"), toml.Parser()); err != nil {
		// config file is optional
	}

	if err := k.Load(env.Provider("INFRA_", "__", func(s string) string {
		s = strings.TrimPrefix(s, "INFRA_")
		return strings.ReplaceAll(strings.ToLower(s), "_", "__")
	}), nil); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	cfg := &Config{
		AppEnv: k.String("app__env"),
		Server: ServerConfig{
			Port:       k.Int("server__port"),
			TLSEnabled: k.Bool("server__tls__enabled"),
			TLSCert:    k.String("server__tls__cert"),
			TLSKey:     k.String("server__tls__key"),
		},
		DB: DBConfig{URL: k.String("db__url")},
		MQTT: MQTTConfig{
			URL:          k.String("mqtt__url"),
			APIURL:       k.String("mqtt__api__url"),
			AdminUsername: k.String("mqtt__admin__username"),
			AdminPassword: k.String("mqtt__admin__password"),
		},
		Redis: RedisConfig{URL: k.String("redis__url"), EnableEvents: k.Bool("redis__enable__events")},
		AI:    AIConfig{URL: k.String("ai__url")},
		Auth:  AuthConfig{JWTSecret: k.String("auth__jwt__secret")},
		Timing: TimingConfig{
			HeartbeatInterval: k.Duration("heartbeat__interval"),
			DeviceTimeout:     k.Duration("device__timeout"),
			AlertInterval:     k.Duration("alert__interval"),
			TwinSyncInterval:  k.Duration("twin__sync__interval"),
			ActionInterval:    k.Duration("action__interval"),
		},
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.DB.URL == "" {
		cfg.DB.URL = "postgres://infra:infra@localhost:5432/inframind?sslmode=disable"
	}
	if cfg.MQTT.URL == "" {
		cfg.MQTT.URL = "mqtt://localhost:1883"
	}
	if cfg.MQTT.APIURL == "" {
		cfg.MQTT.APIURL = "http://localhost:18083"
	}
	if cfg.MQTT.AdminUsername == "" {
		cfg.MQTT.AdminUsername = "mqtt_admin"
	}
	if cfg.MQTT.AdminPassword == "" {
		cfg.MQTT.AdminPassword = "mqtt_admin_secret"
	}
	if cfg.Redis.URL == "" {
		cfg.Redis.URL = "redis://localhost:6379"
	}
	if cfg.AI.URL == "" {
		cfg.AI.URL = "http://localhost:9090"
	}
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "infra-dev-secret-do-not-use-in-prod"
	}
	if cfg.AppEnv == "" {
		cfg.AppEnv = "development"
	}
	if cfg.Timing.HeartbeatInterval <= 0 {
		cfg.Timing.HeartbeatInterval = 60 * time.Second
	}
	if cfg.Timing.DeviceTimeout <= 0 {
		cfg.Timing.DeviceTimeout = 2 * time.Minute
	}
	if cfg.Timing.AlertInterval <= 0 {
		cfg.Timing.AlertInterval = 10 * time.Second
	}
	if cfg.Timing.TwinSyncInterval <= 0 {
		cfg.Timing.TwinSyncInterval = 30 * time.Second
	}
	if cfg.Timing.ActionInterval <= 0 {
		cfg.Timing.ActionInterval = 10 * time.Second
	}

	return cfg, nil
}
