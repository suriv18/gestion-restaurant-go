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
