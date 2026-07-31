package assettype

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
	r.Get("/asset-types", h.List)
	r.Post("/asset-types", h.Create)
	r.Get("/asset-types/{type}", h.GetByType)
	r.Put("/asset-types/{type}", h.Update)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	types, err := h.svc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateAssetTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	at, err := h.svc.Create(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.bus.Publish(eventbus.NewEvent("assettype.created", "backend", at))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(at)
}

func (h *Handler) GetByType(w http.ResponseWriter, r *http.Request) {
	typeName := chi.URLParam(r, "type")
	at, err := h.svc.GetByType(r.Context(), typeName)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(at)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	typeName := chi.URLParam(r, "type")
	var req UpdateAssetTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	at, err := h.svc.Update(r.Context(), typeName, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	h.bus.Publish(eventbus.NewEvent("assettype.updated", "backend", at))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(at)
}
