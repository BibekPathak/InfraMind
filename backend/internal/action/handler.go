package action

import (
	"encoding/json"
	"net/http"
	"strconv"

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
	r.Get("/actions", h.List)
	r.Post("/actions", h.Propose)
	r.Get("/actions/{id}", h.GetByID)
	r.Patch("/actions/{id}/approve", h.Approve)
	r.Patch("/actions/{id}/reject", h.Reject)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	f := Filter{
		AssetID: r.URL.Query().Get("asset_id"),
		Status:  r.URL.Query().Get("status"),
		Type:    r.URL.Query().Get("type"),
		Page:    1,
		Limit:   50,
	}
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		f.Page = p
	}
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		f.Limit = l
	}

	actions, err := h.svc.List(r.Context(), f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actions)
}

func (h *Handler) Propose(w http.ResponseWriter, r *http.Request) {
	var req ProposeActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	a, err := h.svc.Propose(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.bus.Publish(eventbus.NewEvent("action.proposed", "backend", a))
	if h.audit != nil {
		h.audit.Record(r.Context(), "action.proposed", "action", a.ID, actionUserID(r), map[string]any{
			"type": a.Type, "source": a.Source, "assetId": a.AssetID,
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

func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a, err := h.svc.Approve(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.bus.Publish(eventbus.NewEvent("action.approved", "backend", a))
	if h.audit != nil {
		h.audit.Record(r.Context(), "action.approved", "action", id, actionUserID(r), map[string]any{})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a, err := h.svc.Reject(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.bus.Publish(eventbus.NewEvent("action.rejected", "backend", a))
	if h.audit != nil {
		h.audit.Record(r.Context(), "action.rejected", "action", id, actionUserID(r), map[string]any{})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

func actionUserID(r *http.Request) string {
	if id, ok := r.Context().Value(auth.UserIDKey).(string); ok {
		return id
	}
	return "system"
}
