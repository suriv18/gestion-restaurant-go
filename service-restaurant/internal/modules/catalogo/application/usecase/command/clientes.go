// Package command implementa los comandos del directorio de clientes y proveedores.
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
	mensajeSinPermisoClientes = "no tiene permiso para administrar clientes"
	mensajeClienteDuplicado   = "ya existe un cliente con ese documento (aunque esté dado de baja)"
	documentoIncompleto       = "documento_incompleto"
	mensajeDocIncompleto      = "el tipo y el número de documento se indican juntos"
)

func deCliente(err error, contexto string) (appresult.Result[dtoresult.Cliente], error) {
	return deAlmacen[dtoresult.Cliente](contexto, mensajeClienteDuplicado, err)
}

// CrearCliente da de alta un cliente (RES-RF-004). El documento es opcional.
type CrearCliente struct{ clientes outbound.ClienteStore }

func NewCrearCliente(c outbound.ClienteStore) *CrearCliente { return &CrearCliente{c} }

func (uc *CrearCliente) Execute(ctx context.Context, c cmd.CrearCliente) (appresult.Result[dtoresult.Cliente], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoClientes, policy.ClienteAdministrar); ae != nil {
		return rechazar[dtoresult.Cliente](ae)
	}
	nombre, ae := validarNombre(c.Nombre, "nombre")
	if ae != nil {
		return rechazar[dtoresult.Cliente](ae)
	}
	nuevo := outbound.NuevoCliente{Nombre: nombre}
	if vacio(c.TipoDocumento) != vacio(c.NumeroDocumento) {
		return rechazar[dtoresult.Cliente](mapper.Validacion(documentoIncompleto, mensajeDocIncompleto))
	}
	if !vacio(c.TipoDocumento) {
		t, n, ae := validarDocumento(*c.TipoDocumento, *c.NumeroDocumento)
		if ae != nil {
			return rechazar[dtoresult.Cliente](ae)
		}
		nuevo.TipoDocumento, nuevo.NumeroDocumento = &t, &n
	}
	if nuevo.Telefono, ae = opcional(c.Telefono, "telefono", fmtTelefono); ae != nil {
		return rechazar[dtoresult.Cliente](ae)
	}
	if nuevo.Email, ae = opcional(c.Email, "email", fmtEmail); ae != nil {
		return rechazar[dtoresult.Cliente](ae)
	}
	if nuevo.Direccion, ae = opcional(c.Direccion, "direccion", fmtDireccion); ae != nil {
		return rechazar[dtoresult.Cliente](ae)
	}
	cli, err := uc.clientes.Crear(ctx, nuevo)
	if err != nil {
		return deCliente(err, "crear cliente")
	}
	return appresult.Ok(cli), nil
}

// documentoEditado resuelve el par tipo/número de un PATCH de cliente: ausentes → sin cambio;
// ambos nulos → se vacía el documento; ambos con valor → se valida. Un solo lado es inválido.
func documentoEditado(tipo, numero cmd.Campo[string]) (t, n outbound.CampoOpcional, ae *apperror.AppError) {
	if tipo.Presente != numero.Presente {
		return t, n, mapper.Validacion(documentoIncompleto, mensajeDocIncompleto)
	}
	if !tipo.Presente {
		return t, n, nil
	}
	if vacio(tipo.Valor) != vacio(numero.Valor) {
		return t, n, mapper.Validacion(documentoIncompleto, mensajeDocIncompleto)
	}
	if vacio(tipo.Valor) {
		return outbound.CampoOpcional{Presente: true}, outbound.CampoOpcional{Presente: true}, nil
	}
	tv, nv, ae := validarDocumento(*tipo.Valor, *numero.Valor)
	if ae != nil {
		return t, n, ae
	}
	return outbound.CampoOpcional{Presente: true, Valor: &tv}, outbound.CampoOpcional{Presente: true, Valor: &nv}, nil
}

// EditarCliente modifica solo los campos enviados de un cliente activo.
type EditarCliente struct{ clientes outbound.ClienteStore }

func NewEditarCliente(c outbound.ClienteStore) *EditarCliente { return &EditarCliente{c} }

func (uc *EditarCliente) Execute(ctx context.Context, c cmd.EditarCliente) (appresult.Result[dtoresult.Cliente], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoClientes, policy.ClienteAdministrar); ae != nil {
		return rechazar[dtoresult.Cliente](ae)
	}
	actual, found, err := uc.clientes.Obtener(ctx, c.ID)
	if err != nil {
		return tecnico[dtoresult.Cliente]("buscar cliente", err)
	}
	if !found {
		return rechazar[dtoresult.Cliente](mapper.NoEncontrado("cliente"))
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Cliente](apperror.New(apperror.KindConflict, CodigoClienteInactivo, "el cliente está dado de baja: restáurelo para editarlo"))
	}
	if c.Nombre == nil && !c.TipoDocumento.Presente && !c.NumeroDocumento.Presente &&
		!c.Telefono.Presente && !c.Email.Presente && !c.Direccion.Presente {
		return rechazar[dtoresult.Cliente](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosCliente
	if c.Nombre != nil {
		n, ae := validarNombre(*c.Nombre, "nombre")
		if ae != nil {
			return rechazar[dtoresult.Cliente](ae)
		}
		cambios.Nombre = &n
	}
	var ae *apperror.AppError
	if cambios.TipoDocumento, cambios.NumeroDocumento, ae = documentoEditado(c.TipoDocumento, c.NumeroDocumento); ae != nil {
		return rechazar[dtoresult.Cliente](ae)
	}
	if cambios.Telefono, ae = campoOpcional(c.Telefono, "telefono", fmtTelefono); ae != nil {
		return rechazar[dtoresult.Cliente](ae)
	}
	if cambios.Email, ae = campoOpcional(c.Email, "email", fmtEmail); ae != nil {
		return rechazar[dtoresult.Cliente](ae)
	}
	if cambios.Direccion, ae = campoOpcional(c.Direccion, "direccion", fmtDireccion); ae != nil {
		return rechazar[dtoresult.Cliente](ae)
	}
	cli, found, err := uc.clientes.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deCliente(err, "editar cliente")
	}
	if !found {
		return rechazar[dtoresult.Cliente](mapper.NoEncontrado("cliente"))
	}
	return appresult.Ok(cli), nil
}

// CambiarEstadoCliente da de baja (con motivo) o restaura un cliente (RES-RF-030). La
// restauración conserva el mismo id.
type CambiarEstadoCliente struct{ clientes outbound.ClienteStore }

func NewCambiarEstadoCliente(c outbound.ClienteStore) *CambiarEstadoCliente {
	return &CambiarEstadoCliente{c}
}

func (uc *CambiarEstadoCliente) Execute(ctx context.Context, c cmd.CambiarEstadoCliente) (appresult.Result[dtoresult.Cliente], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoClientes, policy.ClienteAdministrar); ae != nil {
		return rechazar[dtoresult.Cliente](ae)
	}
	var motivo string
	if !c.Activo {
		m, ae := validarMotivo(c.Motivo)
		if ae != nil {
			return rechazar[dtoresult.Cliente](ae)
		}
		motivo = m
	}
	actual, found, err := uc.clientes.Obtener(ctx, c.ID)
	if err != nil {
		return tecnico[dtoresult.Cliente]("buscar cliente", err)
	}
	if !found {
		return rechazar[dtoresult.Cliente](mapper.NoEncontrado("cliente"))
	}
	var cli dtoresult.Cliente
	switch {
	case c.Activo && actual.EsActivo:
		return rechazar[dtoresult.Cliente](apperror.New(apperror.KindConflict, CodigoYaActiva, "el cliente ya está activo"))
	case !c.Activo && !actual.EsActivo:
		return rechazar[dtoresult.Cliente](apperror.New(apperror.KindConflict, CodigoYaDadaDeBaja, "el cliente ya está dado de baja"))
	case c.Activo:
		cli, found, err = uc.clientes.Restaurar(ctx, c.ID)
	default:
		cli, found, err = uc.clientes.DarBaja(ctx, c.ID, motivo)
	}
	if err != nil {
		return deCliente(err, "cambiar estado del cliente")
	}
	if !found {
		return rechazar[dtoresult.Cliente](mapper.NoEncontrado("cliente"))
	}
	return appresult.Ok(cli), nil
}
