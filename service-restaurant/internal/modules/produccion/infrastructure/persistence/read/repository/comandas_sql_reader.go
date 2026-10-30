// Package repository contiene el SQL de lectura del módulo produccion.
package repository

import (
	"context"
	"database/sql"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/infrastructure/persistence/internal/cargar"
	"service-restaurant/internal/platform/database"
)

// ComandaSQLReader lee la cola de preparación.
type ComandaSQLReader struct{ db database.DBTX }

func NewComandaSQLReader(db database.DBTX) *ComandaSQLReader { return &ComandaSQLReader{db} }

// ListarComandas devuelve las comandas de las sedes de f.SedeIDs ordenadas por id (cursor estable),
// cada una con todas sus líneas. Con Estado vacío, las que tienen alguna línea PENDIENTE, PREPARACION
// o LISTO; con Estado, las que tienen alguna línea en ese estado.
func (r *ComandaSQLReader) ListarComandas(ctx context.Context, f outbound.FiltroComandas) ([]dtoresult.Comanda, error) {
	var estacion sql.NullInt64
	if f.EstacionID != nil {
		estacion = sql.NullInt64{Int64: *f.EstacionID, Valid: true}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id FROM sch_restaurante.comandas c
		WHERE c.sucursal_id = ANY($1::bigint[]) AND c.id > $2
		  AND ($3::bigint IS NULL OR c.estacion_id = $3)
		  AND EXISTS (SELECT 1 FROM sch_restaurante.comanda_detalles cd
		               WHERE cd.comanda_id = c.id
		                 AND (cd.estado = $4::text
		                      OR ($4::text = '' AND cd.estado IN ('PENDIENTE','PREPARACION','LISTO'))))
		ORDER BY c.id LIMIT $5`, f.SedeIDs, f.DespuesDeID, estacion, f.Estado, f.Limite)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	if len(ids) == 0 {
		return nil, nil
	}
	return cargar.Comandas(ctx, r.db, ids)
}
