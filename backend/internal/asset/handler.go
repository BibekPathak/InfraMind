package asset

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/inframind/backend/internal/audit"
	"github.com/inframind/backend/internal/auth"
	"github.com/inframind/backend/internal/eventbus"
)

type Handler struct {
	svc   *Service
	bus   *eventbus.Bus
	audit *audit.Service
}

func NewHandler(svc *Service, bus *eventbus.Bus, auditSvc *audit.Service) *Handler {
	return &Handler{svc: svc, bus: bus, audit: auditSvc}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/assets", h.List)
	r.Post("/assets", h.Create)
	r.Get("/assets/{id}", h.GetByID)
	r.Get("/assets/{id}/autonomy", h.GetAutonomy)
	r.Patch("/assets/{id}/autonomy", h.UpdateAutonomy)
	r.Delete("/assets/{id}", h.Delete)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	assets, err := h.svc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assets)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	a, err := h.svc.Create(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.bus.Publish(eventbus.NewEvent("asset.created", "backend", a))
	if h.audit != nil {
		h.audit.Record(r.Context(), "asset.created", "asset", a.ID, userID(r), map[string]any{
			"name": a.Name, "type": a.Type,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
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

func (h *Handler) GetAutonomy(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	mode, err := h.svc.GetAutonomyMode(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"autonomyMode": mode})
}

func (h *Handler) UpdateAutonomy(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateAutonomyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	a, err := h.svc.UpdateAutonomyMode(r.Context(), id, req.AutonomyMode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.bus.Publish(eventbus.NewEvent("asset.autonomy_changed", "backend", map[string]any{
		"id":           id,
		"autonomyMode": req.AutonomyMode,
	}))
	if h.audit != nil {
		h.audit.Record(r.Context(), "asset.autonomy_changed", "asset", id, userID(r), map[string]any{
			"autonomyMode": req.AutonomyMode,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.bus.Publish(eventbus.NewEvent("asset.deleted", "backend", map[string]string{"id": id}))
	if h.audit != nil {
		h.audit.Record(r.Context(), "asset.deleted", "asset", id, userID(r), map[string]any{})
	}
	w.WriteHeader(http.StatusNoContent)
}

func userID(r *http.Request) string {
	if id, ok := r.Context().Value(auth.UserIDKey).(string); ok {
		return id
	}
	return "system"
}
