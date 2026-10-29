// Package mapper convierte errores en rechazos esperados de aplicación y resuelve la
// autorización por sede del módulo produccion.
package mapper

import (
	"encoding/base64"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/shared/kernel/apperror"
)

const (
	CodigoNoAutenticado      = "no_autenticado"
	CodigoSinPermiso         = "sin_permiso"
	CodigoNoEncontrado       = "no_encontrado"
	CodigoDuplicado          = "registro_duplicado"
	CodigoReferenciaInvalida = "referencia_invalida"
	CodigoReglaDatos         = "regla_de_datos"
	CodigoSedeNoAutorizada   = "sede_no_autorizada"
	CodigoCursorInvalido     = "cursor_invalido"
)

func NoAutenticado() *apperror.AppError {
	return apperror.New(apperror.KindUnauthorized, CodigoNoAutenticado, "autenticación requerida")
}

// NoEncontrado: para sustantivos femeninos ("sucursal", "línea de comanda").
func NoEncontrado(que string) *apperror.AppError {
	return apperror.New(apperror.KindNotFound, CodigoNoEncontrado, que+" no encontrada")
}

// NoEncontradoM es como NoEncontrado para sustantivos masculinos ("pedido").
func NoEncontradoM(que string) *apperror.AppError {
	return apperror.New(apperror.KindNotFound, CodigoNoEncontrado, que+" no encontrado")
}

func Validacion(code, msg string) *apperror.AppError {
	return apperror.New(apperror.KindValidation, code, msg)
}

// DeAlmacen traduce los errores de almacenamiento conocidos a rechazos esperados. Devuelve nil si
// err no es uno de ellos (error técnico: debe seguir por el canal error).
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
		return apperror.New(apperror.KindBusinessRule, CodigoReglaDatos, mensajeRegla(regla.Mensaje))
	}
	return nil
}

// referenciaRe reconoce el rechazo del trigger fn_referencias_activas:
// "Referencia estacion_id=12: maestro estaciones_preparacion inexistente o dado de baja".
var referenciaRe = regexp.MustCompile(`^Referencia \w+=\d+: maestro (\w+) inexistente o dado de baja$`)

var maestros = map[string]string{"estaciones_preparacion": "estación", "sucursales": "sucursal"}

// mensajeRegla deja pasar los mensajes curados del esquema y reemplaza por un texto para el
// usuario los que exponen nombres internos del trigger de referencias.
func mensajeRegla(msg string) string {
	if !strings.HasPrefix(msg, "Referencia ") {
		return msg
	}
	if m := referenciaRe.FindStringSubmatch(msg); m != nil {
		if nombre, ok := maestros[m[1]]; ok {
			return "la " + nombre + " indicada no existe o está dada de baja"
		}
	}
	return "una referencia no existe o está dada de baja"
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
