package vo

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ErrImagenURLInvalida señala una URL de imagen vacía, demasiado larga o de otro esquema.
var ErrImagenURLInvalida = errors.New("URL de imagen inválida")

const maxImagenURL = 2000

// NewImagenURL recorta y exige una URL absoluta http/https con host, de hasta 2000 caracteres y
// sin espacios ni caracteres de control.
func NewImagenURL(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if n := utf8.RuneCountInString(v); n == 0 || n > maxImagenURL {
		return "", ErrImagenURLInvalida
	}
	for _, r := range v {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return "", ErrImagenURLInvalida
		}
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return "", ErrImagenURLInvalida
	}
	return v, nil
}
