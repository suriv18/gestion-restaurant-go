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

// ProveedoresHandler: proveedores del restaurante (RES-RF-004).
type ProveedoresHandler struct {
	listar  inbound.ListarProveedores
	obtener inbound.ObtenerProveedor
	crear   inbound.CrearProveedor
	editar  inbound.EditarProveedor
	estado  inbound.CambiarEstadoProveedor
}

func NewProveedoresHandler(l inbound.ListarProveedores, o inbound.ObtenerProveedor, c inbound.CrearProveedor,
	e inbound.EditarProveedor, s inbound.CambiarEstadoProveedor) *ProveedoresHandler {
	return &ProveedoresHandler{l, o, c, e, s}
}

// Listar: GET /proveedores?q=&es_activo=&limit=&cursor=
func (h *ProveedoresHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	res, err := h.listar.Execute(r.Context(), query.ListarProveedores{
		Actor: actor, Activo: activo, Texto: q.Get("q"), Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToProveedoresPagina)
}

// Obtener: GET /proveedores/{id}
func (h *ProveedoresHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerProveedor{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToProveedor)
}

// Crear: POST /proveedores (Idempotency-Key obligatoria)
func (h *ProveedoresHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.ProveedorAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	in, err := intencion(r, "POST /proveedores", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearProveedor{
		Intencion: in, Actor: actor, TipoDocumento: body.TipoDocumento, NumeroDocumento: body.NumeroDocumento,
		RazonSocial: body.RazonSocial, ContactoNombre: body.ContactoNombre, Telefono: body.Telefono,
		Email: body.Email, Direccion: body.Direccion,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToProveedor)
}

// Editar: PATCH /proveedores/{id}
func (h *ProveedoresHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.ProveedorEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarProveedor{
		Actor: actor, ID: id, TipoDocumento: body.TipoDocumento, NumeroDocumento: body.NumeroDocumento,
		RazonSocial:    body.RazonSocial,
		ContactoNombre: campo(body.ContactoNombre.Presente, body.ContactoNombre.Valor),
		Telefono:       campo(body.Telefono.Presente, body.Telefono.Valor),
		Email:          campo(body.Email.Presente, body.Email.Valor),
		Direccion:      campo(body.Direccion.Presente, body.Direccion.Valor),
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToProveedor)
}

// Baja: POST /proveedores/{id}/baja con {"motivo": "..."} (Idempotency-Key obligatoria)
func (h *ProveedoresHandler) Baja(w http.ResponseWriter, r *http.Request) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /proveedores/%d/baja", id), body)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoProveedor{
		Intencion: in, Actor: actor, ID: id, Activo: false, Motivo: body.Motivo})
	return responder(w, res, err, http.StatusOK, webmapper.ToProveedor)
}

// Restauracion: POST /proveedores/{id}/restauracion (Idempotency-Key obligatoria)
func (h *ProveedoresHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /proveedores/%d/restauracion", id), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoProveedor{Intencion: in, Actor: actor, ID: id, Activo: true})
	return responder(w, res, err, http.StatusOK, webmapper.ToProveedor)
}
