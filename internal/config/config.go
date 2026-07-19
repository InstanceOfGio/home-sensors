package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port     string
	DBPath   string
	AuthUser string
	AuthPass string

	WeatherEnabled bool
	WeatherLat     float64
	WeatherLon     float64
}

func Load() Config {
	lat, latErr := strconv.ParseFloat(os.Getenv("WEATHER_LAT"), 64)
	lon, lonErr := strconv.ParseFloat(os.Getenv("WEATHER_LON"), 64)

	return Config{
		Port:     getEnv("PORT", "8080"),
		DBPath:   getEnv("DB_PATH", "./data/sensors.db"),
		AuthUser: os.Getenv("BASIC_AUTH_USER"),
		AuthPass: os.Getenv("BASIC_AUTH_PASS"),

		WeatherEnabled: latErr == nil && lonErr == nil,
		WeatherLat:     lat,
		WeatherLon:     lon,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
