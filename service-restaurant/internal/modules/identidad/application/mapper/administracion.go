package mapper

import (
	"encoding/base64"
	"errors"
	"strconv"

	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

const (
	CodigoSinPermiso          = "sin_permiso"
	CodigoNoEncontrado        = "no_encontrado"
	CodigoDuplicado           = "registro_duplicado"
	CodigoReferenciaInvalida  = "referencia_invalida"
	CodigoReglaDatos          = "regla_de_datos"
	CodigoUltimoAdministrador = "ultimo_administrador"
	CodigoAutoBaja            = "no_autobaja"
	CodigoSedeNoAutorizada    = "sede_no_autorizada"
	CodigoCursorInvalido      = "cursor_invalido"
)

// RequiereAdministrar exige un actor autenticado con USUARIO_ADMINISTRAR en alguna sede.
func RequiereAdministrar(a appctx.Actor) *apperror.AppError {
	if a.IsZero() {
		return NoAutenticado()
	}
	if !a.CanAnywhere(policy.UsuarioAdministrar) {
		return apperror.New(apperror.KindForbidden, CodigoSinPermiso, "no tiene permiso para administrar accesos")
	}
	return nil
}

func NoEncontrado(que string) *apperror.AppError {
	return apperror.New(apperror.KindNotFound, CodigoNoEncontrado, que+" no encontrado")
}

func Validacion(code, msg string) *apperror.AppError {
	return apperror.New(apperror.KindValidation, code, msg)
}

// DeAlmacen traduce los errores de almacenamiento conocidos a rechazos esperados.
// Devuelve nil si err no es uno de ellos (error técnico: debe seguir por el canal error).
func DeAlmacen(err error) *apperror.AppError {
	var regla *outbound.ErrReglaDatos
	switch {
	case err == nil:
		return nil
	case errors.Is(err, outbound.ErrDuplicado):
		return apperror.New(apperror.KindConflict, CodigoDuplicado, "ya existe un registro con ese valor único")
	case errors.Is(err, outbound.ErrReferenciaInvalida):
		return apperror.New(apperror.KindValidation, CodigoReferenciaInvalida, "una referencia no existe")
	case errors.As(err, &regla):
		return apperror.New(apperror.KindBusinessRule, CodigoReglaDatos, regla.Mensaje)
	}
	return nil
}

// EncodeCursor / DecodeCursor: el cursor es el último id entregado, opaco para el cliente.
func EncodeCursor(id int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))
}

func DecodeCursor(c string) (int64, *apperror.AppError) {
	if c == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(c)
	if err == nil {
		if id, perr := strconv.ParseInt(string(raw), 10, 64); perr == nil && id >= 0 {
			return id, nil
		}
	}
	return 0, apperror.New(apperror.KindBadRequest, CodigoCursorInvalido, "cursor inválido")
}
