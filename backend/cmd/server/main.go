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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/inframind/backend/internal/action"
	"github.com/inframind/backend/internal/alert"
	"github.com/inframind/backend/internal/asset"
	"github.com/inframind/backend/internal/assettype"
	"github.com/inframind/backend/internal/audit"
	"github.com/inframind/backend/internal/auth"
	"github.com/inframind/backend/internal/config"
	"github.com/inframind/backend/internal/db"
	"github.com/inframind/backend/internal/device"
	"github.com/inframind/backend/internal/eventbus"
	"github.com/inframind/backend/internal/health"
	"github.com/inframind/backend/internal/metrics"
	apimw "github.com/inframind/backend/internal/middleware"
	"github.com/inframind/backend/internal/mqtt"
	"github.com/inframind/backend/internal/organization"
	"github.com/inframind/backend/internal/otel"
	"github.com/inframind/backend/internal/telemetry"
	"github.com/inframind/backend/internal/testing"
	"github.com/inframind/backend/internal/twin"
	"github.com/inframind/backend/internal/workorder"
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

	if cfg.AppEnv != "development" && cfg.Auth.JWTSecret == "infra-dev-secret-do-not-use-in-prod" {
		slog.Error("refusing to start: default JWT secret in non-development environment; set INFRA_AUTH_JWT_SECRET")
		os.Exit(1)
	}

	// OpenTelemetry tracing (no-op when no collector configured)
	shutdownTracing, err := otel.Setup("infra-backend")
	if err != nil {
		slog.Warn("failed to initialize tracing, continuing without it", "error", err)
	} else {
		defer shutdownTracing(context.Background())
	}

	// Prometheus metrics
	promRegistry := prometheus.NewRegistry()
	m := metrics.New(promRegistry)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DB.URL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("database connected")

	auth.InitJWT(cfg.Auth.JWTSecret)
	authSvc := auth.NewAuthService(auth.NewJWTManager(cfg.Auth.JWTSecret))

	bus := eventbus.New()
	bus.SetMetrics(m)

	// Durable cross-instance event delivery via Redis Streams (optional).
	if cfg.Redis.EnableEvents {
		instanceID := fmt.Sprintf("backend-%d", os.Getpid())
		redisBackend, err := eventbus.NewRedisBackend(cfg.Redis.URL, instanceID)
		if err != nil {
			slog.Warn("eventbus: redis backend disabled", "error", err)
		} else {
			bus.SetRedisBackend(redisBackend)
			go redisBackend.Start(ctx, func(evt eventbus.Event) {
				bus.DispatchLocal(evt)
			})
			defer redisBackend.Close()
			slog.Info("eventbus: redis streams enabled", "stream", "infra:events", "instance", instanceID)
		}
	}

	emqxClient := mqtt.NewEMQXClient(cfg.MQTT.APIURL, cfg.MQTT.AdminUsername, cfg.MQTT.AdminPassword)

	// Repositories
	assetRepo := asset.NewRepository(pool)
	deviceRepo := device.NewRepository(pool)
	telemetryRepo := telemetry.NewRepository(pool)
	alertRepo := alert.NewRepository(pool)
	assetTypeRepo := assettype.NewRepository(pool)
	orgRepo := organization.NewRepository(pool)

	// Services
	orgSvc := organization.NewService(orgRepo)
	assetTypeSvc := assettype.NewService(assetTypeRepo)
	assetSvc := asset.NewService(assetRepo)
	assetSvc.SetTypeValidator(assetTypeSvc)
	deviceSvc := device.NewService(deviceRepo, emqxClient)
	healthSvc := health.NewService(cfg.AI.URL, telemetryRepo)
	healthSvc.SetAssetTypeResolver(health.NewDeviceAssetResolver(deviceSvc, assetSvc))
	healthSvc.SetEventPublisher(&recommendationPublisher{bus: bus})
	alertSvc := alert.NewService(alertRepo)
	workOrderRepo := workorder.NewRepository(pool)
	workOrderSvc := workorder.NewService(workOrderRepo)
	actionRepo := action.NewRepository(pool)
	actionSvc := action.NewService(actionRepo)
	auditRepo := audit.NewRepository(pool)
	auditSvc := audit.NewService(auditRepo)

	// WebSocket hub
	wsHub := telemetry.NewWSHub()

	twinRepo := twin.NewRepository(pool)
	twinSvc := twin.NewService(twinRepo, assetSvc, deviceSvc, telemetryRepo, healthSvc)
	twinSync := twin.NewSyncEngineWithInterval(twinSvc, bus, wsHub, cfg.Timing.TwinSyncInterval)

	// Notifier + Alert engine
	notifier := alert.NewLogNotifier()
	alertEngine := alert.NewEngineWithInterval(alertSvc, bus, notifier, telemetryRepo, m, cfg.Timing.AlertInterval)

	// Telemetry ingester (wired to MQTT)
	ingester := telemetry.NewIngester(telemetryRepo, deviceSvc, bus, wsHub, m)

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

	actionExec := action.NewExecutorWithInterval(actionSvc, bus, mqttSub, action.NewPolicyEvaluator(assetSvc), m, cfg.Timing.ActionInterval)
	go actionExec.Run(ctx)

	heartbeatMon := device.NewHeartbeatMonitorWithInterval(pool, bus, cfg.Timing.HeartbeatInterval, cfg.Timing.DeviceTimeout)
	go heartbeatMon.Start(ctx)

	go alertEngine.Start(ctx)

	go twinSync.Start(ctx)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(otel.HTTPMiddleware("infra-backend"))
	r.Use(apimw.CorrelationID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(m.Middleware)
	r.Use(auth.Middleware)
	r.Use(apimw.MaxBody(1 << 20))
	rl := apimw.NewRateLimiter(cfg.Server.RateLimit, time.Minute)
	if redisClient, err := redis.ParseURL(cfg.Redis.URL); err == nil {
		rl.SetRedis(redis.NewClient(redisClient))
	} else {
		slog.Warn("rate limiter using in-memory backend", "error", err)
	}
	r.Use(rl.Middleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link", "X-Total-Count"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Prometheus metrics endpoint (public, no auth)
	r.Get("/metrics", m.Handler().ServeHTTP)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"status":"ok","service":"infra-backend"}`)
		})

		// Read-only views available to all authenticated roles
		telemetry.NewHandler(telemetryRepo, wsHub).Register(r)
		health.NewHandler(healthSvc).Register(r)
		twin.NewHandler(twinSvc, bus).Register(r)
		audit.NewHandler(auditSvc).Register(r)

		// Asset type catalog: viewers can read; writes require operator
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRoleOnMutation(auth.PermWrite))
			assettype.NewHandler(assetTypeSvc, bus).Register(r)
		})

		// Assets, devices, alerts, work orders, actions: reads for all,
		// writes (create/update) for operators and above
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRoleOnMutation(auth.PermWrite))
			asset.NewHandler(assetSvc, bus, auditSvc).Register(r)
			device.NewHandler(deviceSvc, bus, auditSvc).RegisterRoutes(r)
			alert.NewHandler(alertSvc, bus).Register(r)
			workorder.NewHandler(workOrderSvc, bus, auditSvc).Register(r)
			action.NewHandler(actionSvc, bus, auditSvc).Register(r)
		})

		// Organization management: admin only
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAdmin())
			organization.NewHandler(orgSvc, bus, auditSvc).Register(r)
		})

		auth.NewHandler(authSvc).RegisterRoutes(r)
	})

	// Internal testing endpoints (explicitly opt-in via config).
	if cfg.EnableTestEndpoints {
		r.Route("/", func(r chi.Router) {
			testing.NewHandler(mqttSub).Register(r)
		})
		slog.Warn("internal testing endpoints ENABLED - do not use in production")
	}

	// Register event subscriptions
	asset.RegisterEvents(bus)
	device.RegisterEvents(bus)
	telemetry.RegisterEvents(bus)
	alert.RegisterEvents(bus, alertSvc)
	workorder.RegisterEvents(bus, workOrderSvc, deviceSvc)
	action.RegisterEvents(bus, actionSvc, deviceSvc)

	// Server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Server.Port, "tls", cfg.Server.TLSEnabled)
		var err error
		if cfg.Server.TLSEnabled {
			if cfg.Server.TLSCert == "" || cfg.Server.TLSKey == "" {
				slog.Error("tls enabled but cert/key not configured")
				os.Exit(1)
			}
			err = srv.ListenAndServeTLS(cfg.Server.TLSCert, cfg.Server.TLSKey)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
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

type recommendationPublisher struct {
	bus *eventbus.Bus
}

func (p *recommendationPublisher) PublishRecommendation(deviceID string, recs []health.RecommendationResult) {
	items := make([]action.RecommendationItem, 0, len(recs))
	for _, r := range recs {
		items = append(items, action.RecommendationItem{
			ActionType:    r.ActionType,
			ActionPayload: r.ActionPayload,
			Reason:        r.Reason,
		})
	}
	p.bus.Publish(eventbus.NewEvent("ai.recommendation.generated", "health_service", action.RecommendationEvent{
		DeviceID:        deviceID,
		Recommendations: items,
	}))
}
