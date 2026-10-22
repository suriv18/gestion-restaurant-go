package repository

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// Restricción única que tiene un rechazo propio.
const restriccionNombreZona = "zonas_sucursal_id_nombre_key"

// traducirZona distingue la colisión del nombre de zona por el nombre de la restricción.
func traducirZona(err error) error {
	if restriccionUnica(err) == restriccionNombreZona {
		return outbound.ErrNombreZonaDuplicado
	}
	return Translate(err)
}

// ZonaSQLWriter opera sobre sch_restaurante.zonas.
type ZonaSQLWriter struct{ db database.DBTX }

func NewZonaSQLWriter(db database.DBTX) *ZonaSQLWriter { return &ZonaSQLWriter{db} }

func (r *ZonaSQLWriter) Insert(ctx context.Context, n outbound.NuevaZona) (dtoresult.Zona, error) {
	z, err := columnas.ScanZona(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.zonas (sucursal_id, nombre, orden_visualizacion)
		VALUES ($1, $2, $3) RETURNING `+columnas.Zona, n.SucursalID, n.Nombre, n.Orden))
	return z, traducirZona(err)
}

// GetForUpdate lee la zona bloqueando su fila hasta el final de la transacción: serializa la
// baja de la zona con el alta de mesas en ella.
func (r *ZonaSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Zona, bool, error) {
	z, err := columnas.ScanZona(r.db.QueryRowContext(ctx,
		`SELECT `+columnas.Zona+` FROM sch_restaurante.zonas WHERE id = $1 FOR UPDATE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Zona{}, false, err
	}
	return z, true, nil
}

func (r *ZonaSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosZona) (dtoresult.Zona, bool, error) {
	z, err := columnas.ScanZona(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.zonas SET
			nombre = COALESCE($2, nombre),
			orden_visualizacion = COALESCE($3, orden_visualizacion)
		WHERE id = $1 RETURNING `+columnas.Zona, id, c.Nombre, c.Orden))
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
	z, err := columnas.ScanZona(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.zonas SET es_activo = $2 WHERE id = $1 RETURNING `+columnas.Zona, id, activo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Zona{}, false, traducirZona(err)
	}
	return z, true, nil
}
