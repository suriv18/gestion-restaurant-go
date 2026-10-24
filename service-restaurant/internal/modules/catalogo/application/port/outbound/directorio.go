package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
)

// CampoOpcional es un cambio de un campo anulable: si Presente, se escribe Valor (nil = NULL).
type CampoOpcional struct {
	Presente bool
	Valor    *string
}

// FiltroClientes acota el listado por estado y texto (ya sin comodines de LIKE).
type FiltroClientes struct {
	Activo      *bool
	Texto       string
	DespuesDeID int64
	Limite      int
}

// ClienteReader lee clientes sin bloquear filas.
type ClienteReader interface {
	ListarClientes(ctx context.Context, f FiltroClientes) ([]dtoresult.Cliente, error)
	ObtenerCliente(ctx context.Context, id int64) (dtoresult.Cliente, bool, error)
}

type NuevoCliente struct {
	TipoDocumento, NumeroDocumento *string
	Nombre                         string
	Telefono, Email, Direccion     *string
}

// CambiosCliente: solo los campos presentes se modifican.
type CambiosCliente struct {
	Nombre                         *string
	TipoDocumento, NumeroDocumento CampoOpcional
	Telefono, Email, Direccion     CampoOpcional
}

// ClienteStore escribe clientes dentro de la transacción del comando.
type ClienteStore interface {
	Crear(ctx context.Context, n NuevoCliente) (dtoresult.Cliente, error)
	// Obtener lee el cliente bloqueando su fila (FOR UPDATE); found=false si no existe.
	Obtener(ctx context.Context, id int64) (dtoresult.Cliente, bool, error)
	Actualizar(ctx context.Context, id int64, c CambiosCliente) (dtoresult.Cliente, bool, error)
	DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Cliente, bool, error)
	Restaurar(ctx context.Context, id int64) (dtoresult.Cliente, bool, error)
}

type FiltroProveedores struct {
	Activo      *bool
	Texto       string
	DespuesDeID int64
	Limite      int
}

type ProveedorReader interface {
	ListarProveedores(ctx context.Context, f FiltroProveedores) ([]dtoresult.Proveedor, error)
	ObtenerProveedor(ctx context.Context, id int64) (dtoresult.Proveedor, bool, error)
}

type NuevoProveedor struct {
	TipoDocumento, NumeroDocumento, RazonSocial string
	ContactoNombre, Telefono, Email, Direccion  *string
}

type CambiosProveedor struct {
	TipoDocumento, NumeroDocumento, RazonSocial *string
	ContactoNombre, Telefono, Email, Direccion  CampoOpcional
}

type ProveedorStore interface {
	Crear(ctx context.Context, n NuevoProveedor) (dtoresult.Proveedor, error)
	Obtener(ctx context.Context, id int64) (dtoresult.Proveedor, bool, error)
	Actualizar(ctx context.Context, id int64, c CambiosProveedor) (dtoresult.Proveedor, bool, error)
	DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Proveedor, bool, error)
	Restaurar(ctx context.Context, id int64) (dtoresult.Proveedor, bool, error)
}
