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

// MesasHandler: mesas del salón (RES-RF-008).
type MesasHandler struct {
	listar  inbound.ListarMesas
	obtener inbound.ObtenerMesa
	crear   inbound.CrearMesa
	editar  inbound.EditarMesa
	estado  inbound.CambiarEstadoMesa
}

func NewMesasHandler(l inbound.ListarMesas, o inbound.ObtenerMesa, c inbound.CrearMesa,
	e inbound.EditarMesa, s inbound.CambiarEstadoMesa) *MesasHandler {
	return &MesasHandler{l, o, c, e, s}
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

// Crear: POST /mesas (Idempotency-Key obligatoria)
func (h *MesasHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.MesaAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	sede, err := idCadena(body.SucursalID, "sucursal_id")
	if err != nil {
		return err
	}
	zona, err := idCadena(body.ZonaID, "zona_id")
	if err != nil {
		return err
	}
	in, err := intencion(r, "POST /mesas", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearMesa{
		Intencion: in, Actor: actor, SucursalID: sede, ZonaID: zona, NumeroMesa: body.NumeroMesa,
		Capacidad: body.Capacidad, Habilitada: body.Habilitada, PosicionX: body.PosicionX, PosicionY: body.PosicionY,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToMesa)
}

// Editar: PATCH /mesas/{id}
func (h *MesasHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.MesaEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	var zona *int64
	if body.ZonaID != nil {
		z, err := idCadena(*body.ZonaID, "zona_id")
		if err != nil {
			return err
		}
		zona = &z
	}
	res, err := h.editar.Execute(r.Context(), command.EditarMesa{
		Actor: actor, ID: id, ZonaID: zona, NumeroMesa: body.NumeroMesa, Capacidad: body.Capacidad, Habilitada: body.Habilitada,
		PosicionX: command.Campo[string]{Presente: body.PosicionX.Presente, Valor: body.PosicionX.Valor},
		PosicionY: command.Campo[string]{Presente: body.PosicionY.Presente, Valor: body.PosicionY.Valor},
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToMesa)
}

// Baja: POST /mesas/{id}/baja con {"motivo": "..."} (Idempotency-Key obligatoria)
func (h *MesasHandler) Baja(w http.ResponseWriter, r *http.Request) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /mesas/%d/baja", id), body)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoMesa{
		Intencion: in, Actor: actor, ID: id, Activo: false, Motivo: body.Motivo,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToMesa)
}

// Restauracion: POST /mesas/{id}/restauracion (Idempotency-Key obligatoria)
func (h *MesasHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /mesas/%d/restauracion", id), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoMesa{Intencion: in, Actor: actor, ID: id, Activo: true})
	return responder(w, res, err, http.StatusOK, webmapper.ToMesa)
}
