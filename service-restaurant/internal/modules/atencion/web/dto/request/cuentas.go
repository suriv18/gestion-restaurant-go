package request

// CuentaDivision es el cuerpo de dividir-cuenta (esquema DividirCuenta del contrato). Los ids viajan
// como cadenas.
type CuentaDivision struct {
	DetalleIDs   []string `json:"detalle_ids"`
	NombreCuenta *string  `json:"nombre_cuenta,omitempty"`
}
