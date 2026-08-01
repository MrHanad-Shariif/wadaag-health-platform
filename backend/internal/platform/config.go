package platform

import (
	"fmt"
	"os"
	"time"
)

// Config holds process-wide configuration read from the environment.
// Local dev values default sanely so `go run ./cmd/api` works against the
// Docker Compose stack without any .env file present.
type Config struct {
	Port            string
	DatabaseURL     string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Env             string
}

func LoadConfig() (Config, error) {
	cfg := Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://wadaag:wadaag@localhost:5432/wadaag_health?sslmode=disable"),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Env:             getEnv("APP_ENV", "development"),
	}

	if cfg.JWTSecret == "" {
		if cfg.Env != "development" {
			return Config{}, fmt.Errorf("JWT_SECRET must be set outside development")
		}
		cfg.JWTSecret = "dev-only-insecure-secret-do-not-use-in-production"
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
