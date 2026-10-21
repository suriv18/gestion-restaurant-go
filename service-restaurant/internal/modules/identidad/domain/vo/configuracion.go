package vo

import (
	"errors"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata" // zonas IANA embebidas: la imagen de ejecución puede no traerlas
)

// ErrFormatoInvalido señala un valor de configuración que no cumple su formato.
var ErrFormatoInvalido = errors.New("formato inválido")

var (
	ubigeoRe         = regexp.MustCompile(`^[0-9]{6}$`)
	monedaRe         = regexp.MustCompile(`^[A-Z]{3}$`)
	codigoSucursalRe = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{1,19}$`)
)

// Ubigeo es el código de ubicación de seis dígitos.
type Ubigeo struct{ value string }

func NewUbigeo(raw string) (Ubigeo, error) {
	v := strings.TrimSpace(raw)
	if !ubigeoRe.MatchString(v) {
		return Ubigeo{}, ErrFormatoInvalido
	}
	return Ubigeo{v}, nil
}

func (u Ubigeo) String() string { return u.value }

// Moneda es un código de tres letras mayúsculas (ISO 4217, p. ej. PEN).
type Moneda struct{ value string }

func NewMoneda(raw string) (Moneda, error) {
	v := strings.ToUpper(strings.TrimSpace(raw))
	if !monedaRe.MatchString(v) {
		return Moneda{}, ErrFormatoInvalido
	}
	return Moneda{v}, nil
}

func (m Moneda) String() string { return m.value }

// ZonaHoraria es un nombre de zona IANA conocido (America/Lima). Se rechaza "Local".
type ZonaHoraria struct{ value string }

func NewZonaHoraria(raw string) (ZonaHoraria, error) {
	v := strings.TrimSpace(raw)
	if v == "" || strings.EqualFold(v, "local") {
		return ZonaHoraria{}, ErrFormatoInvalido
	}
	if _, err := time.LoadLocation(v); err != nil {
		return ZonaHoraria{}, ErrFormatoInvalido
	}
	return ZonaHoraria{v}, nil
}

func (z ZonaHoraria) String() string { return z.value }

// CodigoSucursal identifica una sede: mayúsculas, dígitos, _ y - (2 a 20).
type CodigoSucursal struct{ value string }

func NewCodigoSucursal(raw string) (CodigoSucursal, error) {
	v := strings.ToUpper(strings.TrimSpace(raw))
	if !codigoSucursalRe.MatchString(v) {
		return CodigoSucursal{}, ErrCodigoInvalido
	}
	return CodigoSucursal{v}, nil
}

func (c CodigoSucursal) String() string { return c.value }
