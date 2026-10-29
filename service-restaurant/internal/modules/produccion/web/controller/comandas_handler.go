package controller

import (
	"fmt"
	"net/http"

	"service-restaurant/internal/modules/produccion/application/dto/command"
	"service-restaurant/internal/modules/produccion/application/port/inbound"
	"service-restaurant/internal/modules/produccion/web/dto/request"
	webmapper "service-restaurant/internal/modules/produccion/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
)

// ComandasHandler: envío a cocina, cola de preparación y estados de las líneas (RES-RF-012).
type ComandasHandler struct {
	enviar       inbound.EnviarComanda
	transicionar inbound.TransicionarLinea
}

func NewComandasHandler(e inbound.EnviarComanda, t inbound.TransicionarLinea) *ComandasHandler {
	return &ComandasHandler{e, t}
}

// Enviar: POST /pedidos/{id}/comandas (Idempotency-Key obligatoria)
func (h *ComandasHandler) Enviar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	pedido, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.ComandaAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	estacion, err := idCadena(body.EstacionID, "estacion_id")
	if err != nil {
		return err
	}
	lineas := make([]command.LineaEnvio, len(body.Lineas))
	for i, l := range body.Lineas {
		det, err := idCadena(l.DetallePedidoID, "detalle_pedido_id")
		if err != nil {
			return err
		}
		lineas[i] = command.LineaEnvio{DetallePedidoID: det, Cantidad: l.Cantidad}
	}
	in, err := intencion(r, fmt.Sprintf("POST /pedidos/%d/comandas", pedido), body)
	if err != nil {
		return err
	}
	res, err := h.enviar.Execute(r.Context(), command.EnviarComanda{
		Intencion: in, Actor: actor, PedidoID: pedido, EstacionID: estacion, Lineas: lineas,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToComanda)
}

// Transicionar: POST /comanda-detalles/{id}/transiciones
func (h *ComandasHandler) Transicionar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.PreparacionTransicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.transicionar.Execute(r.Context(), command.TransicionarLinea{
		Actor: actor, LineaID: id, Estado: body.Estado, Motivo: body.Motivo,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToLinea)
}
