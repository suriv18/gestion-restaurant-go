// Package request decodifica cuerpos de solicitud de forma estricta.
package request

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"service-restaurant/internal/shared/kernel/apperror"
)

func bad(code, msg string) *apperror.AppError {
	return apperror.New(apperror.KindBadRequest, code, msg)
}

// DecodeJSON exige Content-Type application/json, rechaza campos desconocidos y
// contenido extra. El tamaño máximo lo limita http.MaxBytesHandler en el router.
func DecodeJSON(r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		mt, _, err := mime.ParseMediaType(ct)
		if err != nil || mt != "application/json" {
			return bad("content_type_invalido", "Content-Type debe ser application/json")
		}
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooBig *http.MaxBytesError
		switch {
		case errors.As(err, &tooBig):
			return bad("cuerpo_demasiado_grande", "el cuerpo excede el tamaño permitido")
		case errors.Is(err, io.EOF):
			return bad("cuerpo_requerido", "se requiere un cuerpo JSON")
		default:
			return bad("json_invalido", "el cuerpo no es un JSON válido para esta operación")
		}
	}
	if dec.More() {
		return bad("json_invalido", "el cuerpo debe contener un único objeto JSON")
	}
	return nil
}
