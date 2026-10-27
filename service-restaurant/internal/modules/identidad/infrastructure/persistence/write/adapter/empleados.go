package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// EmpleadoWriteAdapter implementa outbound.EmpleadoStore.
type EmpleadoWriteAdapter struct{ repo *repository.EmpleadoSQLWriter }

func NewEmpleadoWriteAdapter(db database.DBTX) *EmpleadoWriteAdapter {
	return &EmpleadoWriteAdapter{repo: repository.NewEmpleadoSQLWriter(db)}
}

var _ outbound.EmpleadoStore = (*EmpleadoWriteAdapter)(nil)

func (a *EmpleadoWriteAdapter) Crear(ctx context.Context, n outbound.NuevoEmpleado) (dtoresult.Empleado, error) {
	return a.repo.Insert(ctx, n)
}
func (a *EmpleadoWriteAdapter) Obtener(ctx context.Context, id int64) (dtoresult.Empleado, bool, error) {
	return a.repo.GetForUpdate(ctx, id)
}
func (a *EmpleadoWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosEmpleado) (dtoresult.Empleado, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *EmpleadoWriteAdapter) DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Empleado, bool, error) {
	return a.repo.SetBaja(ctx, id, motivo)
}
func (a *EmpleadoWriteAdapter) Restaurar(ctx context.Context, id int64) (dtoresult.Empleado, bool, error) {
	return a.repo.SetActivo(ctx, id)
}
func (a *EmpleadoWriteAdapter) Usuario(ctx context.Context, id int64) (outbound.UsuarioRef, bool, error) {
	return a.repo.GetUsuario(ctx, id)
}
func (a *EmpleadoWriteAdapter) UsuarioActivoDe(ctx context.Context, empleadoID int64) (bool, error) {
	return a.repo.UsuarioActivoDe(ctx, empleadoID)
}
