package telemetry

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo *Repository
	hub  *WSHub
}

func NewHandler(repo *Repository, hub *WSHub) *Handler {
	return &Handler{repo: repo, hub: hub}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/devices/{id}/telemetry", h.Query)
	r.Get("/telemetry/live", h.Live)
	r.Get("/telemetry/aggregate", h.Aggregate)
	r.Get("/telemetry/ws", h.WebSocket)
}

func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	limit := 100

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC()

	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = t
		}
	}

	results, err := h.repo.Query(r.Context(), id, from, to, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if results == nil {
		results = []Telemetry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		http.Error(w, `{"error":"device_id query parameter required"}`, http.StatusBadRequest)
		return
	}

	t, err := h.repo.GetLatest(r.Context(), deviceID)
	if err != nil {
		http.Error(w, `{"error":"no telemetry found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func (h *Handler) Aggregate(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		http.Error(w, `{"error":"device_id query parameter required"}`, http.StatusBadRequest)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	window := r.URL.Query().Get("window")

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC()

	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = t
		}
	}
	if window == "" {
		window = "5m"
	}

	results, err := h.repo.Aggregate(r.Context(), deviceID, from, to, window)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *Handler) WebSocket(w http.ResponseWriter, r *http.Request) {
	h.hub.HandleWebSocket(w, r)
}
