// Package mapper convierte errores de dominio en rechazos esperados de aplicación.
package mapper

import (
	"errors"

	"service-restaurant/internal/modules/identidad/domain/domainerror"
	"service-restaurant/internal/shared/kernel/apperror"
)

const (
	CodigoCredencialesInvalidas = "credenciales_invalidas"
	CodigoNoAutenticado         = "no_autenticado"
	CodigoYaInstalado           = "ya_instalado"
	CodigoDatosInvalidos        = "datos_invalidos"
)

// CredencialesInvalidas es siempre idéntico: no revela si el usuario existe, está
// inactivo, bloqueado o es el actor técnico.
func CredencialesInvalidas() *apperror.AppError {
	return apperror.New(apperror.KindUnauthorized, CodigoCredencialesInvalidas, "credenciales inválidas")
}

// NoAutenticado cubre token ausente, inválido, vencido o revocado.
func NoAutenticado() *apperror.AppError {
	return apperror.New(apperror.KindUnauthorized, CodigoNoAutenticado, "autenticación requerida")
}

// DeInvariante convierte un InvariantError conocido en un rechazo esperado.
// Devuelve nil si el error no es una invariante del dominio.
func DeInvariante(err error) *apperror.AppError {
	var ie *domainerror.InvariantError
	if errors.As(err, &ie) {
		return apperror.New(apperror.KindValidation, ie.Code, ie.Message)
	}
	return nil
}
