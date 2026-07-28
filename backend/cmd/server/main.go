package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/inframind/backend/internal/alert"
	"github.com/inframind/backend/internal/asset"
	"github.com/inframind/backend/internal/auth"
	"github.com/inframind/backend/internal/config"
	"github.com/inframind/backend/internal/db"
	"github.com/inframind/backend/internal/device"
	"github.com/inframind/backend/internal/eventbus"
	"github.com/inframind/backend/internal/health"
	"github.com/inframind/backend/internal/mqtt"
	"github.com/inframind/backend/internal/telemetry"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DB.URL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("database connected")

	bus := eventbus.New()

	emqxClient := mqtt.NewEMQXClient(cfg.MQTT.APIURL, cfg.MQTT.AdminUsername, cfg.MQTT.AdminPassword)

	// Repositories
	assetRepo := asset.NewRepository(pool)
	deviceRepo := device.NewRepository(pool)
	telemetryRepo := telemetry.NewRepository(pool)

	// Services
	assetSvc := asset.NewService(assetRepo)
	deviceSvc := device.NewService(deviceRepo, emqxClient)
	healthSvc := health.NewService(cfg.AI.URL)

	// WebSocket hub
	wsHub := telemetry.NewWSHub()

	// Telemetry ingester (wired to MQTT)
	ingester := telemetry.NewIngester(telemetryRepo, deviceSvc, bus, wsHub)

	// MQTT
	mqttSub, err := mqtt.NewSubscriber(cfg.MQTT.URL, cfg.MQTT.AdminUsername, cfg.MQTT.AdminPassword, func(topic string, payload []byte) {
		ingester.HandleMQTTMessage(topic, payload)
	})
	if err != nil {
		slog.Error("failed to connect to mqtt", "error", err)
		os.Exit(1)
	}
	defer mqttSub.Close()

	if err := mqttSub.Subscribe("telemetry/#", 1); err != nil {
		slog.Error("failed to subscribe to telemetry", "error", err)
		os.Exit(1)
	}

	heartbeatMon := device.NewHeartbeatMonitor(pool, bus)
	go heartbeatMon.Start(ctx)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(auth.Middleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link", "X-Total-Count"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"status":"ok","service":"infra-backend"}`)
		})

		asset.NewHandler(assetSvc, bus).Register(r)
		device.NewHandler(deviceSvc, bus).RegisterRoutes(r)
		telemetry.NewHandler(telemetryRepo, wsHub).Register(r)
		alert.NewHandler().Register(r)
		health.NewHandler(healthSvc).Register(r)
	})

	// Register event subscriptions
	asset.RegisterEvents(bus)
	device.RegisterEvents(bus)
	telemetry.RegisterEvents(bus)
	alert.RegisterEvents(bus)

	// Server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ingester.Stop()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
