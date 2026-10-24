package command

import (
	"errors"
	"fmt"

	"service-restaurant/internal/modules/catalogo/application/mapper"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	CodigoDocumentoDuplicado = "documento_duplicado"
	CodigoYaDadaDeBaja       = "ya_dada_de_baja"
	CodigoYaActiva           = "ya_activa"
	CodigoClienteInactivo    = "cliente_inactivo"
	CodigoProveedorInactivo  = "proveedor_inactivo"
)

// rechazar devuelve un rechazo esperado (no un error técnico).
func rechazar[T any](ae *apperror.AppError) (appresult.Result[T], error) {
	return appresult.Fail[T](ae), nil
}

// tecnico envuelve un error inesperado; el Result queda en su valor cero (inválido).
func tecnico[T any](contexto string, err error) (appresult.Result[T], error) {
	return appresult.Result[T]{}, fmt.Errorf("%s: %w", contexto, err)
}

// deAlmacen separa un error de almacenamiento conocido (rechazo) de uno técnico. El documento
// repetido tiene un rechazo propio con el mensaje del maestro.
func deAlmacen[T any](contexto, mensajeDocumento string, err error) (appresult.Result[T], error) {
	if errors.Is(err, outbound.ErrDocumentoDuplicado) {
		return rechazar[T](apperror.New(apperror.KindConflict, CodigoDocumentoDuplicado, mensajeDocumento))
	}
	if ae := mapper.DeAlmacen(err); ae != nil {
		return rechazar[T](ae)
	}
	return tecnico[T](contexto, err)
}
