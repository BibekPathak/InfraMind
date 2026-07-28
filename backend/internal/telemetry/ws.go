package telemetry

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WSEvent struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	AssetID   string `json:"asset_id"`
	Payload   any    `json:"payload"`
}

type WSClient struct {
	conn     *websocket.Conn
	deviceID string
	done     chan struct{}
}

type WSHub struct {
	mu      sync.RWMutex
	clients map[string][]*WSClient
}

func NewWSHub() *WSHub {
	return &WSHub{
		clients: make(map[string][]*WSClient),
	}
}

func (h *WSHub) Subscribe(deviceID string, conn *websocket.Conn) *WSClient {
	client := &WSClient{
		conn:     conn,
		deviceID: deviceID,
		done:     make(chan struct{}),
	}

	h.mu.Lock()
	h.clients[deviceID] = append(h.clients[deviceID], client)
	h.mu.Unlock()

	go func() {
		defer func() {
			h.unsubscribe(client)
			conn.Close()
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	slog.Debug("websocket client subscribed", "deviceId", deviceID)
	return client
}

func (h *WSHub) unsubscribe(client *WSClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients := h.clients[client.deviceID]
	for i, c := range clients {
		if c == client {
			h.clients[client.deviceID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	close(client.done)
	slog.Debug("websocket client unsubscribed", "deviceId", client.deviceID)
}

func (h *WSHub) Broadcast(deviceID string, event WSEvent) {
	h.mu.RLock()
	clients := h.clients[deviceID]
	h.mu.RUnlock()

	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("websocket marshal event", "error", err)
		return
	}

	for _, client := range clients {
		select {
		case <-client.done:
			continue
		default:
		}

		if err := client.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			slog.Warn("websocket write error", "error", err, "deviceId", deviceID)
			go h.unsubscribe(client)
		}
	}
}

func (h *WSHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		http.Error(w, `{"error":"device_id query parameter required"}`, http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade", "error", err)
		return
	}

	h.Subscribe(deviceID, conn)

	conn.WriteJSON(WSEvent{
		Type:      "connected",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		AssetID:   deviceID,
		Payload:   map[string]string{"status": "connected"},
	})
}
