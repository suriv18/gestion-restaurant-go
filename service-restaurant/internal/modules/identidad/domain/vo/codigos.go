package vo

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	ErrCodigoInvalido = errors.New("código inválido")
	ErrTextoInvalido  = errors.New("texto inválido")

	codigoRolRe     = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,39}$`)
	codigoPermisoRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,79}$`)
)

// CodigoRol es el código de un rol: mayúsculas, dígitos y guion bajo (2 a 40).
type CodigoRol struct{ value string }

func NewCodigoRol(raw string) (CodigoRol, error) {
	v := strings.ToUpper(strings.TrimSpace(raw))
	if !codigoRolRe.MatchString(v) {
		return CodigoRol{}, ErrCodigoInvalido
	}
	return CodigoRol{v}, nil
}

func (c CodigoRol) String() string { return c.value }

// CodigoPermiso es el código de un permiso: mayúsculas, dígitos y guion bajo (2 a 80).
type CodigoPermiso struct{ value string }

func NewCodigoPermiso(raw string) (CodigoPermiso, error) {
	v := strings.ToUpper(strings.TrimSpace(raw))
	if !codigoPermisoRe.MatchString(v) {
		return CodigoPermiso{}, ErrCodigoInvalido
	}
	return CodigoPermiso{v}, nil
}

func (c CodigoPermiso) String() string { return c.value }

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
