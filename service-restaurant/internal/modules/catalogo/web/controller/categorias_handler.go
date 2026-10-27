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

// CategoriasHandler: categorías de la carta (RES-RF-005).
type CategoriasHandler struct {
	listar  inbound.ListarCategorias
	obtener inbound.ObtenerCategoria
	crear   inbound.CrearCategoria
	editar  inbound.EditarCategoria
	estado  inbound.CambiarEstadoCategoria
}

func NewCategoriasHandler(l inbound.ListarCategorias, o inbound.ObtenerCategoria, c inbound.CrearCategoria,
	e inbound.EditarCategoria, s inbound.CambiarEstadoCategoria) *CategoriasHandler {
	return &CategoriasHandler{l, o, c, e, s}
}

// Listar: GET /categorias?q=&es_activo=&limit=&cursor=
func (h *CategoriasHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	res, err := h.listar.Execute(r.Context(), query.ListarCategorias{
		Actor: actor, Activo: activo, Texto: q.Get("q"), Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToCategoriasPagina)
}

// Obtener: GET /categorias/{id}
func (h *CategoriasHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerCategoria{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToCategoria)
}

// Crear: POST /categorias (Idempotency-Key obligatoria)
func (h *CategoriasHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.CategoriaAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	in, err := intencion(r, "POST /categorias", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearCategoria{
		Intencion: in, Actor: actor, Nombre: body.Nombre, Descripcion: body.Descripcion, Orden: body.OrdenVisualizacion})
	return responder(w, res, err, http.StatusCreated, webmapper.ToCategoria)
}

// Editar: PATCH /categorias/{id}
func (h *CategoriasHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.CategoriaEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarCategoria{
		Actor: actor, ID: id, Nombre: body.Nombre,
		Descripcion: campo(body.Descripcion.Presente, body.Descripcion.Valor), Orden: body.OrdenVisualizacion})
	return responder(w, res, err, http.StatusOK, webmapper.ToCategoria)
}

// Baja: POST /categorias/{id}/baja, sin motivo (Idempotency-Key obligatoria)
func (h *CategoriasHandler) Baja(w http.ResponseWriter, r *http.Request) error {
	return h.cambiarEstado(w, r, false, "baja")
}

// Restauracion: POST /categorias/{id}/restauracion (Idempotency-Key obligatoria)
func (h *CategoriasHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
	return h.cambiarEstado(w, r, true, "restauracion")
}

// cambiarEstado resuelve las dos acciones de estado, que comparten forma: cuerpo vacío o "{}".
func (h *CategoriasHandler) cambiarEstado(w http.ResponseWriter, r *http.Request, activo bool, accion string) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /categorias/%d/%s", id, accion), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoCategoria{Intencion: in, Actor: actor, ID: id, Activo: activo})
	return responder(w, res, err, http.StatusOK, webmapper.ToCategoria)
}
