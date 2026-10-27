package repository

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// ProductoSQLReader lee productos.
type ProductoSQLReader struct{ db database.DBTX }

func NewProductoSQLReader(db database.DBTX) *ProductoSQLReader { return &ProductoSQLReader{db} }

// desdeProductos es el FROM común: productos con LEFT JOIN a la fila ACTIVA de productos_sucursal
// de la sede $1 (nil = ninguna). La subconsulta renombra sus columnas para no chocar con las de
// productos, así que columnas.Producto sigue valiendo sin calificar y no hay N+1: una sola
// consulta trae cada ítem con su estado en la sede.
const desdeProductos = `
	FROM sch_restaurante.productos
	LEFT JOIN (SELECT producto_id, id AS ps_id, disponible, precio_venta AS ps_precio, estacion_id AS ps_estacion
	           FROM sch_restaurante.productos_sucursal WHERE sucursal_id = $1::bigint AND es_activo) ps
	       ON ps.producto_id = productos.id`

func idSQL(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

// ListarProductos ordena por id (cursor estable). El texto, ya con los comodines escapados, busca
// por nombre (contiene) o por código (empieza por). Con sede, filtra opcionalmente por
// disponibilidad: true = fila activa habilitada; false = fila activa pausada (un producto sin
// fila no está "pausado": simplemente no se ofrece, y se distingue por en_sucursal = null).
func (r *ProductoSQLReader) ListarProductos(ctx context.Context, f outbound.FiltroProductos) ([]dtoresult.Producto, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+columnas.Producto+`, `+columnas.ProductoEnSede+desdeProductos+`
		WHERE id > $2 AND ($3::boolean IS NULL OR es_activo = $3)
		  AND ($4::bigint IS NULL OR categoria_id = $4)
		  AND ($5::text = '' OR nombre ILIKE '%' || $5 || '%' ESCAPE '\'
		       OR codigo ILIKE $5 || '%' ESCAPE '\')
		  AND ($6::boolean IS NULL OR (ps.ps_id IS NOT NULL AND ps.disponible = $6))
		ORDER BY id LIMIT $7`,
		idSQL(f.SucursalID), f.DespuesDeID, activoSQL(f.Activo), idSQL(f.CategoriaID), f.Texto, activoSQL(f.Disponible), f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Producto
	for rows.Next() {
		p, err := columnas.ScanProductoEnSede(rows)
		if err != nil {
			return nil, err
		}
		p.ConSede = f.SucursalID != nil
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ProductoSQLReader) ObtenerProducto(ctx context.Context, id int64, sucursalID *int64) (dtoresult.Producto, bool, error) {
	p, err := columnas.ScanProductoEnSede(r.db.QueryRowContext(ctx,
		`SELECT `+columnas.Producto+`, `+columnas.ProductoEnSede+desdeProductos+` WHERE id = $2`, idSQL(sucursalID), id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Producto{}, false, nil
	}
	p.ConSede = sucursalID != nil
	return p, err == nil, err
}
