// Package command implementa los comandos del módulo produccion.
package command

import (
	"fmt"

	"service-restaurant/internal/modules/produccion/application/mapper"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// rechazar devuelve un rechazo esperado (no un error técnico).
func rechazar[T any](ae *apperror.AppError) (appresult.Result[T], error) {
	return appresult.Fail[T](ae), nil
}

// tecnico envuelve un error inesperado; el Result queda en su valor cero (inválido).
func tecnico[T any](contexto string, err error) (appresult.Result[T], error) {
	return appresult.Result[T]{}, fmt.Errorf("%s: %w", contexto, err)
}

// deAlmacen separa un error de almacenamiento conocido (rechazo) de uno técnico.
func deAlmacen[T any](contexto string, err error) (appresult.Result[T], error) {
	if ae := mapper.DeAlmacen(err); ae != nil {
		return rechazar[T](ae)
	}
	return tecnico[T](contexto, err)
}
