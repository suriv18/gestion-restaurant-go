package command

// Intencion identifica una operación para la idempotencia (ADR-04). La rellena la capa web con
// la clave del encabezado, la operación y la huella del contenido; los casos de uso la ignoran.
type Intencion struct {
	Clave     string
	Operacion string // p. ej. "POST /clientes/7/baja" (con el id concreto)
	Huella    [32]byte
}

// Campo es un valor opcional de un PATCH: Presente distingue "no enviado" de "enviado como
// null" (vaciar). Valor es nil cuando se envió null.
type Campo[T any] struct {
	Presente bool
	Valor    *T
}
