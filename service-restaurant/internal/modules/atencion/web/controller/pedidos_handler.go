package controller

import (
	"fmt"
	"net/http"

	"service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/application/dto/query"
	"service-restaurant/internal/modules/atencion/application/port/inbound"
	"service-restaurant/internal/modules/atencion/web/dto/request"
	webmapper "service-restaurant/internal/modules/atencion/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
)

var (
	estadosPedido = []string{"REGISTRADO", "PREPARACION", "LISTO", "ENTREGADO", "CANCELADO"}
	tiposPedido   = []string{"SALON", "LLEVAR", "DELIVERY"}
)

// PedidosHandler: pedidos, cuentas y consumos del salón (RES-RF-009, RES-RF-010).
type PedidosHandler struct {
	listar  inbound.ListarPedidos
	obtener inbound.ObtenerPedido
	abrir   inbound.AbrirPedido
	agregar inbound.AgregarDetalle
	editar  inbound.EditarDetalle
}

func NewPedidosHandler(l inbound.ListarPedidos, o inbound.ObtenerPedido, a inbound.AbrirPedido,
	ag inbound.AgregarDetalle, e inbound.EditarDetalle) *PedidosHandler {
	return &PedidosHandler{l, o, a, ag, e}
}

// opcionalID convierte un id en cadena anulable (contrato) a *int64.
func opcionalID(valor *string, campo string) (*int64, error) {
	if valor == nil {
		return nil, nil
	}
	n, err := idCadena(*valor, campo)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// Listar: GET /pedidos?sucursal_id=&estado=&tipo=&mesa_id=&limit=&cursor=
func (h *PedidosHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	sede, err := sucursalDe(q)
	if err != nil {
		return err
	}
	estado, err := enumDe(q, "estado", estadosPedido...)
	if err != nil {
		return err
	}
	tipo, err := enumDe(q, "tipo", tiposPedido...)
	if err != nil {
		return err
	}
	var mesa *int64
	if v := q.Get("mesa_id"); v != "" {
		if mesa, err = opcionalID(&v, "mesa_id"); err != nil {
			return err
		}
	}
	res, err := h.listar.Execute(r.Context(), query.ListarPedidos{
		Actor: actor, SucursalID: sede, Estado: estado, Tipo: tipo, MesaID: mesa, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToPedidosPagina)
}

// Obtener: GET /pedidos/{id}
func (h *PedidosHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerPedido{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToPedidoDetalle)
}

// Abrir: POST /pedidos (Idempotency-Key obligatoria)
func (h *PedidosHandler) Abrir(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.PedidoAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	sede, err := idCadena(body.SucursalID, "sucursal_id")
	if err != nil {
		return err
	}
	mesa, err := opcionalID(body.MesaID, "mesa_id")
	if err != nil {
		return err
	}
	cliente, err := opcionalID(body.ClienteID, "cliente_id")
	if err != nil {
		return err
	}
	var reparto *command.Reparto
	if body.Reparto != nil {
		reparto = &command.Reparto{Destinatario: body.Reparto.Destinatario, Telefono: body.Reparto.Telefono,
			Direccion: body.Reparto.Direccion, Referencia: body.Reparto.Referencia, Canal: body.Reparto.Canal}
	}
	in, err := intencion(r, "POST /pedidos", body)
	if err != nil {
		return err
	}
	res, err := h.abrir.Execute(r.Context(), command.AbrirPedido{
		Intencion: in, Actor: actor, SucursalID: sede, Tipo: body.Tipo, MesaID: mesa, Comensales: body.Comensales,
		ClienteID: cliente, Notas: body.Notas, Reparto: reparto,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToPedidoDetalle)
}

// AgregarDetalle: POST /pedidos/{id}/detalles (Idempotency-Key obligatoria)
func (h *PedidosHandler) AgregarDetalle(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	pedido, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.ConsumoAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	producto, err := idCadena(body.ProductoID, "producto_id")
	if err != nil {
		return err
	}
	cuenta, err := idCadena(body.CuentaID, "cuenta_id")
	if err != nil {
		return err
	}
	padre, err := opcionalID(body.DetallePadreID, "detalle_padre_id")
	if err != nil {
		return err
	}
	in, err := intencion(r, fmt.Sprintf("POST /pedidos/%d/detalles", pedido), body)
	if err != nil {
		return err
	}
	res, err := h.agregar.Execute(r.Context(), command.AgregarDetalle{
		Intencion: in, Actor: actor, PedidoID: pedido, CuentaID: cuenta, ProductoID: producto, Cantidad: body.Cantidad,
		DetallePadreID: padre, Instrucciones: body.Instrucciones, DescuentoMonto: body.DescuentoMonto,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToLinea)
}

// EditarDetalle: PATCH /pedidos/{id}/detalles/{detalle_id}
func (h *PedidosHandler) EditarDetalle(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	pedido, err := pathID(r)
	if err != nil {
		return err
	}
	detalle, err := pathIDDe(r, "detalle_id")
	if err != nil {
		return err
	}
	var body request.ConsumoEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarDetalle{
		Actor: actor, PedidoID: pedido, DetalleID: detalle, Cantidad: body.Cantidad, DescuentoMonto: body.DescuentoMonto,
		Instrucciones: command.Campo[string]{Presente: body.Instrucciones.Presente, Valor: body.Instrucciones.Valor},
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToLinea)
}
