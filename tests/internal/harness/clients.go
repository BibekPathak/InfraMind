package harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

// TestJWTSecret matches the backend dev secret.
const TestJWTSecret = "infra-dev-secret-do-not-use-in-prod"

type testClaims struct {
	UserID         string `json:"user_id"`
	Role           string `json:"role"`
	OrganizationID string `json:"organization_id"`
	jwt.RegisteredClaims
}

// MintToken creates a signed JWT for a given org (matches the backend's
// HMAC signing scheme) so tests can simulate users in different tenants.
func MintToken(userID, role, orgID string) (string, error) {
	claims := testClaims{
		UserID:         userID,
		Role:           role,
		OrganizationID: orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "inframind",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(TestJWTSecret))
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// LoginResponse mirrors backend /auth/login.
type LoginResponse struct {
	AccessToken    string `json:"accessToken"`
	UserID         string `json:"userId"`
	Role           string `json:"role"`
	DisplayName    string `json:"displayName"`
	OrganizationID string `json:"organizationId"`
}

// APIClient is a thin HTTP client with JWT auth for the backend under test.
type APIClient struct {
	baseURL string
	token   string
	hc      *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{baseURL: baseURL, hc: httpClient}
}

func (c *APIClient) SetToken(tok string) { c.token = tok }

// Login authenticates and stores the access token.
func (c *APIClient) Login(email, password string) (*LoginResponse, error) {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp, err := c.do("POST", "/api/v1/auth/login", body, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var lr LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	c.token = lr.AccessToken
	return &lr, nil
}

// Do performs a request with the stored token and decodes JSON into out.
func (c *APIClient) Do(method, path string, body any, out any) (int, error) {
	var b []byte
	var err error
	if body != nil {
		b, err = json.Marshal(body)
		if err != nil {
			return 0, err
		}
	}
	resp, err := c.do(method, path, b, c.token)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if out != nil {
		data, _ := io.ReadAll(resp.Body)
		if len(data) > 0 {
			_ = json.Unmarshal(data, out)
		}
	}
	return resp.StatusCode, nil
}

func (c *APIClient) do(method, path string, body []byte, token string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.hc.Do(req)
}

// MQTTPub publishes a raw JSON payload to a topic with QoS 1 using the admin
// credentials (EMQX deny-by-default ACL would reject anonymous publishers).
func (h *Harness) MQTTPub(topic string, payload []byte) error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(h.cfg.MQTTURL)
	opts.SetClientID(fmt.Sprintf("harness-%d", time.Now().UnixNano()))
	opts.SetUsername("mqtt_admin")
	opts.SetPassword("mqtt_admin_secret")
	opts.SetConnectTimeout(5 * time.Second)
	client := mqtt.NewClient(opts)
	tok := client.Connect()
	tok.Wait()
	if tok.Error() != nil {
		return fmt.Errorf("mqtt connect: %w", tok.Error())
	}
	defer client.Disconnect(250)

	ptok := client.Publish(topic, 1, false, payload)
	ptok.Wait()
	if ptok.Error() != nil {
		return fmt.Errorf("mqtt publish %s: %w", topic, ptok.Error())
	}
	return nil
}

// mqttAdminConnectOK checks whether EMQX accepts the admin MQTT connection.
func MQTTAdminConnectOK(brokerURL string) bool {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(fmt.Sprintf("harness-probe-%d", time.Now().UnixNano()))
	opts.SetUsername("mqtt_admin")
	opts.SetPassword("mqtt_admin_secret")
	opts.SetConnectTimeout(3 * time.Second)
	client := mqtt.NewClient(opts)
	tok := client.Connect()
	tok.WaitTimeout(3 * time.Second)
	if tok.Error() != nil {
		return false
	}
	client.Disconnect(100)
	return true
}

// WSEvent is the typed event envelope pushed by the backend.
type WSEvent struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	AssetID   string          `json:"asset_id"`
	Payload   json.RawMessage `json:"payload"`
}

// WSClient subscribes to the backend WebSocket stream for a device.
type WSClient struct {
	conn *websocket.Conn
}

func (h *Harness) WSConnect(deviceID string) (*WSClient, error) {
	wsURL := "ws" + h.cfg.APIURL[len("http"):] + "/api/v1/telemetry/ws?device_id=" + deviceID
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ws dial: %w", err)
	}
	return &WSClient{conn: conn}, nil
}

// Read waits for the next event or times out.
func (w *WSClient) Read(timeout time.Duration) (*WSEvent, error) {
	w.conn.SetReadDeadline(time.Now().Add(timeout))
	var evt WSEvent
	if err := w.conn.ReadJSON(&evt); err != nil {
		return nil, err
	}
	return &evt, nil
}

func (w *WSClient) Close() { w.conn.Close() }

// WaitFor polls fn until it returns true or the deadline expires.
func WaitFor(deadline time.Duration, interval time.Duration, fn func() bool) bool {
	start := time.Now()
	for time.Since(start) < deadline {
		if fn() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}
