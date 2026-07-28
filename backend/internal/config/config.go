package config

import (
	"fmt"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Server ServerConfig
	DB     DBConfig
	MQTT   MQTTConfig
	Redis  RedisConfig
	AI     AIConfig
}

type ServerConfig struct {
	Port int
}

type DBConfig struct {
	URL string
}

type MQTTConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
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
		return s
	}), nil); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{Port: k.Int("server__port")},
		DB:     DBConfig{URL: k.String("db__url")},
		MQTT:   MQTTConfig{URL: k.String("mqtt__url")},
		Redis:  RedisConfig{URL: k.String("redis__url")},
		AI:     AIConfig{URL: k.String("ai__url")},
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
	if cfg.Redis.URL == "" {
		cfg.Redis.URL = "redis://localhost:6379"
	}
	if cfg.AI.URL == "" {
		cfg.AI.URL = "http://localhost:9090"
	}

	return cfg, nil
}
