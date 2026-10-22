// Package request define los cuerpos HTTP del módulo atencion.
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

// Motivo es el cuerpo de la baja con motivo (mesas).
type Motivo struct {
	Motivo string `json:"motivo"`
}

// Confirmacion es el cuerpo (vacío) de las restauraciones y de la baja simple de zonas.
type Confirmacion struct{}
