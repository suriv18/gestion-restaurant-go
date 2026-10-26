package controller

import (
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

// ImpuestosHandler: impuestos (solo lectura) y sus tasas vigentes (RES-RF-027).
type ImpuestosHandler struct {
	listarImpuestos inbound.ListarImpuestos
	obtenerImpuesto inbound.ObtenerImpuesto
	listarTasas     inbound.ListarTasasImpuesto
	obtenerTasa     inbound.ObtenerTasaImpuesto
	crearTasa       inbound.CrearTasaImpuesto
	editarTasa      inbound.EditarTasaImpuesto
}

func NewImpuestosHandler(li inbound.ListarImpuestos, oi inbound.ObtenerImpuesto, lt inbound.ListarTasasImpuesto,
	ot inbound.ObtenerTasaImpuesto, ct inbound.CrearTasaImpuesto, et inbound.EditarTasaImpuesto) *ImpuestosHandler {
	return &ImpuestosHandler{li, oi, lt, ot, ct, et}
}

// ListarImpuestos: GET /impuestos?es_activo=&limit=&cursor=
func (h *ImpuestosHandler) ListarImpuestos(w http.ResponseWriter, r *http.Request) error {
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
	res, err := h.listarImpuestos.Execute(r.Context(), query.ListarImpuestos{Actor: actor, Activo: activo, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToImpuestosPagina)
}

// ObtenerImpuesto: GET /impuestos/{id}
func (h *ImpuestosHandler) ObtenerImpuesto(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtenerImpuesto.Execute(r.Context(), query.ObtenerImpuesto{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToImpuesto)
}

// ListarTasas: GET /tasas-impuesto?impuesto_id=&limit=&cursor=
func (h *ImpuestosHandler) ListarTasas(w http.ResponseWriter, r *http.Request) error {
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
	var impuestoID int64
	if v := q.Get("impuesto_id"); v != "" {
		if impuestoID, err = strconv.ParseInt(v, 10, 64); err != nil || impuestoID <= 0 {
			return apperror.New(apperror.KindBadRequest, "impuesto_id_invalido", "impuesto_id no es un id válido")
		}
	}
	res, err := h.listarTasas.Execute(r.Context(), query.ListarTasasImpuesto{Actor: actor, ImpuestoID: impuestoID, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToTasasPagina)
}

// ObtenerTasa: GET /tasas-impuesto/{id}
func (h *ImpuestosHandler) ObtenerTasa(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtenerTasa.Execute(r.Context(), query.ObtenerTasaImpuesto{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToTasaImpuesto)
}

// CrearTasa: POST /tasas-impuesto (Idempotency-Key obligatoria)
func (h *ImpuestosHandler) CrearTasa(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.TasaImpuestoAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	in, err := intencion(r, "POST /tasas-impuesto", body)
	if err != nil {
		return err
	}
	impuestoID, perr := strconv.ParseInt(body.ImpuestoID, 10, 64)
	if perr != nil || impuestoID <= 0 {
		return apperror.New(apperror.KindValidation, "impuesto_id_invalido", "impuesto_id es obligatorio y debe ser un id válido")
	}
	res, err := h.crearTasa.Execute(r.Context(), command.CrearTasaImpuesto{
		Intencion: in, Actor: actor, ImpuestoID: impuestoID, Tasa: body.Tasa,
		VigenteDesde: body.VigenteDesde, VigenteHasta: body.VigenteHasta, Fundamento: body.Fundamento})
	return responder(w, res, err, http.StatusCreated, webmapper.ToTasaImpuesto)
}

// EditarTasa: PATCH /tasas-impuesto/{id}
func (h *ImpuestosHandler) EditarTasa(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.TasaImpuestoEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editarTasa.Execute(r.Context(), command.EditarTasaImpuesto{
		Actor: actor, ID: id, Tasa: body.Tasa, VigenteDesde: body.VigenteDesde,
		VigenteHasta: campo(body.VigenteHasta.Presente, body.VigenteHasta.Valor),
		Fundamento:   campo(body.Fundamento.Presente, body.Fundamento.Valor)})
	return responder(w, res, err, http.StatusOK, webmapper.ToTasaImpuesto)
}
