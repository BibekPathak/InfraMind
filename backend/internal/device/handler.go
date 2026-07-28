package device

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/inframind/backend/internal/eventbus"
)

type Handler struct {
	svc *Service
	bus *eventbus.Bus
}

func NewHandler(svc *Service, bus *eventbus.Bus) *Handler {
	return &Handler{svc: svc, bus: bus}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/devices", h.Create)
	r.Get("/devices/{id}", h.GetByID)
	r.Post("/devices/{id}/heartbeat", h.Heartbeat)
	r.Get("/devices/{id}/config", h.GetConfig)
	r.Put("/devices/{id}/config", h.UpdateConfig)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	d, mqttUser, mqttPass, err := h.svc.Register(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.bus.Publish(eventbus.NewEvent("device.registered", "backend", d))

	resp := RegistrationResponse{
		Device:       *d,
		MQTTUsername: mqttUser,
		MQTTPassword: mqttPass,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	d, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}

func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.HandleHeartbeat(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.bus.Publish(eventbus.NewEvent("device.heartbeat", "backend", map[string]string{"deviceId": id}))
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cfg, err := h.svc.GetConfig(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req ConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if err := h.svc.UpdateConfig(r.Context(), id, req.Config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.bus.Publish(eventbus.NewEvent("device.configuration_updated", "backend", map[string]any{
		"deviceId": id,
		"config":   req.Config,
	}))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(req.Config)
}
