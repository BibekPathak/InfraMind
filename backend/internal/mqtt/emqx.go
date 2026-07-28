package mqtt

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type EMQXClient struct {
	apiURL   string
	username string
	password string
	client   *http.Client
}

func NewEMQXClient(apiURL, username, password string) *EMQXClient {
	return &EMQXClient{
		apiURL:   apiURL,
		username: username,
		password: password,
		client:   &http.Client{},
	}
}

func (e *EMQXClient) CreateUser(userID, password string) error {
	body := map[string]any{
		"user_id":      userID,
		"password":     password,
		"is_superuser": false,
	}
	return e.post("/api/v5/authentication/password_based:built_in_database/users", body)
}

func (e *EMQXClient) DeleteUser(userID string) error {
	return e.delete(fmt.Sprintf("/api/v5/authentication/password_based:built_in_database/users/%s", userID))
}

func (e *EMQXClient) CreateACLRule(userID, topic string, action string) error {
	body := map[string]any{
		"rules": []map[string]any{
			{
				"topic":      topic,
				"action":     action,
				"permission": "allow",
				"principal":  []string{userID},
			},
		},
	}
	return e.post("/api/v5/authorization/built_in_database/rules", body)
}

func (e *EMQXClient) post(path string, body any) error {
	data, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", e.apiURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	e.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("http post %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("emqx api %s: status %d", path, resp.StatusCode)
	}
	slog.Debug("emqx api success", "path", path, "status", resp.StatusCode)
	return nil
}

func (e *EMQXClient) delete(path string) error {
	req, err := http.NewRequest("DELETE", e.apiURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	e.setAuth(req)

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("http delete %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("emqx api %s: status %d", path, resp.StatusCode)
	}
	return nil
}

func (e *EMQXClient) setAuth(req *http.Request) {
	auth := base64.StdEncoding.EncodeToString([]byte(e.username + ":" + e.password))
	req.Header.Set("Authorization", "Basic "+auth)
}
