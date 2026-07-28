package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
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

	gen := telemetry.NewGenerator(t.DeviceID)
	gen.AddScenario("healthy", scenarios.Healthy(t.DeviceID))
	gen.AddScenario("overloaded", scenarios.Overloaded(t.DeviceID))
	gen.AddScenario("cooling_failure", scenarios.CoolingFailure(t.DeviceID))

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.MQTTURL)
	opts.SetClientID(fmt.Sprintf("simulator-%s", t.DeviceID))
	opts.SetCleanSession(true)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if token.Error() != nil {
		slog.Error("mqtt connection failed", "error", token.Error())
		os.Exit(1)
	}
	defer client.Disconnect(250)
	slog.Info("connected to mqtt", "broker", cfg.MQTTURL)

	tick := 0
	ticker := time.NewTicker(time.Duration(cfg.IntervalMs) * time.Millisecond)
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
