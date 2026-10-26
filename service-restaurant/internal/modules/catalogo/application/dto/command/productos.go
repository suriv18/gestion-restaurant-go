package command

import appctx "service-restaurant/internal/shared/application/context"

// CrearProducto da de alta un producto de la carta. Los decimales llegan como cadena (sin float);
// Factor vacío o nil equivale a "1". CategoriaID 0 significa "no enviada".
type CrearProducto struct {
	Intencion
	Actor       appctx.Actor
	CategoriaID int64
	Codigo      string
	Nombre      string
	Descripcion *string
	Tipo        string
	ArticuloID  *int64
	Factor      *string
	Precio      string
	Afectacion  string
	ImpuestoID  *int64
	ImagenURL   *string
}

// EditarProducto aplica solo los campos presentes. Descripcion, ArticuloID, ImpuestoID e ImagenURL
// admiten null para vaciarse. Las reglas cruzadas se validan sobre el valor resultante.
type EditarProducto struct {
	Actor       appctx.Actor
	ID          int64
	CategoriaID *int64
	Codigo      *string
	Nombre      *string
	Descripcion Campo[string]
	Tipo        *string
	ArticuloID  Campo[int64]
	Factor      *string
	Precio      *string
	Afectacion  *string
	ImpuestoID  Campo[int64]
	ImagenURL   Campo[string]
}

// CambiarEstadoProducto da de baja (Activo=false, con motivo) o restaura un producto.
type CambiarEstadoProducto struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
	Motivo string
}
