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

// RolesHandler: administración del catálogo de roles (RES-RF-033).
type RolesHandler struct {
	crear   inbound.CrearRol
	editar  inbound.EditarRol
	estado  inbound.CambiarEstadoRol
	listar  inbound.ListarRoles
	obtener inbound.ObtenerRol
}

func NewRolesHandler(c inbound.CrearRol, e inbound.EditarRol, s inbound.CambiarEstadoRol, l inbound.ListarRoles, o inbound.ObtenerRol) *RolesHandler {
	return &RolesHandler{c, e, s, l, o}
}

// Listar: GET /roles?limit=&cursor=
func (h *RolesHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	res, err := h.listar.Execute(r.Context(), query.ListarRoles{Actor: actor, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToRolesLista)
}

// Obtener: GET /roles/{id}
func (h *RolesHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerRol{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToRol)
}

// Crear: POST /roles (Idempotency-Key obligatoria)
func (h *RolesHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.RolAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	in, err := intencion(r, "POST /roles", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearRol{
		Intencion: in, Actor: actor, Codigo: body.Codigo, Nombre: body.Nombre, Descripcion: body.Descripcion,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToRol)
}

// Editar: PATCH /roles/{id}
func (h *RolesHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.RolEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarRol{
		Actor: actor, ID: id, Codigo: body.Codigo, Nombre: body.Nombre,
		Descripcion: body.Descripcion.Valor, DescripcionPresente: body.Descripcion.Presente,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToRol)
}

// Baja: POST /roles/{id}/baja (Idempotency-Key obligatoria)
func (h *RolesHandler) Baja(w http.ResponseWriter, r *http.Request) error {
	return h.cambiarEstado(w, r, false)
}

// Restauracion: POST /roles/{id}/restauracion (Idempotency-Key obligatoria)
func (h *RolesHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
	return h.cambiarEstado(w, r, true)
}

func (h *RolesHandler) cambiarEstado(w http.ResponseWriter, r *http.Request, activo bool) error {
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
	}
	// Ni la baja ni la restauración de un rol llevan datos: se acepta sin cuerpo o con «{}» y se
	// rechaza cualquier campo (el actor sale de la sesión, no del cuerpo).
	var c request.Confirmacion
	if err := leerCuerpoOpcionalVacio(r, &c); err != nil {
		return err
	}
	in, err := intencion(r, fmt.Sprintf("POST /roles/%d/%s", id, accion), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoRol{Intencion: in, Actor: actor, ID: id, Activo: activo})
	return responder(w, res, err, http.StatusOK, webmapper.ToRol)
}
