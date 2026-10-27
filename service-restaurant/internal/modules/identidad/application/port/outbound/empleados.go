package outbound

import (
	"context"
	"fmt"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
)

// Colisiones únicas que tienen un rechazo propio.
var (
	ErrDocumentoEmpleadoDuplicado = fmt.Errorf("documento de empleado duplicado: %w", ErrDuplicado)
	ErrUsuarioYaVinculado         = fmt.Errorf("usuario ya vinculado a un empleado: %w", ErrDuplicado)
)

// FiltroEmpleados acota el listado por estado y texto (ya sin comodines de LIKE).
type FiltroEmpleados struct {
	Activo      *bool
	Texto       string
	DespuesDeID int64
	Limite      int
}

// EmpleadoReader lee empleados sin bloquear filas.
type EmpleadoReader interface {
	ListarEmpleados(ctx context.Context, f FiltroEmpleados) ([]dtoresult.Empleado, error)
	ObtenerEmpleado(ctx context.Context, id int64) (dtoresult.Empleado, bool, error)
}

type NuevoEmpleado struct {
	UsuarioID                      *int64
	TipoDocumento, NumeroDocumento string
	Nombres, Apellidos             string
	Telefono, Cargo                *string
	FechaIngreso                   *string
}

// CampoEntero es un cambio de un campo anulable entero: si Presente, se escribe Valor (nil = NULL).
type CampoEntero struct {
	Presente bool
	Valor    *int64
}

// CambiosEmpleado: solo los campos presentes se modifican.
type CambiosEmpleado struct {
	TipoDocumento, NumeroDocumento, Nombres, Apellidos, FechaIngreso *string
	Telefono, Cargo, FechaCese                                       CampoOpcional
	UsuarioID                                                        CampoEntero
}

// UsuarioRef resume el usuario que se vincula a un empleado.
type UsuarioRef struct {
	EsSistema bool
	EsActivo  bool
}

// EmpleadoStore escribe empleados dentro de la transacción del comando.
type EmpleadoStore interface {
	Crear(ctx context.Context, n NuevoEmpleado) (dtoresult.Empleado, error)
	// Obtener lee el empleado bloqueando su fila (FOR UPDATE); found=false si no existe.
	Obtener(ctx context.Context, id int64) (dtoresult.Empleado, bool, error)
	Actualizar(ctx context.Context, id int64, c CambiosEmpleado) (dtoresult.Empleado, bool, error)
	DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Empleado, bool, error)
	Restaurar(ctx context.Context, id int64) (dtoresult.Empleado, bool, error)
	// Usuario lee el usuario en modo compartido: no puede darse de baja hasta que termine el comando.
	Usuario(ctx context.Context, id int64) (UsuarioRef, bool, error)
	// UsuarioActivoDe indica si el empleado tiene un usuario vinculado y activo.
	UsuarioActivoDe(ctx context.Context, empleadoID int64) (bool, error)
}
