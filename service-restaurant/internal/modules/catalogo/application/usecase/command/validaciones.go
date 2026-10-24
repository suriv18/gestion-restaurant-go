package command

import (
	"strings"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	"service-restaurant/internal/modules/catalogo/application/mapper"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/vo"
	"service-restaurant/internal/shared/kernel/apperror"
)

const (
	maxNombre    = 250
	maxContacto  = 150
	maxDireccion = 500
	maxMotivo    = 2000
)

// Validaciones de campos compartidas por el alta y la edición: cada una devuelve el valor ya
// normalizado o el rechazo de validación (422) con el código y el mensaje del campo.

func validarNombre(raw, campo string) (string, *apperror.AppError) {
	n, err := vo.TextoUnaLinea(raw, maxNombre)
	if err != nil {
		return "", mapper.Validacion(campo+"_invalido", "el nombre es obligatorio (máximo 250 caracteres)")
	}
	return n, nil
}

func validarMotivo(raw string) (string, *apperror.AppError) {
	m, err := vo.TextoObligatorio(raw, maxMotivo)
	if err != nil {
		return "", mapper.Validacion("motivo_requerido", "la baja exige un motivo (máximo 2000 caracteres)")
	}
	return m, nil
}

func validarDocumento(tipo, numero string) (string, string, *apperror.AppError) {
	t, n, err := vo.NewDocumento(tipo, numero)
	if err != nil {
		return "", "", mapper.Validacion("documento_invalido",
			"el documento no es válido: el tipo admite letras y dígitos (hasta 10), el número hasta 20 caracteres; DNI exige 8 dígitos y RUC 11")
	}
	return t, n, nil
}

// formato valida y normaliza un texto no vacío.
type formato func(string) (string, error)

func texto(max int, multilinea bool) formato {
	return func(s string) (string, error) {
		if multilinea {
			return vo.TextoObligatorio(s, max)
		}
		return vo.TextoUnaLinea(s, max)
	}
}

var (
	fmtTelefono  formato = vo.NewTelefono
	fmtEmail     formato = vo.NewEmail
	fmtDireccion         = texto(maxDireccion, true)
	fmtContacto          = texto(maxContacto, false)
)

func mensajeDe(campo string) string {
	switch campo {
	case "telefono":
		return "el teléfono admite dígitos, espacios y + ( ) - . (5 a 30 caracteres)"
	case "email":
		return "el correo no es válido (máximo 254 caracteres)"
	case "direccion":
		return "la dirección admite hasta 500 caracteres"
	case "contacto_nombre":
		return "el nombre de contacto admite hasta 150 caracteres"
	}
	return "valor inválido"
}

// opcional normaliza un campo anulable del alta: nil o texto en blanco → nil (sin valor).
func opcional(raw *string, campo string, f formato) (*string, *apperror.AppError) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	v, err := f(*raw)
	if err != nil {
		return nil, mapper.Validacion(campo+"_invalido", mensajeDe(campo))
	}
	return &v, nil
}

// campoOpcional normaliza un campo anulable de un PATCH: ausente → no se toca; null o texto en
// blanco → se vacía; con valor → se valida y se normaliza.
func campoOpcional(c cmd.Campo[string], campo string, f formato) (outbound.CampoOpcional, *apperror.AppError) {
	if !c.Presente {
		return outbound.CampoOpcional{}, nil
	}
	v, ae := opcional(c.Valor, campo, f)
	if ae != nil {
		return outbound.CampoOpcional{}, ae
	}
	return outbound.CampoOpcional{Presente: true, Valor: v}, nil
}

// vacio indica que el texto opcional no trae valor.
func vacio(p *string) bool { return p == nil || strings.TrimSpace(*p) == "" }
