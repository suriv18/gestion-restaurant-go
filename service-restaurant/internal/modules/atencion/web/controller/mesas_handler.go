package controller

import (
	"net/http"

	"service-restaurant/internal/modules/atencion/application/dto/query"
	"service-restaurant/internal/modules/atencion/application/port/inbound"
	webmapper "service-restaurant/internal/modules/atencion/web/mapper"
)

// MesasHandler: mesas del salón (RES-RF-008).
type MesasHandler struct {
	listar  inbound.ListarMesas
	obtener inbound.ObtenerMesa
}

func NewMesasHandler(l inbound.ListarMesas, o inbound.ObtenerMesa) *MesasHandler {
	return &MesasHandler{l, o}
}

// Listar: GET /mesas?sucursal_id=&es_activo=&limit=&cursor=
func (h *MesasHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	sede, err := sucursalDe(q)
	if err != nil {
		return err
	}
	res, err := h.listar.Execute(r.Context(), query.ListarMesas{
		Actor: actor, SucursalID: sede, Activo: activo, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToMesasPagina)
}

// Obtener: GET /mesas/{id}
func (h *MesasHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerMesa{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToMesa)
}
