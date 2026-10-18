package config

import (
	"database/sql"
	"log/slog"

	"net/http"
	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/inbound"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/application/usecase/command"
	uquery "service-restaurant/internal/modules/identidad/application/usecase/query"
	readadapter "service-restaurant/internal/modules/identidad/infrastructure/persistence/read/adapter"
	writeadapter "service-restaurant/internal/modules/identidad/infrastructure/persistence/write/adapter"
	"service-restaurant/internal/modules/identidad/web/controller"
)

// administracion agrupa los casos de uso de RES-RF-033 ya decorados.
type administracion struct {
	roles    *controller.RolesHandler
	permisos *controller.PermisosHandler
	usuarios *controller.UsuariosHandler
}

func armarAdministracion(db *sql.DB, logger *slog.Logger, hasher outbound.PasswordHasher, clock outbound.Clock) *administracion {
	// Cada ejecución de escritura construye sus adaptadores sobre el mismo *sql.Tx.
	sinIntencion := func(actorID int64) (int64, cmd.Intencion) { return actorID, cmd.Intencion{} }

	crearRol := escritura[cmd.CrearRol, dtoresult.Rol]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearRol, dtoresult.Rol] {
			return command.NewCrearRol(writeadapter.NewRolWriteAdapter(q))
		},
		meta: func(c cmd.CrearRol) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editarRol := escritura[cmd.EditarRol, dtoresult.Rol]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarRol, dtoresult.Rol] {
			return command.NewEditarRol(writeadapter.NewRolWriteAdapter(q))
		},
		meta: func(c cmd.EditarRol) (int64, cmd.Intencion) { return sinIntencion(c.Actor.UserID) }}
	estadoRol := escritura[cmd.CambiarEstadoRol, dtoresult.Rol]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarEstadoRol, dtoresult.Rol] {
			return command.NewCambiarEstadoRol(writeadapter.NewRolWriteAdapter(q), writeadapter.NewAccesosWriteAdapter(q))
		},
		meta: func(c cmd.CambiarEstadoRol) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	crearPermiso := escritura[cmd.CrearPermiso, dtoresult.Permiso]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearPermiso, dtoresult.Permiso] {
			return command.NewCrearPermiso(writeadapter.NewPermisoWriteAdapter(q))
		},
		meta: func(c cmd.CrearPermiso) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editarPermiso := escritura[cmd.EditarPermiso, dtoresult.Permiso]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarPermiso, dtoresult.Permiso] {
			return command.NewEditarPermiso(writeadapter.NewPermisoWriteAdapter(q))
		},
		meta: func(c cmd.EditarPermiso) (int64, cmd.Intencion) { return sinIntencion(c.Actor.UserID) }}
	estadoPermiso := escritura[cmd.CambiarEstadoPermiso, dtoresult.Permiso]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarEstadoPermiso, dtoresult.Permiso] {
			return command.NewCambiarEstadoPermiso(writeadapter.NewPermisoWriteAdapter(q), writeadapter.NewAccesosWriteAdapter(q))
		},
		meta: func(c cmd.CambiarEstadoPermiso) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	crearUsuario := escritura[cmd.CrearUsuario, dtoresult.UsuarioAdmin]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearUsuario, dtoresult.UsuarioAdmin] {
			return command.NewCrearUsuario(writeadapter.NewAccesosWriteAdapter(q), hasher)
		},
		meta: func(c cmd.CrearUsuario) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	asignar := escritura[cmd.AsignarAccesos, dtoresult.UsuarioAdmin]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.AsignarAccesos, dtoresult.UsuarioAdmin] {
			return command.NewAsignarAccesos(writeadapter.NewAccesosWriteAdapter(q))
		},
		meta: func(c cmd.AsignarAccesos) (int64, cmd.Intencion) { return sinIntencion(c.Actor.UserID) }}
	estadoUsuario := escritura[cmd.CambiarEstadoUsuario, dtoresult.UsuarioAdmin]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarEstadoUsuario, dtoresult.UsuarioAdmin] {
			return command.NewCambiarEstadoUsuario(writeadapter.NewAccesosWriteAdapter(q), clock)
		},
		meta: func(c cmd.CambiarEstadoUsuario) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	// Lecturas: sobre el pool, sin transacción.
	catalogo := readadapter.NewCatalogoReadAdapter(db)
	equipo := readadapter.NewUsuarioReadAdapter(db)

	return &administracion{
		roles: controller.NewRolesHandler(
			inbound.CrearRol(crearRol), inbound.EditarRol(editarRol), inbound.CambiarEstadoRol(estadoRol),
			uquery.NewListarRoles(catalogo), uquery.NewObtenerRol(catalogo)),
		permisos: controller.NewPermisosHandler(
			inbound.CrearPermiso(crearPermiso), inbound.EditarPermiso(editarPermiso), inbound.CambiarEstadoPermiso(estadoPermiso),
			uquery.NewListarPermisos(catalogo), uquery.NewObtenerPermiso(catalogo)),
		usuarios: controller.NewUsuariosHandler(
			inbound.CrearUsuario(crearUsuario), inbound.AsignarAccesos(asignar), inbound.CambiarEstadoUsuario(estadoUsuario),
			uquery.NewListarUsuarios(equipo), uquery.NewListarSucursalesAsignables(equipo)),
	}
}
