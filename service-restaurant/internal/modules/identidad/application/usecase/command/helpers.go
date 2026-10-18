package command

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// rechazar devuelve un rechazo esperado (no un error técnico).
func rechazar[T any](ae *apperror.AppError) (appresult.Result[T], error) {
	return appresult.Fail[T](ae), nil
}

// tecnico envuelve un error inesperado; el Result queda en su valor cero (inválido).
func tecnico[T any](contexto string, err error) (appresult.Result[T], error) {
	return appresult.Result[T]{}, fmt.Errorf("%s: %w", contexto, err)
}

// deAlmacen separa un error de almacenamiento conocido (rechazo) de uno técnico.
func deAlmacen[T any](contexto string, err error) (appresult.Result[T], error) {
	if ae := mapper.DeAlmacen(err); ae != nil {
		return rechazar[T](ae)
	}
	return tecnico[T](contexto, err)
}

// textoOpcional normaliza un texto opcional: nil o vacío → nil; demasiado largo → error.
func textoOpcional(p *string, max int) (*string, bool) {
	if p == nil {
		return nil, true
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil, true
	}
	if utf8.RuneCountInString(v) > max {
		return nil, false
	}
	return &v, true
}

// positivosUnicos descarta repetidos y exige ids > 0; ok=false si alguno es inválido.
func positivosUnicos(ids []int64) ([]int64, bool) {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, false
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, true
}

// diferenciaSimetrica devuelve los elementos que están en uno solo de los conjuntos.
func diferenciaSimetrica(a, b []int64) []int64 {
	enA := make(map[int64]bool, len(a))
	for _, x := range a {
		enA[x] = true
	}
	enB := make(map[int64]bool, len(b))
	for _, x := range b {
		enB[x] = true
	}
	var out []int64
	for x := range enA {
		if !enB[x] {
			out = append(out, x)
		}
	}
	for x := range enB {
		if !enA[x] {
			out = append(out, x)
		}
	}
	return out
}
