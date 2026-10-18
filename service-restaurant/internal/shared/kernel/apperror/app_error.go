// Package apperror define los rechazos esperados del negocio.
package apperror

// Kind clasifica un rechazo esperado; la capa web lo traduce a HTTP.
type Kind string

const (
	KindBadRequest   Kind = "bad_request"
	KindValidation   Kind = "validation"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindBusinessRule Kind = "business_rule"
)

// AppError es un rechazo esperado (no un fallo técnico).
type AppError struct {
	Kind    Kind
	Code    string
	Message string
}

func (e *AppError) Error() string { return e.Code + ": " + e.Message }

// New crea un AppError.
func New(kind Kind, code, message string) *AppError {
	return &AppError{Kind: kind, Code: code, Message: message}
}
