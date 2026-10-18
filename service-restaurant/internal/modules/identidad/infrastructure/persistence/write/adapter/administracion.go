package adapter

import (
	"context"
	"time"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// RolWriteAdapter implementa outbound.RolStore.
type RolWriteAdapter struct{ repo *repository.RolSQLWriter }

func NewRolWriteAdapter(db database.DBTX) *RolWriteAdapter {
	return &RolWriteAdapter{repo: repository.NewRolSQLWriter(db)}
}

var _ outbound.RolStore = (*RolWriteAdapter)(nil)

func (a *RolWriteAdapter) Crear(ctx context.Context, codigo, nombre string, descripcion *string) (dtoresult.Rol, error) {
	return a.repo.Insert(ctx, codigo, nombre, descripcion)
}
func (a *RolWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosRol) (dtoresult.Rol, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *RolWriteAdapter) CambiarActivo(ctx context.Context, id int64, activo bool) (dtoresult.Rol, bool, error) {
	return a.repo.SetActivo(ctx, id, activo)
}

// PermisoWriteAdapter implementa outbound.PermisoStore.
type PermisoWriteAdapter struct{ repo *repository.PermisoSQLWriter }

func NewPermisoWriteAdapter(db database.DBTX) *PermisoWriteAdapter {
	return &PermisoWriteAdapter{repo: repository.NewPermisoSQLWriter(db)}
}

var _ outbound.PermisoStore = (*PermisoWriteAdapter)(nil)

func (a *PermisoWriteAdapter) Crear(ctx context.Context, codigo, descripcion string) (dtoresult.Permiso, error) {
	return a.repo.Insert(ctx, codigo, descripcion)
}
func (a *PermisoWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosPermiso) (dtoresult.Permiso, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *PermisoWriteAdapter) CambiarActivo(ctx context.Context, id int64, activo bool) (dtoresult.Permiso, bool, error) {
	return a.repo.SetActivo(ctx, id, activo)
}

// AccesosWriteAdapter implementa outbound.AccesosStore.
type AccesosWriteAdapter struct{ repo *repository.AccesosSQLWriter }

func NewAccesosWriteAdapter(db database.DBTX) *AccesosWriteAdapter {
	return &AccesosWriteAdapter{repo: repository.NewAccesosSQLWriter(db)}
}

var _ outbound.AccesosStore = (*AccesosWriteAdapter)(nil)

func (a *AccesosWriteAdapter) CrearUsuario(ctx context.Context, username string, email *string, hash string) (dtoresult.UsuarioAdmin, error) {
	return a.repo.InsertUsuario(ctx, username, email, hash)
}
func (a *AccesosWriteAdapter) ObtenerUsuario(ctx context.Context, id int64) (dtoresult.UsuarioAdmin, bool, error) {
	return a.repo.GetUsuario(ctx, id)
}
func (a *AccesosWriteAdapter) SucursalesActivas(ctx context.Context, usuarioID int64) ([]int64, error) {
	return a.repo.SucursalesActivas(ctx, usuarioID)
}
func (a *AccesosWriteAdapter) ReemplazarAsignaciones(ctx context.Context, usuarioID int64, rolIDs, sucursalIDs []int64) error {
	return a.repo.ReemplazarAsignaciones(ctx, usuarioID, rolIDs, sucursalIDs)
}
func (a *AccesosWriteAdapter) CambiarActivoUsuario(ctx context.Context, id int64, activo bool, motivo string) (dtoresult.UsuarioAdmin, error) {
	return a.repo.SetActivoUsuario(ctx, id, activo, motivo)
}
func (a *AccesosWriteAdapter) RevocarSesionesDeUsuario(ctx context.Context, usuarioID int64, ahora time.Time) error {
	return a.repo.RevocarSesiones(ctx, usuarioID, ahora)
}
func (a *AccesosWriteAdapter) ContarAdministradoresEfectivos(ctx context.Context) (int, error) {
	return a.repo.ContarAdministradores(ctx)
}
