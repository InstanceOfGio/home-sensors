package config

import "os"

type Config struct {
	Port     string
	DBPath   string
	AuthUser string
	AuthPass string
}

func Load() Config {
	return Config{
		Port:     getEnv("PORT", "8080"),
		DBPath:   getEnv("DB_PATH", "./data/sensors.db"),
		AuthUser: os.Getenv("BASIC_AUTH_USER"),
		AuthPass: os.Getenv("BASIC_AUTH_PASS"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
