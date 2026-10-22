package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/platform/database"
)

// Las posiciones son numeric(10,2): se leen como texto y se escriben vía ::text::numeric para
// no perder exactitud ni depender de la codificación del driver.
const mesaCols = `id, sucursal_id, zona_id, numero_mesa, capacidad, habilitada, posicion_x::text, posicion_y::text,
	es_activo, motivo_eliminacion, eliminado_en, eliminado_por, creado_en, creado_por, actualizado_en, actualizado_por`

func textoNulo(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func scanMesa(s scanner) (dtoresult.Mesa, error) {
	var m dtoresult.Mesa
	var px, py, motivo sql.NullString
	var elimEn sql.NullTime
	var elimPor sql.NullInt64
	err := s.Scan(&m.ID, &m.SucursalID, &m.ZonaID, &m.NumeroMesa, &m.Capacidad, &m.Habilitada, &px, &py,
		&m.EsActivo, &motivo, &elimEn, &elimPor, &m.CreadoEn, &m.CreadoPor, &m.ActualizadoEn, &m.ActualizadoPor)
	m.PosicionX, m.PosicionY, m.MotivoEliminacion = textoNulo(px), textoNulo(py), textoNulo(motivo)
	if elimEn.Valid {
		t := elimEn.Time
		m.EliminadoEn = &t
	}
	if elimPor.Valid {
		v := elimPor.Int64
		m.EliminadoPor = &v
	}
	return m, err
}

// traducirMesa distingue la colisión del número de mesa (mesas_sucursal_id_numero_mesa_key).
func traducirMesa(err error) error {
	err = Translate(err)
	if errors.Is(err, outbound.ErrDuplicado) && strings.Contains(err.Error(), "mesas_sucursal_id_numero_mesa_key") {
		return outbound.ErrNumeroMesaDuplicado
	}
	return err
}

// MesaSQLWriter opera sobre sch_restaurante.mesas.
type MesaSQLWriter struct{ db database.DBTX }

func NewMesaSQLWriter(db database.DBTX) *MesaSQLWriter { return &MesaSQLWriter{db} }

func (r *MesaSQLWriter) Insert(ctx context.Context, n outbound.NuevaMesa) (dtoresult.Mesa, error) {
	m, err := scanMesa(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.mesas (sucursal_id, zona_id, numero_mesa, capacidad, habilitada, posicion_x, posicion_y)
		VALUES ($1, $2, $3, $4, $5, $6::text::numeric, $7::text::numeric) RETURNING `+mesaCols,
		n.SucursalID, n.ZonaID, n.NumeroMesa, n.Capacidad, n.Habilitada, n.PosicionX, n.PosicionY))
	return m, traducirMesa(err)
}

// GetForUpdate lee la mesa bloqueando su fila hasta el final de la transacción.
func (r *MesaSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	m, err := scanMesa(r.db.QueryRowContext(ctx,
		`SELECT `+mesaCols+` FROM sch_restaurante.mesas WHERE id = $1 FOR UPDATE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Mesa{}, false, err
	}
	return m, true, nil
}

// GetZona lee la zona en modo compartido: no puede desactivarse hasta que termine el comando.
func (r *MesaSQLWriter) GetZona(ctx context.Context, zonaID int64) (outbound.ZonaRef, bool, error) {
	var z outbound.ZonaRef
	err := r.db.QueryRowContext(ctx,
		`SELECT sucursal_id, es_activo FROM sch_restaurante.zonas WHERE id = $1 FOR SHARE`, zonaID).Scan(&z.SucursalID, &z.EsActiva)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.ZonaRef{}, false, err
	}
	return z, true, nil
}

func (r *MesaSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosMesa) (dtoresult.Mesa, bool, error) {
	m, err := scanMesa(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.mesas SET
			zona_id = COALESCE($2, zona_id),
			numero_mesa = COALESCE($3, numero_mesa),
			capacidad = COALESCE($4, capacidad),
			habilitada = COALESCE($5, habilitada),
			posicion_x = CASE WHEN $6 THEN $7::text::numeric ELSE posicion_x END,
			posicion_y = CASE WHEN $8 THEN $9::text::numeric ELSE posicion_y END
		WHERE id = $1 RETURNING `+mesaCols, id, c.ZonaID, c.NumeroMesa, c.Capacidad, c.Habilitada,
		c.PosicionX.Presente, c.PosicionX.Valor, c.PosicionY.Presente, c.PosicionY.Valor))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Mesa{}, false, traducirMesa(err)
	}
	return m, true, nil
}

// SetBaja desactiva la mesa; el trigger de metadatos completa actor y fecha de la baja.
func (r *MesaSQLWriter) SetBaja(ctx context.Context, id int64, motivo string) (dtoresult.Mesa, bool, error) {
	m, err := scanMesa(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.mesas SET es_activo = false, motivo_eliminacion = $2
		WHERE id = $1 RETURNING `+mesaCols, id, motivo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Mesa{}, false, traducirMesa(err)
	}
	return m, true, nil
}

// SetActiva reactiva la misma fila (conserva el ID); el trigger limpia los marcadores de baja.
func (r *MesaSQLWriter) SetActiva(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	m, err := scanMesa(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.mesas SET es_activo = true WHERE id = $1 RETURNING `+mesaCols, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Mesa{}, false, traducirMesa(err)
	}
	return m, true, nil
}
