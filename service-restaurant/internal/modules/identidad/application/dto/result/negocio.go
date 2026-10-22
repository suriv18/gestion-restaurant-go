package result

import "time"

// Negocio es la empresa única del despliegue (RES-RF-003). El RUC no se edita.
type Negocio struct {
	ID              int64
	RUC             string
	RazonSocial     string
	NombreComercial string
	DireccionFiscal string
	Ubigeo          *string
	Telefono        *string
	Email           *string
	Moneda          string
	ZonaHoraria     string
	EsActivo        bool
	CreadoEn        time.Time
	CreadoPor       int64
	ActualizadoEn   time.Time
	ActualizadoPor  int64
}
