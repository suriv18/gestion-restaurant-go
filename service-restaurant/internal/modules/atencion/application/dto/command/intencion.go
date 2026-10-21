// Package command agrupa los DTO de comandos del módulo atencion.
package command

// Intencion identifica una operación para la idempotencia (ADR-04). La rellena la capa web con
// la clave del encabezado, la operación y la huella del contenido; los casos de uso la ignoran.
type Intencion struct {
	Clave     string
	Operacion string // p. ej. "POST /zonas/7/baja" (con el id concreto)
	Huella    [32]byte
}
