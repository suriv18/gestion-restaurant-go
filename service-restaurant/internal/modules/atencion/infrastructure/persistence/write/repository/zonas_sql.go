package repository

import (
	"context"
	"errors"
	"strings"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/platform/database"
)

const zonaCols = `id, sucursal_id, nombre, orden_visualizacion, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`

func scanZona(s scanner) (dtoresult.Zona, error) {
	var z dtoresult.Zona
	err := s.Scan(&z.ID, &z.SucursalID, &z.Nombre, &z.OrdenVisualizacion, &z.EsActivo,
		&z.CreadoEn, &z.CreadoPor, &z.ActualizadoEn, &z.ActualizadoPor)
	return z, err
}

// traducirZona distingue la colisión del nombre de zona (zonas_sucursal_id_nombre_key).
func traducirZona(err error) error {
	err = Translate(err)
	if errors.Is(err, outbound.ErrDuplicado) && strings.Contains(err.Error(), "zonas_sucursal_id_nombre_key") {
		return outbound.ErrNombreZonaDuplicado
	}
	return err
}

// ZonaSQLWriter opera sobre sch_restaurante.zonas.
type ZonaSQLWriter struct{ db database.DBTX }

func NewZonaSQLWriter(db database.DBTX) *ZonaSQLWriter { return &ZonaSQLWriter{db} }

func (r *ZonaSQLWriter) Insert(ctx context.Context, n outbound.NuevaZona) (dtoresult.Zona, error) {
	z, err := scanZona(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.zonas (sucursal_id, nombre, orden_visualizacion)
		VALUES ($1, $2, $3) RETURNING `+zonaCols, n.SucursalID, n.Nombre, n.Orden))
	return z, traducirZona(err)
}

// GetForUpdate lee la zona bloqueando su fila hasta el final de la transacción: serializa la
// baja de la zona con el alta de mesas en ella.
func (r *ZonaSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Zona, bool, error) {
	z, err := scanZona(r.db.QueryRowContext(ctx,
		`SELECT `+zonaCols+` FROM sch_restaurante.zonas WHERE id = $1 FOR UPDATE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Zona{}, false, err
	}
	return z, true, nil
}

func (r *ZonaSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosZona) (dtoresult.Zona, bool, error) {
	z, err := scanZona(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.zonas SET
			nombre = COALESCE($2, nombre),
			orden_visualizacion = COALESCE($3, orden_visualizacion)
		WHERE id = $1 RETURNING `+zonaCols, id, c.Nombre, c.Orden))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Zona{}, false, traducirZona(err)
	}
	return z, true, nil
}

func (r *ZonaSQLWriter) CountMesasActivas(ctx context.Context, zonaID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT count(*) FROM sch_restaurante.mesas WHERE zona_id = $1 AND es_activo`, zonaID).Scan(&n)
	return n, err
}

// SetActivo desactiva o reactiva la misma fila (conserva el ID).
func (r *ZonaSQLWriter) SetActivo(ctx context.Context, id int64, activo bool) (dtoresult.Zona, bool, error) {
	z, err := scanZona(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.zonas SET es_activo = $2 WHERE id = $1 RETURNING `+zonaCols, id, activo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Zona{}, false, traducirZona(err)
	}
	return z, true, nil
}
