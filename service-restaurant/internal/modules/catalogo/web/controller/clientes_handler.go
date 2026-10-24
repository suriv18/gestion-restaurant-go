package controller

import (
	"fmt"
	"net/http"

	"service-restaurant/internal/modules/catalogo/application/dto/command"
	"service-restaurant/internal/modules/catalogo/application/dto/query"
	"service-restaurant/internal/modules/catalogo/application/port/inbound"
	"service-restaurant/internal/modules/catalogo/web/dto/request"
	webmapper "service-restaurant/internal/modules/catalogo/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
)

// ClientesHandler: clientes del restaurante (RES-RF-004).
type ClientesHandler struct {
	listar  inbound.ListarClientes
	obtener inbound.ObtenerCliente
	crear   inbound.CrearCliente
	editar  inbound.EditarCliente
	estado  inbound.CambiarEstadoCliente
}

func NewClientesHandler(l inbound.ListarClientes, o inbound.ObtenerCliente, c inbound.CrearCliente,
	e inbound.EditarCliente, s inbound.CambiarEstadoCliente) *ClientesHandler {
	return &ClientesHandler{l, o, c, e, s}
}

// Listar: GET /clientes?q=&es_activo=&limit=&cursor=
func (h *ClientesHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	res, err := h.listar.Execute(r.Context(), query.ListarClientes{
		Actor: actor, Activo: activo, Texto: q.Get("q"), Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToClientesPagina)
}

// Obtener: GET /clientes/{id}
func (h *ClientesHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerCliente{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToCliente)
}

// Crear: POST /clientes (Idempotency-Key obligatoria)
func (h *ClientesHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.ClienteAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	in, err := intencion(r, "POST /clientes", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearCliente{
		Intencion: in, Actor: actor, TipoDocumento: body.TipoDocumento, NumeroDocumento: body.NumeroDocumento,
		Nombre: body.NombreRazonSocial, Telefono: body.Telefono, Email: body.Email, Direccion: body.Direccion,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToCliente)
}

// Editar: PATCH /clientes/{id}
func (h *ClientesHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.ClienteEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarCliente{
		Actor: actor, ID: id, Nombre: body.NombreRazonSocial,
		TipoDocumento:   campo(body.TipoDocumento.Presente, body.TipoDocumento.Valor),
		NumeroDocumento: campo(body.NumeroDocumento.Presente, body.NumeroDocumento.Valor),
		Telefono:        campo(body.Telefono.Presente, body.Telefono.Valor),
		Email:           campo(body.Email.Presente, body.Email.Valor),
		Direccion:       campo(body.Direccion.Presente, body.Direccion.Valor),
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToCliente)
}

// Baja: POST /clientes/{id}/baja con {"motivo": "..."} (Idempotency-Key obligatoria)
func (h *ClientesHandler) Baja(w http.ResponseWriter, r *http.Request) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /clientes/%d/baja", id), body)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoCliente{
		Intencion: in, Actor: actor, ID: id, Activo: false, Motivo: body.Motivo})
	return responder(w, res, err, http.StatusOK, webmapper.ToCliente)
}

// Restauracion: POST /clientes/{id}/restauracion (Idempotency-Key obligatoria)
func (h *ClientesHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /clientes/%d/restauracion", id), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoCliente{Intencion: in, Actor: actor, ID: id, Activo: true})
	return responder(w, res, err, http.StatusOK, webmapper.ToCliente)
}
