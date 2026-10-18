// Package adapter implementa los puertos de salida de lectura.
package adapter

import (
	"context"

	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/read/mapper"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
	appctx "service-restaurant/internal/shared/application/context"
)

// SesionReadAdapter implementa outbound.SesionReader.
type SesionReadAdapter struct{ repo *repository.SesionSQLReader }

func NewSesionReadAdapter(db database.DBTX) *SesionReadAdapter {
	return &SesionReadAdapter{repo: repository.NewSesionSQLReader(db)}
}

var _ outbound.SesionReader = (*SesionReadAdapter)(nil)

func (a *SesionReadAdapter) BuscarPorID(ctx context.Context, id int64) (outbound.SesionAutenticada, bool, error) {
	p, found, err := a.repo.FindByID(ctx, id)
	if err != nil || !found {
		return outbound.SesionAutenticada{}, false, err
	}
	return mapper.SesionAAplicacion(p), true, nil
}

func (a *SesionReadAdapter) PermisosEfectivos(ctx context.Context, usuarioID int64) ([]appctx.Grant, error) {
	gs, err := a.repo.Grants(ctx, usuarioID)
	if err != nil {
		return nil, err
	}
	return mapper.GrantsAAplicacion(gs), nil
}
