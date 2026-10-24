// Package repository contiene el SQL parametrizado de lectura del directorio de clientes y
// proveedores.
package repository

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

func activoSQL(a *bool) sql.NullBool {
	if a == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *a, Valid: true}
}

// ClienteSQLReader lee clientes.
type ClienteSQLReader struct{ db database.DBTX }

func NewClienteSQLReader(db database.DBTX) *ClienteSQLReader { return &ClienteSQLReader{db} }

// ListarClientes ordena por id (cursor estable). El texto, ya con los comodines escapados, busca
// por nombre (contiene) o por número de documento (empieza por).
func (r *ClienteSQLReader) ListarClientes(ctx context.Context, f outbound.FiltroClientes) ([]dtoresult.Cliente, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+columnas.Cliente+` FROM sch_restaurante.clientes
		WHERE id > $1 AND ($2::boolean IS NULL OR es_activo = $2)
		  AND ($3::text = '' OR nombre_razon_social ILIKE '%' || $3 || '%' ESCAPE '\'
		       OR numero_documento ILIKE $3 || '%' ESCAPE '\')
		ORDER BY id LIMIT $4`, f.DespuesDeID, activoSQL(f.Activo), f.Texto, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Cliente
	for rows.Next() {
		c, err := columnas.ScanCliente(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *ClienteSQLReader) ObtenerCliente(ctx context.Context, id int64) (dtoresult.Cliente, bool, error) {
	c, err := columnas.ScanCliente(r.db.QueryRowContext(ctx, `SELECT `+columnas.Cliente+` FROM sch_restaurante.clientes WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Cliente{}, false, nil
	}
	return c, err == nil, err
}

// ProveedorSQLReader lee proveedores.
type ProveedorSQLReader struct{ db database.DBTX }

func NewProveedorSQLReader(db database.DBTX) *ProveedorSQLReader { return &ProveedorSQLReader{db} }

// ListarProveedores busca por razón social (contiene) o por número de documento (empieza por).
func (r *ProveedorSQLReader) ListarProveedores(ctx context.Context, f outbound.FiltroProveedores) ([]dtoresult.Proveedor, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+columnas.Proveedor+` FROM sch_restaurante.proveedores
		WHERE id > $1 AND ($2::boolean IS NULL OR es_activo = $2)
		  AND ($3::text = '' OR razon_social ILIKE '%' || $3 || '%' ESCAPE '\'
		       OR numero_documento ILIKE $3 || '%' ESCAPE '\')
		ORDER BY id LIMIT $4`, f.DespuesDeID, activoSQL(f.Activo), f.Texto, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Proveedor
	for rows.Next() {
		p, err := columnas.ScanProveedor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ProveedorSQLReader) ObtenerProveedor(ctx context.Context, id int64) (dtoresult.Proveedor, bool, error) {
	p, err := columnas.ScanProveedor(r.db.QueryRowContext(ctx, `SELECT `+columnas.Proveedor+` FROM sch_restaurante.proveedores WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Proveedor{}, false, nil
	}
	return p, err == nil, err
}
