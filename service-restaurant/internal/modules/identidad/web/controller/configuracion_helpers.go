package controller

import (
	"net/url"

	"service-restaurant/internal/shared/kernel/apperror"
)

// activoDe lee el filtro opcional es_activo ("true" o "false").
func activoDe(q url.Values) (*bool, error) {
	switch v := q.Get("es_activo"); v {
	case "":
		return nil, nil
	case "true":
		t := true
		return &t, nil
	case "false":
		f := false
		return &f, nil
	}
	return nil, apperror.New(apperror.KindBadRequest, "es_activo_invalido", "es_activo debe ser true o false")
}
