package response

import "time"

// Mesa: las posiciones son decimales exactos serializados como cadena ("12.50").
type Mesa struct {
	ID                string     `json:"id"`
	SucursalID        string     `json:"sucursal_id"`
	ZonaID            string     `json:"zona_id"`
	NumeroMesa        string     `json:"numero_mesa"`
	Capacidad         int        `json:"capacidad"`
	Habilitada        bool       `json:"habilitada"`
	PosicionX         *string    `json:"posicion_x"`
	PosicionY         *string    `json:"posicion_y"`
	EsActivo          bool       `json:"es_activo"`
	MotivoEliminacion *string    `json:"motivo_eliminacion"`
	EliminadoEn       *time.Time `json:"eliminado_en"`
	EliminadoPor      *string    `json:"eliminado_por"`
	CreadoEn          time.Time  `json:"creado_en"`
	CreadoPor         string     `json:"creado_por"`
	ActualizadoEn     time.Time  `json:"actualizado_en"`
	ActualizadoPor    string     `json:"actualizado_por"`
}

type MesasPagina struct {
	Items      []Mesa  `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
