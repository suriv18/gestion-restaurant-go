package repository

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// EstacionSQLWriter opera sobre sch_restaurante.estaciones_preparacion. Actor y fechas de
// auditoría los completa el trigger de metadatos desde app.usuario_id; la sede activa la
// exige el trigger de referencias activas.
type EstacionSQLWriter struct{ db database.DBTX }

func NewEstacionSQLWriter(db database.DBTX) *EstacionSQLWriter { return &EstacionSQLWriter{db} }

func (r *EstacionSQLWriter) Insert(ctx context.Context, n outbound.NuevaEstacion) (dtoresult.Estacion, error) {
	e, err := columnas.ScanEstacion(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.estaciones_preparacion (sucursal_id, codigo, nombre)
		VALUES ($1, $2, $3) RETURNING `+columnas.Estacion, n.SucursalID, n.Codigo, n.Nombre))
	return e, Translate(err)
}

// GetForUpdate lee la estación bloqueando su fila hasta el final de la transacción.
func (r *EstacionSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Estacion, bool, error) {
	e, err := columnas.ScanEstacion(r.db.QueryRowContext(ctx,
		`SELECT `+columnas.Estacion+` FROM sch_restaurante.estaciones_preparacion WHERE id = $1 FOR UPDATE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Estacion{}, false, err
	}
	return e, true, nil
}

func (r *EstacionSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosEstacion) (dtoresult.Estacion, bool, error) {
	e, err := columnas.ScanEstacion(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.estaciones_preparacion SET
			codigo = COALESCE($2, codigo),
			nombre = COALESCE($3, nombre)
		WHERE id = $1 RETURNING `+columnas.Estacion, id, c.Codigo, c.Nombre))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Estacion{}, false, Translate(err)
	}
	return e, true, nil
}

// SetActivo activa o desactiva la fila; la baja de una estación no lleva motivo.
func (r *EstacionSQLWriter) SetActivo(ctx context.Context, id int64, activo bool) (dtoresult.Estacion, bool, error) {
	e, err := columnas.ScanEstacion(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.estaciones_preparacion SET es_activo = $2 WHERE id = $1 RETURNING `+columnas.Estacion, id, activo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Estacion{}, false, Translate(err)
	}
	return e, true, nil
}

// EnUso indica si alguna fila activa de productos_sucursal usa la estación.
func (r *EstacionSQLWriter) EnUso(ctx context.Context, id int64) (bool, error) {
	var existe bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM sch_restaurante.productos_sucursal WHERE estacion_id = $1 AND es_activo)`, id).Scan(&existe)
	return existe, err
}
