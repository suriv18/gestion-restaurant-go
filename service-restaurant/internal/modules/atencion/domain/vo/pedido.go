package vo

import (
	"math/big"
	"regexp"
	"strings"
)

const maxComensales = 500

var (
	cantidadRe = regexp.MustCompile(`^[0-9]{1,11}(\.[0-9]{1,3})?$`)
	dineroRe   = regexp.MustCompile(`^[0-9]{1,12}(\.[0-9]{1,2})?$`)
)

// TipoPedido es el canal de atención de un pedido.
type TipoPedido string

const (
	TipoSalon    TipoPedido = "SALON"
	TipoLlevar   TipoPedido = "LLEVAR"
	TipoDelivery TipoPedido = "DELIVERY"
)

func NewTipoPedido(raw string) (TipoPedido, error) {
	switch t := TipoPedido(strings.TrimSpace(raw)); t {
	case TipoSalon, TipoLlevar, TipoDelivery:
		return t, nil
	}
	return "", ErrValorInvalido
}

func (t TipoPedido) String() string { return string(t) }

// Comensales es el número de personas de un pedido (1 a 500).
type Comensales struct{ value int }

func NewComensales(n int) (Comensales, error) {
	if n < 1 || n > maxComensales {
		return Comensales{}, ErrValorInvalido
	}
	return Comensales{n}, nil
}

func (c Comensales) Int() int { return c.value }

// Cantidad es una cantidad decimal exacta de numeric(14,3), estrictamente positiva.
type Cantidad struct {
	text string
	r    *big.Rat
}

func NewCantidad(raw string) (Cantidad, error) {
	v := strings.TrimSpace(raw)
	if !cantidadRe.MatchString(v) {
		return Cantidad{}, ErrValorInvalido
	}
	r, ok := new(big.Rat).SetString(v)
	if !ok || r.Sign() <= 0 {
		return Cantidad{}, ErrValorInvalido
	}
	return Cantidad{v, r}, nil
}

func (c Cantidad) String() string { return c.text }

// Dinero es un importe decimal exacto de numeric(14,2), mayor o igual que cero.
type Dinero struct {
	text string
	r    *big.Rat
}

func NewDinero(raw string) (Dinero, error) {
	v := strings.TrimSpace(raw)
	if !dineroRe.MatchString(v) {
		return Dinero{}, ErrValorInvalido
	}
	r, ok := new(big.Rat).SetString(v)
	if !ok {
		return Dinero{}, ErrValorInvalido
	}
	return Dinero{v, r}, nil
}

func (d Dinero) String() string { return d.text }
func (d Dinero) EsCero() bool   { return d.r.Sign() == 0 }

// DescuentoCabe indica si el descuento no supera el importe bruto de la línea:
// round(cantidad × precio_unitario, 2), con el redondeo de PostgreSQL (mitad hacia arriba).
// Es la misma regla que el CHECK de detalles_pedido, para rechazar antes con un código propio.
func DescuentoCabe(c Cantidad, precioUnitario string, d Dinero) bool {
	p, ok := new(big.Rat).SetString(strings.TrimSpace(precioUnitario))
	if !ok {
		return false
	}
	bruto := redondear2(new(big.Rat).Mul(c.r, p))
	return d.r.Cmp(bruto) <= 0
}

// redondear2 redondea un racional no negativo a 2 decimales, la mitad hacia arriba.
func redondear2(r *big.Rat) *big.Rat {
	x := new(big.Rat).Mul(r, big.NewRat(100, 1))
	n := new(big.Int).Quo(x.Num(), x.Denom())
	resto := new(big.Rat).Sub(x, new(big.Rat).SetInt(n))
	if resto.Cmp(big.NewRat(1, 2)) >= 0 {
		n.Add(n, big.NewInt(1))
	}
	return new(big.Rat).SetFrac(n, big.NewInt(100))
}
