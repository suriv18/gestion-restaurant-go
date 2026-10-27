package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// EmpleadoReadAdapter implementa outbound.EmpleadoReader.
type EmpleadoReadAdapter struct{ repo *repository.EmpleadoSQLReader }

func NewEmpleadoReadAdapter(db database.DBTX) *EmpleadoReadAdapter {
	return &EmpleadoReadAdapter{repo: repository.NewEmpleadoSQLReader(db)}
}

var _ outbound.EmpleadoReader = (*EmpleadoReadAdapter)(nil)

func (a *EmpleadoReadAdapter) ListarEmpleados(ctx context.Context, f outbound.FiltroEmpleados) ([]dtoresult.Empleado, error) {
	return a.repo.ListEmpleados(ctx, f)
}
func (a *EmpleadoReadAdapter) ObtenerEmpleado(ctx context.Context, id int64) (dtoresult.Empleado, bool, error) {
	return a.repo.GetEmpleado(ctx, id)
}
