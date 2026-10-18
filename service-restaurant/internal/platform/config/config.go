// Package config carga la configuración desde variables de entorno.
package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Addr            string
	DatabaseURL     string
	LogLevel        string
	ShutdownTimeout time.Duration
	MaxBodyBytes    int64
}

// Load lee el entorno; DATABASE_URL es obligatoria.
func Load() (Config, error) {
	c := Config{
		Addr:            get("APP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		LogLevel:        get("LOG_LEVEL", "info"),
		ShutdownTimeout: 15 * time.Second,
		MaxBodyBytes:    1 << 20,
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: DATABASE_URL es obligatoria")
	}
	return c, nil
}

func get(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
