package controller

import (
	"net/http"

	"service-restaurant/internal/modules/atencion/application/dto/query"
	"service-restaurant/internal/modules/atencion/application/port/inbound"
	webmapper "service-restaurant/internal/modules/atencion/web/mapper"
)

// ZonasHandler: salones, terrazas y niveles del local (RES-RF-008).
type ZonasHandler struct {
	listar  inbound.ListarZonas
	obtener inbound.ObtenerZona
}

func NewZonasHandler(l inbound.ListarZonas, o inbound.ObtenerZona) *ZonasHandler {
	return &ZonasHandler{l, o}
}

// Listar: GET /zonas?sucursal_id=&es_activo=&limit=&cursor=
func (h *ZonasHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	res, err := h.listar.Execute(r.Context(), query.ListarZonas{
		Actor: actor, SucursalID: sede, Activo: activo, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToZonasPagina)
}

// Obtener: GET /zonas/{id}
func (h *ZonasHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerZona{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToZona)
}
