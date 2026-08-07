package testing

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/inframind/backend/internal/mqtt"
)

// Handler exposes internal-only testing endpoints. These routes are registered
// only when EnableTestEndpoints is true (config-gated availability, not a
// test-mode branch in production logic).
type Handler struct {
	pub *mqtt.Subscriber
}

func NewHandler(pub *mqtt.Subscriber) *Handler {
	return &Handler{pub: pub}
}

// Register mounts internal endpoints under /internal.
func (h *Handler) Register(r chi.Router) {
	r.Route("/internal/testing", func(r chi.Router) {
		r.Post("/fault", h.InjectFault)
		r.Get("/fault", h.ListFaults)
	})
}

// FaultRequest tells the simulator to immediately switch a device to a fault
// scenario, making demos and tests deterministic.
type FaultRequest struct {
	DeviceID string `json:"deviceId"`
	Fault    string `json:"fault"` // healthy, overloaded, cooling_failure, ...
}

// InjectFault publishes a control message to simulator/{device_id}/fault.
func (h *Handler) InjectFault(w http.ResponseWriter, r *http.Request) {
	var req FaultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, `{"error":"deviceId is required"}`, http.StatusBadRequest)
		return
	}
	if h.pub == nil {
		http.Error(w, `{"error":"mqtt publisher unavailable"}`, http.StatusInternalServerError)
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"deviceId": req.DeviceID,
		"fault":    req.Fault,
	})

	topic := fmt.Sprintf("simulator/%s/fault", req.DeviceID)
	if err := h.pub.Publish(topic, 1, payload); err != nil {
		http.Error(w, `{"error":"failed to publish fault"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "injected", "topic": topic})
}

// ListFaults is a stub for discoverability (documentation / future).
func (h *Handler) ListFaults(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{
		"faults": {"healthy", "overloaded", "cooling_failure", "sensor_failure", "voltage_sag"},
	})
}
