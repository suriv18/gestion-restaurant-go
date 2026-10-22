package controller

import (
	"fmt"
	"net/http"

	"service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/application/dto/query"
	"service-restaurant/internal/modules/atencion/application/port/inbound"
	"service-restaurant/internal/modules/atencion/web/dto/request"
	webmapper "service-restaurant/internal/modules/atencion/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
)

// ZonasHandler: salones, terrazas y niveles del local (RES-RF-008).
type ZonasHandler struct {
	listar  inbound.ListarZonas
	obtener inbound.ObtenerZona
	crear   inbound.CrearZona
	editar  inbound.EditarZona
	estado  inbound.CambiarEstadoZona
}

func NewZonasHandler(l inbound.ListarZonas, o inbound.ObtenerZona, c inbound.CrearZona,
	e inbound.EditarZona, s inbound.CambiarEstadoZona) *ZonasHandler {
	return &ZonasHandler{l, o, c, e, s}
}

// Listar: GET /zonas?sucursal_id=&es_activo=&limit=&cursor=
func (h *ZonasHandler) Listar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	q, err := consulta(r)
	if err != nil {
		return err
	}
	limite, err := limiteDe(q)
	if err != nil {
		return err
	}
	activo, err := activoDe(q)
	if err != nil {
		return err
	}
	sede, err := sucursalDe(q)
	if err != nil {
		return err
	}
	res, err := h.listar.Execute(r.Context(), query.ListarZonas{
		Actor: actor, SucursalID: sede, Activo: activo, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToZonasPagina)
}

// Obtener: GET /zonas/{id}
func (h *ZonasHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerZona{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToZona)
}

// Crear: POST /zonas (Idempotency-Key obligatoria)
func (h *ZonasHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.ZonaAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	sede, err := idCadena(body.SucursalID, "sucursal_id")
	if err != nil {
		return err
	}
	in, err := intencion(r, "POST /zonas", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearZona{
		Intencion: in, Actor: actor, SucursalID: sede, Nombre: body.Nombre, OrdenVisualizacion: body.OrdenVisualizacion,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToZona)
}

// Editar: PATCH /zonas/{id}
func (h *ZonasHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.ZonaEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarZona{
		Actor: actor, ID: id, Nombre: body.Nombre, OrdenVisualizacion: body.OrdenVisualizacion,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToZona)
}

// Baja: POST /zonas/{id}/baja (sin motivo; Idempotency-Key obligatoria)
func (h *ZonasHandler) Baja(w http.ResponseWriter, r *http.Request) error {
	return h.cambiarEstado(w, r, false)
}

// Restauracion: POST /zonas/{id}/restauracion (Idempotency-Key obligatoria)
func (h *ZonasHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
	return h.cambiarEstado(w, r, true)
}

func (h *ZonasHandler) cambiarEstado(w http.ResponseWriter, r *http.Request, activo bool) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var c request.Confirmacion
	if err := leerCuerpoOpcionalVacio(r, &c); err != nil {
		return err
	}
	accion := "baja"
	if activo {
		accion = "restauracion"
	}
	in, err := intencion(r, fmt.Sprintf("POST /zonas/%d/%s", id, accion), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoZona{Intencion: in, Actor: actor, ID: id, Activo: activo})
	return responder(w, res, err, http.StatusOK, webmapper.ToZona)
}
