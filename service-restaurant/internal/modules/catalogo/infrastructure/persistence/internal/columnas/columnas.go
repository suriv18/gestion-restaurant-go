// Package columnas reúne las columnas y el escaneo de filas de clientes y proveedores que
// comparten el lado de lectura y el de escritura. Vive bajo persistence/internal: solo la
// persistencia del módulo puede importarlo, y no acopla un lado con el otro.
package columnas

import (
	"database/sql"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
)

// Fila la cumplen *sql.Row y *sql.Rows.
type Fila interface{ Scan(...any) error }

const (
	Cliente = `id, tipo_documento, numero_documento, nombre_razon_social, telefono, email, direccion,
	es_activo, motivo_eliminacion, eliminado_en, eliminado_por, creado_en, creado_por, actualizado_en, actualizado_por`

	Proveedor = `id, tipo_documento, numero_documento, razon_social, contacto_nombre, telefono, email, direccion,
	es_activo, motivo_eliminacion, eliminado_en, eliminado_por, creado_en, creado_por, actualizado_en, actualizado_por`
)

func textoNulo(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

// baja traduce los marcadores de baja nulos de la fila.
func baja(motivo sql.NullString, en sql.NullTime, por sql.NullInt64) dtoresult.Baja {
	b := dtoresult.Baja{MotivoEliminacion: textoNulo(motivo)}
	if en.Valid {
		t := en.Time
		b.EliminadoEn = &t
	}
	if por.Valid {
		v := por.Int64
		b.EliminadoPor = &v
	}
	return b
}

// ScanCliente lee una fila con las columnas de Cliente.
func ScanCliente(s Fila) (dtoresult.Cliente, error) {
	var c dtoresult.Cliente
	var tipo, numero, tel, email, dir, motivo sql.NullString
	var elimEn sql.NullTime
	var elimPor sql.NullInt64
	err := s.Scan(&c.ID, &tipo, &numero, &c.Nombre, &tel, &email, &dir,
		&c.EsActivo, &motivo, &elimEn, &elimPor, &c.CreadoEn, &c.CreadoPor, &c.ActualizadoEn, &c.ActualizadoPor)
	c.TipoDocumento, c.NumeroDocumento = textoNulo(tipo), textoNulo(numero)
	c.Telefono, c.Email, c.Direccion = textoNulo(tel), textoNulo(email), textoNulo(dir)
	c.Baja = baja(motivo, elimEn, elimPor)
	return c, err
}

// ScanProveedor lee una fila con las columnas de Proveedor.
func ScanProveedor(s Fila) (dtoresult.Proveedor, error) {
	var p dtoresult.Proveedor
	var contacto, tel, email, dir, motivo sql.NullString
	var elimEn sql.NullTime
	var elimPor sql.NullInt64
	err := s.Scan(&p.ID, &p.TipoDocumento, &p.NumeroDocumento, &p.RazonSocial, &contacto, &tel, &email, &dir,
		&p.EsActivo, &motivo, &elimEn, &elimPor, &p.CreadoEn, &p.CreadoPor, &p.ActualizadoEn, &p.ActualizadoPor)
	p.ContactoNombre, p.Telefono, p.Email, p.Direccion = textoNulo(contacto), textoNulo(tel), textoNulo(email), textoNulo(dir)
	p.Baja = baja(motivo, elimEn, elimPor)
	return p, err
}
