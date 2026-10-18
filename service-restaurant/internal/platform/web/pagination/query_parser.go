// Package pagination interpreta parámetros de paginación de la query string.
package pagination

import (
	"net/url"
	"strconv"

	"service-restaurant/internal/shared/application/pagination"
	"service-restaurant/internal/shared/kernel/apperror"
)

// ParsePageRequest lee page y size; valores inválidos son un rechazo de validación.
func ParsePageRequest(q url.Values) (pagination.PageRequest, error) {
	p := pagination.PageRequest{Size: pagination.DefaultSize}
	if v := q.Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return p, apperror.New(apperror.KindValidation, "page_invalid", "page debe ser un entero >= 0")
		}
		p.Page = n
	}
	if v := q.Get("size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > pagination.MaxSize {
			return p, apperror.New(apperror.KindValidation, "size_invalid", "size debe estar entre 1 y 100")
		}
		p.Size = n
	}
	return p, nil
}
