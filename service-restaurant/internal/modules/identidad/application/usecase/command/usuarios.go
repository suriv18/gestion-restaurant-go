package command

import (
	"context"
	"net/mail"
	"strings"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	"service-restaurant/internal/modules/identidad/domain/vo"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	maxEmail  = 254
	maxMotivo = 2000
)

// CrearUsuario da de alta un usuario humano. La contraseña solo se usa para derivar su hash.
type CrearUsuario struct {
	accesos outbound.AccesosStore
	hasher  outbound.PasswordHasher
}

func NewCrearUsuario(a outbound.AccesosStore, h outbound.PasswordHasher) *CrearUsuario {
	return &CrearUsuario{a, h}
}

func (uc *CrearUsuario) Execute(ctx context.Context, c cmd.CrearUsuario) (appresult.Result[dtoresult.UsuarioAdmin], error) {
	if ae := mapper.RequiereAdministrar(c.Actor); ae != nil {
		return rechazar[dtoresult.UsuarioAdmin](ae)
	}
	username, err := vo.NewUsername(c.Username)
	if err != nil {
		return rechazar[dtoresult.UsuarioAdmin](mapper.Validacion("username_invalido", "el username es obligatorio (máximo 80 caracteres, sin caracteres de control)"))
	}
	if err := policy.ValidarClaveNueva(c.Password); err != nil {
		if ae := mapper.DeInvariante(err); ae != nil {
			return rechazar[dtoresult.UsuarioAdmin](ae)
		}
		return tecnico[dtoresult.UsuarioAdmin]("validar contraseña", err)
	}
	email, ok := emailOpcional(c.Email)
	if !ok {
		return rechazar[dtoresult.UsuarioAdmin](mapper.Validacion("email_invalido", "el correo electrónico no es válido"))
	}
	hash, err := uc.hasher.Hash(c.Password)
	if err != nil {
		return tecnico[dtoresult.UsuarioAdmin]("derivar hash", err)
	}
	u, err := uc.accesos.CrearUsuario(ctx, username.String(), email, hash)
	if err != nil {
		return deAlmacen[dtoresult.UsuarioAdmin]("crear usuario", err)
	}
	return appresult.Ok(u), nil
}

func emailOpcional(p *string) (*string, bool) {
	v, ok := textoOpcional(p, maxEmail)
	if !ok || v == nil {
		return nil, ok
	}
	addr, err := mail.ParseAddress(*v)
	if err != nil || addr.Address != *v || strings.ContainsAny(*v, " <>") {
		return nil, false
	}
	return v, true
}

// AsignarAccesos reemplaza los roles y las sedes de un usuario.
//
// Reglas: el administrador solo agrega o quita sedes donde él mismo tiene
// USUARIO_ADMINISTRAR; el actor técnico no existe para la API; y el cambio no puede dejar
// al sistema sin administrador (se aplica, se cuenta y, si queda en 0, se revierte).
type AsignarAccesos struct{ accesos outbound.AccesosStore }

func NewAsignarAccesos(a outbound.AccesosStore) *AsignarAccesos { return &AsignarAccesos{a} }

func (uc *AsignarAccesos) Execute(ctx context.Context, c cmd.AsignarAccesos) (appresult.Result[dtoresult.UsuarioAdmin], error) {
	if ae := mapper.RequiereAdministrar(c.Actor); ae != nil {
		return rechazar[dtoresult.UsuarioAdmin](ae)
	}
	roles, ok1 := positivosUnicos(c.RolIDs)
	sedes, ok2 := positivosUnicos(c.SucursalIDs)
	if !ok1 || !ok2 || len(roles) == 0 || len(sedes) == 0 {
		return rechazar[dtoresult.UsuarioAdmin](mapper.Validacion("asignaciones_invalidas", "se requiere al menos un rol y una sucursal, con ids válidos"))
	}
	objetivo, found, err := uc.accesos.ObtenerUsuario(ctx, c.UsuarioID)
	if err != nil {
		return tecnico[dtoresult.UsuarioAdmin]("buscar usuario", err)
	}
	if !found || objetivo.EsSistema {
		return rechazar[dtoresult.UsuarioAdmin](mapper.NoEncontrado("usuario"))
	}
	actuales, err := uc.accesos.SucursalesActivas(ctx, c.UsuarioID)
	if err != nil {
		return tecnico[dtoresult.UsuarioAdmin]("leer sucursales", err)
	}
	permitidas := make(map[int64]bool)
	for _, s := range c.Actor.SucursalesCon(policy.UsuarioAdministrar) {
		permitidas[s] = true
	}
	for _, s := range diferenciaSimetrica(actuales, sedes) {
		if !permitidas[s] {
			return rechazar[dtoresult.UsuarioAdmin](apperror.New(apperror.KindForbidden, mapper.CodigoSedeNoAutorizada,
				"solo puede agregar o quitar sucursales donde usted administra accesos"))
		}
	}
	if err := uc.accesos.ReemplazarAsignaciones(ctx, c.UsuarioID, roles, sedes); err != nil {
		return deAlmacen[dtoresult.UsuarioAdmin]("reemplazar asignaciones", err)
	}
	if ae, err := ultimoAdministrador(ctx, uc.accesos); err != nil {
		return tecnico[dtoresult.UsuarioAdmin]("verificar administradores", err)
	} else if ae != nil {
		return rechazar[dtoresult.UsuarioAdmin](ae)
	}
	u, _, err := uc.accesos.ObtenerUsuario(ctx, c.UsuarioID)
	if err != nil {
		return tecnico[dtoresult.UsuarioAdmin]("releer usuario", err)
	}
	return appresult.Ok(u), nil
}

// CambiarEstadoUsuario da de baja (con motivo) o restaura a un usuario. La baja revoca sus
// sesiones; nadie puede darse de baja a sí mismo ni dejar al sistema sin administrador.
type CambiarEstadoUsuario struct {
	accesos outbound.AccesosStore
	clock   outbound.Clock
}

func NewCambiarEstadoUsuario(a outbound.AccesosStore, c outbound.Clock) *CambiarEstadoUsuario {
	return &CambiarEstadoUsuario{a, c}
}

func (uc *CambiarEstadoUsuario) Execute(ctx context.Context, c cmd.CambiarEstadoUsuario) (appresult.Result[dtoresult.UsuarioAdmin], error) {
	if ae := mapper.RequiereAdministrar(c.Actor); ae != nil {
		return rechazar[dtoresult.UsuarioAdmin](ae)
	}
	motivo := ""
	if !c.Activo {
		m, err := vo.TextoObligatorio(c.Motivo, maxMotivo)
		if err != nil {
			return rechazar[dtoresult.UsuarioAdmin](mapper.Validacion("motivo_requerido", "la baja exige un motivo (máximo 2000 caracteres)"))
		}
		motivo = m
		if c.UsuarioID == c.Actor.UserID {
			return rechazar[dtoresult.UsuarioAdmin](apperror.New(apperror.KindBusinessRule, mapper.CodigoAutoBaja,
				"la baja de su propio usuario debe realizarla otro administrador"))
		}
	}
	objetivo, found, err := uc.accesos.ObtenerUsuario(ctx, c.UsuarioID)
	if err != nil {
		return tecnico[dtoresult.UsuarioAdmin]("buscar usuario", err)
	}
	if !found || objetivo.EsSistema {
		return rechazar[dtoresult.UsuarioAdmin](mapper.NoEncontrado("usuario"))
	}
	u, err := uc.accesos.CambiarActivoUsuario(ctx, c.UsuarioID, c.Activo, motivo)
	if err != nil {
		return deAlmacen[dtoresult.UsuarioAdmin]("cambiar estado del usuario", err)
	}
	if !c.Activo {
		// Restaurar al usuario más adelante no debe revivir sesiones antiguas.
		if err := uc.accesos.RevocarSesionesDeUsuario(ctx, c.UsuarioID, uc.clock.Now()); err != nil {
			return tecnico[dtoresult.UsuarioAdmin]("revocar sesiones", err)
		}
		if ae, err := ultimoAdministrador(ctx, uc.accesos); err != nil {
			return tecnico[dtoresult.UsuarioAdmin]("verificar administradores", err)
		} else if ae != nil {
			return rechazar[dtoresult.UsuarioAdmin](ae)
		}
	}
	return appresult.Ok(u), nil
}
