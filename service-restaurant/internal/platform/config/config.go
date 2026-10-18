// Package config carga la configuración desde variables de entorno.
package config

import (
	"fmt"
	"os"
	"time"
)

// MinJWTSecretBytes es el mínimo de la clave HMAC (256 bits).
const MinJWTSecretBytes = 32

type Config struct {
	Addr            string
	DatabaseURL     string
	LogLevel        string
	ShutdownTimeout time.Duration
	MaxBodyBytes    int64

	// JWT (HS256). El secreto debe tener al menos 32 bytes y no se versiona.
	JWTSecret   []byte
	JWTIssuer   string
	JWTAudience string
}

// Load lee el entorno; DATABASE_URL es obligatoria.
func Load() (Config, error) {
	c := Config{
		Addr:            get("APP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		LogLevel:        get("LOG_LEVEL", "info"),
		ShutdownTimeout: 15 * time.Second,
		MaxBodyBytes:    1 << 20,
		JWTSecret:       []byte(os.Getenv("JWT_SECRET")),
		JWTIssuer:       get("JWT_ISSUER", "service-restaurant"),
		JWTAudience:     get("JWT_AUDIENCE", "service-restaurant-api"),
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: DATABASE_URL es obligatoria")
	}
	if len(c.JWTSecret) < MinJWTSecretBytes {
		return Config{}, fmt.Errorf("config: JWT_SECRET es obligatoria y debe tener al menos %d bytes", MinJWTSecretBytes)
	}
	return c, nil
}

func get(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
