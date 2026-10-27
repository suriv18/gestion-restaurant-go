// Package adapter implementa los puertos de salida de escritura del directorio.
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// ClienteWriteAdapter implementa outbound.ClienteStore.
type ClienteWriteAdapter struct{ repo *repository.ClienteSQLWriter }

func NewClienteWriteAdapter(db database.DBTX) *ClienteWriteAdapter {
	return &ClienteWriteAdapter{repo: repository.NewClienteSQLWriter(db)}
}

var _ outbound.ClienteStore = (*ClienteWriteAdapter)(nil)

func (a *ClienteWriteAdapter) Crear(ctx context.Context, n outbound.NuevoCliente) (dtoresult.Cliente, error) {
	return a.repo.Insert(ctx, n)
}
func (a *ClienteWriteAdapter) Obtener(ctx context.Context, id int64) (dtoresult.Cliente, bool, error) {
	return a.repo.GetForUpdate(ctx, id)
}
func (a *ClienteWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosCliente) (dtoresult.Cliente, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *ClienteWriteAdapter) DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Cliente, bool, error) {
	return a.repo.SetBaja(ctx, id, motivo)
}
func (a *ClienteWriteAdapter) Restaurar(ctx context.Context, id int64) (dtoresult.Cliente, bool, error) {
	return a.repo.SetActivo(ctx, id)
}

// ProveedorWriteAdapter implementa outbound.ProveedorStore.
type ProveedorWriteAdapter struct {
	repo *repository.ProveedorSQLWriter
}

func NewProveedorWriteAdapter(db database.DBTX) *ProveedorWriteAdapter {
	return &ProveedorWriteAdapter{repo: repository.NewProveedorSQLWriter(db)}
}

var _ outbound.ProveedorStore = (*ProveedorWriteAdapter)(nil)

func (a *ProveedorWriteAdapter) Crear(ctx context.Context, n outbound.NuevoProveedor) (dtoresult.Proveedor, error) {
	return a.repo.Insert(ctx, n)
}
func (a *ProveedorWriteAdapter) Obtener(ctx context.Context, id int64) (dtoresult.Proveedor, bool, error) {
	return a.repo.GetForUpdate(ctx, id)
}
func (a *ProveedorWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosProveedor) (dtoresult.Proveedor, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *ProveedorWriteAdapter) DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Proveedor, bool, error) {
	return a.repo.SetBaja(ctx, id, motivo)
}
func (a *ProveedorWriteAdapter) Restaurar(ctx context.Context, id int64) (dtoresult.Proveedor, bool, error) {
	return a.repo.SetActivo(ctx, id)
}
