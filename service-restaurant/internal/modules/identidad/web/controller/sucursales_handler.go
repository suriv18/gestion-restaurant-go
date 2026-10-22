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

// SucursalesHandler: sedes del negocio (RES-RF-003).
type SucursalesHandler struct {
	listar  inbound.ListarSucursales
	obtener inbound.ObtenerSucursal
	crear   inbound.CrearSucursal
	editar  inbound.EditarSucursal
	estado  inbound.CambiarEstadoSucursal
}

func NewSucursalesHandler(l inbound.ListarSucursales, o inbound.ObtenerSucursal, c inbound.CrearSucursal,
	e inbound.EditarSucursal, s inbound.CambiarEstadoSucursal) *SucursalesHandler {
	return &SucursalesHandler{l, o, c, e, s}
}

// Listar: GET /sucursales?limit=&cursor=&es_activo=
func (h *SucursalesHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	res, err := h.listar.Execute(r.Context(), query.ListarSucursales{Actor: actor, Limite: limite, Cursor: q.Get("cursor"), Activo: activo})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursalesPagina)
}

// Obtener: GET /sucursales/{id}
func (h *SucursalesHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerSucursal{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursal)
}

// Crear: POST /sucursales (Idempotency-Key obligatoria)
func (h *SucursalesHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.SucursalAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	in, err := intencion(r, "POST /sucursales", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearSucursal{
		Intencion: in, Actor: actor, Codigo: body.Codigo, Nombre: body.Nombre, Direccion: body.Direccion,
		Ubigeo: body.Ubigeo, CodigoEstablecimientoSunat: body.CodigoEstablecimientoSunat, Telefono: body.Telefono,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToSucursal)
}

// Editar: PATCH /sucursales/{id}
func (h *SucursalesHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.SucursalEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarSucursal{
		Actor: actor, ID: id, Codigo: body.Codigo, Nombre: body.Nombre, Direccion: body.Direccion,
		Ubigeo:                     command.Campo[string]{Presente: body.Ubigeo.Presente, Valor: body.Ubigeo.Valor},
		CodigoEstablecimientoSunat: command.Campo[string]{Presente: body.CodigoEstablecimientoSunat.Presente, Valor: body.CodigoEstablecimientoSunat.Valor},
		Telefono:                   command.Campo[string]{Presente: body.Telefono.Presente, Valor: body.Telefono.Valor},
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursal)
}

// Baja: POST /sucursales/{id}/baja con {"motivo": "..."} (Idempotency-Key obligatoria)
func (h *SucursalesHandler) Baja(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.Motivo
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	in, err := intencion(r, fmt.Sprintf("POST /sucursales/%d/baja", id), body)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoSucursal{
		Intencion: in, Actor: actor, ID: id, Activo: false, Motivo: body.Motivo,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursal)
}

// Restauracion: POST /sucursales/{id}/restauracion (Idempotency-Key obligatoria)
func (h *SucursalesHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /sucursales/%d/restauracion", id), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoSucursal{Intencion: in, Actor: actor, ID: id, Activo: true})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursal)
}
