// Package policy contiene las reglas de permiso del módulo atencion.
package policy

// Permisos de zonas y mesas (migración 00008). Se declaran aquí porque atencion no importa
// identidad; el catálogo de permisos vive en la base.
const (
	MesaVer         = "MESA_VER"
	MesaAdministrar = "MESA_ADMINISTRAR"
)

// Permisos de pedidos (migración 00011; PEDIDO_CREAR y DESCUENTO_AUTORIZAR existen desde 00001).
const (
	PedidoVer          = "PEDIDO_VER"
	PedidoCrear        = "PEDIDO_CREAR"
	PedidoEditar       = "PEDIDO_EDITAR"
	PedidoAnular       = "PEDIDO_ANULAR" // existe desde 00001; la migración 00013 lo asigna a CAJERO
	DescuentoAutorizar = "DESCUENTO_AUTORIZAR"
)
