package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// NegocioWriteAdapter implementa outbound.NegocioStore.
type NegocioWriteAdapter struct{ repo *repository.NegocioSQLWriter }

func NewNegocioWriteAdapter(db database.DBTX) *NegocioWriteAdapter {
	return &NegocioWriteAdapter{repo: repository.NewNegocioSQLWriter(db)}
}

var _ outbound.NegocioStore = (*NegocioWriteAdapter)(nil)

func (a *NegocioWriteAdapter) Actualizar(ctx context.Context, c outbound.CambiosNegocio) (dtoresult.Negocio, bool, error) {
	return a.repo.Update(ctx, c)
}

// SucursalWriteAdapter implementa outbound.SucursalStore.
type SucursalWriteAdapter struct {
	repo    *repository.SucursalSQLWriter
	accesos *repository.AccesosSQLWriter
}

func NewSucursalWriteAdapter(db database.DBTX) *SucursalWriteAdapter {
	return &SucursalWriteAdapter{repo: repository.NewSucursalSQLWriter(db), accesos: repository.NewAccesosSQLWriter(db)}
}

func (a *SucursalWriteAdapter) ContarAdministradoresEfectivos(ctx context.Context) (int, error) {
	return a.accesos.ContarAdministradores(ctx)
}

var _ outbound.SucursalStore = (*SucursalWriteAdapter)(nil)

func (a *SucursalWriteAdapter) Crear(ctx context.Context, n outbound.NuevaSucursal) (dtoresult.Sucursal, error) {
	return a.repo.Insert(ctx, n)
}
func (a *SucursalWriteAdapter) Obtener(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	return a.repo.GetForUpdate(ctx, id)
}
func (a *SucursalWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosSucursal) (dtoresult.Sucursal, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *SucursalWriteAdapter) BloquearNegocio(ctx context.Context) error {
	return a.repo.LockNegocio(ctx)
}
func (a *SucursalWriteAdapter) ContarActivas(ctx context.Context) (int, error) {
	return a.repo.CountActivas(ctx)
}
func (a *SucursalWriteAdapter) DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Sucursal, bool, error) {
	return a.repo.SetBaja(ctx, id, motivo)
}
func (a *SucursalWriteAdapter) Restaurar(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	return a.repo.SetActiva(ctx, id)
}
