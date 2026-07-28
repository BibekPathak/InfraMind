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
	"github.com/inframind/simulator/device"
	"github.com/inframind/simulator/scenarios"
	"github.com/inframind/simulator/telemetry"
)

type deviceConfig struct {
	Config map[string]any `json:"config"`
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

	t := device.NewTransformer(cfg.DeviceID)
	slog.Info("simulator starting", "deviceId", t.DeviceID, "intervalMs", cfg.IntervalMs)

	fetchedCfg := fetchDeviceConfig(cfg.BackendURL, cfg.DeviceID)
	if fetchedCfg != nil {
		slog.Info("fetched device config from backend", "config", fetchedCfg)
	}

	gen := telemetry.NewGenerator(t.DeviceID)
	gen.AddScenario("healthy", scenarios.Healthy(t.DeviceID))
	gen.AddScenario("overloaded", scenarios.Overloaded(t.DeviceID))
	gen.AddScenario("cooling_failure", scenarios.CoolingFailure(t.DeviceID))
	gen.AddScenario("sensor_failure", scenarios.SensorFailure(t.DeviceID))
	gen.AddScenario("voltage_sag", scenarios.VoltageSag(t.DeviceID))

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.MQTTURL)
	opts.SetClientID(fmt.Sprintf("simulator-%s", t.DeviceID))
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
	if fetchedCfg != nil {
		if v, ok := fetchedCfg["interval_ms"]; ok {
			if iv, ok := v.(float64); ok && iv > 0 {
				interval = time.Duration(iv) * time.Millisecond
				slog.Info("using config overridden interval", "intervalMs", iv)
			}
		}
	}

	tick := 0
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("publishing telemetry", "topic", fmt.Sprintf("telemetry/%s/data", t.DeviceID))

	for {
		select {
		case <-ticker.C:
			reading := gen.Generate(tick)
			tick++

			payload, err := json.Marshal(reading)
			if err != nil {
				slog.Error("failed to marshal telemetry", "error", err)
				continue
			}

			topic := fmt.Sprintf("telemetry/%s/data", t.DeviceID)
			token := client.Publish(topic, 1, false, payload)
			token.Wait()

			slog.Info("telemetry published",
				"topic", topic,
				"scenario", reading.Scenario,
				"temp", reading.Temperature,
				"current", reading.Current,
			)

		case <-sigChan:
			slog.Info("shutting down simulator")
			return
		}
	}
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
