package vo

import "errors"

var (
	ErrTipoProductoInvalido = errors.New("tipo de producto inválido")
	ErrAfectacionInvalida   = errors.New("afectación del IGV inválida")
)

// Tipos de producto de la carta. Solo REVENTA lleva artículo de inventario.
const (
	TipoPreparado = "PREPARADO"
	TipoReventa   = "REVENTA"
	TipoAdicional = "ADICIONAL"
	TipoServicio  = "SERVICIO"
)

// Afectación del IGV: 10 gravado, 20 exonerado, 30 inafecto. Solo la gravada exige impuesto.
const (
	AfectacionGravado   = "10"
	AfectacionExonerado = "20"
	AfectacionInafecto  = "30"
)

// NewTipoProducto exige uno de los cuatro tipos, en mayúsculas exactas.
func NewTipoProducto(raw string) (string, error) {
	switch raw {
	case TipoPreparado, TipoReventa, TipoAdicional, TipoServicio:
		return raw, nil
	}
	return "", ErrTipoProductoInvalido
}

// NewAfectacionIGV exige "10", "20" o "30".
func NewAfectacionIGV(raw string) (string, error) {
	switch raw {
	case AfectacionGravado, AfectacionExonerado, AfectacionInafecto:
		return raw, nil
	}
	return "", ErrAfectacionInvalida
}
