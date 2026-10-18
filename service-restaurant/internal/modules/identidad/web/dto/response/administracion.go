package response

import "time"

type Rol struct {
	ID             string    `json:"id"`
	Codigo         string    `json:"codigo"`
	Nombre         string    `json:"nombre"`
	Descripcion    *string   `json:"descripcion"`
	EsActivo       bool      `json:"es_activo"`
	CreadoEn       time.Time `json:"creado_en"`
	CreadoPor      string    `json:"creado_por"`
	ActualizadoEn  time.Time `json:"actualizado_en"`
	ActualizadoPor string    `json:"actualizado_por"`
}

type Permiso struct {
	ID             string    `json:"id"`
	Codigo         string    `json:"codigo"`
	Descripcion    string    `json:"descripcion"`
	EsActivo       bool      `json:"es_activo"`
	CreadoEn       time.Time `json:"creado_en"`
	CreadoPor      string    `json:"creado_por"`
	ActualizadoEn  time.Time `json:"actualizado_en"`
	ActualizadoPor string    `json:"actualizado_por"`
}

// Usuario nunca incluye el hash de la contraseña.
type Usuario struct {
	ID               string     `json:"id"`
	Username         string     `json:"username"`
	Email            *string    `json:"email"`
	EsSistema        bool       `json:"es_sistema"`
	EsActivo         bool       `json:"es_activo"`
	UltimoAccesoEn   *time.Time `json:"ultimo_acceso_en"`
	BloqueadoHasta   *time.Time `json:"bloqueado_hasta"`
	IntentosFallidos int        `json:"intentos_fallidos"`
	CreadoEn         time.Time  `json:"creado_en"`
	CreadoPor        string     `json:"creado_por"`
	ActualizadoEn    time.Time  `json:"actualizado_en"`
	ActualizadoPor   string     `json:"actualizado_por"`
}

type RolesLista struct {
	Items      []Rol   `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

type PermisosLista struct {
	Items      []Permiso `json:"items"`
	NextCursor *string   `json:"nextCursor"`
}
