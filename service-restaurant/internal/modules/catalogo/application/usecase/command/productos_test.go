package command

import (
	"context"
	"testing"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

// storeProductos es un ProductoStore en memoria de un solo producto.
type storeProductos struct {
	prod       dtoresult.Producto
	err        error
	cambios    outbound.CambiosProducto
	escrituras int
}

func (s *storeProductos) Crear(context.Context, outbound.NuevoProducto) (dtoresult.Producto, error) {
	s.escrituras++
	return s.prod, s.err
}
func (s *storeProductos) Obtener(context.Context, int64) (dtoresult.Producto, bool, error) {
	return s.prod, true, nil
}
func (s *storeProductos) Actualizar(_ context.Context, _ int64, c outbound.CambiosProducto) (dtoresult.Producto, bool, error) {
	s.escrituras++
	s.cambios = c
	return s.prod, true, s.err
}
func (s *storeProductos) DarBaja(context.Context, int64, string) (dtoresult.Producto, bool, error) {
	s.escrituras++
	return s.prod, true, s.err
}
func (s *storeProductos) Restaurar(context.Context, int64) (dtoresult.Producto, bool, error) {
	s.escrituras++
	return s.prod, true, s.err
}

func adminCarta() appctx.Actor {
	return appctx.NewActor(7, "ana", 1, []appctx.Grant{{SucursalID: 1, Permiso: policy.CartaAdministrar}})
}

func id64(n int64) *int64 { return &n }

// Las invariantes cruzadas del PATCH se evalúan sobre el valor resultante (fila actual + cambios).
func TestEditarProductoInvariantesSobreValorResultante(t *testing.T) {
	ctx := context.Background()
	gravado := dtoresult.Producto{ID: 1, Tipo: "PREPARADO", AfectacionIGV: "10", ImpuestoID: id64(3), EsActivo: true}
	reventa := dtoresult.Producto{ID: 1, Tipo: "REVENTA", ArticuloID: id64(9), AfectacionIGV: "20", EsActivo: true}
	afect20 := "20"
	afect10 := "10"
	reventaTipo := "REVENTA"
	preparado := "PREPARADO"

	casos := []struct {
		nombre string
		actual dtoresult.Producto
		c      cmd.EditarProducto
		code   string // vacío = se acepta
	}{
		{"vaciar impuesto de un gravado", gravado, cmd.EditarProducto{ImpuestoID: cmd.Campo[int64]{Presente: true}}, "impuesto_requerido"},
		{"pasar a exonerado y vaciar impuesto", gravado, cmd.EditarProducto{Afectacion: &afect20, ImpuestoID: cmd.Campo[int64]{Presente: true}}, ""},
		{"gravar sin impuesto", reventa, cmd.EditarProducto{Afectacion: &afect10}, "impuesto_requerido"},
		{"pasar a REVENTA sin artículo", gravado, cmd.EditarProducto{Tipo: &reventaTipo}, "articulo_requerido"},
		{"pasar a REVENTA con artículo", gravado, cmd.EditarProducto{Tipo: &reventaTipo, ArticuloID: cmd.Campo[int64]{Presente: true, Valor: id64(5)}}, ""},
		{"dejar de ser REVENTA conservando artículo", reventa, cmd.EditarProducto{Tipo: &preparado}, "articulo_no_permitido"},
		{"dejar de ser REVENTA vaciando artículo", reventa, cmd.EditarProducto{Tipo: &preparado, ArticuloID: cmd.Campo[int64]{Presente: true}}, ""},
		{"vaciar artículo de un REVENTA", reventa, cmd.EditarProducto{ArticuloID: cmd.Campo[int64]{Presente: true}}, "articulo_requerido"},
		{"sin campos", gravado, cmd.EditarProducto{}, "sin_cambios"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			st := &storeProductos{prod: tc.actual}
			tc.c.Actor, tc.c.ID = adminCarta(), 1
			res, err := NewEditarProducto(st).Execute(ctx, tc.c)
			if err != nil {
				t.Fatalf("error técnico: %v", err)
			}
			if tc.code == "" {
				if res.Err() != nil || st.escrituras != 1 {
					t.Fatalf("debía aceptarse: err=%v escrituras=%d", res.Err(), st.escrituras)
				}
				return
			}
			if e := res.Err(); e == nil || e.Kind != apperror.KindValidation || e.Code != tc.code || st.escrituras != 0 {
				t.Fatalf("rechazo = %v (escrituras %d), quería %s", e, st.escrituras, tc.code)
			}
		})
	}
}

func TestProductoSoloEditaActivosYNormalizaDecimales(t *testing.T) {
	ctx := context.Background()
	baja := &storeProductos{prod: dtoresult.Producto{ID: 1, Tipo: "SERVICIO", AfectacionIGV: "20"}}
	nombre := "X"
	res, _ := NewEditarProducto(baja).Execute(ctx, cmd.EditarProducto{Actor: adminCarta(), ID: 1, Nombre: &nombre})
	if e := res.Err(); e == nil || e.Kind != apperror.KindConflict || e.Code != CodigoProductoInactivo {
		t.Fatalf("producto de baja: %v", e)
	}

	st := &storeProductos{prod: dtoresult.Producto{ID: 1, Tipo: "SERVICIO", AfectacionIGV: "20", EsActivo: true}}
	precio, factor := "7.5", "3"
	if res, _ = NewEditarProducto(st).Execute(ctx, cmd.EditarProducto{Actor: adminCarta(), ID: 1, Precio: &precio, Factor: &factor}); res.Err() != nil {
		t.Fatalf("edición válida: %v", res.Err())
	}
	if *st.cambios.Precio != "7.50" || *st.cambios.Factor != "3.000000" {
		t.Fatalf("decimales sin normalizar: %s %s", *st.cambios.Precio, *st.cambios.Factor)
	}
}

// El mensaje técnico del trigger se sustituye por uno curado; al restaurar es un conflicto de estado.
func TestProductoTraduceReferenciasInactivas(t *testing.T) {
	ctx := context.Background()
	trigger := &outbound.ErrReglaDatos{Mensaje: "Referencia categoria_id=5: maestro categorias inexistente o dado de baja"}
	st := &storeProductos{prod: dtoresult.Producto{ID: 1, EsActivo: false}, err: trigger}

	res, err := NewCambiarEstadoProducto(st).Execute(ctx, cmd.CambiarEstadoProducto{Actor: adminCarta(), ID: 1, Activo: true})
	if err != nil {
		t.Fatalf("error técnico: %v", err)
	}
	if e := res.Err(); e == nil || e.Kind != apperror.KindConflict || e.Code != CodigoReferenciaInactiva {
		t.Fatalf("restaurar: %v", e)
	}

	st = &storeProductos{err: trigger}
	res, _ = NewCrearProducto(st).Execute(ctx, cmd.CrearProducto{Actor: adminCarta(), CategoriaID: 5, Codigo: "a", Nombre: "A",
		Tipo: "SERVICIO", Precio: "1", Afectacion: "20"})
	if e := res.Err(); e == nil || e.Kind != apperror.KindValidation || e.Code != "referencia_invalida" {
		t.Fatalf("crear: %v", e)
	}
}
