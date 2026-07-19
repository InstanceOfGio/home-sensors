package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"home-sensors/internal/storage"
	"home-sensors/internal/webhook"
	"home-sensors/internal/weather"
	hub "home-sensors/internal/websocket"
)

func NewRouter(store *storage.Store, wh *webhook.Handler, wsHub *hub.Hub, weatherCache *weather.Cache, staticFS http.FileSystem, log *slog.Logger, authUser, authPass string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	if authUser != "" && authPass != "" {
		r.Use(basicAuth(authUser, authPass))
	} else {
		log.Warn("BASIC_AUTH_USER/BASIC_AUTH_PASS not set, API is running without authentication")
	}

	h := &Handlers{store: store, weather: weatherCache, log: log}

	r.Post("/webhook/minew", wh.ServeHTTP)
	r.Get("/ws", wsHub.ServeWS)

	r.Route("/api", func(r chi.Router) {
		r.Get("/devices", h.ListDevices)
		r.Patch("/devices/{id}", h.UpdateDevice)
		r.Get("/current", h.Current)
		r.Get("/history", h.History)
		if weatherCache != nil {
			r.Get("/weather", h.Weather)
		}
	})

	r.Handle("/*", http.FileServer(staticFS))

	return r
}
