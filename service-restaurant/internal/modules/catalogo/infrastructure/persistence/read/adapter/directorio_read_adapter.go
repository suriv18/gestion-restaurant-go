// Package adapter implementa los puertos de salida de lectura del directorio.
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// ClienteReadAdapter implementa outbound.ClienteReader.
type ClienteReadAdapter struct{ repo *repository.ClienteSQLReader }

func NewClienteReadAdapter(db database.DBTX) *ClienteReadAdapter {
	return &ClienteReadAdapter{repo: repository.NewClienteSQLReader(db)}
}

var _ outbound.ClienteReader = (*ClienteReadAdapter)(nil)

func (a *ClienteReadAdapter) ListarClientes(ctx context.Context, f outbound.FiltroClientes) ([]dtoresult.Cliente, error) {
	return a.repo.ListarClientes(ctx, f)
}
func (a *ClienteReadAdapter) ObtenerCliente(ctx context.Context, id int64) (dtoresult.Cliente, bool, error) {
	return a.repo.ObtenerCliente(ctx, id)
}

// ProveedorReadAdapter implementa outbound.ProveedorReader.
type ProveedorReadAdapter struct {
	repo *repository.ProveedorSQLReader
}

func NewProveedorReadAdapter(db database.DBTX) *ProveedorReadAdapter {
	return &ProveedorReadAdapter{repo: repository.NewProveedorSQLReader(db)}
}

var _ outbound.ProveedorReader = (*ProveedorReadAdapter)(nil)

func (a *ProveedorReadAdapter) ListarProveedores(ctx context.Context, f outbound.FiltroProveedores) ([]dtoresult.Proveedor, error) {
	return a.repo.ListarProveedores(ctx, f)
}
func (a *ProveedorReadAdapter) ObtenerProveedor(ctx context.Context, id int64) (dtoresult.Proveedor, bool, error) {
	return a.repo.ObtenerProveedor(ctx, id)
}
