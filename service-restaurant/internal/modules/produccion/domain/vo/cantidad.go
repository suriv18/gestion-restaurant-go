// Package vo contiene los objetos de valor del módulo produccion; solo usa la biblioteca estándar.
package vo

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var ErrValorInvalido = errors.New("valor inválido")

var cantidadRe = regexp.MustCompile(`^[0-9]{1,11}(\.[0-9]{1,3})?$`)

// Cantidad es una cantidad decimal exacta de numeric(14,3), guardada en milésimas para sumar y
// restar sin coma flotante.
type Cantidad struct{ milis int64 }

func parsear(raw string) (Cantidad, error) {
	v := strings.TrimSpace(raw)
	if !cantidadRe.MatchString(v) {
		return Cantidad{}, ErrValorInvalido
	}
	entera, fraccion, _ := strings.Cut(v, ".")
	fraccion += strings.Repeat("0", 3-len(fraccion))
	n, err := strconv.ParseInt(entera+fraccion, 10, 64)
	if err != nil {
		return Cantidad{}, ErrValorInvalido
	}
	return Cantidad{n}, nil
}

// NewCantidad exige una cantidad estrictamente positiva: la que llega en una solicitud.
func NewCantidad(raw string) (Cantidad, error) {
	c, err := parsear(raw)
	if err != nil || c.milis <= 0 {
		return Cantidad{}, ErrValorInvalido
	}
	return c, nil
}

// CantidadDeAlmacen lee una cantidad que viene de la base; acepta cero (nada enviado todavía).
func CantidadDeAlmacen(raw string) (Cantidad, error) { return parsear(raw) }

func (c Cantidad) Milis() int64           { return c.milis }
func (c Cantidad) Excede(o Cantidad) bool { return c.milis > o.milis }

// Menos resta o de c sin bajar de cero.
func (c Cantidad) Menos(o Cantidad) Cantidad {
	if o.milis >= c.milis {
		return Cantidad{}
	}
	return Cantidad{c.milis - o.milis}
}

// String devuelve la cantidad con tres decimales, igual que numeric(14,3).
func (c Cantidad) String() string {
	return fmt.Sprintf("%d.%03d", c.milis/1000, c.milis%1000)
}
