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

type RolAlta struct {
	Codigo      string  `json:"codigo"`
	Nombre      string  `json:"nombre"`
	Descripcion *string `json:"descripcion"`
}

type RolEdicion struct {
	Codigo      *string          `json:"codigo,omitempty"`
	Nombre      *string          `json:"nombre,omitempty"`
	Descripcion Opcional[string] `json:"descripcion"`
}

type PermisoAlta struct {
	Codigo      string `json:"codigo"`
	Descripcion string `json:"descripcion"`
}

type PermisoEdicion struct {
	Codigo      *string `json:"codigo,omitempty"`
	Descripcion *string `json:"descripcion,omitempty"`
}

// UsuarioAlta: nombre se acepta por contrato pero no se persiste (los nombres de personas
// pertenecen a empleados; usuarios solo guarda username y email). Con empleado, el alta crea
// además al empleado vinculado en la misma transacción.
type UsuarioAlta struct {
	Username string         `json:"username"`
	Nombre   *string        `json:"nombre"`
	Email    *string        `json:"email"`
	Password string         `json:"password"`
	Empleado *EmpleadoDatos `json:"empleado"`
}

// Asignaciones: los ids viajan como cadenas, según el contrato.
type Asignaciones struct {
	RolIDs      []string `json:"rol_ids"`
	SucursalIDs []string `json:"sucursal_ids"`
}

type Motivo struct {
	Motivo string `json:"motivo"`
}

// Confirmacion es el cuerpo (vacío) de las restauraciones y de las bajas sin motivo.
type Confirmacion struct{}
