package vo

import (
	"errors"
	"regexp"
	"strings"
)

// ErrCodigoProductoInvalido señala un código de producto fuera de formato.
var ErrCodigoProductoInvalido = errors.New("código de producto inválido")

// Letras, dígitos, "-", "_" y "." (1 a 40 caracteres, sin espacios), siempre en mayúsculas.
var codigoProductoRe = regexp.MustCompile(`^[A-Z0-9._-]{1,40}$`)

// NewCodigoProducto recorta, pasa a mayúscula solo a-z (ToUpper plegaría letras no ASCII) y valida.
func NewCodigoProducto(raw string) (string, error) {
	v := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r - 'a' + 'A'
		}
		return r
	}, strings.TrimSpace(raw))
	if !codigoProductoRe.MatchString(v) {
		return "", ErrCodigoProductoInvalido
	}
	return v, nil
}
