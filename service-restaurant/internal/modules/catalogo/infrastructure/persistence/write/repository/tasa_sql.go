package repository

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// TasaSQLWriter opera sobre sch_restaurante.tasas_impuesto. Actor y fechas de auditoría los
// completa el trigger de metadatos. La tasa viaja como texto y PostgreSQL la convierte a
// numeric(7,4); las fechas, como texto a date.
type TasaSQLWriter struct{ db database.DBTX }

func NewTasaSQLWriter(db database.DBTX) *TasaSQLWriter { return &TasaSQLWriter{db} }

// ImpuestoActivo bloquea el impuesto FOR SHARE (no puede darse de baja durante el alta).
func (r *TasaSQLWriter) ImpuestoActivo(ctx context.Context, impuestoID int64) (bool, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM sch_restaurante.impuestos WHERE id = $1 AND es_activo FOR SHARE`, impuestoID).Scan(&id)
	if missing, err := notFound(err); missing || err != nil {
		return false, err
	}
	return true, nil
}

func (r *TasaSQLWriter) Insert(ctx context.Context, n outbound.NuevaTasa) (dtoresult.TasaImpuesto, error) {
	t, err := columnas.ScanTasaImpuesto(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.tasas_impuesto (impuesto_id, tasa, vigente_desde, vigente_hasta, fundamento)
		VALUES ($1, $2::text::numeric, $3::text::date, $4::text::date, $5) RETURNING `+columnas.TasaImpuesto,
		n.ImpuestoID, n.Tasa, n.VigenteDesde, n.VigenteHasta, n.Fundamento))
	return t, Translate(err)
}

// GetForUpdate lee la tasa bloqueando su fila hasta el final de la transacción.
func (r *TasaSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.TasaImpuesto, bool, error) {
	t, err := columnas.ScanTasaImpuesto(r.db.QueryRowContext(ctx,
		`SELECT `+columnas.TasaImpuesto+` FROM sch_restaurante.tasas_impuesto WHERE id = $1 FOR UPDATE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.TasaImpuesto{}, false, err
	}
	return t, true, nil
}

// Update escribe el estado completo resultante (el caso de uso ya fusionó los cambios).
func (r *TasaSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosTasa) (dtoresult.TasaImpuesto, bool, error) {
	t, err := columnas.ScanTasaImpuesto(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.tasas_impuesto SET tasa = $2::text::numeric, vigente_desde = $3::text::date,
			vigente_hasta = $4::text::date, fundamento = $5
		WHERE id = $1 RETURNING `+columnas.TasaImpuesto, id, c.Tasa, c.VigenteDesde, c.VigenteHasta, c.Fundamento))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.TasaImpuesto{}, false, Translate(err)
	}
	return t, true, nil
}
