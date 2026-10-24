package command

import (
	"context"
	"errors"
	"strings"
	"time"

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
	maxNombresEmpleado = 120
	maxCargoEmpleado   = 80
	formatoFecha       = "2006-01-02"

	mensajeSinPermisoEmpleados = "no tiene permiso para administrar empleados"

	CodigoDocumentoDuplicado    = "documento_duplicado"
	CodigoUsuarioYaVinculado    = "usuario_ya_vinculado"
	CodigoUsuarioInvalido       = "usuario_invalido"
	CodigoEmpleadoInactivo      = "empleado_inactivo"
	CodigoEmpleadoConUsuario    = "empleado_con_usuario_activo"
	CodigoFechasIncoherentes    = "fechas_incoherentes"
	CodigoEmpleadoYaDadoDeBaja  = "ya_dado_de_baja"
	CodigoEmpleadoYaActivo      = "ya_activo"
	prefijoReferenciaUsuarioReg = "Referencia usuario_id="
)

// deEmpleado traduce los rechazos propios del empleado y delega el resto.
func deEmpleado[T any](contexto string, err error) (appresult.Result[T], error) {
	var regla *outbound.ErrReglaDatos
	switch {
	case errors.Is(err, outbound.ErrDocumentoEmpleadoDuplicado):
		return rechazar[T](apperror.New(apperror.KindConflict, CodigoDocumentoDuplicado,
			"ya existe un empleado con ese documento (aunque esté dado de baja)"))
	case errors.Is(err, outbound.ErrUsuarioYaVinculado):
		return rechazar[T](apperror.New(apperror.KindConflict, CodigoUsuarioYaVinculado, "ese usuario ya está vinculado a otro empleado"))
	case errors.As(err, &regla) && strings.HasPrefix(regla.Mensaje, prefijoReferenciaUsuarioReg):
		return rechazar[T](mapper.Validacion(CodigoUsuarioInvalido, "el usuario vinculado no existe o está dado de baja"))
	}
	return deAlmacen[T](contexto, err)
}

func validarTextoEmpleado(raw, campo string, max int, msg string) (string, *apperror.AppError) {
	v, err := vo.TextoUnaLinea(raw, max)
	if err != nil {
		return "", mapper.Validacion(campo+"_invalido", msg)
	}
	return v, nil
}

func validarDocumentoEmpleado(tipo, numero string) (string, string, *apperror.AppError) {
	t, n, err := vo.NewDocumento(tipo, numero)
	if err != nil {
		return "", "", mapper.Validacion("documento_invalido",
			"el documento no es válido: el tipo admite letras y dígitos (hasta 10), el número hasta 20 caracteres; DNI exige 8 dígitos y RUC 11")
	}
	return t, n, nil
}

func validarFecha(raw, campo string) (string, *apperror.AppError) {
	f, err := time.Parse(formatoFecha, strings.TrimSpace(raw))
	if err != nil {
		return "", mapper.Validacion(campo+"_invalida", "la fecha debe tener el formato AAAA-MM-DD")
	}
	return f.Format(formatoFecha), nil
}

type formatoTexto func(string) (string, error)

func opcionalEmpleado(p *string, campo, msg string, f formatoTexto) (*string, *apperror.AppError) {
	if p == nil || strings.TrimSpace(*p) == "" {
		return nil, nil
	}
	v, err := f(*p)
	if err != nil {
		return nil, mapper.Validacion(campo+"_invalido", msg)
	}
	return &v, nil
}

func fmtCargo(s string) (string, error) { return vo.TextoUnaLinea(s, maxCargoEmpleado) }

const (
	msgTelefono = "el teléfono admite dígitos, espacios y + ( ) - . (5 a 30 caracteres)"
	msgCargo    = "el cargo admite hasta 80 caracteres"
)

func campoEmpleado(c cmd.Campo[string], campo, msg string, f formatoTexto) (outbound.CampoOpcional, *apperror.AppError) {
	if !c.Presente {
		return outbound.CampoOpcional{}, nil
	}
	v, ae := opcionalEmpleado(c.Valor, campo, msg, f)
	if ae != nil {
		return outbound.CampoOpcional{}, ae
	}
	return outbound.CampoOpcional{Presente: true, Valor: v}, nil
}

// validarDatosEmpleado normaliza los datos personales; no toca usuario ni fechas.
func validarDatosEmpleado(d cmd.DatosEmpleado) (outbound.NuevoEmpleado, *apperror.AppError) {
	var n outbound.NuevoEmpleado
	var ae *apperror.AppError
	if n.Nombres, ae = validarTextoEmpleado(d.Nombres, "nombres", maxNombresEmpleado, "los nombres son obligatorios (máximo 120 caracteres)"); ae != nil {
		return n, ae
	}
	if n.Apellidos, ae = validarTextoEmpleado(d.Apellidos, "apellidos", maxNombresEmpleado, "los apellidos son obligatorios (máximo 120 caracteres)"); ae != nil {
		return n, ae
	}
	if n.TipoDocumento, n.NumeroDocumento, ae = validarDocumentoEmpleado(d.TipoDocumento, d.NumeroDocumento); ae != nil {
		return n, ae
	}
	if n.Telefono, ae = opcionalEmpleado(d.Telefono, "telefono", msgTelefono, vo.NewTelefono); ae != nil {
		return n, ae
	}
	if n.Cargo, ae = opcionalEmpleado(d.Cargo, "cargo", msgCargo, fmtCargo); ae != nil {
		return n, ae
	}
	return n, nil
}

// usuarioVinculable verifica que el usuario exista, no sea el técnico y esté activo.
func usuarioVinculable(ctx context.Context, store outbound.EmpleadoStore, id int64) (*apperror.AppError, error) {
	ref, found, err := store.Usuario(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found || ref.EsSistema || !ref.EsActivo {
		return mapper.Validacion(CodigoUsuarioInvalido, "el usuario no existe o está dado de baja"), nil
	}
	return nil, nil
}

// CrearEmpleado da de alta a un empleado, con o sin usuario vinculado.
type CrearEmpleado struct{ empleados outbound.EmpleadoStore }

func NewCrearEmpleado(e outbound.EmpleadoStore) *CrearEmpleado { return &CrearEmpleado{e} }

func (uc *CrearEmpleado) Execute(ctx context.Context, c cmd.CrearEmpleado) (appresult.Result[dtoresult.Empleado], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoEmpleados, policy.EmpleadoAdministrar); ae != nil {
		return rechazar[dtoresult.Empleado](ae)
	}
	nuevo, ae := validarDatosEmpleado(c.DatosEmpleado)
	if ae != nil {
		return rechazar[dtoresult.Empleado](ae)
	}
	if c.FechaIngreso != nil && strings.TrimSpace(*c.FechaIngreso) != "" {
		f, ae := validarFecha(*c.FechaIngreso, "fecha_ingreso")
		if ae != nil {
			return rechazar[dtoresult.Empleado](ae)
		}
		nuevo.FechaIngreso = &f
	}
	if c.UsuarioID != nil {
		if *c.UsuarioID <= 0 {
			return rechazar[dtoresult.Empleado](mapper.Validacion(CodigoUsuarioInvalido, "el usuario no existe o está dado de baja"))
		}
		ae, err := usuarioVinculable(ctx, uc.empleados, *c.UsuarioID)
		if err != nil {
			return tecnico[dtoresult.Empleado]("buscar usuario", err)
		}
		if ae != nil {
			return rechazar[dtoresult.Empleado](ae)
		}
		nuevo.UsuarioID = c.UsuarioID
	}
	e, err := uc.empleados.Crear(ctx, nuevo)
	if err != nil {
		return deEmpleado[dtoresult.Empleado]("crear empleado", err)
	}
	return appresult.Ok(e), nil
}

// EditarEmpleado modifica solo los campos enviados de un empleado activo.
type EditarEmpleado struct{ empleados outbound.EmpleadoStore }

func NewEditarEmpleado(e outbound.EmpleadoStore) *EditarEmpleado { return &EditarEmpleado{e} }

func (uc *EditarEmpleado) Execute(ctx context.Context, c cmd.EditarEmpleado) (appresult.Result[dtoresult.Empleado], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoEmpleados, policy.EmpleadoAdministrar); ae != nil {
		return rechazar[dtoresult.Empleado](ae)
	}
	actual, found, err := uc.empleados.Obtener(ctx, c.ID)
	if err != nil {
		return tecnico[dtoresult.Empleado]("buscar empleado", err)
	}
	if !found {
		return rechazar[dtoresult.Empleado](mapper.NoEncontrado("empleado"))
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Empleado](apperror.New(apperror.KindConflict, CodigoEmpleadoInactivo, "el empleado está dado de baja: restáurelo para editarlo"))
	}
	if c.TipoDocumento == nil && c.NumeroDocumento == nil && c.Nombres == nil && c.Apellidos == nil && c.FechaIngreso == nil &&
		!c.Telefono.Presente && !c.Cargo.Presente && !c.FechaCese.Presente && !c.UsuarioID.Presente {
		return rechazar[dtoresult.Empleado](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosEmpleado
	var ae *apperror.AppError
	if c.Nombres != nil {
		v, ae := validarTextoEmpleado(*c.Nombres, "nombres", maxNombresEmpleado, "los nombres son obligatorios (máximo 120 caracteres)")
		if ae != nil {
			return rechazar[dtoresult.Empleado](ae)
		}
		cambios.Nombres = &v
	}
	if c.Apellidos != nil {
		v, ae := validarTextoEmpleado(*c.Apellidos, "apellidos", maxNombresEmpleado, "los apellidos son obligatorios (máximo 120 caracteres)")
		if ae != nil {
			return rechazar[dtoresult.Empleado](ae)
		}
		cambios.Apellidos = &v
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
		t, n, ae := validarDocumentoEmpleado(tipo, numero)
		if ae != nil {
			return rechazar[dtoresult.Empleado](ae)
		}
		cambios.TipoDocumento, cambios.NumeroDocumento = &t, &n
	}
	if cambios.Telefono, ae = campoEmpleado(c.Telefono, "telefono", msgTelefono, vo.NewTelefono); ae != nil {
		return rechazar[dtoresult.Empleado](ae)
	}
	if cambios.Cargo, ae = campoEmpleado(c.Cargo, "cargo", msgCargo, fmtCargo); ae != nil {
		return rechazar[dtoresult.Empleado](ae)
	}
	ingreso, cese := actual.FechaIngreso, actual.FechaCese
	if c.FechaIngreso != nil {
		f, ae := validarFecha(*c.FechaIngreso, "fecha_ingreso")
		if ae != nil {
			return rechazar[dtoresult.Empleado](ae)
		}
		ingreso, cambios.FechaIngreso = f, &f
	}
	if c.FechaCese.Presente {
		cambios.FechaCese.Presente = true
		cese = nil
		if c.FechaCese.Valor != nil && strings.TrimSpace(*c.FechaCese.Valor) != "" {
			f, ae := validarFecha(*c.FechaCese.Valor, "fecha_cese")
			if ae != nil {
				return rechazar[dtoresult.Empleado](ae)
			}
			cambios.FechaCese.Valor, cese = &f, &f
		}
	}
	if cese != nil && *cese < ingreso {
		return rechazar[dtoresult.Empleado](mapper.Validacion(CodigoFechasIncoherentes, "la fecha de cese no puede ser anterior a la de ingreso"))
	}
	if c.UsuarioID.Presente {
		cambios.UsuarioID.Presente = true
		if c.UsuarioID.Valor != nil {
			if *c.UsuarioID.Valor <= 0 {
				return rechazar[dtoresult.Empleado](mapper.Validacion(CodigoUsuarioInvalido, "el usuario no existe o está dado de baja"))
			}
			ae, err := usuarioVinculable(ctx, uc.empleados, *c.UsuarioID.Valor)
			if err != nil {
				return tecnico[dtoresult.Empleado]("buscar usuario", err)
			}
			if ae != nil {
				return rechazar[dtoresult.Empleado](ae)
			}
			cambios.UsuarioID.Valor = c.UsuarioID.Valor
		}
	}
	e, found, err := uc.empleados.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deEmpleado[dtoresult.Empleado]("editar empleado", err)
	}
	if !found {
		return rechazar[dtoresult.Empleado](mapper.NoEncontrado("empleado"))
	}
	return appresult.Ok(e), nil
}

// CambiarEstadoEmpleado da de baja (con motivo) o restaura un empleado. La baja se rechaza
// mientras su usuario siga activo: primero se da de baja el usuario.
type CambiarEstadoEmpleado struct{ empleados outbound.EmpleadoStore }

func NewCambiarEstadoEmpleado(e outbound.EmpleadoStore) *CambiarEstadoEmpleado {
	return &CambiarEstadoEmpleado{e}
}

func (uc *CambiarEstadoEmpleado) Execute(ctx context.Context, c cmd.CambiarEstadoEmpleado) (appresult.Result[dtoresult.Empleado], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoEmpleados, policy.EmpleadoAdministrar); ae != nil {
		return rechazar[dtoresult.Empleado](ae)
	}
	var motivo string
	if !c.Activo {
		m, err := vo.TextoObligatorio(c.Motivo, maxMotivo)
		if err != nil {
			return rechazar[dtoresult.Empleado](mapper.Validacion("motivo_requerido", "la baja exige un motivo (máximo 2000 caracteres)"))
		}
		motivo = m
	}
	actual, found, err := uc.empleados.Obtener(ctx, c.ID)
	if err != nil {
		return tecnico[dtoresult.Empleado]("buscar empleado", err)
	}
	if !found {
		return rechazar[dtoresult.Empleado](mapper.NoEncontrado("empleado"))
	}
	var e dtoresult.Empleado
	switch {
	case c.Activo && actual.EsActivo:
		return rechazar[dtoresult.Empleado](apperror.New(apperror.KindConflict, CodigoEmpleadoYaActivo, "el empleado ya está activo"))
	case !c.Activo && !actual.EsActivo:
		return rechazar[dtoresult.Empleado](apperror.New(apperror.KindConflict, CodigoEmpleadoYaDadoDeBaja, "el empleado ya está dado de baja"))
	case c.Activo:
		e, found, err = uc.empleados.Restaurar(ctx, c.ID)
	default:
		activo, errU := uc.empleados.UsuarioActivoDe(ctx, c.ID)
		if errU != nil {
			return tecnico[dtoresult.Empleado]("buscar usuario del empleado", errU)
		}
		if activo {
			return rechazar[dtoresult.Empleado](apperror.New(apperror.KindConflict, CodigoEmpleadoConUsuario,
				"el empleado tiene un usuario activo: dé de baja primero su usuario"))
		}
		e, found, err = uc.empleados.DarBaja(ctx, c.ID, motivo)
	}
	if err != nil {
		return deEmpleado[dtoresult.Empleado]("cambiar estado del empleado", err)
	}
	if !found {
		return rechazar[dtoresult.Empleado](mapper.NoEncontrado("empleado"))
	}
	return appresult.Ok(e), nil
}
