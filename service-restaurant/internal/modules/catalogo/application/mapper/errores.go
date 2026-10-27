// Package mapper convierte errores en rechazos esperados de aplicación y resuelve la
// autorización del directorio de clientes y proveedores.
package mapper

import (
	"encoding/base64"
	"errors"
	"strconv"

	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

const (
	CodigoNoAutenticado      = "no_autenticado"
	CodigoSinPermiso         = "sin_permiso"
	CodigoNoEncontrado       = "no_encontrado"
	CodigoDuplicado          = "registro_duplicado"
	CodigoReferenciaInvalida = "referencia_invalida"
	CodigoReglaDatos         = "regla_de_datos"
	CodigoCursorInvalido     = "cursor_invalido"
)

func NoAutenticado() *apperror.AppError {
	return apperror.New(apperror.KindUnauthorized, CodigoNoAutenticado, "autenticación requerida")
}

// NoEncontrado: "cliente" y "proveedor" son masculinos, por eso "no encontrado".
func NoEncontrado(que string) *apperror.AppError {
	return apperror.New(apperror.KindNotFound, CodigoNoEncontrado, que+" no encontrado")
}

func Validacion(code, msg string) *apperror.AppError {
	return apperror.New(apperror.KindValidation, code, msg)
}

// Requiere exige un actor autenticado con alguno de los permisos en al menos una sede. Los
// maestros del directorio son globales: no hay alcance por sucursal.
func Requiere(a appctx.Actor, mensaje string, permisos ...string) *apperror.AppError {
	if a.IsZero() {
		return NoAutenticado()
	}
	for _, p := range permisos {
		if a.CanAnywhere(p) {
			return nil
		}
	}
	return apperror.New(apperror.KindForbidden, CodigoSinPermiso, mensaje)
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

// EncodeCursor: el cursor es el último id entregado, opaco para el cliente.
func EncodeCursor(id int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))
}

// DecodeCursor acepta solo el id en decimal canónico (sin "+" ni ceros a la izquierda): así cada
// página tiene un único cursor válido.
func DecodeCursor(c string) (int64, *apperror.AppError) {
	if c == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(c)
	if err == nil {
		if id, perr := strconv.ParseInt(string(raw), 10, 64); perr == nil && id >= 0 && strconv.FormatInt(id, 10) == string(raw) {
			return id, nil
		}
	}
	return 0, apperror.New(apperror.KindBadRequest, CodigoCursorInvalido, "cursor inválido")
}
