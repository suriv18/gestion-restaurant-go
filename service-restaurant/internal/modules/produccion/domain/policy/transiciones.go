package policy

import "service-restaurant/internal/modules/produccion/domain/vo"

// Estados de una línea de comanda (comanda_detalles.estado).
const (
	EstadoPendiente   = "PENDIENTE"
	EstadoPreparacion = "PREPARACION"
	EstadoListo       = "LISTO"
	EstadoEntregado   = "ENTREGADO"
	EstadoCancelado   = "CANCELADO"
)

// siguiente es el único paso permitido desde cada estado. CANCELADO no se alcanza desde aquí: es
// de la rebanada 3.
var siguiente = map[string]string{
	EstadoPendiente:   EstadoPreparacion,
	EstadoPreparacion: EstadoListo,
	EstadoListo:       EstadoEntregado,
}

// EstadoDestinoValido indica si el valor es un destino que el contrato admite en una transición.
func EstadoDestinoValido(destino string) bool {
	switch destino {
	case EstadoPreparacion, EstadoListo, EstadoEntregado, EstadoCancelado:
		return true
	}
	return false
}

// Permite indica si de actual se puede pasar a destino: solo al paso siguiente, sin saltos ni
// retrocesos.
func Permite(actual, destino string) bool {
	s, ok := siguiente[actual]
	return ok && s == destino
}

// PermisosDe devuelve los permisos que habilitan la transición hacia destino (basta uno). La
// entrega también la puede confirmar quien edita pedidos: el mesero que lleva el plato.
func PermisosDe(destino string) []string {
	if destino == EstadoEntregado {
		return []string{CocinaOperar, PedidoEditar}
	}
	return []string{CocinaOperar}
}

// Pendiente es lo que falta por enviar de una línea: su cantidad menos lo ya enviado (sin contar
// lo cancelado).
func Pendiente(total, enviado vo.Cantidad) vo.Cantidad { return total.Menos(enviado) }
