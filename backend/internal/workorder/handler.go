package workorder

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
	r.Post("/work-orders", h.Create)
	r.Get("/work-orders", h.List)
	r.Get("/work-orders/{id}", h.GetByID)
	r.Patch("/work-orders/{id}/assign", h.Assign)
	r.Patch("/work-orders/{id}/status", h.UpdateStatus)
	r.Get("/work-orders/{id}/timeline", h.Timeline)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	wo, err := h.svc.Create(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.bus.Publish(eventbus.NewEvent("workorder.created", "backend", wo))
	if h.audit != nil {
		h.audit.Record(r.Context(), "workorder.created", "work_order", wo.ID, woUserID(r), map[string]any{
			"assetId": wo.AssetID, "type": wo.Type, "priority": wo.Priority,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(wo)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	f := Filter{
		AssetID:  r.URL.Query().Get("asset_id"),
		Status:   r.URL.Query().Get("status"),
		Priority: r.URL.Query().Get("priority"),
		Page:     1,
		Limit:    50,
	}
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		f.Page = p
	}
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		f.Limit = l
	}

	orders, err := h.svc.List(r.Context(), f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wo, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wo)
}

func (h *Handler) Assign(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req AssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	wo, err := h.svc.Assign(r.Context(), id, req.AssignedTo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.bus.Publish(eventbus.NewEvent("workorder.assigned", "backend", wo))
	if h.audit != nil {
		h.audit.Record(r.Context(), "workorder.assigned", "work_order", id, woUserID(r), map[string]any{
			"assignedTo": req.AssignedTo,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wo)
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req StatusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	wo, err := h.svc.UpdateStatus(r.Context(), id, req.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.bus.Publish(eventbus.NewEvent("workorder.status_changed", "backend", wo))
	if h.audit != nil {
		h.audit.Record(r.Context(), "workorder.status_changed", "work_order", id, woUserID(r), map[string]any{
			"status": req.Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wo)
}

func (h *Handler) Timeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wo, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	timeline := wo.Timeline
	if timeline == nil {
		timeline = []TimelineEvent{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timeline)
}

func woUserID(r *http.Request) string {
	if id, ok := r.Context().Value(auth.UserIDKey).(string); ok {
		return id
	}
	return "system"
}
