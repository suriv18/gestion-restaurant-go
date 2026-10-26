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

// ListarProductos ordena por id (cursor estable). El texto, ya con los comodines escapados, busca
// por nombre (contiene) o por código (empieza por).
func (r *ProductoSQLReader) ListarProductos(ctx context.Context, f outbound.FiltroProductos) ([]dtoresult.Producto, error) {
	var categoria sql.NullInt64
	if f.CategoriaID != nil {
		categoria = sql.NullInt64{Int64: *f.CategoriaID, Valid: true}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+columnas.Producto+` FROM sch_restaurante.productos
		WHERE id > $1 AND ($2::boolean IS NULL OR es_activo = $2)
		  AND ($3::bigint IS NULL OR categoria_id = $3)
		  AND ($4::text = '' OR nombre ILIKE '%' || $4 || '%' ESCAPE '\'
		       OR codigo ILIKE $4 || '%' ESCAPE '\')
		ORDER BY id LIMIT $5`, f.DespuesDeID, activoSQL(f.Activo), categoria, f.Texto, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Producto
	for rows.Next() {
		p, err := columnas.ScanProducto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ProductoSQLReader) ObtenerProducto(ctx context.Context, id int64) (dtoresult.Producto, bool, error) {
	p, err := columnas.ScanProducto(r.db.QueryRowContext(ctx, `SELECT `+columnas.Producto+` FROM sch_restaurante.productos WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Producto{}, false, nil
	}
	return p, err == nil, err
}
