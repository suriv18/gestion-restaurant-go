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

// ErrReglaDatos señala una regla de integridad de la base (triggers de la V4).
type ErrReglaDatos struct{ Mensaje string }

func (e *ErrReglaDatos) Error() string { return "regla de datos: " + e.Mensaje }
