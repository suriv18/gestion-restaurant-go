package repository

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
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
