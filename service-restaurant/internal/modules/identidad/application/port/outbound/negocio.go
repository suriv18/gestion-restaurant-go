package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
)

// CampoOpcional es un cambio de un campo anulable: si Presente, se escribe Valor (nil = NULL).
type CampoOpcional struct {
	Presente bool
	Valor    *string
}

// CambiosNegocio: solo los campos presentes se modifican.
type CambiosNegocio struct {
	RazonSocial, NombreComercial, DireccionFiscal, Moneda, ZonaHoraria *string
	Ubigeo, Telefono, Email                                            CampoOpcional
}

// NegocioStore escribe la empresa única. found=false si no existe.
type NegocioStore interface {
	Actualizar(ctx context.Context, c CambiosNegocio) (dtoresult.Negocio, bool, error)
}

// NegocioReader lee la empresa única.
type NegocioReader interface {
	ObtenerNegocio(ctx context.Context) (dtoresult.Negocio, bool, error)
}
