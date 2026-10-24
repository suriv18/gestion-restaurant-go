package result

import "time"

// Baja reúne los marcadores de la baja lógica de un maestro.
type Baja struct {
	MotivoEliminacion *string
	EliminadoEn       *time.Time
	EliminadoPor      *int64
}

// Auditoria reúne los datos de creación y última modificación.
type Auditoria struct {
	CreadoEn       time.Time
	CreadoPor      int64
	ActualizadoEn  time.Time
	ActualizadoPor int64
}

// Cliente es quien compra o reserva; el documento es opcional.
type Cliente struct {
	ID              int64
	TipoDocumento   *string
	NumeroDocumento *string
	Nombre          string
	Telefono        *string
	Email           *string
	Direccion       *string
	EsActivo        bool
	Baja
	Auditoria
}

// PaginaClientes: NextCursor vacío indica la última página.
type PaginaClientes struct {
	Items      []Cliente
	NextCursor string
}

// Proveedor es quien abastece al restaurante; su documento es obligatorio.
type Proveedor struct {
	ID              int64
	TipoDocumento   string
	NumeroDocumento string
	RazonSocial     string
	ContactoNombre  *string
	Telefono        *string
	Email           *string
	Direccion       *string
	EsActivo        bool
	Baja
	Auditoria
}

type PaginaProveedores struct {
	Items      []Proveedor
	NextCursor string
}
