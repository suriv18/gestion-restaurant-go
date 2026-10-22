// Package vo contiene los objetos de valor del módulo atencion; solo usa la biblioteca estándar.
package vo

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrTextoInvalido = errors.New("texto inválido")
	ErrValorInvalido = errors.New("valor inválido")
)

// TextoObligatorio recorta y exige 1..max caracteres (runas). Admite saltos de línea y
// tabulaciones (p. ej. un motivo en varias líneas); rechaza cualquier otro carácter de control
// (C0, DEL, C1) y los separadores de línea y párrafo U+2028/U+2029.
func TextoObligatorio(raw string, max int) (string, error) {
	return validarTexto(raw, max, true)
}

// TextoUnaLinea es como TextoObligatorio pero además rechaza \n y \t: para nombres y códigos.
func TextoUnaLinea(raw string, max int) (string, error) {
	return validarTexto(raw, max, false)
}

func validarTexto(raw string, max int, multilinea bool) (string, error) {
	v := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(v)
	if n == 0 || n > max {
		return "", ErrTextoInvalido
	}
	for _, r := range v {
		if multilinea && (r == '\n' || r == '\t') {
			continue
		}
		if unicode.IsControl(r) || r == ' ' || r == ' ' {
			return "", ErrTextoInvalido
		}
	}
	return v, nil
}
