package outbound

import (
	"errors"
	"fmt"
)

// Errores de almacenamiento que los casos de uso traducen a rechazos esperados.
var (
	ErrDuplicado          = errors.New("registro duplicado")
	ErrReferenciaInvalida = errors.New("referencia inexistente")
)

// ErrDocumentoDuplicado distingue la colisión de (tipo, número) de documento. El índice único
// sigue vigente tras la baja: un documento dado de baja no se libera.
var ErrDocumentoDuplicado = fmt.Errorf("documento duplicado: %w", ErrDuplicado)

// ErrCategoriaDuplicada distingue la colisión del nombre de categoría (único también tras la baja).
var ErrCategoriaDuplicada = fmt.Errorf("categoría duplicada: %w", ErrDuplicado)

// ErrReglaDatos señala una regla de integridad de la base (triggers de la V4).
type ErrReglaDatos struct{ Mensaje string }

func (e *ErrReglaDatos) Error() string { return "regla de datos: " + e.Mensaje }

// ErrVigenciaSolapada señala que la vigencia de una tasa se cruza con otra del mismo impuesto
// (restricción de exclusión, SQLSTATE 23P01).
var ErrVigenciaSolapada = errors.New("vigencia de tasa solapada")

// ErrEstacionDuplicada distingue la colisión de (sucursal, código) de estación (único también
// tras la baja).
var ErrEstacionDuplicada = fmt.Errorf("estación duplicada: %w", ErrDuplicado)

// ErrProductoDuplicado distingue la colisión del código de producto (único también tras la baja).
var ErrProductoDuplicado = fmt.Errorf("producto duplicado: %w", ErrDuplicado)
