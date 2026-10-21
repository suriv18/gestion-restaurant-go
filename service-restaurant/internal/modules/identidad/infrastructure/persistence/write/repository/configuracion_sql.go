package repository

import (
	"context"
	"database/sql"

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
