package columnas

import (
	"database/sql"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
)

// Las cantidades y los importes son numeric: se leen como texto para no perder exactitud.
const (
	Pedido = `id, sucursal_id, codigo_publico::text, clave_idempotencia::text, mesa_id, reserva_id, cliente_id, mesero_id,
		tipo, estado, moneda, comensales, abierto_en, cerrado_en, notas, motivo_cancelacion, version_bloqueo,
		creado_en, creado_por, actualizado_en, actualizado_por`

	Cuenta = `id, numero, cliente_id, etiqueta, estado`

	Linea = `id, pedido_id, cuenta_id, producto_id, detalle_padre_id, descripcion, cantidad::text, precio_unitario::text,
		descuento_total::text, afectacion_igv, tasa_impuesto::text, importe_total::text, base_imponible::text, estado,
		notas_preparacion, creado_en, creado_por, actualizado_en, actualizado_por, anulado_en, anulado_por,
		motivo_anulacion`
)

func enteroNulo(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

// ScanPedido lee una fila con las columnas de Pedido.
func ScanPedido(s Fila) (dtoresult.Pedido, error) {
	var p dtoresult.Pedido
	var mesa, reserva, cliente, mesero sql.NullInt64
	var cerrado sql.NullTime
	var notas, motivo sql.NullString
	err := s.Scan(&p.ID, &p.SucursalID, &p.CodigoPublico, &p.ClaveIdempotencia, &mesa, &reserva, &cliente, &mesero,
		&p.Tipo, &p.Estado, &p.Moneda, &p.Comensales, &p.AbiertoEn, &cerrado, &notas, &motivo, &p.VersionBloqueo,
		&p.CreadoEn, &p.CreadoPor, &p.ActualizadoEn, &p.ActualizadoPor)
	p.MesaID, p.ReservaID, p.ClienteID, p.MeseroID = enteroNulo(mesa), enteroNulo(reserva), enteroNulo(cliente), enteroNulo(mesero)
	if cerrado.Valid {
		t := cerrado.Time
		p.CerradoEn = &t
	}
	p.Notas, p.MotivoCancelacion = textoNulo(notas), textoNulo(motivo)
	return p, err
}

// ScanCuenta lee una fila con las columnas de Cuenta (sin líneas ni totales).
func ScanCuenta(s Fila) (dtoresult.Cuenta, error) {
	var c dtoresult.Cuenta
	var cliente sql.NullInt64
	var etiqueta sql.NullString
	err := s.Scan(&c.ID, &c.Numero, &cliente, &etiqueta, &c.Estado)
	c.ClienteID, c.Etiqueta = enteroNulo(cliente), textoNulo(etiqueta)
	return c, err
}

// ScanLinea lee una fila con las columnas de Linea.
func ScanLinea(s Fila) (dtoresult.Linea, error) {
	var l dtoresult.Linea
	var padre, anuladoPor sql.NullInt64
	var notas, motivo sql.NullString
	var anuladoEn sql.NullTime
	err := s.Scan(&l.ID, &l.PedidoID, &l.CuentaID, &l.ProductoID, &padre, &l.Descripcion, &l.Cantidad, &l.PrecioUnitario,
		&l.DescuentoTotal, &l.AfectacionIGV, &l.TasaImpuesto, &l.ImporteTotal, &l.BaseImponible, &l.Estado,
		&notas, &l.CreadoEn, &l.CreadoPor, &l.ActualizadoEn, &l.ActualizadoPor, &anuladoEn, &anuladoPor, &motivo)
	l.DetallePadreID, l.NotasPreparacion = enteroNulo(padre), textoNulo(notas)
	l.AnuladoPor, l.MotivoAnulacion = enteroNulo(anuladoPor), textoNulo(motivo)
	if anuladoEn.Valid {
		t := anuladoEn.Time
		l.AnuladoEn = &t
	}
	return l, err
}
