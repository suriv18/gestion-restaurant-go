package vo

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	maxTipoDocumento   = 10
	maxNumeroDocumento = 20
	maxEmail           = 254
)

var (
	tipoDocumentoRe   = regexp.MustCompile(`^[A-Z0-9]{1,10}$`)
	numeroDocumentoRe = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{0,19}$`)
	digitosRe         = regexp.MustCompile(`^[0-9]+$`)
	telefonoRe        = regexp.MustCompile(`^[0-9+()\-. ]{5,30}$`)
)

// longitudes de los documentos peruanos que se validan por su tipo.
var longitudPorTipo = map[string]int{"DNI": 8, "RUC": 11}

// NewDocumento normaliza el par (tipo, número): ambos se recortan y pasan a mayúsculas. El tipo
// admite letras y dígitos (hasta 10); el número, letras, dígitos y guiones (hasta 20). DNI exige 8
// dígitos y RUC 11. Devuelve ErrValorInvalido si el par no cumple.
func NewDocumento(tipo, numero string) (string, string, error) {
	t := strings.ToUpper(strings.TrimSpace(tipo))
	n := strings.ToUpper(strings.TrimSpace(numero))
	if !tipoDocumentoRe.MatchString(t) || !numeroDocumentoRe.MatchString(n) {
		return "", "", ErrValorInvalido
	}
	if largo, ok := longitudPorTipo[t]; ok && (len(n) != largo || !digitosRe.MatchString(n)) {
		return "", "", ErrValorInvalido
	}
	return t, n, nil
}

// NewEmail valida una dirección simple (sin nombre visible) de hasta 254 caracteres.
func NewEmail(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" || utf8.RuneCountInString(v) > maxEmail {
		return "", ErrValorInvalido
	}
	a, err := mail.ParseAddress(v)
	if err != nil || a.Address != v || a.Name != "" || !strings.Contains(v[strings.LastIndex(v, "@"):], ".") {
		return "", ErrValorInvalido
	}
	return v, nil
}

// NewTelefono admite dígitos, espacios y + ( ) - . con 5 a 30 caracteres.
func NewTelefono(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if !telefonoRe.MatchString(v) {
		return "", ErrValorInvalido
	}
	return v, nil
}
