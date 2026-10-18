package controller

import (
	"fmt"
	"net/http"

	"service-restaurant/internal/modules/identidad/application/dto/command"
	"service-restaurant/internal/modules/identidad/application/dto/query"
	"service-restaurant/internal/modules/identidad/application/port/inbound"
	"service-restaurant/internal/modules/identidad/web/dto/request"
	webmapper "service-restaurant/internal/modules/identidad/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
)

// PermisosHandler: administración del catálogo de permisos (RES-RF-033).
type PermisosHandler struct {
	crear   inbound.CrearPermiso
	editar  inbound.EditarPermiso
	estado  inbound.CambiarEstadoPermiso
	listar  inbound.ListarPermisos
	obtener inbound.ObtenerPermiso
}

func NewPermisosHandler(c inbound.CrearPermiso, e inbound.EditarPermiso, s inbound.CambiarEstadoPermiso, l inbound.ListarPermisos, o inbound.ObtenerPermiso) *PermisosHandler {
	return &PermisosHandler{c, e, s, l, o}
}

func (h *PermisosHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	res, err := h.listar.Execute(r.Context(), query.ListarPermisos{Actor: actor, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToPermisosLista)
}

func (h *PermisosHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerPermiso{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToPermiso)
}

func (h *PermisosHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.PermisoAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	in, err := intencion(r, "POST /permisos", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearPermiso{
		Intencion: in, Actor: actor, Codigo: body.Codigo, Descripcion: body.Descripcion,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToPermiso)
}

func (h *PermisosHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.PermisoEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarPermiso{Actor: actor, ID: id, Codigo: body.Codigo, Descripcion: body.Descripcion})
	return responder(w, res, err, http.StatusOK, webmapper.ToPermiso)
}

func (h *PermisosHandler) Baja(w http.ResponseWriter, r *http.Request) error {
	return h.cambiarEstado(w, r, false)
}

func (h *PermisosHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
	return h.cambiarEstado(w, r, true)
}

func (h *PermisosHandler) cambiarEstado(w http.ResponseWriter, r *http.Request, activo bool) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	accion := "baja"
	if activo {
		accion = "restauracion"
		var c request.Confirmacion
		if err := leerCuerpoOpcionalVacio(r, &c); err != nil {
			return err
		}
	}
	in, err := intencion(r, fmt.Sprintf("POST /permisos/%d/%s", id, accion), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoPermiso{Intencion: in, Actor: actor, ID: id, Activo: activo})
	return responder(w, res, err, http.StatusOK, webmapper.ToPermiso)
}
