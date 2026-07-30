package health

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
	r.Get("/health/{deviceId}", h.GetHealth)
	r.Get("/health/{deviceId}/analysis", h.GetAnalysis)
}

func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")

	tempStr := r.URL.Query().Get("temperature")
	currentStr := r.URL.Query().Get("current")
	voltageStr := r.URL.Query().Get("voltage")
	humidityStr := r.URL.Query().Get("humidity")

	temp, _ := strconv.ParseFloat(tempStr, 64)
	current, _ := strconv.ParseFloat(currentStr, 64)
	voltage, _ := strconv.ParseFloat(voltageStr, 64)
	humidity, _ := strconv.ParseFloat(humidityStr, 64)

	health, err := h.svc.Calculate(r.Context(), deviceID, temp, current, voltage, humidity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (h *Handler) GetAnalysis(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")

	tempStr := r.URL.Query().Get("temperature")
	currentStr := r.URL.Query().Get("current")
	voltageStr := r.URL.Query().Get("voltage")
	humidityStr := r.URL.Query().Get("humidity")

	temp, _ := strconv.ParseFloat(tempStr, 64)
	current, _ := strconv.ParseFloat(currentStr, 64)
	voltage, _ := strconv.ParseFloat(voltageStr, 64)
	humidity, _ := strconv.ParseFloat(humidityStr, 64)

	analysis, err := h.svc.Analyze(r.Context(), deviceID, temp, current, voltage, humidity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}
