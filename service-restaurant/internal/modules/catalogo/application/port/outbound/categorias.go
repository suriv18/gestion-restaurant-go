package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
)

// FiltroCategorias acota el listado por estado y nombre (texto ya sin comodines de LIKE).
type FiltroCategorias struct {
	Activo      *bool
	Texto       string
	DespuesDeID int64
	Limite      int
}

// CategoriaReader lee categorías sin bloquear filas.
type CategoriaReader interface {
	ListarCategorias(ctx context.Context, f FiltroCategorias) ([]dtoresult.Categoria, error)
	ObtenerCategoria(ctx context.Context, id int64) (dtoresult.Categoria, bool, error)
}

type NuevaCategoria struct {
	Nombre      string
	Descripcion *string
	Orden       int
}

// CambiosCategoria: solo los campos presentes se modifican.
type CambiosCategoria struct {
	Nombre      *string
	Descripcion CampoOpcional
	Orden       *int
}

// CategoriaStore escribe categorías dentro de la transacción del comando.
type CategoriaStore interface {
	Crear(ctx context.Context, n NuevaCategoria) (dtoresult.Categoria, error)
	// Obtener lee la categoría bloqueando su fila (FOR UPDATE); found=false si no existe.
	Obtener(ctx context.Context, id int64) (dtoresult.Categoria, bool, error)
	Actualizar(ctx context.Context, id int64, c CambiosCategoria) (dtoresult.Categoria, bool, error)
	DarBaja(ctx context.Context, id int64) (dtoresult.Categoria, bool, error)
	Restaurar(ctx context.Context, id int64) (dtoresult.Categoria, bool, error)
	// TieneProductosActivos indica si algún producto activo usa la categoría. Se invoca con la
	// fila de la categoría ya bloqueada, para que un alta concurrente de producto no la deje huérfana.
	TieneProductosActivos(ctx context.Context, id int64) (bool, error)
}
