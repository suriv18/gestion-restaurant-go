// Package vo contiene value objects del dominio de identidad (solo biblioteca estándar).
package vo

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxUsername coincide con VARCHAR(80) de usuarios.username.
const MaxUsername = 80

var ErrUsernameInvalid = errors.New("username inválido")

// Username es el nombre de acceso ya validado.
type Username struct{ value string }

// NewUsername rechaza vacíos, demasiado largos o con caracteres de control.
func NewUsername(raw string) (Username, error) {
	v := strings.TrimSpace(raw)
	if v == "" || utf8.RuneCountInString(v) > MaxUsername {
		return Username{}, ErrUsernameInvalid
	}
	for _, r := range v {
		if unicode.IsControl(r) {
			return Username{}, ErrUsernameInvalid
		}
	}
	return Username{value: v}, nil
}

func (u Username) String() string { return u.value }

// Key es la forma de comparación: coincide con el índice único lower(btrim(username)).
func (u Username) Key() string { return strings.ToLower(u.value) }
