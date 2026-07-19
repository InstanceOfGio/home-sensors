package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"home-sensors/frontend"
	"home-sensors/internal/api"
	"home-sensors/internal/config"
	"home-sensors/internal/storage"
	"home-sensors/internal/webhook"
	"home-sensors/internal/weather"
	hub "home-sensors/internal/websocket"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	cfg := config.Load()

	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		log.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	store := storage.New(db)
	wsHub := hub.New(log)
	wh := webhook.NewHandler(store, wsHub, log)

	staticSub, err := fs.Sub(frontend.FS, "static")
	if err != nil {
		log.Error("failed to load embedded frontend", "err", err)
		os.Exit(1)
	}

	bgCtx, stopBg := context.WithCancel(context.Background())
	defer stopBg()

	var weatherCache *weather.Cache
	if cfg.WeatherEnabled {
		weatherCache = weather.NewCache(cfg.WeatherLat, cfg.WeatherLon, log)
		weatherCache.Start(bgCtx)
	} else {
		log.Warn("WEATHER_LAT/WEATHER_LON not set, weather forecast disabled")
	}

	router := api.NewRouter(store, wh, wsHub, weatherCache, http.FS(staticSub), log, cfg.AuthUser, cfg.AuthPass)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Info("server starting", "port", cfg.Port, "db_path", cfg.DBPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
	}
}
