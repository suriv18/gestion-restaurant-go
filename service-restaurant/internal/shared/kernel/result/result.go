// Package result modela el resultado de un caso de uso: éxito o rechazo esperado.
// El valor cero de Result[T] es inválido; solo Ok y Fail crean valores válidos.
package result

import "service-restaurant/internal/shared/kernel/apperror"

type Result[T any] struct {
	value T
	err   *apperror.AppError
	valid bool
}

// Ok construye un resultado exitoso.
func Ok[T any](value T) Result[T] { return Result[T]{value: value, valid: true} }

// Fail construye un rechazo esperado; un error nil produce un Result inválido.
func Fail[T any](err *apperror.AppError) Result[T] {
	if err == nil {
		return Result[T]{}
	}
	return Result[T]{err: err, valid: true}
}

// IsValid es false para el valor cero (error de programación).
func (r Result[T]) IsValid() bool { return r.valid }

// IsOk indica éxito; solo es true si el resultado es válido.
func (r Result[T]) IsOk() bool { return r.valid && r.err == nil }

// Value devuelve el valor (cero si es fallido).
func (r Result[T]) Value() T { return r.value }

// Err devuelve el rechazo, o nil si es exitoso.
func (r Result[T]) Err() *apperror.AppError { return r.err }

// CanCommit es el predicado para el runner de transacciones.
func CanCommit[T any](r Result[T]) bool { return r.IsOk() }
