// Package response define las respuestas HTTP del directorio; los ids viajan como cadenas.
package response

import "time"

type Cliente struct {
	ID                string     `json:"id"`
	TipoDocumento     *string    `json:"tipo_documento"`
	NumeroDocumento   *string    `json:"numero_documento"`
	NombreRazonSocial string     `json:"nombre_razon_social"`
	Telefono          *string    `json:"telefono"`
	Email             *string    `json:"email"`
	Direccion         *string    `json:"direccion"`
	EsActivo          bool       `json:"es_activo"`
	MotivoEliminacion *string    `json:"motivo_eliminacion"`
	EliminadoEn       *time.Time `json:"eliminado_en"`
	EliminadoPor      *string    `json:"eliminado_por"`
	CreadoEn          time.Time  `json:"creado_en"`
	CreadoPor         string     `json:"creado_por"`
	ActualizadoEn     time.Time  `json:"actualizado_en"`
	ActualizadoPor    string     `json:"actualizado_por"`
}

type ClientesPagina struct {
	Items      []Cliente `json:"items"`
	NextCursor *string   `json:"nextCursor"`
}

type Proveedor struct {
	ID                string     `json:"id"`
	TipoDocumento     string     `json:"tipo_documento"`
	NumeroDocumento   string     `json:"numero_documento"`
	RazonSocial       string     `json:"razon_social"`
	ContactoNombre    *string    `json:"contacto_nombre"`
	Telefono          *string    `json:"telefono"`
	Email             *string    `json:"email"`
	Direccion         *string    `json:"direccion"`
	EsActivo          bool       `json:"es_activo"`
	MotivoEliminacion *string    `json:"motivo_eliminacion"`
	EliminadoEn       *time.Time `json:"eliminado_en"`
	EliminadoPor      *string    `json:"eliminado_por"`
	CreadoEn          time.Time  `json:"creado_en"`
	CreadoPor         string     `json:"creado_por"`
	ActualizadoEn     time.Time  `json:"actualizado_en"`
	ActualizadoPor    string     `json:"actualizado_por"`
}

type ProveedoresPagina struct {
	Items      []Proveedor `json:"items"`
	NextCursor *string     `json:"nextCursor"`
}
