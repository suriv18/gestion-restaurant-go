package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"service-restaurant/internal/modules/catalogo/application/dto/command"
	"service-restaurant/internal/modules/catalogo/application/dto/query"
	"service-restaurant/internal/modules/catalogo/application/port/inbound"
	"service-restaurant/internal/modules/catalogo/web/dto/request"
	webmapper "service-restaurant/internal/modules/catalogo/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
	"service-restaurant/internal/shared/kernel/apperror"
)

// EstacionesHandler: estaciones de preparación, por sede (RES-RF-005).
type EstacionesHandler struct {
	listar  inbound.ListarEstaciones
	obtener inbound.ObtenerEstacion
	crear   inbound.CrearEstacion
	editar  inbound.EditarEstacion
	estado  inbound.CambiarEstadoEstacion
}

func NewEstacionesHandler(l inbound.ListarEstaciones, o inbound.ObtenerEstacion, c inbound.CrearEstacion,
	e inbound.EditarEstacion, s inbound.CambiarEstadoEstacion) *EstacionesHandler {
	return &EstacionesHandler{l, o, c, e, s}
}

// idCadena convierte un id en cadena (contrato) a int64 positivo, en decimal canónico.
func idCadena(valor, campo string) (int64, error) {
	n, err := strconv.ParseInt(valor, 10, 64)
	if err != nil || n <= 0 || strconv.FormatInt(n, 10) != valor {
		return 0, apperror.New(apperror.KindBadRequest, campo+"_invalido", campo+" debe ser un entero positivo en cadena")
	}
	return n, nil
}

// Listar: GET /estaciones?sucursal_id=&q=&es_activo=&limit=&cursor=
func (h *EstacionesHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	var sede *int64
	if v := q.Get("sucursal_id"); v != "" {
		n, err := idCadena(v, "sucursal_id")
		if err != nil {
			return err
		}
		sede = &n
	}
	res, err := h.listar.Execute(r.Context(), query.ListarEstaciones{
		Actor: actor, SucursalID: sede, Activo: activo, Texto: q.Get("q"), Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToEstacionesPagina)
}

// Obtener: GET /estaciones/{id}
func (h *EstacionesHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerEstacion{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToEstacion)
}

// Crear: POST /estaciones (Idempotency-Key obligatoria)
func (h *EstacionesHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.EstacionAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	sede, err := idCadena(body.SucursalID, "sucursal_id")
	if err != nil {
		return err
	}
	in, err := intencion(r, "POST /estaciones", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearEstacion{
		Intencion: in, Actor: actor, SucursalID: sede, Codigo: body.Codigo, Nombre: body.Nombre})
	return responder(w, res, err, http.StatusCreated, webmapper.ToEstacion)
}

// Editar: PATCH /estaciones/{id}
func (h *EstacionesHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.EstacionEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarEstacion{Actor: actor, ID: id, Codigo: body.Codigo, Nombre: body.Nombre})
	return responder(w, res, err, http.StatusOK, webmapper.ToEstacion)
}

// Baja: POST /estaciones/{id}/baja, sin motivo (Idempotency-Key obligatoria)
func (h *EstacionesHandler) Baja(w http.ResponseWriter, r *http.Request) error {
	return h.cambiarEstado(w, r, false, "baja")
}

// Restauracion: POST /estaciones/{id}/restauracion (Idempotency-Key obligatoria)
func (h *EstacionesHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
	return h.cambiarEstado(w, r, true, "restauracion")
}

// cambiarEstado resuelve las dos acciones de estado, que comparten forma: cuerpo vacío o "{}".
func (h *EstacionesHandler) cambiarEstado(w http.ResponseWriter, r *http.Request, activo bool, accion string) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /estaciones/%d/%s", id, accion), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoEstacion{Intencion: in, Actor: actor, ID: id, Activo: activo})
	return responder(w, res, err, http.StatusOK, webmapper.ToEstacion)
}
