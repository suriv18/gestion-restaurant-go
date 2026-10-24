// Package request define los cuerpos HTTP del directorio de clientes y proveedores. Un campo
// desconocido responde 400.
package request

import "encoding/json"

// Opcional distingue un campo ausente de uno enviado como null (PATCH).
type Opcional[T any] struct {
	Presente bool
	Valor    *T
}

func (o *Opcional[T]) UnmarshalJSON(b []byte) error {
	o.Presente = true
	if string(b) == "null" {
		o.Valor = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Valor = &v
	return nil
}

// MarshalJSON permite calcular la huella canónica de la solicitud.
func (o Opcional[T]) MarshalJSON() ([]byte, error) {
	if !o.Presente || o.Valor == nil {
		return []byte("null"), nil
	}
	return json.Marshal(o.Valor)
}

// Motivo es el cuerpo de la baja con motivo.
type Motivo struct {
	Motivo string `json:"motivo"`
}

// Confirmacion es el cuerpo (vacío) de las restauraciones.
type Confirmacion struct{}

// ClienteAlta: el documento es opcional, pero tipo y número van juntos.
type ClienteAlta struct {
	TipoDocumento     *string `json:"tipo_documento"`
	NumeroDocumento   *string `json:"numero_documento"`
	NombreRazonSocial string  `json:"nombre_razon_social"`
	Telefono          *string `json:"telefono"`
	Email             *string `json:"email"`
	Direccion         *string `json:"direccion"`
}

type ClienteEdicion struct {
	NombreRazonSocial *string          `json:"nombre_razon_social,omitempty"`
	TipoDocumento     Opcional[string] `json:"tipo_documento"`
	NumeroDocumento   Opcional[string] `json:"numero_documento"`
	Telefono          Opcional[string] `json:"telefono"`
	Email             Opcional[string] `json:"email"`
	Direccion         Opcional[string] `json:"direccion"`
}

type ProveedorAlta struct {
	TipoDocumento   string  `json:"tipo_documento"`
	NumeroDocumento string  `json:"numero_documento"`
	RazonSocial     string  `json:"razon_social"`
	ContactoNombre  *string `json:"contacto_nombre"`
	Telefono        *string `json:"telefono"`
	Email           *string `json:"email"`
	Direccion       *string `json:"direccion"`
}

type ProveedorEdicion struct {
	TipoDocumento   *string          `json:"tipo_documento,omitempty"`
	NumeroDocumento *string          `json:"numero_documento,omitempty"`
	RazonSocial     *string          `json:"razon_social,omitempty"`
	ContactoNombre  Opcional[string] `json:"contacto_nombre"`
	Telefono        Opcional[string] `json:"telefono"`
	Email           Opcional[string] `json:"email"`
	Direccion       Opcional[string] `json:"direccion"`
}
