package alert

import (
	"encoding/json"
	"net/http"
	"strconv"

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
	r.Get("/alerts", h.List)
	r.Get("/alerts/{id}", h.GetByID)
	r.Patch("/alerts/{id}/acknowledge", h.Acknowledge)
	r.Patch("/alerts/{id}/resolve", h.Resolve)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := AlertFilter{
		DeviceID: r.URL.Query().Get("device_id"),
		Status:   r.URL.Query().Get("status"),
		Severity: r.URL.Query().Get("severity"),
		Page:     1,
		Limit:    50,
	}

	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		filter.Page = p
	}
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		filter.Limit = l
	}

	alerts, err := h.svc.List(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

func (h *Handler) Acknowledge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a, err := h.svc.Acknowledge(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.bus.Publish(eventbus.NewEvent("alert.acknowledged", "backend", a))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a, err := h.svc.Resolve(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.bus.Publish(eventbus.NewEvent("alert.resolved", "backend", a))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}
