package vo

import (
	"errors"
	"strings"
)

// ErrDecimalInvalido señala un decimal mal formado, fuera de rango o con demasiados decimales.
var ErrDecimalInvalido = errors.New("decimal inválido")

// Límites de las columnas numeric de productos: precio_venta numeric(14,2) y factor_consumo numeric(18,6).
const (
	maxEnterosPrecio = 11
	escalaPrecio     = 2
	maxEnterosFactor = 12
	escalaFactor     = 6
)

// NewPrecioVenta valida un precio (0 a 99999999999.99, hasta 2 decimales) y lo devuelve con
// exactamente 2 decimales, igual que numeric(14,2)::text ("10" -> "10.00").
func NewPrecioVenta(raw string) (string, error) {
	return normalizarDecimal(raw, maxEnterosPrecio, escalaPrecio, true)
}

// NewFactorConsumo valida un factor (> 0, hasta 6 decimales, 12 enteros) y lo devuelve con
// exactamente 6 decimales, igual que numeric(18,6)::text ("1" -> "1.000000").
func NewFactorConsumo(raw string) (string, error) {
	return normalizarDecimal(raw, maxEnterosFactor, escalaFactor, false)
}

// normalizarDecimal acepta solo dígitos con un punto decimal opcional rodeado de dígitos: sin
// signo, espacios, exponente ni separadores de miles. Nunca pasa por float. Los ceros a la
// izquierda se descartan y la parte decimal se completa hasta la escala.
func normalizarDecimal(raw string, maxEnteros, escala int, admiteCero bool) (string, error) {
	entero, frac, tienePunto := strings.Cut(raw, ".")
	if !soloDigitos(entero) || (tienePunto && !soloDigitos(frac)) || len(frac) > escala {
		return "", ErrDecimalInvalido
	}
	entero = strings.TrimLeft(entero, "0")
	if entero == "" {
		entero = "0"
	}
	if len(entero) > maxEnteros {
		return "", ErrDecimalInvalido
	}
	frac += strings.Repeat("0", escala-len(frac))
	if !admiteCero && entero == "0" && strings.Trim(frac, "0") == "" {
		return "", ErrDecimalInvalido
	}
	return entero + "." + frac, nil
}

func soloDigitos(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
