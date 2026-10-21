package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// ConfiguracionReadAdapter implementa los lectores de negocio y sucursales.
type ConfiguracionReadAdapter struct {
	repo *repository.ConfiguracionSQLReader
}

func NewConfiguracionReadAdapter(db database.DBTX) *ConfiguracionReadAdapter {
	return &ConfiguracionReadAdapter{repo: repository.NewConfiguracionSQLReader(db)}
}

var _ outbound.NegocioReader = (*ConfiguracionReadAdapter)(nil)

func (a *ConfiguracionReadAdapter) ObtenerNegocio(ctx context.Context) (dtoresult.Negocio, bool, error) {
	return a.repo.GetNegocio(ctx)
}

var _ outbound.SucursalReader = (*ConfiguracionReadAdapter)(nil)

func (a *ConfiguracionReadAdapter) ListarSucursales(ctx context.Context, f outbound.FiltroSucursales) ([]dtoresult.Sucursal, error) {
	return a.repo.ListSucursales(ctx, f)
}
func (a *ConfiguracionReadAdapter) ObtenerSucursal(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	return a.repo.GetSucursal(ctx, id)
}
