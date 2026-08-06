package harness

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Config holds resolved addresses for the test stack.
type Config struct {
	DBURL      string
	DBHost     string
	DBPort     string
	MQTTURL    string
	MQTTPort   string
	APIURL     string
	APIPort    string
	AIURL      string
	RedisURL   string
	BackendPid int
}

// Harness bootstraps a full running stack: real containers (TimescaleDB,
// EMQX, Redis) plus the real backend binary and real AI service.
type Harness struct {
	t        *testing.T
	ctx      context.Context
	cancel   context.CancelFunc
	cfg      *Config
	containers []tc.Container
	backend  *exec.Cmd
	ai       *exec.Cmd
	pool     *pgxpool.Pool
	buildOnce sync.Once
	builtBin  string
	buildErr  error
	mu       sync.Mutex
}

var (
	globalHarness *Harness
	globalOnce    sync.Once
	globalErr     error
)

// Global returns a process-wide harness shared by all tests in the suite.
// Built once via sync.Once; tests call CloseGlobal in TestMain.
func Global(t *testing.T) (*Harness, error) {
	globalOnce.Do(func() {
		globalHarness, globalErr = New(t)
	})
	return globalHarness, globalErr
}

func CloseGlobal() {
	if globalHarness != nil {
		globalHarness.Close()
		globalHarness = nil
	}
}

func New(t *testing.T) (*Harness, error) {
	ctx, cancel := context.WithCancel(context.Background())
	h := &Harness{t: t, ctx: ctx, cancel: cancel}

	if err := h.startContainers(); err != nil {
		h.Close()
		return nil, fmt.Errorf("start containers: %w", err)
	}
	if err := h.runMigrations(); err != nil {
		h.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	if err := h.buildBackend(); err != nil {
		h.Close()
		return nil, fmt.Errorf("build backend: %w", err)
	}
	if err := h.startAI(); err != nil {
		h.Close()
		return nil, fmt.Errorf("start ai: %w", err)
	}
	if err := h.startBackend(); err != nil {
		h.Close()
		return nil, fmt.Errorf("start backend: %w", err)
	}
	if err := h.connectPool(); err != nil {
		h.Close()
		return nil, fmt.Errorf("connect pool: %w", err)
	}

	return h, nil
}

func (h *Harness) startContainers() error {
	// TimescaleDB (PostgreSQL + TimescaleDB extension)
	dbC, err := tc.GenericContainer(h.ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image: "timescale/timescaledb:latest-pg16",
			Env: map[string]string{
				"POSTGRES_USER":     "infra",
				"POSTGRES_PASSWORD": "infra",
				"POSTGRES_DB":       "inframind",
			},
			ExposedPorts: []string{"5432/tcp"},
			WaitingFor: wait.ForExec([]string{"pg_isready", "-U", "infra", "-d", "inframind"}).
				WithStartupTimeout(120 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return fmt.Errorf("start timescaledb: %w", err)
	}
	h.containers = append(h.containers, dbC)
	dbPort, _ := dbC.MappedPort(h.ctx, "5432")
	dbHost, _ := dbC.Host(h.ctx)

	// EMQX broker
	emqxC, err := tc.GenericContainer(h.ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image: "emqx/emqx:latest",
			Env: map[string]string{
				"EMQX_DASHBOARD__DEFAULT_PASSWORD": "infra123",
				"EMQX_AUTH__BUILTIN__TYPE":         "internal",
				"EMQX_AUTH__BUILTIN__BOOTSTRAP_USERS_FILE": "/opt/emqx/etc/users.bootstrap",
				"EMQX_AUTHORIZATION__SOURCES__1__TYPE":      "built_in_database",
				"EMQX_AUTHORIZATION__NO_MATCH":              "deny",
			},
			ExposedPorts: []string{"1883/tcp"},
			WaitingFor:   wait.ForListeningPort("1883/tcp").WithStartupTimeout(120 * time.Second),
			Mounts: tc.ContainerMounts{
				tc.BindMount(filepath.Join(mustRepoRoot(), "deployments", "emqx", "users.bootstrap"), "/opt/emqx/etc/users.bootstrap"),
			},
		},
		Started: true,
	})
	if err != nil {
		return fmt.Errorf("start emqx: %w", err)
	}
	h.containers = append(h.containers, emqxC)
	mqttPort, _ := emqxC.MappedPort(h.ctx, "1883")
	mqttHost, _ := emqxC.Host(h.ctx)

	// EMQX reports the listener open before it accepts MQTT CONNECT packets.
	// Poll with an actual admin connection before proceeding.
	mqttURL := fmt.Sprintf("mqtt://%s:%s", mqttHost, mqttPort.Port())
	if !WaitFor(60_000_000_000, 1_000_000_000, func() bool {
		return mqttAdminConnectOK(mqttURL)
	}) {
		return fmt.Errorf("emqx did not accept mqtt admin connection")
	}

	// Redis
	redisC, err := tc.GenericContainer(h.ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	if err != nil {
		return fmt.Errorf("start redis: %w", err)
	}
	h.containers = append(h.containers, redisC)
	redisPort, _ := redisC.MappedPort(h.ctx, "6379")
	redisHost, _ := redisC.Host(h.ctx)

	h.cfg = &Config{
		DBURL:    fmt.Sprintf("postgres://infra:infra@%s:%s/inframind?sslmode=disable", dbHost, dbPort.Port()),
		DBHost:   dbHost,
		DBPort:   dbPort.Port(),
		MQTTURL:  fmt.Sprintf("mqtt://%s:%s", mqttHost, mqttPort.Port()),
		MQTTPort: mqttPort.Port(),
		AIURL:    "http://localhost:19090",
		RedisURL: fmt.Sprintf("redis://%s:%s", redisHost, redisPort.Port()),
	}
	return nil
}

func (h *Harness) runMigrations() error {
	repoRoot := mustRepoRoot()
	dir := filepath.Join(repoRoot, "backend", "internal", "db", "migrations")

	db, err := sql.Open("pgx", h.cfg.DBURL)
	if err != nil {
		return fmt.Errorf("open sql for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.Up(db, dir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

func (h *Harness) buildBackend() error {
	var err error
	h.buildOnce.Do(func() {
		repoRoot := mustRepoRoot()
		bin := filepath.Join(os.TempDir(), "infra-backend-test")
		cmd := exec.Command("go", "build", "-o", bin, "./cmd/server")
		cmd.Dir = filepath.Join(repoRoot, "backend")
		out, buildErr := cmd.CombinedOutput()
		if buildErr != nil {
			h.buildErr = fmt.Errorf("build backend: %w\n%s", buildErr, out)
			return
		}
		h.builtBin = bin
	})
	if h.buildErr != nil {
		return h.buildErr
	}
	return err
}

func (h *Harness) startAI() error {
	repoRoot := mustRepoRoot()
	aiDir := filepath.Join(repoRoot, "ai")
	cmd := exec.Command("python", "-m", "uvicorn", "main:app", "--host", "127.0.0.1", "--port", "19090")
	cmd.Dir = aiDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ai: %w", err)
	}
	h.ai = cmd

	// Wait for AI to become ready.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := httpGet(h.cfg.AIURL + "/health"); err == nil && resp == 200 {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("ai service did not become ready")
}

func (h *Harness) startBackend() error {
	port := "18080"
	h.cfg.APIPort = port
	h.cfg.APIURL = "http://localhost:" + port

	env := []string{
		"INFRA_APP_ENV=development",
		"INFRA_SERVER_PORT=" + port,
		"INFRA_DB_URL=" + h.cfg.DBURL,
		"INFRA_MQTT_URL=" + h.cfg.MQTTURL,
		"INFRA_MQTT_API_URL=http://localhost:18083",
		"INFRA_MQTT_ADMIN_USERNAME=mqtt_admin",
		"INFRA_MQTT_ADMIN_PASSWORD=mqtt_admin_secret",
		"INFRA_REDIS_URL=" + h.cfg.RedisURL,
		"INFRA_REDIS_ENABLE_EVENTS=false",
		"INFRA_AI_URL=" + h.cfg.AIURL,
		"INFRA_AUTH_JWT_SECRET=infra-dev-secret-do-not-use-in-prod",
		"INFRA_HEARTBEAT_INTERVAL=2s",
		"INFRA_DEVICE_TIMEOUT=6s",
		"INFRA_ALERT_INTERVAL=1s",
		"INFRA_TWIN_SYNC_INTERVAL=1s",
		"INFRA_ACTION_INTERVAL=1s",
	}
	env = append(env, os.Environ()...)

	cmd := exec.Command(h.builtBin)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start backend: %w", err)
	}
	h.backend = cmd
	h.cfg.BackendPid = cmd.Process.Pid

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := httpGet(h.cfg.APIURL + "/api/v1/health"); err == nil && resp == 200 {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("backend did not become ready")
}

func (h *Harness) connectPool() error {
	pool, err := pgxpool.New(h.ctx, h.cfg.DBURL)
	if err != nil {
		return err
	}
	h.pool = pool
	return nil
}

// Close tears down the whole stack.
func (h *Harness) Close() {
	if h.backend != nil && h.backend.Process != nil {
		h.backend.Process.Kill()
		h.backend.Wait()
	}
	if h.ai != nil && h.ai.Process != nil {
		h.ai.Process.Kill()
		h.ai.Wait()
	}
	if h.pool != nil {
		h.pool.Close()
	}
	for _, c := range h.containers {
		c.Terminate(h.ctx)
	}
	h.cancel()
}

// Config returns the resolved stack config.
func (h *Harness) Config() *Config { return h.cfg }

// Pool returns the DB pool (for direct assertions).
func (h *Harness) Pool() *pgxpool.Pool { return h.pool }

func mustRepoRoot() string {
	wd, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(wd, "docker-compose.yml")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return ""
		}
		wd = parent
	}
}

func httpGet(url string) (int, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}
