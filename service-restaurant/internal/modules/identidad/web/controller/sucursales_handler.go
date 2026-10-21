package controller

import (
	"net/http"

	"service-restaurant/internal/modules/identidad/application/dto/query"
	"service-restaurant/internal/modules/identidad/application/port/inbound"
	webmapper "service-restaurant/internal/modules/identidad/web/mapper"
)

// SucursalesHandler: sedes del negocio (RES-RF-003).
type SucursalesHandler struct {
	listar  inbound.ListarSucursales
	obtener inbound.ObtenerSucursal
}

func NewSucursalesHandler(l inbound.ListarSucursales, o inbound.ObtenerSucursal) *SucursalesHandler {
	return &SucursalesHandler{l, o}
}

// Listar: GET /sucursales?limit=&cursor=&es_activo=
func (h *SucursalesHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	res, err := h.listar.Execute(r.Context(), query.ListarSucursales{Actor: actor, Limite: limite, Cursor: q.Get("cursor"), Activo: activo})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursalesPagina)
}

// Obtener: GET /sucursales/{id}
func (h *SucursalesHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerSucursal{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursal)
}
