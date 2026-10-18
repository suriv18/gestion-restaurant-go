// Package domainerror contiene los errores tipados de invariantes del dominio.
package domainerror

// InvariantError señala una regla de negocio incumplida. El caso de uso lo convierte
// en un rechazo esperado (AppError); los demás errores continúan por el canal error.
type InvariantError struct {
	Code    string
	Message string
}

func (e *InvariantError) Error() string { return e.Code + ": " + e.Message }

// New crea un InvariantError.
func New(code, message string) *InvariantError {
	return &InvariantError{Code: code, Message: message}
}
