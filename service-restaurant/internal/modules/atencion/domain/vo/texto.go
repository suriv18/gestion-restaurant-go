// Package vo contiene los objetos de valor del módulo atencion; solo usa la biblioteca estándar.
package vo

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrTextoInvalido = errors.New("texto inválido")
	ErrValorInvalido = errors.New("valor inválido")
)

// TextoObligatorio recorta y exige 1..max caracteres sin caracteres de control.
func TextoObligatorio(raw string, max int) (string, error) {
	v := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(v)
	if n == 0 || n > max {
		return "", ErrTextoInvalido
	}
	for _, r := range v {
		if r < 0x20 && r != '\n' && r != '\t' {
			return "", ErrTextoInvalido
		}
	}
	return v, nil
}
