package command

import (
	"context"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/mapper"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/policy"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	mensajeSinPermisoProveedores = "no tiene permiso para administrar proveedores"
	mensajeProveedorDuplicado    = "ya existe un proveedor con ese documento (aunque esté dado de baja)"
)

func deProveedor(err error, contexto string) (appresult.Result[dtoresult.Proveedor], error) {
	return deAlmacen[dtoresult.Proveedor](contexto, mensajeProveedorDuplicado, err)
}

// CrearProveedor da de alta un proveedor (RES-RF-004). Su documento es obligatorio.
type CrearProveedor struct{ proveedores outbound.ProveedorStore }

func NewCrearProveedor(p outbound.ProveedorStore) *CrearProveedor { return &CrearProveedor{p} }

func (uc *CrearProveedor) Execute(ctx context.Context, c cmd.CrearProveedor) (appresult.Result[dtoresult.Proveedor], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoProveedores, policy.ProveedorAdministrar); ae != nil {
		return rechazar[dtoresult.Proveedor](ae)
	}
	razon, ae := validarNombre(c.RazonSocial, "razon_social")
	if ae != nil {
		return rechazar[dtoresult.Proveedor](ae)
	}
	tipo, numero, ae := validarDocumento(c.TipoDocumento, c.NumeroDocumento)
	if ae != nil {
		return rechazar[dtoresult.Proveedor](ae)
	}
	nuevo := outbound.NuevoProveedor{TipoDocumento: tipo, NumeroDocumento: numero, RazonSocial: razon}
	if nuevo.ContactoNombre, ae = opcional(c.ContactoNombre, "contacto_nombre", fmtContacto); ae != nil {
		return rechazar[dtoresult.Proveedor](ae)
	}
	if nuevo.Telefono, ae = opcional(c.Telefono, "telefono", fmtTelefono); ae != nil {
		return rechazar[dtoresult.Proveedor](ae)
	}
	if nuevo.Email, ae = opcional(c.Email, "email", fmtEmail); ae != nil {
		return rechazar[dtoresult.Proveedor](ae)
	}
	if nuevo.Direccion, ae = opcional(c.Direccion, "direccion", fmtDireccion); ae != nil {
		return rechazar[dtoresult.Proveedor](ae)
	}
	p, err := uc.proveedores.Crear(ctx, nuevo)
	if err != nil {
		return deProveedor(err, "crear proveedor")
	}
	return appresult.Ok(p), nil
}

// EditarProveedor modifica solo los campos enviados de un proveedor activo.
type EditarProveedor struct{ proveedores outbound.ProveedorStore }

func NewEditarProveedor(p outbound.ProveedorStore) *EditarProveedor { return &EditarProveedor{p} }

func (uc *EditarProveedor) Execute(ctx context.Context, c cmd.EditarProveedor) (appresult.Result[dtoresult.Proveedor], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoProveedores, policy.ProveedorAdministrar); ae != nil {
		return rechazar[dtoresult.Proveedor](ae)
	}
	actual, found, err := uc.proveedores.Obtener(ctx, c.ID)
	if err != nil {
		return tecnico[dtoresult.Proveedor]("buscar proveedor", err)
	}
	if !found {
		return rechazar[dtoresult.Proveedor](mapper.NoEncontrado("proveedor"))
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Proveedor](apperror.New(apperror.KindConflict, CodigoProveedorInactivo, "el proveedor está dado de baja: restáurelo para editarlo"))
	}
	if c.TipoDocumento == nil && c.NumeroDocumento == nil && c.RazonSocial == nil &&
		!c.ContactoNombre.Presente && !c.Telefono.Presente && !c.Email.Presente && !c.Direccion.Presente {
		return rechazar[dtoresult.Proveedor](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosProveedor
	if c.RazonSocial != nil {
		r, ae := validarNombre(*c.RazonSocial, "razon_social")
		if ae != nil {
			return rechazar[dtoresult.Proveedor](ae)
		}
		cambios.RazonSocial = &r
	}
	if c.TipoDocumento != nil || c.NumeroDocumento != nil {
		// Un lado del par puede venir solo: el otro se toma del registro bloqueado.
		tipo, numero := actual.TipoDocumento, actual.NumeroDocumento
		if c.TipoDocumento != nil {
			tipo = *c.TipoDocumento
		}
		if c.NumeroDocumento != nil {
			numero = *c.NumeroDocumento
		}
		t, n, ae := validarDocumento(tipo, numero)
		if ae != nil {
			return rechazar[dtoresult.Proveedor](ae)
		}
		cambios.TipoDocumento, cambios.NumeroDocumento = &t, &n
	}
	var ae *apperror.AppError
	if cambios.ContactoNombre, ae = campoOpcional(c.ContactoNombre, "contacto_nombre", fmtContacto); ae != nil {
		return rechazar[dtoresult.Proveedor](ae)
	}
	if cambios.Telefono, ae = campoOpcional(c.Telefono, "telefono", fmtTelefono); ae != nil {
		return rechazar[dtoresult.Proveedor](ae)
	}
	if cambios.Email, ae = campoOpcional(c.Email, "email", fmtEmail); ae != nil {
		return rechazar[dtoresult.Proveedor](ae)
	}
	if cambios.Direccion, ae = campoOpcional(c.Direccion, "direccion", fmtDireccion); ae != nil {
		return rechazar[dtoresult.Proveedor](ae)
	}
	p, found, err := uc.proveedores.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deProveedor(err, "editar proveedor")
	}
	if !found {
		return rechazar[dtoresult.Proveedor](mapper.NoEncontrado("proveedor"))
	}
	return appresult.Ok(p), nil
}

// CambiarEstadoProveedor da de baja (con motivo) o restaura un proveedor (RES-RF-030).
type CambiarEstadoProveedor struct{ proveedores outbound.ProveedorStore }

func NewCambiarEstadoProveedor(p outbound.ProveedorStore) *CambiarEstadoProveedor {
	return &CambiarEstadoProveedor{p}
}

func (uc *CambiarEstadoProveedor) Execute(ctx context.Context, c cmd.CambiarEstadoProveedor) (appresult.Result[dtoresult.Proveedor], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoProveedores, policy.ProveedorAdministrar); ae != nil {
		return rechazar[dtoresult.Proveedor](ae)
	}
	var motivo string
	if !c.Activo {
		m, ae := validarMotivo(c.Motivo)
		if ae != nil {
			return rechazar[dtoresult.Proveedor](ae)
		}
		motivo = m
	}
	actual, found, err := uc.proveedores.Obtener(ctx, c.ID)
	if err != nil {
		return tecnico[dtoresult.Proveedor]("buscar proveedor", err)
	}
	if !found {
		return rechazar[dtoresult.Proveedor](mapper.NoEncontrado("proveedor"))
	}
	var p dtoresult.Proveedor
	switch {
	case c.Activo && actual.EsActivo:
		return rechazar[dtoresult.Proveedor](apperror.New(apperror.KindConflict, CodigoYaActiva, "el proveedor ya está activo"))
	case !c.Activo && !actual.EsActivo:
		return rechazar[dtoresult.Proveedor](apperror.New(apperror.KindConflict, CodigoYaDadaDeBaja, "el proveedor ya está dado de baja"))
	case c.Activo:
		p, found, err = uc.proveedores.Restaurar(ctx, c.ID)
	default:
		p, found, err = uc.proveedores.DarBaja(ctx, c.ID, motivo)
	}
	if err != nil {
		return deProveedor(err, "cambiar estado del proveedor")
	}
	if !found {
		return rechazar[dtoresult.Proveedor](mapper.NoEncontrado("proveedor"))
	}
	return appresult.Ok(p), nil
}
