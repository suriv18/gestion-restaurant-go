package repository

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// ProductoSQLWriter opera sobre sch_restaurante.productos. Actor y fechas de auditoría y de baja
// los completa el trigger de metadatos desde app.usuario_id.
type ProductoSQLWriter struct{ db database.DBTX }

func NewProductoSQLWriter(db database.DBTX) *ProductoSQLWriter { return &ProductoSQLWriter{db} }

func (r *ProductoSQLWriter) Insert(ctx context.Context, n outbound.NuevoProducto) (dtoresult.Producto, error) {
	p, err := columnas.ScanProducto(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.productos
			(categoria_id, codigo, nombre, descripcion, tipo, articulo_id, factor_consumo, precio_venta, afectacion_igv, impuesto_id, imagen_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7::numeric, $8::numeric, $9, $10, $11) RETURNING `+columnas.Producto,
		n.CategoriaID, n.Codigo, n.Nombre, n.Descripcion, n.Tipo, n.ArticuloID, n.Factor, n.Precio, n.Afectacion, n.ImpuestoID, n.ImagenURL))
	return p, Translate(err)
}

// GetForUpdate lee el producto bloqueando su fila hasta el final de la transacción.
func (r *ProductoSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Producto, bool, error) {
	p, err := columnas.ScanProducto(r.db.QueryRowContext(ctx,
		`SELECT `+columnas.Producto+` FROM sch_restaurante.productos WHERE id = $1 FOR UPDATE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Producto{}, false, err
	}
	return p, true, nil
}

func (r *ProductoSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosProducto) (dtoresult.Producto, bool, error) {
	p, err := columnas.ScanProducto(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.productos SET
			categoria_id = COALESCE($2::bigint, categoria_id),
			codigo = COALESCE($3::text, codigo),
			nombre = COALESCE($4::text, nombre),
			descripcion = CASE WHEN $5 THEN $6::text ELSE descripcion END,
			tipo = COALESCE($7::text, tipo),
			articulo_id = CASE WHEN $8 THEN $9::bigint ELSE articulo_id END,
			factor_consumo = COALESCE($10::numeric, factor_consumo),
			precio_venta = COALESCE($11::numeric, precio_venta),
			afectacion_igv = COALESCE($12::text, afectacion_igv),
			impuesto_id = CASE WHEN $13 THEN $14::bigint ELSE impuesto_id END,
			imagen_url = CASE WHEN $15 THEN $16::text ELSE imagen_url END
		WHERE id = $1 RETURNING `+columnas.Producto, id,
		c.CategoriaID, c.Codigo, c.Nombre, c.Descripcion.Presente, c.Descripcion.Valor, c.Tipo,
		c.ArticuloID.Presente, c.ArticuloID.Valor, c.Factor, c.Precio, c.Afectacion,
		c.ImpuestoID.Presente, c.ImpuestoID.Valor, c.ImagenURL.Presente, c.ImagenURL.Valor))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Producto{}, false, Translate(err)
	}
	return p, true, nil
}

// SetBaja desactiva el producto con su motivo; el trigger completa quién y cuándo.
func (r *ProductoSQLWriter) SetBaja(ctx context.Context, id int64, motivo string) (dtoresult.Producto, bool, error) {
	p, err := columnas.ScanProducto(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.productos SET es_activo = false, motivo_eliminacion = $2
		WHERE id = $1 RETURNING `+columnas.Producto, id, motivo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Producto{}, false, Translate(err)
	}
	return p, true, nil
}

// SetActivo reactiva la misma fila (conserva el ID); el trigger limpia los marcadores de baja y
// el de referencias activas exige que categoría, artículo e impuesto sigan activos.
func (r *ProductoSQLWriter) SetActivo(ctx context.Context, id int64) (dtoresult.Producto, bool, error) {
	p, err := columnas.ScanProducto(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.productos SET es_activo = true WHERE id = $1 RETURNING `+columnas.Producto, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Producto{}, false, Translate(err)
	}
	return p, true, nil
}
