package command

import (
	"strconv"
	"strings"
	"time"

	"service-restaurant/internal/modules/catalogo/application/mapper"
	"service-restaurant/internal/shared/kernel/apperror"
)

const (
	maxFundamento = 2000
	layoutFecha   = "2006-01-02"
	// Las vigencias se acotan a un rango razonable: fuera de él casi seguro es un error de tecleo.
	anioMinimo = 1900
	anioMaximo = 2999
)

var fmtFundamento = texto(maxFundamento, true)

// validarTasa comprueba una tasa 0..100 con hasta 4 decimales, sin pasar por float: solo
// dígitos, un punto opcional y entre 1 y 4 decimales. Devuelve el valor normalizado a 4
// decimales (la misma forma que entrega numeric(7,4)::text), p. ej. "18" → "18.0000".
func validarTasa(raw string) (string, *apperror.AppError) {
	malo := mapper.Validacion("tasa_invalida", "la tasa debe ser un decimal entre 0 y 100 con hasta 4 decimales (p. ej. 18 o 10.5)")
	ent, dec, tienePunto := strings.Cut(raw, ".")
	if ent == "" || len(ent) > 3 || !soloDigitos(ent) {
		return "", malo
	}
	if tienePunto && (dec == "" || len(dec) > 4 || !soloDigitos(dec)) {
		return "", malo
	}
	dec += strings.Repeat("0", 4-len(dec))
	// Valor máximo: 100.0000. Con tres dígitos enteros solo "100" (y decimales en cero) es válido.
	n := 0
	for _, c := range ent {
		n = n*10 + int(c-'0')
	}
	if n > 100 || (n == 100 && dec != "0000") {
		return "", malo
	}
	return strconv.Itoa(n) + "." + dec, nil
}

func soloDigitos(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

// parsearFecha valida una fecha YYYY-MM-DD estricta dentro del rango admitido.
func parsearFecha(raw, campo string) (time.Time, *apperror.AppError) {
	f, err := time.Parse(layoutFecha, raw)
	if err != nil || len(raw) != len(layoutFecha) || f.Year() < anioMinimo || f.Year() > anioMaximo {
		return time.Time{}, mapper.Validacion(campo+"_invalido", "la fecha debe tener el formato AAAA-MM-DD (años 1900 a 2999)")
	}
	return f, nil
}

// vigencia valida el par (desde, hasta) y devuelve ambos normalizados; hasta nil = abierta.
func vigencia(desde string, hasta *string) (string, *string, *apperror.AppError) {
	d, ae := parsearFecha(desde, "vigente_desde")
	if ae != nil {
		return "", nil, ae
	}
	if hasta == nil {
		return d.Format(layoutFecha), nil, nil
	}
	h, ae := parsearFecha(*hasta, "vigente_hasta")
	if ae != nil {
		return "", nil, ae
	}
	if !h.After(d) {
		return "", nil, mapper.Validacion(CodigoVigenciaInvalida, "vigente_hasta debe ser posterior a vigente_desde (la vigencia es [desde, hasta))")
	}
	s := h.Format(layoutFecha)
	return d.Format(layoutFecha), &s, nil
}
