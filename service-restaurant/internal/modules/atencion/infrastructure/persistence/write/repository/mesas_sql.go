package repository

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// Restricciones únicas que tienen un rechazo propio.
const restriccionNumeroMesa = "mesas_sucursal_id_numero_mesa_key"

// traducirMesa distingue la colisión del número de mesa por el nombre de la restricción.
func traducirMesa(err error) error {
	if restriccionUnica(err) == restriccionNumeroMesa {
		return outbound.ErrNumeroMesaDuplicado
	}
	return Translate(err)
}

// MesaSQLWriter opera sobre sch_restaurante.mesas.
type MesaSQLWriter struct{ db database.DBTX }

func NewMesaSQLWriter(db database.DBTX) *MesaSQLWriter { return &MesaSQLWriter{db} }

func (r *MesaSQLWriter) Insert(ctx context.Context, n outbound.NuevaMesa) (dtoresult.Mesa, error) {
	m, err := columnas.ScanMesa(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.mesas (sucursal_id, zona_id, numero_mesa, capacidad, habilitada, posicion_x, posicion_y)
		VALUES ($1, $2, $3, $4, $5, $6::text::numeric, $7::text::numeric) RETURNING `+columnas.Mesa,
		n.SucursalID, n.ZonaID, n.NumeroMesa, n.Capacidad, n.Habilitada, n.PosicionX, n.PosicionY))
	return m, traducirMesa(err)
}

// GetForUpdate lee la mesa bloqueando su fila hasta el final de la transacción.
func (r *MesaSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	m, err := columnas.ScanMesa(r.db.QueryRowContext(ctx,
		`SELECT `+columnas.Mesa+` FROM sch_restaurante.mesas WHERE id = $1 FOR UPDATE`, id))
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
	m, err := columnas.ScanMesa(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.mesas SET
			zona_id = COALESCE($2, zona_id),
			numero_mesa = COALESCE($3, numero_mesa),
			capacidad = COALESCE($4, capacidad),
			habilitada = COALESCE($5, habilitada),
			posicion_x = CASE WHEN $6 THEN $7::text::numeric ELSE posicion_x END,
			posicion_y = CASE WHEN $8 THEN $9::text::numeric ELSE posicion_y END
		WHERE id = $1 RETURNING `+columnas.Mesa, id, c.ZonaID, c.NumeroMesa, c.Capacidad, c.Habilitada,
		c.PosicionX.Presente, c.PosicionX.Valor, c.PosicionY.Presente, c.PosicionY.Valor))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Mesa{}, false, traducirMesa(err)
	}
	return m, true, nil
}

// SetBaja desactiva la mesa; el trigger de metadatos completa actor y fecha de la baja.
func (r *MesaSQLWriter) SetBaja(ctx context.Context, id int64, motivo string) (dtoresult.Mesa, bool, error) {
	m, err := columnas.ScanMesa(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.mesas SET es_activo = false, motivo_eliminacion = $2
		WHERE id = $1 RETURNING `+columnas.Mesa, id, motivo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Mesa{}, false, traducirMesa(err)
	}
	return m, true, nil
}

// SetActiva reactiva la misma fila (conserva el ID); el trigger limpia los marcadores de baja.
func (r *MesaSQLWriter) SetActiva(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	m, err := columnas.ScanMesa(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.mesas SET es_activo = true WHERE id = $1 RETURNING `+columnas.Mesa, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Mesa{}, false, traducirMesa(err)
	}
	return m, true, nil
}
