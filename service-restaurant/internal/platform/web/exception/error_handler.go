// Package exception mapea errores a Problem Details seguros.
package exception

import (
	"errors"
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/middleware"
	"service-restaurant/internal/platform/web/response"
	"service-restaurant/internal/shared/kernel/apperror"
)

// StatusFor traduce un Kind a código HTTP.
func StatusFor(k apperror.Kind) int {
	switch k {
	case apperror.KindBadRequest:
		return http.StatusBadRequest
	case apperror.KindValidation, apperror.KindBusinessRule:
		return http.StatusUnprocessableEntity
	case apperror.KindNotFound:
		return http.StatusNotFound
	case apperror.KindConflict:
		return http.StatusConflict
	case apperror.KindUnauthorized:
		return http.StatusUnauthorized
	case apperror.KindForbidden:
		return http.StatusForbidden
	}
	return http.StatusInternalServerError
}

// Handle escribe el problema. Errores no conocidos devuelven 500 sin filtrar detalles.
func Handle(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	rid := middleware.RequestIDFrom(r.Context())
	var ae *apperror.AppError
	if errors.As(err, &ae) {
		response.WriteProblem(w, response.Problem{
			Status: StatusFor(ae.Kind), Detail: ae.Message, Code: ae.Code, RequestID: rid,
		})
		return
	}
	logger.Error("error inesperado", "error", err, "request_id", rid, "path", r.URL.Path)
	response.WriteProblem(w, response.Problem{
		Status: http.StatusInternalServerError, Detail: "error interno", RequestID: rid,
	})
}
