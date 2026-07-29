package twin

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

func (h *Handler) Register(r chi.Router) {
	r.Get("/twins", h.List)
	r.Get("/twins/{id}", h.GetByID)
	r.Get("/assets/{id}/twin", h.GetByAssetID)
	r.Post("/twins/{id}/events", h.AddEvent)
	r.Get("/twins/{id}/events", h.ListEvents)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	twins, err := h.svc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(twins)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	twin, err := h.svc.GetByAssetID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(twin)
}

func (h *Handler) GetByAssetID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	twin, err := h.svc.GetByAssetID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(twin)
}

func (h *Handler) AddEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req AddEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	twin, err := h.svc.AddEvent(r.Context(), id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.bus.Publish(eventbus.NewEvent("twin.event_added", "backend", map[string]any{
		"assetId": id,
		"event":   req,
	}))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(twin)
}

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	twin, err := h.svc.GetByAssetID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	events := twin.MaintenanceHistory
	if events == nil {
		events = []TwinEvent{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
