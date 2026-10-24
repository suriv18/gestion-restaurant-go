package query

import (
	"context"
	"fmt"
	"strings"

	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	mensajeSinPermisoEmpleados = "no tiene permiso para consultar empleados"
	maxTextoBusqueda           = 150
)

var escapeLike = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// textoBusqueda recorta, limita y escapa los comodines de LIKE de la búsqueda.
func textoBusqueda(raw string) string {
	t := strings.TrimSpace(raw)
	if r := []rune(t); len(r) > maxTextoBusqueda {
		t = string(r[:maxTextoBusqueda])
	}
	return escapeLike.Replace(t)
}

// ListarEmpleados pide limite+1 filas para saber si hay más páginas.
type ListarEmpleados struct{ empleados outbound.EmpleadoReader }

func NewListarEmpleados(e outbound.EmpleadoReader) *ListarEmpleados { return &ListarEmpleados{e} }

func (uc *ListarEmpleados) Execute(ctx context.Context, q qry.ListarEmpleados) (appresult.Result[dtoresult.PaginaEmpleados], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoEmpleados, policy.EmpleadoVer, policy.EmpleadoAdministrar); ae != nil {
		return appresult.Fail[dtoresult.PaginaEmpleados](ae), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaEmpleados](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.empleados.ListarEmpleados(ctx, outbound.FiltroEmpleados{
		Activo: q.Activo, Texto: textoBusqueda(q.Texto), DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaEmpleados]{}, fmt.Errorf("listar empleados: %w", err)
	}
	pagina := dtoresult.PaginaEmpleados{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Empleado{}
	}
	return appresult.Ok(pagina), nil
}

type ObtenerEmpleado struct{ empleados outbound.EmpleadoReader }

func NewObtenerEmpleado(e outbound.EmpleadoReader) *ObtenerEmpleado { return &ObtenerEmpleado{e} }

func (uc *ObtenerEmpleado) Execute(ctx context.Context, q qry.ObtenerEmpleado) (appresult.Result[dtoresult.Empleado], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoEmpleados, policy.EmpleadoVer, policy.EmpleadoAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Empleado](ae), nil
	}
	e, found, err := uc.empleados.ObtenerEmpleado(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Empleado]{}, fmt.Errorf("obtener empleado: %w", err)
	}
	if !found {
		return appresult.Fail[dtoresult.Empleado](mapper.NoEncontrado("empleado")), nil
	}
	return appresult.Ok(e), nil
}
