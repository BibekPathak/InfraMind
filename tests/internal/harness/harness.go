package harness

import (
	"context"
	"database/sql"
	"fmt"
	"net"
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
	t          *testing.T
	ctx        context.Context
	cancel     context.CancelFunc
	cfg        *Config
	containers []tc.Container
	byName     map[string]tc.Container
	backend    *exec.Cmd
	ai         *exec.Cmd
	pool       *pgxpool.Pool
	buildOnce  sync.Once
	builtBin   string
	buildErr   error
	mu         sync.Mutex
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
	h := &Harness{t: t, ctx: ctx, cancel: cancel, byName: make(map[string]tc.Container)}

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
	h.byName["timescaledb"] = dbC
	dbPort, _ := dbC.MappedPort(h.ctx, "5432")
	dbHost, _ := dbC.Host(h.ctx)

	// EMQX broker
	emqxC, err := tc.GenericContainer(h.ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image: "emqx/emqx:latest",
			Env: map[string]string{
				"EMQX_DASHBOARD__DEFAULT_PASSWORD":       "infra123",
				"EMQX_AUTHENTICATION__1__MECHANISM":      "password_based",
				"EMQX_AUTHENTICATION__1__BACKEND":        "built_in_database",
				"EMQX_AUTHENTICATION__1__BOOTSTRAP_FILE": "/opt/emqx/etc/users.json",
				"EMQX_AUTHORIZATION__SOURCES__1__TYPE":   "built_in_database",
				"EMQX_AUTHORIZATION__NO_MATCH":           "deny",
			},
			ExposedPorts: []string{"1883/tcp"},
			WaitingFor:   wait.ForListeningPort("1883/tcp").WithStartupTimeout(120 * time.Second),
			Mounts: tc.ContainerMounts{
				tc.BindMount(filepath.Join(mustRepoRoot(), "deployments", "emqx", "users.json"), "/opt/emqx/etc/users.json"),
			},
		},
		Started: true,
	})
	if err != nil {
		return fmt.Errorf("start emqx: %w", err)
	}
	h.containers = append(h.containers, emqxC)
	h.byName["emqx"] = emqxC
	mqttPort, _ := emqxC.MappedPort(h.ctx, "1883")
	mqttHost, _ := emqxC.Host(h.ctx)

	// EMQX reports the listener open before it accepts MQTT CONNECT packets.
	// Poll with an actual admin connection before proceeding.
	mqttURL := fmt.Sprintf("mqtt://%s:%s", mqttHost, mqttPort.Port())
	if !WaitFor(60_000_000_000, 1_000_000_000, func() bool {
		return MQTTAdminConnectOK(mqttURL)
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
	h.byName["redis"] = redisC
	redisPort, _ := redisC.MappedPort(h.ctx, "6379")
	redisHost, _ := redisC.Host(h.ctx)

	h.cfg = &Config{
		DBURL:    fmt.Sprintf("postgres://infra:infra@%s:%s/inframind?sslmode=disable", dbHost, dbPort.Port()),
		DBHost:   dbHost,
		DBPort:   dbPort.Port(),
		MQTTURL:  fmt.Sprintf("mqtt://%s:%s", mqttHost, mqttPort.Port()),
		MQTTPort: mqttPort.Port(),
		AIURL:    "",
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
	port := freePort()
	h.cfg.AIURL = "http://localhost:" + port
	cmd := exec.Command("python", "-m", "uvicorn", "main:app", "--host", "127.0.0.1", "--port", port)
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
	port := freePort()
	h.cfg.APIPort = port
	h.cfg.APIURL = "http://localhost:" + port

	env := []string{
		"INFRA_APP_ENV=development",
		"INFRA_SERVER_PORT=" + port,
		"INFRA_SERVER_RATE_LIMIT=100000",
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

// StopContainer stops a named container (timescaledb, emqx, redis).
func (h *Harness) StopContainer(name string) error {
	c, ok := h.byName[name]
	if !ok {
		return fmt.Errorf("unknown container %q", name)
	}
	return c.Stop(h.ctx, nil)
}

// StartContainer starts a previously stopped named container.
func (h *Harness) StartContainer(name string) error {
	c, ok := h.byName[name]
	if !ok {
		return fmt.Errorf("unknown container %q", name)
	}
	return c.Start(h.ctx)
}

// RestartContainer stops and restarts a named container.
func (h *Harness) RestartContainer(name string) error {
	if err := h.StopContainer(name); err != nil {
		return fmt.Errorf("stop %s: %w", name, err)
	}
	if err := h.StartContainer(name); err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}
	return nil
}

// RestartBackend kills and restarts the backend binary subprocess, then waits
// for its health endpoint to become ready again.
func (h *Harness) RestartBackend() error {
	if h.backend != nil && h.backend.Process != nil {
		h.backend.Process.Kill()
		h.backend.Wait()
	}

	port := freePort()
	h.cfg.APIPort = port
	h.cfg.APIURL = "http://localhost:" + port

	env := []string{
		"INFRA_APP_ENV=development",
		"INFRA_SERVER_PORT=" + port,
		"INFRA_SERVER_RATE_LIMIT=100000",
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
		return fmt.Errorf("restart backend: %w", err)
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
	return fmt.Errorf("backend did not become ready after restart")
}

// RestartAI kills and restarts the AI service subprocess.
func (h *Harness) RestartAI() error {
	if err := h.StopAI(); err != nil {
		return err
	}
	return h.startAI()
}

// ContainerByName returns a named container and whether it exists.
func (h *Harness) ContainerByName(name string) (tc.Container, bool) {
	c, ok := h.byName[name]
	return c, ok
}

// RefreshDBPort re-reads the TimescaleDB mapped port after a container restart
// (testcontainers may reassign host ports) and updates the config + pool.
func (h *Harness) RefreshDBPort() error {
	c, ok := h.byName["timescaledb"]
	if !ok {
		return fmt.Errorf("timescaledb container not found")
	}
	p, err := c.MappedPort(h.ctx, "5432")
	if err != nil {
		return fmt.Errorf("read db port: %w", err)
	}
	host, _ := c.Host(h.ctx)
	h.cfg.DBPort = p.Port()
	h.cfg.DBHost = host
	h.cfg.DBURL = fmt.Sprintf("postgres://infra:infra@%s:%s/inframind?sslmode=disable", host, p.Port())

	// Re-establish the DB pool against the new port.
	if h.pool != nil {
		h.pool.Close()
	}
	pool, err := pgxpool.New(h.ctx, h.cfg.DBURL)
	if err != nil {
		return fmt.Errorf("refresh db pool: %w", err)
	}
	h.pool = pool
	return nil
}

// RefreshMQTTPort re-reads the EMQX mapped port after a container restart.
func (h *Harness) RefreshMQTTPort() error {
	c, ok := h.byName["emqx"]
	if !ok {
		return fmt.Errorf("emqx container not found")
	}
	p, err := c.MappedPort(h.ctx, "1883")
	if err != nil {
		return fmt.Errorf("read mqtt port: %w", err)
	}
	host, _ := c.Host(h.ctx)
	h.cfg.MQTTPort = p.Port()
	h.cfg.MQTTURL = fmt.Sprintf("mqtt://%s:%s", host, p.Port())
	return nil
}

// RefreshRedisPort re-reads the Redis mapped port after a container restart.
func (h *Harness) RefreshRedisPort() error {
	c, ok := h.byName["redis"]
	if !ok {
		return fmt.Errorf("redis container not found")
	}
	p, err := c.MappedPort(h.ctx, "6379")
	if err != nil {
		return fmt.Errorf("read redis port: %w", err)
	}
	host, _ := c.Host(h.ctx)
	h.cfg.RedisURL = fmt.Sprintf("redis://%s:%s", host, p.Port())
	return nil
}

// Ctx returns the harness context.
func (h *Harness) Ctx() context.Context { return h.ctx }

// StopAI kills the AI subprocess without restarting it.
func (h *Harness) StopAI() error {
	if h.ai != nil && h.ai.Process != nil {
		h.ai.Process.Kill()
		h.ai.Wait()
		h.ai = nil
	}
	return nil
}

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

// freePort returns a currently-free TCP port for binding local subprocesses,
// avoiding conflicts with orphaned processes from prior runs.
func freePort() string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "0"
	}
	defer l.Close()
	return fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port)
}

func httpGet(url string) (int, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}
