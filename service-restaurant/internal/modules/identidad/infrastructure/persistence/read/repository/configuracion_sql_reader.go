package repository

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/platform/database"
)

// ConfiguracionSQLReader lee negocio y sucursales.
type ConfiguracionSQLReader struct{ db database.DBTX }

func NewConfiguracionSQLReader(db database.DBTX) *ConfiguracionSQLReader {
	return &ConfiguracionSQLReader{db}
}

type filaConfig interface{ Scan(...any) error }

func cadenaNula(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

const negocioColsLectura = `id, ruc, razon_social, nombre_comercial, direccion_fiscal, ubigeo, telefono, email,
	moneda, zona_horaria, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`

func leerNegocio(s filaConfig) (dtoresult.Negocio, error) {
	var n dtoresult.Negocio
	var ubigeo, telefono, email sql.NullString
	err := s.Scan(&n.ID, &n.RUC, &n.RazonSocial, &n.NombreComercial, &n.DireccionFiscal, &ubigeo, &telefono, &email,
		&n.Moneda, &n.ZonaHoraria, &n.EsActivo, &n.CreadoEn, &n.CreadoPor, &n.ActualizadoEn, &n.ActualizadoPor)
	n.Ubigeo, n.Telefono, n.Email = cadenaNula(ubigeo), cadenaNula(telefono), cadenaNula(email)
	return n, err
}

// GetNegocio devuelve la empresa única; found=false si aún no se instaló.
func (r *ConfiguracionSQLReader) GetNegocio(ctx context.Context) (dtoresult.Negocio, bool, error) {
	n, err := leerNegocio(r.db.QueryRowContext(ctx,
		`SELECT `+negocioColsLectura+` FROM sch_restaurante.negocio WHERE singleton`))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Negocio{}, false, nil
	}
	return n, err == nil, err
}

const sucursalColsLectura = `id, negocio_id, codigo, nombre, direccion, ubigeo, codigo_establecimiento_sunat, telefono,
	es_activo, motivo_eliminacion, eliminado_en, eliminado_por, creado_en, creado_por, actualizado_en, actualizado_por`

func leerSucursal(s filaConfig) (dtoresult.Sucursal, error) {
	var x dtoresult.Sucursal
	var ubigeo, sunat, tel, motivo sql.NullString
	var elimEn sql.NullTime
	var elimPor sql.NullInt64
	err := s.Scan(&x.ID, &x.NegocioID, &x.Codigo, &x.Nombre, &x.Direccion, &ubigeo, &sunat, &tel,
		&x.EsActivo, &motivo, &elimEn, &elimPor, &x.CreadoEn, &x.CreadoPor, &x.ActualizadoEn, &x.ActualizadoPor)
	x.Ubigeo, x.CodigoEstablecimientoSunat, x.Telefono, x.MotivoEliminacion =
		cadenaNula(ubigeo), cadenaNula(sunat), cadenaNula(tel), cadenaNula(motivo)
	if elimEn.Valid {
		t := elimEn.Time
		x.EliminadoEn = &t
	}
	if elimPor.Valid {
		v := elimPor.Int64
		x.EliminadoPor = &v
	}
	return x, err
}

// ListSucursales devuelve las sedes activas de f.SedeIDs y, si f.IncluirInactivas, todas las
// inactivas; ordenadas por id para un cursor estable.
func (r *ConfiguracionSQLReader) ListSucursales(ctx context.Context, f outbound.FiltroSucursales) ([]dtoresult.Sucursal, error) {
	var activo sql.NullBool
	if f.Activo != nil {
		activo = sql.NullBool{Bool: *f.Activo, Valid: true}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+sucursalColsLectura+` FROM sch_restaurante.sucursales
		WHERE ((es_activo AND id = ANY($1::bigint[])) OR (NOT es_activo AND $2))
		  AND id > $3 AND ($4::boolean IS NULL OR es_activo = $4)
		ORDER BY id LIMIT $5`, f.SedeIDs, f.IncluirInactivas, f.DespuesDeID, activo, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Sucursal
	for rows.Next() {
		s, err := leerSucursal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ConfiguracionSQLReader) GetSucursal(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	s, err := leerSucursal(r.db.QueryRowContext(ctx,
		`SELECT `+sucursalColsLectura+` FROM sch_restaurante.sucursales WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Sucursal{}, false, nil
	}
	return s, err == nil, err
}
