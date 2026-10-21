// Package controller expone el módulo atencion por HTTP.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/platform/idempotency"
	"service-restaurant/internal/platform/web/middleware"
	webrequest "service-restaurant/internal/platform/web/request"
	"service-restaurant/internal/platform/web/response"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

var errResultadoInvalido = errors.New("el caso de uso devolvió un Result inválido")

// responder traduce el Result de un caso de uso a la respuesta HTTP.
func responder[T, R any](w http.ResponseWriter, res appresult.Result[T], err error, status int, f func(T) R) error {
	if err != nil {
		return err
	}
	if !res.IsValid() {
		return errResultadoInvalido
	}
	if res.Err() != nil {
		return res.Err()
	}
	w.Header().Set("Cache-Control", "no-store")
	response.WriteJSON(w, status, f(res.Value()))
	return nil
}

func actorDe(r *http.Request) (appctx.Actor, error) {
	a, ok := middleware.ActorFrom(r.Context())
	if !ok {
		return appctx.Actor{}, mapper.NoAutenticado()
	}
	return a, nil
}

// pathID lee el id numérico positivo de la ruta.
func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperror.New(apperror.KindBadRequest, "id_invalido", "el id de la ruta no es válido")
	}
	return id, nil
}

// idCadena convierte un id en cadena (contrato) a int64 positivo.
func idCadena(valor, campo string) (int64, error) {
	n, err := strconv.ParseInt(valor, 10, 64)
	if err != nil || n <= 0 {
		return 0, apperror.New(apperror.KindBadRequest, campo+"_invalido", campo+" debe ser un entero positivo en cadena")
	}
	return n, nil
}

// intencion arma la intención idempotente: exige Idempotency-Key (UUID) y calcula la huella
// sobre la operación y el cuerpo canónico (el DTO tipado vuelto a serializar).
func intencion(r *http.Request, operacion string, cuerpo any) (cmd.Intencion, error) {
	clave, err := idempotency.ParseKey(r.Header.Get("Idempotency-Key"))
	if err != nil {
		return cmd.Intencion{}, err
	}
	var raw []byte
	if cuerpo != nil {
		if raw, err = json.Marshal(cuerpo); err != nil {
			return cmd.Intencion{}, err
		}
	}
	return cmd.Intencion{Clave: clave, Operacion: operacion, Huella: idempotency.Fingerprint(operacion, raw)}, nil
}

// leerCuerpoOpcionalVacio acepta ausencia de cuerpo o "{}" (restauraciones y bajas simples).
func leerCuerpoOpcionalVacio(r *http.Request, dst any) error {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 {
		return nil
	}
	return webrequest.DecodeJSON(r, dst)
}

// consulta analiza la query string de forma estricta: un cursor corrupto no puede tratarse
// como "primera página".
func consulta(r *http.Request) (url.Values, error) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, apperror.New(apperror.KindBadRequest, "query_invalida", "los parámetros de la consulta están mal codificados")
	}
	return q, nil
}

// limiteDe lee limit (1..100, por defecto 25).
func limiteDe(q url.Values) (int, error) {
	v := q.Get("limit")
	if v == "" {
		return 25, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > 100 {
		return 0, apperror.New(apperror.KindBadRequest, "limit_invalido", "limit debe estar entre 1 y 100")
	}
	return n, nil
}

// activoDe lee el filtro opcional es_activo ("true" o "false").
func activoDe(q url.Values) (*bool, error) {
	switch q.Get("es_activo") {
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

// sucursalDe lee el filtro opcional sucursal_id.
func sucursalDe(q url.Values) (*int64, error) {
	v := q.Get("sucursal_id")
	if v == "" {
		return nil, nil
	}
	n, err := idCadena(v, "sucursal_id")
	if err != nil {
		return nil, err
	}
	return &n, nil
}
