// Package query implementa las consultas del directorio de clientes y proveedores.
package query

import (
	"context"
	"fmt"
	"strings"

	qry "service-restaurant/internal/modules/catalogo/application/dto/query"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/mapper"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/policy"
	"service-restaurant/internal/shared/application/pagination"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	mensajeSinPermisoClientes    = "no tiene permiso para consultar clientes"
	mensajeSinPermisoProveedores = "no tiene permiso para consultar proveedores"
	maxTexto                     = 150
)

// tamanio normaliza el límite pedido (por defecto 25, máximo 100; nunca menos de 1).
func tamanio(limite int) int {
	return pagination.PageRequest{Size: limite}.Normalize().Size
}

var escapeLike = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// textoBusqueda recorta, limita y escapa los comodines de LIKE de la búsqueda.
func textoBusqueda(raw string) string {
	t := strings.TrimSpace(raw)
	if r := []rune(t); len(r) > maxTexto {
		t = string(r[:maxTexto])
	}
	return escapeLike.Replace(t)
}

// ListarClientes pide limite+1 filas para saber si hay más páginas.
type ListarClientes struct{ clientes outbound.ClienteReader }

func NewListarClientes(c outbound.ClienteReader) *ListarClientes { return &ListarClientes{c} }

func (uc *ListarClientes) Execute(ctx context.Context, q qry.ListarClientes) (appresult.Result[dtoresult.PaginaClientes], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoClientes, policy.ClienteVer, policy.ClienteAdministrar); ae != nil {
		return appresult.Fail[dtoresult.PaginaClientes](ae), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaClientes](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.clientes.ListarClientes(ctx, outbound.FiltroClientes{
		Activo: q.Activo, Texto: textoBusqueda(q.Texto), DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaClientes]{}, fmt.Errorf("listar clientes: %w", err)
	}
	pagina := dtoresult.PaginaClientes{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Cliente{}
	}
	return appresult.Ok(pagina), nil
}

type ObtenerCliente struct{ clientes outbound.ClienteReader }

func NewObtenerCliente(c outbound.ClienteReader) *ObtenerCliente { return &ObtenerCliente{c} }

func (uc *ObtenerCliente) Execute(ctx context.Context, q qry.ObtenerCliente) (appresult.Result[dtoresult.Cliente], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoClientes, policy.ClienteVer, policy.ClienteAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Cliente](ae), nil
	}
	c, found, err := uc.clientes.ObtenerCliente(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Cliente]{}, fmt.Errorf("obtener cliente: %w", err)
	}
	if !found {
		return appresult.Fail[dtoresult.Cliente](mapper.NoEncontrado("cliente")), nil
	}
	return appresult.Ok(c), nil
}

type ListarProveedores struct{ proveedores outbound.ProveedorReader }

func NewListarProveedores(p outbound.ProveedorReader) *ListarProveedores {
	return &ListarProveedores{p}
}

func (uc *ListarProveedores) Execute(ctx context.Context, q qry.ListarProveedores) (appresult.Result[dtoresult.PaginaProveedores], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoProveedores, policy.ProveedorVer, policy.ProveedorAdministrar); ae != nil {
		return appresult.Fail[dtoresult.PaginaProveedores](ae), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaProveedores](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.proveedores.ListarProveedores(ctx, outbound.FiltroProveedores{
		Activo: q.Activo, Texto: textoBusqueda(q.Texto), DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaProveedores]{}, fmt.Errorf("listar proveedores: %w", err)
	}
	pagina := dtoresult.PaginaProveedores{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Proveedor{}
	}
	return appresult.Ok(pagina), nil
}

type ObtenerProveedor struct{ proveedores outbound.ProveedorReader }

func NewObtenerProveedor(p outbound.ProveedorReader) *ObtenerProveedor {
	return &ObtenerProveedor{p}
}

func (uc *ObtenerProveedor) Execute(ctx context.Context, q qry.ObtenerProveedor) (appresult.Result[dtoresult.Proveedor], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoProveedores, policy.ProveedorVer, policy.ProveedorAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Proveedor](ae), nil
	}
	p, found, err := uc.proveedores.ObtenerProveedor(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Proveedor]{}, fmt.Errorf("obtener proveedor: %w", err)
	}
	if !found {
		return appresult.Fail[dtoresult.Proveedor](mapper.NoEncontrado("proveedor")), nil
	}
	return appresult.Ok(p), nil
}
