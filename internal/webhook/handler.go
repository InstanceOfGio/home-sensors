package webhook

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"home-sensors/internal/models"
	"home-sensors/internal/storage"
	hub "home-sensors/internal/websocket"
)

type Handler struct {
	store *storage.Store
	hub   *hub.Hub
	log   *slog.Logger
}

func NewHandler(store *storage.Store, h *hub.Hub, log *slog.Logger) *Handler {
	return &Handler{store: store, hub: h, log: log}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var payload models.MinewPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	stored := 0

	for _, adv := range payload.Adv {
		if adv.Type != "ht" {
			continue
		}

		device, err := h.store.UpsertDevice(ctx, adv.Mac)
		if err != nil {
			h.log.Error("upsert device failed", "mac", adv.Mac, "err", err)
			continue
		}

		ts := adv.Tm
		if ts.IsZero() {
			ts = time.Now().UTC()
		}

		if err := h.store.InsertReading(ctx, device.ID, adv.Temperature, adv.Humidity, adv.Battery, adv.Rssi, ts); err != nil {
			h.log.Error("insert reading failed", "mac", adv.Mac, "err", err)
			continue
		}
		stored++

		h.hub.Broadcast(hub.Update{
			DeviceID:    device.ID,
			Mac:         device.Mac,
			Name:        device.Name,
			Room:        device.Room,
			Temperature: adv.Temperature,
			Humidity:    adv.Humidity,
			Battery:     adv.Battery,
			Rssi:        adv.Rssi,
			UpdatedAt:   ts,
		})
	}

	h.log.Info("webhook processed", "gw", payload.Gw, "seq", payload.Seq, "received", len(payload.Adv), "stored", stored)
	w.WriteHeader(http.StatusNoContent)
}
