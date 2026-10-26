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

	Categoria = `id, nombre, descripcion, orden_visualizacion, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`
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

// ScanCategoria lee una fila con las columnas de Categoria.
func ScanCategoria(s Fila) (dtoresult.Categoria, error) {
	var c dtoresult.Categoria
	var desc sql.NullString
	err := s.Scan(&c.ID, &c.Nombre, &desc, &c.Orden, &c.EsActivo, &c.CreadoEn, &c.CreadoPor, &c.ActualizadoEn, &c.ActualizadoPor)
	c.Descripcion = textoNulo(desc)
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

// Impuesto y TasaImpuesto: la tasa y las fechas se leen como texto (numeric(7,4)::text y
// date::text) para no pasar por float ni por zona horaria.
const (
	Impuesto = `id, codigo, nombre, descripcion, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`

	TasaImpuesto = `id, impuesto_id, tasa::text, vigente_desde::text, vigente_hasta::text, fundamento,
	creado_en, creado_por, actualizado_en, actualizado_por`
)

// ScanImpuesto lee una fila con las columnas de Impuesto.
func ScanImpuesto(s Fila) (dtoresult.Impuesto, error) {
	var i dtoresult.Impuesto
	var desc sql.NullString
	err := s.Scan(&i.ID, &i.Codigo, &i.Nombre, &desc, &i.EsActivo, &i.CreadoEn, &i.CreadoPor, &i.ActualizadoEn, &i.ActualizadoPor)
	i.Descripcion = textoNulo(desc)
	return i, err
}

// ScanTasaImpuesto lee una fila con las columnas de TasaImpuesto.
func ScanTasaImpuesto(s Fila) (dtoresult.TasaImpuesto, error) {
	var t dtoresult.TasaImpuesto
	var hasta, fund sql.NullString
	err := s.Scan(&t.ID, &t.ImpuestoID, &t.Tasa, &t.VigenteDesde, &hasta, &fund,
		&t.CreadoEn, &t.CreadoPor, &t.ActualizadoEn, &t.ActualizadoPor)
	t.VigenteHasta, t.Fundamento = textoNulo(hasta), textoNulo(fund)
	return t, err
}

// Producto: precio y factor se leen como texto (numeric::text) para no pasar por float; la
// respuesta trae el precio con 2 decimales y el factor con 6, tal como los guarda PostgreSQL.
const Producto = `id, categoria_id, codigo, nombre, descripcion, tipo, articulo_id, factor_consumo::text, precio_venta::text,
	afectacion_igv, impuesto_id, imagen_url, es_activo, motivo_eliminacion, eliminado_en, eliminado_por,
	creado_en, creado_por, actualizado_en, actualizado_por`

// ScanProducto lee una fila con las columnas de Producto.
func ScanProducto(s Fila) (dtoresult.Producto, error) { return scanProducto(s) }

// ProductoEnSede son las columnas que el LEFT JOIN a productos_sucursal añade a Producto (en este
// orden): ps_id (NULL = sin fila activa en la sede), disponible, precio de la sede y estación.
const ProductoEnSede = `ps.ps_id, ps.disponible, ps.ps_precio::text, ps.ps_estacion`

// ScanProductoEnSede lee Producto seguido de ProductoEnSede y completa EnSucursal.
func ScanProductoEnSede(s Fila) (dtoresult.Producto, error) {
	var psID, estacion sql.NullInt64
	var disp sql.NullBool
	var precio sql.NullString
	p, err := scanProducto(s, &psID, &disp, &precio, &estacion)
	if err != nil {
		return p, err
	}
	p.ConSede = true
	if psID.Valid {
		e := dtoresult.EnSucursal{Disponible: disp.Bool, PrecioSucursal: textoNulo(precio), PrecioEfectivo: p.PrecioVenta, EstacionID: enteroNulo(estacion)}
		if e.PrecioSucursal != nil {
			e.PrecioEfectivo = *e.PrecioSucursal
		}
		p.EnSucursal = &e
	}
	return p, nil
}

func scanProducto(s Fila, extra ...any) (dtoresult.Producto, error) {
	var p dtoresult.Producto
	var desc, imagen, motivo sql.NullString
	var articulo, impuesto, elimPor sql.NullInt64
	var elimEn sql.NullTime
	dest := append([]any{&p.ID, &p.CategoriaID, &p.Codigo, &p.Nombre, &desc, &p.Tipo, &articulo, &p.FactorConsumo, &p.PrecioVenta,
		&p.AfectacionIGV, &impuesto, &imagen, &p.EsActivo, &motivo, &elimEn, &elimPor,
		&p.CreadoEn, &p.CreadoPor, &p.ActualizadoEn, &p.ActualizadoPor}, extra...)
	err := s.Scan(dest...)
	p.Descripcion, p.ImagenURL = textoNulo(desc), textoNulo(imagen)
	p.ArticuloID, p.ImpuestoID = enteroNulo(articulo), enteroNulo(impuesto)
	p.Baja = baja(motivo, elimEn, elimPor)
	return p, err
}

func enteroNulo(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

// Estacion: estación de preparación de una sede.
const Estacion = `id, sucursal_id, codigo, nombre, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`

// ScanEstacion lee una fila con las columnas de Estacion.
func ScanEstacion(s Fila) (dtoresult.Estacion, error) {
	var e dtoresult.Estacion
	err := s.Scan(&e.ID, &e.SucursalID, &e.Codigo, &e.Nombre, &e.EsActivo, &e.CreadoEn, &e.CreadoPor, &e.ActualizadoEn, &e.ActualizadoPor)
	return e, err
}

// ProductoSucursal: fila de productos_sucursal (disponibilidad por sede). El precio se lee como
// texto para no pasar por float.
const ProductoSucursal = `id, producto_id, sucursal_id, estacion_id, precio_venta::text, disponible, es_activo,
	creado_en, creado_por, actualizado_en, actualizado_por`

// ScanProductoSucursal lee una fila con las columnas de ProductoSucursal.
func ScanProductoSucursal(s Fila) (dtoresult.ProductoSucursal, error) {
	var p dtoresult.ProductoSucursal
	var estacion sql.NullInt64
	var precio sql.NullString
	err := s.Scan(&p.ID, &p.ProductoID, &p.SucursalID, &estacion, &precio, &p.Disponible, &p.EsActivo,
		&p.CreadoEn, &p.CreadoPor, &p.ActualizadoEn, &p.ActualizadoPor)
	p.EstacionID, p.PrecioVenta = enteroNulo(estacion), textoNulo(precio)
	return p, err
}
