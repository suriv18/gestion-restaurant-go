// Package bootstrap compone plataforma y módulos; no contiene reglas de negocio.
package bootstrap

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	atencion "service-restaurant/internal/modules/atencion/infrastructure/config"
	catalogo "service-restaurant/internal/modules/catalogo/infrastructure/config"
	identidad "service-restaurant/internal/modules/identidad/infrastructure/config"
	produccion "service-restaurant/internal/modules/produccion/infrastructure/config"
	"service-restaurant/internal/platform/config"
	"service-restaurant/internal/platform/database"
)

type App struct {
	cfg        config.Config
	logger     *slog.Logger
	db         *sql.DB
	identidad  *identidad.Module
	atencion   *atencion.Module
	catalogo   *catalogo.Module
	produccion *produccion.Module
	server     *http.Server
}

func newIdentidad(cfg config.Config, logger *slog.Logger, db *sql.DB) (*identidad.Module, error) {
	return identidad.New(identidad.Deps{
		DB:        db,
		Logger:    logger,
		JWTSecret: cfg.JWTSecret, JWTIssuer: cfg.JWTIssuer, JWTAudience: cfg.JWTAudience,
	})
}

// New abre recursos, ensambla los módulos y arma el servidor.
func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	mod, err := newIdentidad(cfg, logger, db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	atn, err := atencion.New(atencion.Deps{DB: db, Logger: logger})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	cat, err := catalogo.New(catalogo.Deps{DB: db, Logger: logger})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	prd, err := produccion.New(produccion.Deps{DB: db, Logger: logger})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	a := &App{cfg: cfg, logger: logger, db: db, identidad: mod, atencion: atn, catalogo: cat, produccion: prd}
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
