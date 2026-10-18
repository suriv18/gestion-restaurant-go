package bootstrap

import (
	"context"
	"log/slog"

	identidad "service-restaurant/internal/modules/identidad/infrastructure/config"
	"service-restaurant/internal/platform/config"
	"service-restaurant/internal/platform/database"
)

// AdminParams son los datos del alta inicial. Los datos de negocio y sucursal solo se
// usan si todavía no existen.
type AdminParams struct {
	Username, Password                                 string
	RUC, RazonSocial, NombreComercial, DireccionFiscal string
	SucursalCodigo, SucursalNombre, SucursalDireccion  string
}

// AdminInstalado identifica lo creado.
type AdminInstalado struct{ UsuarioID, SucursalID int64 }

// InstalarAdministrador abre la base, ensambla identidad y crea el primer administrador.
// Es el paso de instalación (cmd/admin); no forma parte del arranque de la API.
func InstalarAdministrador(ctx context.Context, cfg config.Config, logger *slog.Logger, p AdminParams) (AdminInstalado, error) {
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return AdminInstalado{}, err
	}
	defer db.Close()
	mod, err := newIdentidad(cfg, logger, db)
	if err != nil {
		return AdminInstalado{}, err
	}
	r, err := mod.InstalarAdministrador(ctx, identidad.AdminParams(p))
	return AdminInstalado(r), err
}
