package policy

import (
	"unicode"
	"unicode/utf8"

	"service-restaurant/internal/modules/identidad/domain/domainerror"
)

const (
	ClaveMinima = 12
	// ClaveMaxima acota el trabajo de argon2 ante entradas abusivas.
	ClaveMaxima = 128
)

// ValidarClaveNueva exige longitud y mezcla de letras y dígitos al crear una contraseña.
// No se aplica al login: allí una clave fuera de rango simplemente no coincide.
func ValidarClaveNueva(clave string) error {
	n := utf8.RuneCountInString(clave)
	if n < ClaveMinima || n > ClaveMaxima {
		return domainerror.New("clave_debil", "la contraseña debe tener entre 12 y 128 caracteres")
	}
	var letra, digito bool
	for _, r := range clave {
		switch {
		case unicode.IsLetter(r):
			letra = true
		case unicode.IsDigit(r):
			digito = true
		}
	}
	if !letra || !digito {
		return domainerror.New("clave_debil", "la contraseña debe combinar letras y números")
	}
	return nil
}

// ClaveAdmisibleParaLogin descarta de entrada claves que no pueden ser válidas.
func ClaveAdmisibleParaLogin(clave string) bool {
	n := utf8.RuneCountInString(clave)
	return n >= 1 && n <= ClaveMaxima
}
