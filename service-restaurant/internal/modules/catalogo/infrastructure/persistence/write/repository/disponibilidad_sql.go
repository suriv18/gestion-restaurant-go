package repository

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// DisponibilidadSQLWriter opera sobre sch_restaurante.productos_sucursal. Actor y fechas de
// auditoría los completa el trigger de metadatos desde app.usuario_id.
type DisponibilidadSQLWriter struct{ db database.DBTX }

func NewDisponibilidadSQLWriter(db database.DBTX) *DisponibilidadSQLWriter {
	return &DisponibilidadSQLWriter{db}
}

// LockProducto lee el producto con FOR SHARE: bloquea una baja o edición concurrente hasta el
// final de la transacción sin impedir que otras sedes habiliten el mismo producto.
func (r *DisponibilidadSQLWriter) LockProducto(ctx context.Context, id int64) (dtoresult.Producto, bool, error) {
	p, err := columnas.ScanProducto(r.db.QueryRowContext(ctx,
		`SELECT `+columnas.Producto+` FROM sch_restaurante.productos WHERE id = $1 FOR SHARE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Producto{}, false, err
	}
	return p, true, nil
}

// Upsert inserta la fila o, si ya existe para (producto, sucursal), actualiza la disponibilidad y
// solo los campos presentes (precio, estación); además la reactiva (es_activo = true). En el
// camino de inserción un campo ausente queda NULL (precio general / sin estación).
func (r *DisponibilidadSQLWriter) Upsert(ctx context.Context, productoID, sucursalID int64, c outbound.CambiosDisponibilidad) (dtoresult.ProductoSucursal, error) {
	ps, err := columnas.ScanProductoSucursal(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.productos_sucursal AS ps (producto_id, sucursal_id, disponible, precio_venta, estacion_id)
		VALUES ($1, $2, $3, $4::numeric, $5::bigint)
		ON CONFLICT (producto_id, sucursal_id) DO UPDATE SET
			disponible = EXCLUDED.disponible,
			precio_venta = CASE WHEN $6::boolean THEN EXCLUDED.precio_venta ELSE ps.precio_venta END,
			estacion_id = CASE WHEN $7::boolean THEN EXCLUDED.estacion_id ELSE ps.estacion_id END,
			es_activo = true
		RETURNING `+columnas.ProductoSucursal,
		productoID, sucursalID, c.Disponible, c.Precio.Valor, c.Estacion.Valor, c.Precio.Presente, c.Estacion.Presente))
	return ps, Translate(err)
}
