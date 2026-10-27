package repository

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// ClienteSQLWriter opera sobre sch_restaurante.clientes. Actor y fechas de auditoría los
// completa el trigger de metadatos desde app.usuario_id.
type ClienteSQLWriter struct{ db database.DBTX }

func NewClienteSQLWriter(db database.DBTX) *ClienteSQLWriter { return &ClienteSQLWriter{db} }

func (r *ClienteSQLWriter) Insert(ctx context.Context, n outbound.NuevoCliente) (dtoresult.Cliente, error) {
	c, err := columnas.ScanCliente(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.clientes (tipo_documento, numero_documento, nombre_razon_social, telefono, email, direccion)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+columnas.Cliente,
		n.TipoDocumento, n.NumeroDocumento, n.Nombre, n.Telefono, n.Email, n.Direccion))
	return c, Translate(err)
}

// GetForUpdate lee el cliente bloqueando su fila hasta el final de la transacción.
func (r *ClienteSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Cliente, bool, error) {
	c, err := columnas.ScanCliente(r.db.QueryRowContext(ctx,
		`SELECT `+columnas.Cliente+` FROM sch_restaurante.clientes WHERE id = $1 FOR UPDATE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Cliente{}, false, err
	}
	return c, true, nil
}

func (r *ClienteSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosCliente) (dtoresult.Cliente, bool, error) {
	cli, err := columnas.ScanCliente(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.clientes SET
			nombre_razon_social = COALESCE($2, nombre_razon_social),
			tipo_documento = CASE WHEN $3 THEN $4::text ELSE tipo_documento END,
			numero_documento = CASE WHEN $5 THEN $6::text ELSE numero_documento END,
			telefono = CASE WHEN $7 THEN $8::text ELSE telefono END,
			email = CASE WHEN $9 THEN $10::text ELSE email END,
			direccion = CASE WHEN $11 THEN $12::text ELSE direccion END
		WHERE id = $1 RETURNING `+columnas.Cliente, id, c.Nombre,
		c.TipoDocumento.Presente, c.TipoDocumento.Valor, c.NumeroDocumento.Presente, c.NumeroDocumento.Valor,
		c.Telefono.Presente, c.Telefono.Valor, c.Email.Presente, c.Email.Valor, c.Direccion.Presente, c.Direccion.Valor))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Cliente{}, false, Translate(err)
	}
	return cli, true, nil
}

// SetBaja desactiva el cliente; el trigger de metadatos completa actor y fecha de la baja.
func (r *ClienteSQLWriter) SetBaja(ctx context.Context, id int64, motivo string) (dtoresult.Cliente, bool, error) {
	c, err := columnas.ScanCliente(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.clientes SET es_activo = false, motivo_eliminacion = $2
		WHERE id = $1 RETURNING `+columnas.Cliente, id, motivo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Cliente{}, false, Translate(err)
	}
	return c, true, nil
}

// SetActivo reactiva la misma fila (conserva el ID); el trigger limpia los marcadores de baja.
func (r *ClienteSQLWriter) SetActivo(ctx context.Context, id int64) (dtoresult.Cliente, bool, error) {
	c, err := columnas.ScanCliente(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.clientes SET es_activo = true WHERE id = $1 RETURNING `+columnas.Cliente, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Cliente{}, false, Translate(err)
	}
	return c, true, nil
}

// ProveedorSQLWriter opera sobre sch_restaurante.proveedores.
type ProveedorSQLWriter struct{ db database.DBTX }

func NewProveedorSQLWriter(db database.DBTX) *ProveedorSQLWriter { return &ProveedorSQLWriter{db} }

func (r *ProveedorSQLWriter) Insert(ctx context.Context, n outbound.NuevoProveedor) (dtoresult.Proveedor, error) {
	p, err := columnas.ScanProveedor(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.proveedores (tipo_documento, numero_documento, razon_social, contacto_nombre, telefono, email, direccion)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+columnas.Proveedor,
		n.TipoDocumento, n.NumeroDocumento, n.RazonSocial, n.ContactoNombre, n.Telefono, n.Email, n.Direccion))
	return p, Translate(err)
}

func (r *ProveedorSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Proveedor, bool, error) {
	p, err := columnas.ScanProveedor(r.db.QueryRowContext(ctx,
		`SELECT `+columnas.Proveedor+` FROM sch_restaurante.proveedores WHERE id = $1 FOR UPDATE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Proveedor{}, false, err
	}
	return p, true, nil
}

func (r *ProveedorSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosProveedor) (dtoresult.Proveedor, bool, error) {
	p, err := columnas.ScanProveedor(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.proveedores SET
			tipo_documento = COALESCE($2, tipo_documento),
			numero_documento = COALESCE($3, numero_documento),
			razon_social = COALESCE($4, razon_social),
			contacto_nombre = CASE WHEN $5 THEN $6::text ELSE contacto_nombre END,
			telefono = CASE WHEN $7 THEN $8::text ELSE telefono END,
			email = CASE WHEN $9 THEN $10::text ELSE email END,
			direccion = CASE WHEN $11 THEN $12::text ELSE direccion END
		WHERE id = $1 RETURNING `+columnas.Proveedor, id, c.TipoDocumento, c.NumeroDocumento, c.RazonSocial,
		c.ContactoNombre.Presente, c.ContactoNombre.Valor, c.Telefono.Presente, c.Telefono.Valor,
		c.Email.Presente, c.Email.Valor, c.Direccion.Presente, c.Direccion.Valor))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Proveedor{}, false, Translate(err)
	}
	return p, true, nil
}

func (r *ProveedorSQLWriter) SetBaja(ctx context.Context, id int64, motivo string) (dtoresult.Proveedor, bool, error) {
	p, err := columnas.ScanProveedor(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.proveedores SET es_activo = false, motivo_eliminacion = $2
		WHERE id = $1 RETURNING `+columnas.Proveedor, id, motivo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Proveedor{}, false, Translate(err)
	}
	return p, true, nil
}

func (r *ProveedorSQLWriter) SetActivo(ctx context.Context, id int64) (dtoresult.Proveedor, bool, error) {
	p, err := columnas.ScanProveedor(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.proveedores SET es_activo = true WHERE id = $1 RETURNING `+columnas.Proveedor, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Proveedor{}, false, Translate(err)
	}
	return p, true, nil
}
