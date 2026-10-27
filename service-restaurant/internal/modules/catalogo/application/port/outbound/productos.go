package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
)

// FiltroProductos acota el listado (texto ya sin comodines de LIKE). Activo nil = sin filtro de estado.
type FiltroProductos struct {
	Activo      *bool
	CategoriaID *int64
	Texto       string
	// SucursalID (opcional) añade el estado en esa sede; Disponible (solo con SucursalID) filtra por
	// él: true = habilitados, false = pausados (fila activa con disponible=false).
	SucursalID  *int64
	Disponible  *bool
	DespuesDeID int64
	Limite      int
}

// ProductoReader lee productos sin bloquear filas.
type ProductoReader interface {
	ListarProductos(ctx context.Context, f FiltroProductos) ([]dtoresult.Producto, error)
	ObtenerProducto(ctx context.Context, id int64, sucursalID *int64) (dtoresult.Producto, bool, error)
}

// NuevoProducto trae los valores ya validados y normalizados (decimales con su escala).
type NuevoProducto struct {
	CategoriaID int64
	Codigo      string
	Nombre      string
	Descripcion *string
	Tipo        string
	ArticuloID  *int64
	Factor      string
	Precio      string
	Afectacion  string
	ImpuestoID  *int64
	ImagenURL   *string
}

// CampoID es un cambio de una referencia anulable: si Presente, se escribe Valor (nil = NULL).
type CampoID struct {
	Presente bool
	Valor    *int64
}

// CambiosProducto: solo los campos presentes se modifican.
type CambiosProducto struct {
	CategoriaID *int64
	Codigo      *string
	Nombre      *string
	Descripcion CampoOpcional
	Tipo        *string
	ArticuloID  CampoID
	Factor      *string
	Precio      *string
	Afectacion  *string
	ImpuestoID  CampoID
	ImagenURL   CampoOpcional
}

// ProductoStore escribe productos dentro de la transacción del comando.
type ProductoStore interface {
	Crear(ctx context.Context, n NuevoProducto) (dtoresult.Producto, error)
	// Obtener lee el producto bloqueando su fila (FOR UPDATE); found=false si no existe.
	Obtener(ctx context.Context, id int64) (dtoresult.Producto, bool, error)
	Actualizar(ctx context.Context, id int64, c CambiosProducto) (dtoresult.Producto, bool, error)
	DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Producto, bool, error)
	Restaurar(ctx context.Context, id int64) (dtoresult.Producto, bool, error)
}
