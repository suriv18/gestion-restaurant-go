package command

import (
	"context"
	"errors"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/mapper"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/policy"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	mensajeSinPermisoCarta     = "no tiene permiso para administrar la carta"
	mensajeCategoriaDuplicada  = "ya existe una categoría con ese nombre (aunque esté dada de baja)"
	mensajeCategoriaConProduct = "la categoría tiene productos activos: dé de baja o reasigne esos productos antes"
)

func deCategoria(err error, contexto string) (appresult.Result[dtoresult.Categoria], error) {
	if errors.Is(err, outbound.ErrCategoriaDuplicada) {
		return rechazar[dtoresult.Categoria](apperror.New(apperror.KindConflict, CodigoCategoriaDuplicada, mensajeCategoriaDuplicada))
	}
	return deAlmacen[dtoresult.Categoria](contexto, "", err)
}

// CrearCategoria da de alta una categoría de la carta.
type CrearCategoria struct{ categorias outbound.CategoriaStore }

func NewCrearCategoria(c outbound.CategoriaStore) *CrearCategoria { return &CrearCategoria{c} }

func (uc *CrearCategoria) Execute(ctx context.Context, c cmd.CrearCategoria) (appresult.Result[dtoresult.Categoria], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoCarta, policy.CartaAdministrar); ae != nil {
		return rechazar[dtoresult.Categoria](ae)
	}
	nombre, ae := validarNombreHasta(c.Nombre, "nombre", maxNombreCategoria)
	if ae != nil {
		return rechazar[dtoresult.Categoria](ae)
	}
	nueva := outbound.NuevaCategoria{Nombre: nombre}
	if nueva.Descripcion, ae = opcional(c.Descripcion, "descripcion", fmtDescripcion); ae != nil {
		return rechazar[dtoresult.Categoria](ae)
	}
	if nueva.Orden, ae = validarOrden(c.Orden); ae != nil {
		return rechazar[dtoresult.Categoria](ae)
	}
	cat, err := uc.categorias.Crear(ctx, nueva)
	if err != nil {
		return deCategoria(err, "crear categoría")
	}
	return appresult.Ok(cat), nil
}

// EditarCategoria modifica solo los campos enviados de una categoría activa.
type EditarCategoria struct{ categorias outbound.CategoriaStore }

func NewEditarCategoria(c outbound.CategoriaStore) *EditarCategoria { return &EditarCategoria{c} }

func (uc *EditarCategoria) Execute(ctx context.Context, c cmd.EditarCategoria) (appresult.Result[dtoresult.Categoria], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoCarta, policy.CartaAdministrar); ae != nil {
		return rechazar[dtoresult.Categoria](ae)
	}
	actual, found, err := uc.categorias.Obtener(ctx, c.ID)
	if err != nil {
		return tecnico[dtoresult.Categoria]("buscar categoría", err)
	}
	if !found {
		return rechazar[dtoresult.Categoria](mapper.NoEncontrada("categoría"))
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Categoria](apperror.New(apperror.KindConflict, CodigoCategoriaInactiva, "la categoría está dada de baja: restáurela para editarla"))
	}
	if c.Nombre == nil && !c.Descripcion.Presente && c.Orden == nil {
		return rechazar[dtoresult.Categoria](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosCategoria
	var ae *apperror.AppError
	if c.Nombre != nil {
		n, ae := validarNombreHasta(*c.Nombre, "nombre", maxNombreCategoria)
		if ae != nil {
			return rechazar[dtoresult.Categoria](ae)
		}
		cambios.Nombre = &n
	}
	if cambios.Descripcion, ae = campoOpcional(c.Descripcion, "descripcion", fmtDescripcion); ae != nil {
		return rechazar[dtoresult.Categoria](ae)
	}
	if c.Orden != nil {
		o, ae := validarOrden(c.Orden)
		if ae != nil {
			return rechazar[dtoresult.Categoria](ae)
		}
		cambios.Orden = &o
	}
	cat, found, err := uc.categorias.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deCategoria(err, "editar categoría")
	}
	if !found {
		return rechazar[dtoresult.Categoria](mapper.NoEncontrada("categoría"))
	}
	return appresult.Ok(cat), nil
}

// CambiarEstadoCategoria da de baja (sin motivo) o restaura una categoría. La baja se rechaza si
// aún tiene productos activos: la comprobación ocurre con la fila de la categoría bloqueada.
type CambiarEstadoCategoria struct{ categorias outbound.CategoriaStore }

func NewCambiarEstadoCategoria(c outbound.CategoriaStore) *CambiarEstadoCategoria {
	return &CambiarEstadoCategoria{c}
}

func (uc *CambiarEstadoCategoria) Execute(ctx context.Context, c cmd.CambiarEstadoCategoria) (appresult.Result[dtoresult.Categoria], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoCarta, policy.CartaAdministrar); ae != nil {
		return rechazar[dtoresult.Categoria](ae)
	}
	actual, found, err := uc.categorias.Obtener(ctx, c.ID)
	if err != nil {
		return tecnico[dtoresult.Categoria]("buscar categoría", err)
	}
	if !found {
		return rechazar[dtoresult.Categoria](mapper.NoEncontrada("categoría"))
	}
	var cat dtoresult.Categoria
	switch {
	case c.Activo && actual.EsActivo:
		return rechazar[dtoresult.Categoria](apperror.New(apperror.KindConflict, CodigoYaActiva, "la categoría ya está activa"))
	case !c.Activo && !actual.EsActivo:
		return rechazar[dtoresult.Categoria](apperror.New(apperror.KindConflict, CodigoYaDadaDeBaja, "la categoría ya está dada de baja"))
	case c.Activo:
		cat, found, err = uc.categorias.Restaurar(ctx, c.ID)
	default:
		enUso, uerr := uc.categorias.TieneProductosActivos(ctx, c.ID)
		if uerr != nil {
			return tecnico[dtoresult.Categoria]("comprobar productos de la categoría", uerr)
		}
		if enUso {
			return rechazar[dtoresult.Categoria](apperror.New(apperror.KindConflict, CodigoCategoriaConProductosActivos, mensajeCategoriaConProduct))
		}
		cat, found, err = uc.categorias.DarBaja(ctx, c.ID)
	}
	if err != nil {
		return deCategoria(err, "cambiar estado de la categoría")
	}
	if !found {
		return rechazar[dtoresult.Categoria](mapper.NoEncontrada("categoría"))
	}
	return appresult.Ok(cat), nil
}
