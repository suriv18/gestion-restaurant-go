package vo

import (
	"regexp"
	"strings"
)

const (
	maxNumeroMesa = 20
	maxCapacidad  = 500
	maxOrden      = 9999
)

var posicionRe = regexp.MustCompile(`^-?[0-9]{1,8}(\.[0-9]{1,2})?$`)

// NumeroMesa identifica la mesa dentro de su sede (1 a 20 caracteres, sin saltos ni tabulaciones).
type NumeroMesa struct{ value string }

func NewNumeroMesa(raw string) (NumeroMesa, error) {
	v, err := TextoUnaLinea(raw, maxNumeroMesa)
	if err != nil {
		return NumeroMesa{}, ErrValorInvalido
	}
	return NumeroMesa{v}, nil
}

func (n NumeroMesa) String() string { return n.value }

// Capacidad es el número de comensales de la mesa (1 a 500).
type Capacidad struct{ value int }

func NewCapacidad(n int) (Capacidad, error) {
	if n < 1 || n > maxCapacidad {
		return Capacidad{}, ErrValorInvalido
	}
	return Capacidad{n}, nil
}

func (c Capacidad) Int() int { return c.value }

// Orden es el orden de visualización de una zona (0 a 9999).
type Orden struct{ value int }

func NewOrden(n int) (Orden, error) {
	if n < 0 || n > maxOrden {
		return Orden{}, ErrValorInvalido
	}
	return Orden{n}, nil
}

func (o Orden) Int() int { return o.value }

// Posicion es una coordenada decimal exacta de numeric(10,2): hasta 8 enteros y 2 decimales.
type Posicion struct{ value string }

func NewPosicion(raw string) (Posicion, error) {
	v := strings.TrimSpace(raw)
	if !posicionRe.MatchString(v) {
		return Posicion{}, ErrValorInvalido
	}
	return Posicion{v}, nil
}

func (p Posicion) String() string { return p.value }
