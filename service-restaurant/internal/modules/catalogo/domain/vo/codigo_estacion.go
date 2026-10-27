package vo

import (
	"errors"
	"regexp"
	"strings"
)

// ErrCodigoEstacionInvalido señala un código de estación fuera de formato.
var ErrCodigoEstacionInvalido = errors.New("código de estación inválido")

// Mismo criterio que el código de sucursal: mayúsculas, dígitos, _ y -, empezando por letra o
// dígito (1 a 20 caracteres; una estación puede llamarse "A").
var codigoEstacionRe = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{0,19}$`)

// NewCodigoEstacion recorta, pasa a mayúscula solo a-z (strings.ToUpper plegaría letras no ASCII
// como "ı" -> "I" y las dejaría pasar la expresión regular) y valida el formato.
func NewCodigoEstacion(raw string) (string, error) {
	v := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r - 'a' + 'A'
		}
		return r
	}, strings.TrimSpace(raw))
	if !codigoEstacionRe.MatchString(v) {
		return "", ErrCodigoEstacionInvalido
	}
	return v, nil
}
