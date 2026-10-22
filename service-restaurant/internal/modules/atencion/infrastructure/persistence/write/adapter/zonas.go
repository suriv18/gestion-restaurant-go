// Package adapter implementa los puertos de salida de escritura del módulo atencion.
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// ZonaWriteAdapter implementa outbound.ZonaStore.
type ZonaWriteAdapter struct{ repo *repository.ZonaSQLWriter }

func NewZonaWriteAdapter(db database.DBTX) *ZonaWriteAdapter {
	return &ZonaWriteAdapter{repo: repository.NewZonaSQLWriter(db)}
}

var _ outbound.ZonaStore = (*ZonaWriteAdapter)(nil)

func (a *ZonaWriteAdapter) Crear(ctx context.Context, n outbound.NuevaZona) (dtoresult.Zona, error) {
	return a.repo.Insert(ctx, n)
}
func (a *ZonaWriteAdapter) Obtener(ctx context.Context, id int64) (dtoresult.Zona, bool, error) {
	return a.repo.GetForUpdate(ctx, id)
}
func (a *ZonaWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosZona) (dtoresult.Zona, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *ZonaWriteAdapter) ContarMesasActivas(ctx context.Context, zonaID int64) (int, error) {
	return a.repo.CountMesasActivas(ctx, zonaID)
}
func (a *ZonaWriteAdapter) CambiarActivo(ctx context.Context, id int64, activo bool) (dtoresult.Zona, bool, error) {
	return a.repo.SetActivo(ctx, id, activo)
}
