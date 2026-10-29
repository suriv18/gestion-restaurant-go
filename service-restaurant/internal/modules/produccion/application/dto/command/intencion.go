// Package command agrupa los DTO de comandos del módulo produccion.
package command

// Intencion identifica una operación para la idempotencia (ADR-04). La rellena la capa web con
// la clave del encabezado, la operación y la huella del contenido; los casos de uso solo leen Clave.
type Intencion struct {
	Clave     string
	Operacion string // p. ej. "POST /pedidos/7/comandas" (con el id concreto)
	Huella    [32]byte
}
