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

type Sucursal struct {
	ID                         string     `json:"id"`
	NegocioID                  string     `json:"negocio_id"`
	Codigo                     string     `json:"codigo"`
	Nombre                     string     `json:"nombre"`
	Direccion                  string     `json:"direccion"`
	Ubigeo                     *string    `json:"ubigeo"`
	CodigoEstablecimientoSunat *string    `json:"codigo_establecimiento_sunat"`
	Telefono                   *string    `json:"telefono"`
	EsActivo                   bool       `json:"es_activo"`
	MotivoEliminacion          *string    `json:"motivo_eliminacion"`
	EliminadoEn                *time.Time `json:"eliminado_en"`
	EliminadoPor               *string    `json:"eliminado_por"`
	CreadoEn                   time.Time  `json:"creado_en"`
	CreadoPor                  string     `json:"creado_por"`
	ActualizadoEn              time.Time  `json:"actualizado_en"`
	ActualizadoPor             string     `json:"actualizado_por"`
}

type SucursalesPagina struct {
	Items      []Sucursal `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}
