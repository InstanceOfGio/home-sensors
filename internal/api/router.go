package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"home-sensors/internal/storage"
	"home-sensors/internal/webhook"
	hub "home-sensors/internal/websocket"
)

func NewRouter(store *storage.Store, wh *webhook.Handler, wsHub *hub.Hub, staticFS http.FileSystem, log *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := &Handlers{store: store, log: log}

	r.Post("/webhook/minew", wh.ServeHTTP)
	r.Get("/ws", wsHub.ServeWS)

	r.Route("/api", func(r chi.Router) {
		r.Get("/devices", h.ListDevices)
		r.Patch("/devices/{id}", h.UpdateDevice)
		r.Get("/current", h.Current)
		r.Get("/history", h.History)
	})

	r.Handle("/*", http.FileServer(staticFS))

	return r
}
