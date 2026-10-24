package controller

import (
	"fmt"
	"net/http"

	"service-restaurant/internal/modules/identidad/application/dto/command"
	"service-restaurant/internal/modules/identidad/application/dto/query"
	"service-restaurant/internal/modules/identidad/application/port/inbound"
	"service-restaurant/internal/modules/identidad/web/dto/request"
	webmapper "service-restaurant/internal/modules/identidad/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
)

// UsuariosHandler: alta, asignaciones, baja y restauración de usuarios (RES-RF-033).
type UsuariosHandler struct {
	crear   inbound.CrearUsuario
	asignar inbound.AsignarAccesos
	estado  inbound.CambiarEstadoUsuario
	listar  inbound.ListarUsuarios
	sedes   inbound.ListarSucursalesAsignables
}

func NewUsuariosHandler(c inbound.CrearUsuario, a inbound.AsignarAccesos, s inbound.CambiarEstadoUsuario,
	l inbound.ListarUsuarios, sedes inbound.ListarSucursalesAsignables) *UsuariosHandler {
	return &UsuariosHandler{c, a, s, l, sedes}
}

// Listar: GET /usuarios?limit=&cursor= — el equipo con sus roles y sedes vigentes.
func (h *UsuariosHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	res, err := h.listar.Execute(r.Context(), query.ListarUsuarios{Actor: actor, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToUsuariosLista)
}

// SucursalesAsignables: GET /sucursales-asignables — las sedes que el actor puede asignar.
func (h *UsuariosHandler) SucursalesAsignables(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	res, err := h.sedes.Execute(r.Context(), query.ListarSucursalesAsignables{Actor: actor})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursalesLista)
}

// Crear: POST /usuarios (Idempotency-Key obligatoria). La contraseña nunca se registra:
// la huella de idempotencia se calcula sobre el cuerpo sin ella.
func (h *UsuariosHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.UsuarioAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	// Huella: usuario y correo identifican la intención; la contraseña se incluye solo como
	// su propio hash SHA-256 para detectar un cambio de contenido sin conservar el secreto.
	canonico := struct {
		Username string                 `json:"username"`
		Nombre   *string                `json:"nombre"`
		Email    *string                `json:"email"`
		Password [32]byte               `json:"password_sha256"`
		Empleado *request.EmpleadoDatos `json:"empleado"`
	}{body.Username, body.Nombre, body.Email, sha256Of(body.Password), body.Empleado}
	in, err := intencion(r, "POST /usuarios", canonico)
	if err != nil {
		return err
	}
	var empleado *command.DatosEmpleado
	if body.Empleado != nil {
		d := datosEmpleado(*body.Empleado)
		empleado = &d
	}
	res, err := h.crear.Execute(r.Context(), command.CrearUsuario{
		Intencion: in, Actor: actor, Username: body.Username, Password: body.Password, Email: body.Email, Empleado: empleado,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToUsuario)
}

// Asignaciones: PUT /usuarios/{id}/asignaciones (reemplaza roles y sedes; idempotente por naturaleza)
func (h *UsuariosHandler) Asignaciones(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.Asignaciones
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	roles, err := idsDe(body.RolIDs)
	if err != nil {
		return err
	}
	sedes, err := idsDe(body.SucursalIDs)
	if err != nil {
		return err
	}
	res, err := h.asignar.Execute(r.Context(), command.AsignarAccesos{Actor: actor, UsuarioID: id, RolIDs: roles, SucursalIDs: sedes})
	return responder(w, res, err, http.StatusOK, webmapper.ToUsuario)
}

// Baja: POST /usuarios/{id}/baja con {"motivo": "..."} (Idempotency-Key obligatoria)
func (h *UsuariosHandler) Baja(w http.ResponseWriter, r *http.Request) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /usuarios/%d/baja", id), body)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoUsuario{
		Intencion: in, Actor: actor, UsuarioID: id, Activo: false, Motivo: body.Motivo,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToUsuario)
}

// Restauracion: POST /usuarios/{id}/restauracion (Idempotency-Key obligatoria)
func (h *UsuariosHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /usuarios/%d/restauracion", id), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoUsuario{
		Intencion: in, Actor: actor, UsuarioID: id, Activo: true,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToUsuario)
}
