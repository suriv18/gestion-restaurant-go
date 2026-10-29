// Package policy contiene las reglas de permiso y de estado del módulo produccion.
package policy

// Permisos que usa el módulo. Se declaran aquí porque produccion no importa identidad ni
// atencion; el catálogo de permisos vive en la base (00001, 00011 y 00012).
const (
	PedidoVer    = "PEDIDO_VER"
	PedidoCrear  = "PEDIDO_CREAR"
	PedidoEditar = "PEDIDO_EDITAR"
	CocinaOperar = "COCINA_OPERAR"
)
