package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"service-restaurant/internal/modules/identidad/application/dto/command"
	"service-restaurant/internal/modules/identidad/application/dto/query"
	"service-restaurant/internal/modules/identidad/application/port/inbound"
	"service-restaurant/internal/modules/identidad/web/dto/request"
	webmapper "service-restaurant/internal/modules/identidad/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
	"service-restaurant/internal/shared/kernel/apperror"
)

// EmpleadosHandler: personal operativo del restaurante.
type EmpleadosHandler struct {
	listar  inbound.ListarEmpleados
	obtener inbound.ObtenerEmpleado
	crear   inbound.CrearEmpleado
	editar  inbound.EditarEmpleado
	estado  inbound.CambiarEstadoEmpleado
}

func NewEmpleadosHandler(l inbound.ListarEmpleados, o inbound.ObtenerEmpleado, c inbound.CrearEmpleado,
	e inbound.EditarEmpleado, s inbound.CambiarEstadoEmpleado) *EmpleadosHandler {
	return &EmpleadosHandler{l, o, c, e, s}
}

// usuarioIDDe convierte un id en cadena del cuerpo a int64 positivo (decimal canónico).
func usuarioIDDe(valor string) (int64, error) {
	n, err := strconv.ParseInt(valor, 10, 64)
	if err != nil || n <= 0 || strconv.FormatInt(n, 10) != valor {
		return 0, apperror.New(apperror.KindBadRequest, "usuario_id_invalido", "usuario_id debe ser un entero positivo en cadena")
	}
	return n, nil
}

// Listar: GET /empleados?q=&es_activo=&limit=&cursor=
func (h *EmpleadosHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	res, err := h.listar.Execute(r.Context(), query.ListarEmpleados{
		Actor: actor, Activo: activo, Texto: q.Get("q"), Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToEmpleadosLista)
}

// Obtener: GET /empleados/{id}
func (h *EmpleadosHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerEmpleado{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToEmpleado)
}

// Crear: POST /empleados (Idempotency-Key obligatoria)
func (h *EmpleadosHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.EmpleadoAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	var usuarioID *int64
	if body.UsuarioID != nil {
		n, err := usuarioIDDe(*body.UsuarioID)
		if err != nil {
			return err
		}
		usuarioID = &n
	}
	in, err := intencion(r, "POST /empleados", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearEmpleado{
		Intencion: in, Actor: actor, DatosEmpleado: datosEmpleado(body.EmpleadoDatos),
		FechaIngreso: body.FechaIngreso, UsuarioID: usuarioID,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToEmpleado)
}

// Editar: PATCH /empleados/{id}
func (h *EmpleadosHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.EmpleadoEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	usuario := command.Campo[int64]{Presente: body.UsuarioID.Presente}
	if body.UsuarioID.Valor != nil {
		n, err := usuarioIDDe(*body.UsuarioID.Valor)
		if err != nil {
			return err
		}
		usuario.Valor = &n
	}
	res, err := h.editar.Execute(r.Context(), command.EditarEmpleado{
		Actor: actor, ID: id, TipoDocumento: body.TipoDocumento, NumeroDocumento: body.NumeroDocumento,
		Nombres: body.Nombres, Apellidos: body.Apellidos, FechaIngreso: body.FechaIngreso,
		Telefono:  command.Campo[string]{Presente: body.Telefono.Presente, Valor: body.Telefono.Valor},
		Cargo:     command.Campo[string]{Presente: body.Cargo.Presente, Valor: body.Cargo.Valor},
		FechaCese: command.Campo[string]{Presente: body.FechaCese.Presente, Valor: body.FechaCese.Valor},
		UsuarioID: usuario,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToEmpleado)
}

// Baja: POST /empleados/{id}/baja con {"motivo": "..."} (Idempotency-Key obligatoria)
func (h *EmpleadosHandler) Baja(w http.ResponseWriter, r *http.Request) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /empleados/%d/baja", id), body)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoEmpleado{
		Intencion: in, Actor: actor, ID: id, Activo: false, Motivo: body.Motivo})
	return responder(w, res, err, http.StatusOK, webmapper.ToEmpleado)
}

// Restauracion: POST /empleados/{id}/restauracion (Idempotency-Key obligatoria)
func (h *EmpleadosHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /empleados/%d/restauracion", id), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoEmpleado{Intencion: in, Actor: actor, ID: id, Activo: true})
	return responder(w, res, err, http.StatusOK, webmapper.ToEmpleado)
}

// datosEmpleado convierte los datos personales del cuerpo en los del comando.
func datosEmpleado(d request.EmpleadoDatos) command.DatosEmpleado {
	return command.DatosEmpleado{
		TipoDocumento: d.TipoDocumento, NumeroDocumento: d.NumeroDocumento, Nombres: d.Nombres, Apellidos: d.Apellidos,
		Telefono: d.Telefono, Cargo: d.Cargo,
	}
}
