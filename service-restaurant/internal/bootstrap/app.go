// Package bootstrap compone plataforma y módulos; no contiene reglas de negocio.
package bootstrap

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"service-restaurant/internal/platform/config"
	"service-restaurant/internal/platform/database"
)

type App struct {
	cfg    config.Config
	logger *slog.Logger
	db     *sql.DB
	server *http.Server
}

// New abre recursos y arma el servidor.
func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	a := &App{cfg: cfg, logger: logger, db: db}
	a.server = &http.Server{
		Addr:              cfg.Addr,
		Handler:           a.router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return a, nil
}
