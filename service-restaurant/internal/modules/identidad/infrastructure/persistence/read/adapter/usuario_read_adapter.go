package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// UsuarioReadAdapter implementa outbound.UsuarioReader.
type UsuarioReadAdapter struct{ repo *repository.UsuarioSQLReader }

func NewUsuarioReadAdapter(db database.DBTX) *UsuarioReadAdapter {
	return &UsuarioReadAdapter{repo: repository.NewUsuarioSQLReader(db)}
}

var _ outbound.UsuarioReader = (*UsuarioReadAdapter)(nil)

func (a *UsuarioReadAdapter) ListarUsuarios(ctx context.Context, despuesDeID int64, limite int) ([]dtoresult.UsuarioConAccesos, error) {
	return a.repo.ListUsuarios(ctx, despuesDeID, limite)
}

func (a *UsuarioReadAdapter) SucursalesActivasPorID(ctx context.Context, ids []int64) ([]dtoresult.SucursalResumen, error) {
	return a.repo.ListSucursalesActivas(ctx, ids)
}
