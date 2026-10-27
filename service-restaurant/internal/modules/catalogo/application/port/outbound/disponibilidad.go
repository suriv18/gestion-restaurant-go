package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
)

// CambiosDisponibilidad es lo que escribe el upsert: Disponible siempre; Precio y Estacion solo
// si están presentes (ausente = conservar lo guardado; Valor nil = NULL).
type CambiosDisponibilidad struct {
	Disponible bool
	Precio     CampoOpcional
	Estacion   CampoID
}

// DisponibilidadStore escribe productos_sucursal dentro de la transacción del comando.
type DisponibilidadStore interface {
	// BloquearProducto lee el producto con FOR SHARE: una baja o edición concurrente espera, y
	// no se habilita un plato mientras se da de baja. found=false si no existe.
	BloquearProducto(ctx context.Context, id int64) (dtoresult.Producto, bool, error)
	// Guardar hace el upsert por (producto, sucursal) y devuelve siempre la fila. Reactiva la
	// fila si estaba inactiva. No escribe nada en productos.
	Guardar(ctx context.Context, productoID, sucursalID int64, c CambiosDisponibilidad) (dtoresult.ProductoSucursal, error)
}
