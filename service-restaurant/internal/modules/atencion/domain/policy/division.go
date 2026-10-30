package policy

import (
	"errors"
	"slices"
)

// Rechazos de la selección de líneas al dividir una cuenta.
var (
	ErrSeleccionVacia    = errors.New("la selección de líneas está vacía")
	ErrLineaRepetida     = errors.New("una línea se repite en la selección")
	ErrLineaInexistente  = errors.New("la línea no pertenece al pedido")
	ErrLineaAnulada      = errors.New("la línea está anulada")
	ErrAdicionalSinPadre = errors.New("un adicional no viaja sin la línea de la que cuelga")
)

// LineaPedido es lo que la división de cuentas necesita saber de una línea del pedido.
type LineaPedido struct {
	ID       int64
	PadreID  *int64
	CuentaID int64
	Activa   bool
}

// SeleccionDeDivision resuelve qué líneas se mueven a la cuenta nueva: las pedidas más los
// adicionales activos que cuelgan de ellas. Un adicional pedido sin su padre en la selección se
// rechaza. El resultado va ordenado y sin repetidos.
func SeleccionDeDivision(pedidas []int64, lineas []LineaPedido) ([]int64, error) {
	if len(pedidas) == 0 {
		return nil, ErrSeleccionVacia
	}
	porID := make(map[int64]LineaPedido, len(lineas))
	for _, l := range lineas {
		porID[l.ID] = l
	}
	pedida := make(map[int64]bool, len(pedidas))
	for _, id := range pedidas {
		if pedida[id] {
			return nil, ErrLineaRepetida
		}
		pedida[id] = true
	}
	for _, id := range pedidas {
		l, ok := porID[id]
		switch {
		case !ok:
			return nil, ErrLineaInexistente
		case !l.Activa:
			return nil, ErrLineaAnulada
		case l.PadreID != nil && !pedida[*l.PadreID]:
			return nil, ErrAdicionalSinPadre
		}
	}
	movidas := slices.Clone(pedidas)
	for _, l := range lineas {
		if l.PadreID != nil && pedida[*l.PadreID] && !pedida[l.ID] && l.Activa {
			movidas = append(movidas, l.ID)
		}
	}
	slices.Sort(movidas)
	return movidas, nil
}
