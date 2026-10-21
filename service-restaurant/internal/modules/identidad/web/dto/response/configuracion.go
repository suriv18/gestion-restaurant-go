package response

import "time"

type Negocio struct {
	ID              string    `json:"id"`
	Singleton       bool      `json:"singleton"`
	RUC             string    `json:"ruc"`
	RazonSocial     string    `json:"razon_social"`
	NombreComercial string    `json:"nombre_comercial"`
	DireccionFiscal string    `json:"direccion_fiscal"`
	Ubigeo          *string   `json:"ubigeo"`
	Telefono        *string   `json:"telefono"`
	Email           *string   `json:"email"`
	Moneda          string    `json:"moneda"`
	ZonaHoraria     string    `json:"zona_horaria"`
	EsActivo        bool      `json:"es_activo"`
	CreadoEn        time.Time `json:"creado_en"`
	CreadoPor       string    `json:"creado_por"`
	ActualizadoEn   time.Time `json:"actualizado_en"`
	ActualizadoPor  string    `json:"actualizado_por"`
}
