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

// CategoriaSQLReader lee categorías.
type CategoriaSQLReader struct{ db database.DBTX }

func NewCategoriaSQLReader(db database.DBTX) *CategoriaSQLReader { return &CategoriaSQLReader{db} }

// ListarCategorias ordena por id (cursor estable); el texto, ya con los comodines escapados,
// busca por nombre (contiene).
func (r *CategoriaSQLReader) ListarCategorias(ctx context.Context, f outbound.FiltroCategorias) ([]dtoresult.Categoria, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+columnas.Categoria+` FROM sch_restaurante.categorias
		WHERE id > $1 AND ($2::boolean IS NULL OR es_activo = $2)
		  AND ($3::text = '' OR nombre ILIKE '%' || $3 || '%' ESCAPE '\')
		ORDER BY id LIMIT $4`, f.DespuesDeID, activoSQL(f.Activo), f.Texto, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Categoria
	for rows.Next() {
		c, err := columnas.ScanCategoria(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CategoriaSQLReader) ObtenerCategoria(ctx context.Context, id int64) (dtoresult.Categoria, bool, error) {
	c, err := columnas.ScanCategoria(r.db.QueryRowContext(ctx, `SELECT `+columnas.Categoria+` FROM sch_restaurante.categorias WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Categoria{}, false, nil
	}
	return c, err == nil, err
}
