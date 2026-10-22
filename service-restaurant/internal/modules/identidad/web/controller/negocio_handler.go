package controller

import (
	"net/http"

	"service-restaurant/internal/modules/identidad/application/dto/command"
	"service-restaurant/internal/modules/identidad/application/dto/query"
	"service-restaurant/internal/modules/identidad/application/port/inbound"
	"service-restaurant/internal/modules/identidad/web/dto/request"
	webmapper "service-restaurant/internal/modules/identidad/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
)

// NegocioHandler: lectura y edición de la empresa única (RES-RF-003).
type NegocioHandler struct {
	obtener inbound.ObtenerNegocio
	editar  inbound.EditarNegocio
}

func NewNegocioHandler(o inbound.ObtenerNegocio, e inbound.EditarNegocio) *NegocioHandler {
	return &NegocioHandler{o, e}
}

// Obtener: GET /negocio
func (h *NegocioHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerNegocio{Actor: actor})
	return responder(w, res, err, http.StatusOK, webmapper.ToNegocio)
}

// Editar: PATCH /negocio
func (h *NegocioHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.NegocioEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarNegocio{
		Actor: actor, RazonSocial: body.RazonSocial, NombreComercial: body.NombreComercial,
		DireccionFiscal: body.DireccionFiscal, Moneda: body.Moneda, ZonaHoraria: body.ZonaHoraria,
		Ubigeo:   command.Campo[string]{Presente: body.Ubigeo.Presente, Valor: body.Ubigeo.Valor},
		Telefono: command.Campo[string]{Presente: body.Telefono.Presente, Valor: body.Telefono.Valor},
		Email:    command.Campo[string]{Presente: body.Email.Presente, Valor: body.Email.Valor},
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToNegocio)
}
