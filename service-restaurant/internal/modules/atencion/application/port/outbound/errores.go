// Package outbound declara los puertos de salida del módulo atencion.
package outbound

import "errors"

// Errores de almacenamiento que los casos de uso traducen a rechazos esperados.
var (
	ErrDuplicado          = errors.New("registro duplicado")
	ErrReferenciaInvalida = errors.New("referencia inexistente")
)

// ErrReglaDatos señala una regla de integridad de la base (triggers de la V4).
type ErrReglaDatos struct{ Mensaje string }

func (e *ErrReglaDatos) Error() string { return "regla de datos: " + e.Mensaje }
