package repository

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// CategoriaSQLWriter opera sobre sch_restaurante.categorias. Actor y fechas de auditoría los
// completa el trigger de metadatos desde app.usuario_id.
type CategoriaSQLWriter struct{ db database.DBTX }

func NewCategoriaSQLWriter(db database.DBTX) *CategoriaSQLWriter { return &CategoriaSQLWriter{db} }

func (r *CategoriaSQLWriter) Insert(ctx context.Context, n outbound.NuevaCategoria) (dtoresult.Categoria, error) {
	c, err := columnas.ScanCategoria(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.categorias (nombre, descripcion, orden_visualizacion)
		VALUES ($1, $2, $3) RETURNING `+columnas.Categoria, n.Nombre, n.Descripcion, n.Orden))
	return c, Translate(err)
}

// GetForUpdate lee la categoría bloqueando su fila hasta el final de la transacción.
func (r *CategoriaSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Categoria, bool, error) {
	c, err := columnas.ScanCategoria(r.db.QueryRowContext(ctx,
		`SELECT `+columnas.Categoria+` FROM sch_restaurante.categorias WHERE id = $1 FOR UPDATE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Categoria{}, false, err
	}
	return c, true, nil
}

func (r *CategoriaSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosCategoria) (dtoresult.Categoria, bool, error) {
	cat, err := columnas.ScanCategoria(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.categorias SET
			nombre = COALESCE($2, nombre),
			descripcion = CASE WHEN $3 THEN $4::text ELSE descripcion END,
			orden_visualizacion = COALESCE($5::integer, orden_visualizacion)
		WHERE id = $1 RETURNING `+columnas.Categoria, id, c.Nombre, c.Descripcion.Presente, c.Descripcion.Valor, c.Orden))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Categoria{}, false, Translate(err)
	}
	return cat, true, nil
}

// SetActivo activa o desactiva la fila; la baja de una categoría no lleva motivo.
func (r *CategoriaSQLWriter) SetActivo(ctx context.Context, id int64, activo bool) (dtoresult.Categoria, bool, error) {
	c, err := columnas.ScanCategoria(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.categorias SET es_activo = $2 WHERE id = $1 RETURNING `+columnas.Categoria, id, activo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Categoria{}, false, Translate(err)
	}
	return c, true, nil
}

// TieneProductosActivos indica si algún producto activo pertenece a la categoría.
func (r *CategoriaSQLWriter) TieneProductosActivos(ctx context.Context, id int64) (bool, error) {
	var existe bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM sch_restaurante.productos WHERE categoria_id = $1 AND es_activo)`, id).Scan(&existe)
	return existe, err
}
