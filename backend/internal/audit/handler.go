package audit

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/audit-logs", h.List)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	f := Filter{
		ResourceType: r.URL.Query().Get("resource_type"),
		Action:       r.URL.Query().Get("action"),
		Page:         1,
		Limit:        50,
	}
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		f.Page = p
	}
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		f.Limit = l
	}

	logs, err := h.svc.List(r.Context(), f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
