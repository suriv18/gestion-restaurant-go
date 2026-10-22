package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/platform/database"
)

const negocioCols = `id, ruc, razon_social, nombre_comercial, direccion_fiscal, ubigeo, telefono, email,
	moneda, zona_horaria, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`

func textoNulo(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func scanNegocio(s scanner) (dtoresult.Negocio, error) {
	var n dtoresult.Negocio
	var ubigeo, telefono, email sql.NullString
	err := s.Scan(&n.ID, &n.RUC, &n.RazonSocial, &n.NombreComercial, &n.DireccionFiscal, &ubigeo, &telefono, &email,
		&n.Moneda, &n.ZonaHoraria, &n.EsActivo, &n.CreadoEn, &n.CreadoPor, &n.ActualizadoEn, &n.ActualizadoPor)
	n.Ubigeo, n.Telefono, n.Email = textoNulo(ubigeo), textoNulo(telefono), textoNulo(email)
	return n, err
}

// NegocioSQLWriter opera sobre sch_restaurante.negocio (singleton).
type NegocioSQLWriter struct{ db database.DBTX }

func NewNegocioSQLWriter(db database.DBTX) *NegocioSQLWriter { return &NegocioSQLWriter{db} }

// Update aplica solo los campos presentes. El RUC y el singleton no forman parte del SQL.
func (r *NegocioSQLWriter) Update(ctx context.Context, c outbound.CambiosNegocio) (dtoresult.Negocio, bool, error) {
	n, err := scanNegocio(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.negocio SET
			razon_social = COALESCE($1, razon_social),
			nombre_comercial = COALESCE($2, nombre_comercial),
			direccion_fiscal = COALESCE($3, direccion_fiscal),
			ubigeo = CASE WHEN $4 THEN $5 ELSE ubigeo END,
			telefono = CASE WHEN $6 THEN $7 ELSE telefono END,
			email = CASE WHEN $8 THEN $9 ELSE email END,
			moneda = COALESCE($10, moneda),
			zona_horaria = COALESCE($11, zona_horaria)
		WHERE singleton RETURNING `+negocioCols,
		c.RazonSocial, c.NombreComercial, c.DireccionFiscal,
		c.Ubigeo.Presente, c.Ubigeo.Valor, c.Telefono.Presente, c.Telefono.Valor, c.Email.Presente, c.Email.Valor,
		c.Moneda, c.ZonaHoraria))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Negocio{}, false, Translate(err)
	}
	return n, true, nil
}

const sucursalCols = `id, negocio_id, codigo, nombre, direccion, ubigeo, codigo_establecimiento_sunat, telefono,
	es_activo, motivo_eliminacion, eliminado_en, eliminado_por, creado_en, creado_por, actualizado_en, actualizado_por`

func scanSucursal(s scanner) (dtoresult.Sucursal, error) {
	var x dtoresult.Sucursal
	var ubigeo, sunat, tel, motivo sql.NullString
	var elimEn sql.NullTime
	var elimPor sql.NullInt64
	err := s.Scan(&x.ID, &x.NegocioID, &x.Codigo, &x.Nombre, &x.Direccion, &ubigeo, &sunat, &tel,
		&x.EsActivo, &motivo, &elimEn, &elimPor, &x.CreadoEn, &x.CreadoPor, &x.ActualizadoEn, &x.ActualizadoPor)
	x.Ubigeo, x.CodigoEstablecimientoSunat, x.Telefono, x.MotivoEliminacion =
		textoNulo(ubigeo), textoNulo(sunat), textoNulo(tel), textoNulo(motivo)
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

// traducirSucursal distingue la colisión del código de sede (sucursales_codigo_key) de otras unicidades.
func traducirSucursal(err error) error {
	err = Translate(err)
	if errors.Is(err, outbound.ErrDuplicado) && strings.Contains(err.Error(), "sucursales_codigo_key") {
		return outbound.ErrCodigoSucursalDuplicado
	}
	return err
}

// SucursalSQLWriter opera sobre sch_restaurante.sucursales.
type SucursalSQLWriter struct{ db database.DBTX }

func NewSucursalSQLWriter(db database.DBTX) *SucursalSQLWriter { return &SucursalSQLWriter{db} }

// Insert crea la sede en la empresa única; negocio_id sale de la base, nunca del cliente.
func (r *SucursalSQLWriter) Insert(ctx context.Context, n outbound.NuevaSucursal) (dtoresult.Sucursal, error) {
	s, err := scanSucursal(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.sucursales (negocio_id, codigo, nombre, direccion, ubigeo, codigo_establecimiento_sunat, telefono)
		SELECT id, $1, $2, $3, $4, $5, $6 FROM sch_restaurante.negocio WHERE singleton
		RETURNING `+sucursalCols, n.Codigo, n.Nombre, n.Direccion, n.Ubigeo, n.CodigoEstablecimientoSunat, n.Telefono))
	return s, traducirSucursal(err)
}

// GetForUpdate lee la sede bloqueando su fila hasta el final de la transacción.
func (r *SucursalSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	s, err := scanSucursal(r.db.QueryRowContext(ctx,
		`SELECT `+sucursalCols+` FROM sch_restaurante.sucursales WHERE id = $1 FOR UPDATE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Sucursal{}, false, err
	}
	return s, true, nil
}

func (r *SucursalSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosSucursal) (dtoresult.Sucursal, bool, error) {
	s, err := scanSucursal(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.sucursales SET
			codigo = COALESCE($2, codigo),
			nombre = COALESCE($3, nombre),
			direccion = COALESCE($4, direccion),
			ubigeo = CASE WHEN $5 THEN $6 ELSE ubigeo END,
			codigo_establecimiento_sunat = CASE WHEN $7 THEN $8 ELSE codigo_establecimiento_sunat END,
			telefono = CASE WHEN $9 THEN $10 ELSE telefono END
		WHERE id = $1 RETURNING `+sucursalCols, id, c.Codigo, c.Nombre, c.Direccion,
		c.Ubigeo.Presente, c.Ubigeo.Valor, c.CodigoEstablecimientoSunat.Presente, c.CodigoEstablecimientoSunat.Valor,
		c.Telefono.Presente, c.Telefono.Valor))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Sucursal{}, false, traducirSucursal(err)
	}
	return s, true, nil
}

// LockNegocio bloquea la fila de la empresa: serializa las bajas de sedes (las altas solo esperan por los locks de FK).
func (r *SucursalSQLWriter) LockNegocio(ctx context.Context) error {
	var id int64
	return r.db.QueryRowContext(ctx, `SELECT id FROM sch_restaurante.negocio WHERE singleton FOR UPDATE`).Scan(&id)
}

func (r *SucursalSQLWriter) CountActivas(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM sch_restaurante.sucursales WHERE es_activo`).Scan(&n)
	return n, err
}

// SetBaja desactiva la sede; el trigger de metadatos completa actor y fecha de la baja.
func (r *SucursalSQLWriter) SetBaja(ctx context.Context, id int64, motivo string) (dtoresult.Sucursal, bool, error) {
	s, err := scanSucursal(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.sucursales SET es_activo = false, motivo_eliminacion = $2
		WHERE id = $1 RETURNING `+sucursalCols, id, motivo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Sucursal{}, false, traducirSucursal(err)
	}
	return s, true, nil
}

// SetActiva reactiva la misma fila (conserva el ID); el trigger limpia los marcadores de baja.
func (r *SucursalSQLWriter) SetActiva(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	s, err := scanSucursal(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.sucursales SET es_activo = true WHERE id = $1 RETURNING `+sucursalCols, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Sucursal{}, false, traducirSucursal(err)
	}
	return s, true, nil
}
