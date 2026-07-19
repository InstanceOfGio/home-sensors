package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"home-sensors/internal/storage"
	"home-sensors/internal/weather"
)

type Handlers struct {
	store   *storage.Store
	weather *weather.Cache
	log     *slog.Logger
}

func (h *Handlers) ListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.store.ListDevices(r.Context())
	if err != nil {
		h.log.Error("list devices failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, devices)
}

type updateDeviceRequest struct {
	Name string `json:"name"`
	Room string `json:"room"`
	Type string `json:"type"`
}

func (h *Handlers) UpdateDevice(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req updateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	device, err := h.store.UpdateDevice(r.Context(), id, req.Name, req.Room, req.Type)
	if err != nil {
		h.log.Error("update device failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, device)
}

func (h *Handlers) Current(w http.ResponseWriter, r *http.Request) {
	states, err := h.store.CurrentState(r.Context())
	if err != nil {
		h.log.Error("current state failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, states)
}

func (h *Handlers) History(w http.ResponseWriter, r *http.Request) {
	rangeParam := r.URL.Query().Get("range")

	var since time.Time
	switch rangeParam {
	case "7d":
		since = time.Now().AddDate(0, 0, -7)
	case "30d":
		since = time.Now().AddDate(0, 0, -30)
	case "24h", "":
		since = time.Now().Add(-24 * time.Hour)
	default:
		http.Error(w, "invalid range (use 24h, 7d or 30d)", http.StatusBadRequest)
		return
	}

	points, err := h.store.History(r.Context(), since)
	if err != nil {
		h.log.Error("history failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, points)
}

func (h *Handlers) Weather(w http.ResponseWriter, r *http.Request) {
	points, fetchedAt := h.weather.Get()
	if fetchedAt.IsZero() {
		http.Error(w, "weather forecast not available yet", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, points)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
