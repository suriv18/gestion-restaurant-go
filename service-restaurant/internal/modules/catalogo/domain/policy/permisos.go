package policy

// Permisos del directorio de clientes y proveedores (migración 00009). Se declaran aquí porque
// catalogo no importa identidad; el catálogo de permisos vive en la base.
const (
	ClienteVer           = "CLIENTE_VER"
	ClienteAdministrar   = "CLIENTE_ADMINISTRAR"
	ProveedorVer         = "PROVEEDOR_VER"
	ProveedorAdministrar = "PROVEEDOR_ADMINISTRAR"
)

// Permisos de la carta (migración 00010). CartaAdministrar implica ver.
const (
	CartaVer         = "CARTA_VER"
	CartaAdministrar = "CARTA_ADMINISTRAR"
)

// Permisos de impuestos y tasas (migración 00010). CartaAdministrar también habilita la lectura
// de impuestos y tasas, que necesita el formulario de productos.
const (
	ImpuestoVer         = "IMPUESTO_VER"
	ImpuestoAdministrar = "IMPUESTO_ADMINISTRAR"
)

// Permisos de estaciones de preparación (migración 00010). Son por sede; EstacionAdministrar implica ver.
const (
	EstacionVer         = "ESTACION_VER"
	EstacionAdministrar = "ESTACION_ADMINISTRAR"
)
