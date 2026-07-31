package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/inframind/simulator/config"
	"github.com/inframind/simulator/scenarios"
	"github.com/inframind/simulator/telemetry"
)

type deviceConfig struct {
	Config map[string]any `json:"config"`
}

type simDevice struct {
	spec   config.DeviceSpec
	gen    *telemetry.Generator
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	devices := buildDevices(cfg)
	slog.Info("simulator starting", "deviceCount", len(devices), "intervalMs", cfg.IntervalMs)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.MQTTURL)
	opts.SetClientID("simulator-multi")
	opts.SetCleanSession(true)

	if cfg.MQTTUsername != "" {
		opts.SetUsername(cfg.MQTTUsername)
		opts.SetPassword(cfg.MQTTPassword)
		slog.Info("authenticating to mqtt", "username", cfg.MQTTUsername)
	}

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		slog.Info("mqtt connected", "broker", cfg.MQTTURL)
	})
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		slog.Error("mqtt connection lost", "error", err)
	})

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if token.Error() != nil {
		slog.Error("mqtt connection failed", "error", token.Error())
		os.Exit(1)
	}
	defer client.Disconnect(250)
	slog.Info("connected to mqtt", "broker", cfg.MQTTURL)

	interval := time.Duration(cfg.IntervalMs) * time.Millisecond

	tick := 0
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			for _, d := range devices {
				publishReading(client, d.gen, tick)
			}
			tick++

		case <-sigChan:
			slog.Info("shutting down simulator")
			return
		}
	}
}

func buildDevices(cfg *config.Config) []simDevice {
	devices := make([]simDevice, 0, len(cfg.Devices))
	for _, spec := range cfg.Devices {
		gen := newGenerator(spec)
		if gen == nil {
			slog.Warn("unsupported device type, skipping", "deviceId", spec.ID, "type", spec.Type)
			continue
		}
		fetchedCfg := fetchDeviceConfig(cfg.BackendURL, spec.ID)
		if fetchedCfg != nil {
			slog.Info("fetched device config from backend", "deviceId", spec.ID, "config", fetchedCfg)
		}
		devices = append(devices, simDevice{spec: spec, gen: gen})
		slog.Info("simulator device online", "deviceId", spec.ID, "type", spec.Type)
	}
	return devices
}

func newGenerator(spec config.DeviceSpec) *telemetry.Generator {
	gen := telemetry.NewGenerator(spec.ID)

	switch spec.Type {
	case "transformer":
		gen.AddScenario("healthy", scenarios.Healthy(spec.ID))
		gen.AddScenario("overloaded", scenarios.Overloaded(spec.ID))
		gen.AddScenario("cooling_failure", scenarios.CoolingFailure(spec.ID))
		gen.AddScenario("sensor_failure", scenarios.SensorFailure(spec.ID))
		gen.AddScenario("voltage_sag", scenarios.VoltageSag(spec.ID))
	case "pump":
		gen.AddScenario("healthy", scenarios.PumpHealthy(spec.ID))
		gen.AddScenario("overloaded", scenarios.PumpOverloaded(spec.ID))
		gen.AddScenario("cavitation", scenarios.PumpCavitation(spec.ID))
	case "motor":
		gen.AddScenario("healthy", scenarios.MotorHealthy(spec.ID))
		gen.AddScenario("bearing_wear", scenarios.MotorBearingWear(spec.ID))
		gen.AddScenario("overloaded", scenarios.MotorOverload(spec.ID))
	default:
		return nil
	}

	return gen
}

func publishReading(client mqtt.Client, gen *telemetry.Generator, tick int) {
	reading := gen.Generate(tick)

	payload, err := json.Marshal(reading)
	if err != nil {
		slog.Error("failed to marshal telemetry", "error", err, "deviceId", reading.DeviceID)
		return
	}

	topic := fmt.Sprintf("telemetry/%s/data", reading.DeviceID)
	token := client.Publish(topic, 1, false, payload)
	token.Wait()

	slog.Info("telemetry published",
		"topic", topic,
		"deviceId", reading.DeviceID,
		"scenario", reading.Scenario,
		"temp", reading.Temperature,
	)
}

func fetchDeviceConfig(backendURL, deviceID string) map[string]any {
	url := fmt.Sprintf("%s/api/v1/devices/%s/config", backendURL, deviceID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Warn("failed to create config request", "error", err)
		return nil
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("failed to fetch device config, using defaults", "deviceId", deviceID, "error", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("device config not found, using defaults", "deviceId", deviceID, "status", resp.StatusCode)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("failed to read config response", "error", err)
		return nil
	}

	var cfg map[string]any
	if err := json.Unmarshal(body, &cfg); err != nil {
		slog.Warn("failed to parse config", "error", err)
		return nil
	}

	return cfg
}
