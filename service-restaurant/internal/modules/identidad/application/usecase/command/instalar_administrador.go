package command

import (
	"context"
	"fmt"
	"regexp"
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

var rucRe = regexp.MustCompile(`^[0-9]{11}$`)

// InstalarAdministrador crea el primer administrador (la V4 solo trae el actor técnico).
// Se niega a correr si ya existe un usuario humano.
type InstalarAdministrador struct {
	instalacion outbound.InstalacionStore
	actor       outbound.ActorContext
	hasher      outbound.PasswordHasher
}

func NewInstalarAdministrador(i outbound.InstalacionStore, a outbound.ActorContext, h outbound.PasswordHasher) *InstalarAdministrador {
	return &InstalarAdministrador{i, a, h}
}

func (uc *InstalarAdministrador) Execute(ctx context.Context, c cmd.InstalarAdministrador) (appresult.Result[dtoresult.AdministradorInstalado], error) {
	fail := func(code, msg string) (appresult.Result[dtoresult.AdministradorInstalado], error) {
		return appresult.Fail[dtoresult.AdministradorInstalado](apperror.New(apperror.KindValidation, code, msg)), nil
	}

	username, err := vo.NewUsername(c.Username)
	if err != nil {
		return fail(mapper.CodigoDatosInvalidos, "username inválido")
	}
	if err := policy.ValidarClaveNueva(c.Password); err != nil {
		if ae := mapper.DeInvariante(err); ae != nil {
			return appresult.Fail[dtoresult.AdministradorInstalado](ae), nil
		}
		return appresult.Result[dtoresult.AdministradorInstalado]{}, err
	}
	for name, v := range map[string]string{
		"razón social": c.RazonSocial, "nombre comercial": c.NombreComercial, "dirección fiscal": c.DireccionFiscal,
		"código de sucursal": c.SucursalCodigo, "nombre de sucursal": c.SucursalNombre, "dirección de sucursal": c.SucursalDireccion,
	} {
		if strings.TrimSpace(v) == "" {
			return fail(mapper.CodigoDatosInvalidos, name+" es obligatorio")
		}
	}
	if !rucRe.MatchString(c.RUC) {
		return fail(mapper.CodigoDatosInvalidos, "el RUC debe tener 11 dígitos")
	}

	existe, err := uc.instalacion.ExisteUsuarioHumano(ctx)
	if err != nil {
		return appresult.Result[dtoresult.AdministradorInstalado]{}, fmt.Errorf("instalar administrador: %w", err)
	}
	if existe {
		return appresult.Fail[dtoresult.AdministradorInstalado](
			apperror.New(apperror.KindConflict, mapper.CodigoYaInstalado, "ya existe un usuario humano; la instalación inicial no se repite")), nil
	}

	sistema, err := uc.actor.UsuarioSistemaID(ctx)
	if err != nil {
		return appresult.Result[dtoresult.AdministradorInstalado]{}, fmt.Errorf("instalar administrador: actor técnico: %w", err)
	}
	if err := uc.actor.Fijar(ctx, sistema); err != nil {
		return appresult.Result[dtoresult.AdministradorInstalado]{}, fmt.Errorf("instalar administrador: fijar actor: %w", err)
	}
	sucursalID, err := uc.instalacion.AsegurarSucursal(ctx, outbound.DatosInstalacion{
		RUC: c.RUC, RazonSocial: c.RazonSocial, NombreComercial: c.NombreComercial, DireccionFiscal: c.DireccionFiscal,
		SucursalCodigo: c.SucursalCodigo, SucursalNombre: c.SucursalNombre, SucursalDireccion: c.SucursalDireccion,
	})
	if err != nil {
		return appresult.Result[dtoresult.AdministradorInstalado]{}, fmt.Errorf("instalar administrador: sucursal: %w", err)
	}
	hash, err := uc.hasher.Hash(c.Password)
	if err != nil {
		return appresult.Result[dtoresult.AdministradorInstalado]{}, fmt.Errorf("instalar administrador: hash: %w", err)
	}
	id, err := uc.instalacion.CrearAdministrador(ctx, username.String(), hash, sucursalID)
	if err != nil {
		return appresult.Result[dtoresult.AdministradorInstalado]{}, fmt.Errorf("instalar administrador: crear usuario: %w", err)
	}
	return appresult.Ok(dtoresult.AdministradorInstalado{UsuarioID: id, SucursalID: sucursalID}), nil
}
