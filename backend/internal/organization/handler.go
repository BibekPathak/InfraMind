package organization

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
	r.Get("/organizations", h.List)
	r.Post("/organizations", h.Create)
	r.Get("/organizations/{id}", h.GetByID)
	r.Patch("/organizations/{id}", h.Update)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgs, err := h.svc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orgs)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	o, err := h.svc.Create(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.bus.Publish(eventbus.NewEvent("organization.created", "backend", o))
	if h.audit != nil {
		h.audit.Record(r.Context(), "organization.created", "organization", o.ID, orgUserID(r), map[string]any{
			"name": o.Name, "slug": o.Slug,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(o)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	o, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	o, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.bus.Publish(eventbus.NewEvent("organization.updated", "backend", o))
	if h.audit != nil {
		h.audit.Record(r.Context(), "organization.updated", "organization", id, orgUserID(r), map[string]any{})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}

func orgUserID(r *http.Request) string {
	if id, ok := r.Context().Value(auth.UserIDKey).(string); ok {
		return id
	}
	return "system"
}
