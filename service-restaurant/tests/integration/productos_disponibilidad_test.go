package integration

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

type dispResp struct {
	ID             string  `json:"id"`
	ProductoID     string  `json:"producto_id"`
	SucursalID     string  `json:"sucursal_id"`
	EstacionID     *string `json:"estacion_id"`
	PrecioVenta    *string `json:"precio_venta"`
	Disponible     bool    `json:"disponible"`
	EsActivo       bool    `json:"es_activo"`
	CreadoEn       string  `json:"creado_en"`
	CreadoPor      string  `json:"creado_por"`
	ActualizadoEn  string  `json:"actualizado_en"`
	ActualizadoPor string  `json:"actualizado_por"`
}

type enSucursalResp struct {
	Disponible     bool    `json:"disponible"`
	PrecioSucursal *string `json:"precio_sucursal"`
	PrecioEfectivo string  `json:"precio_efectivo"`
	EstacionID     *string `json:"estacion_id"`
}

type productoSedeResp struct {
	productoResp
	EnSucursal *enSucursalResp `json:"en_sucursal"`
}

type productosSedePagina struct {
	Items      []productoSedeResp `json:"items"`
	NextCursor *string            `json:"nextCursor"`
}

func (p productosSedePagina) ids() []string {
	out := make([]string, 0, len(p.Items))
	for _, x := range p.Items {
		out = append(out, x.ID)
	}
	return out
}

// adminCartaSede devuelve el token de un usuario propio de la prueba con CARTA_ADMINISTRAR en las sedes dadas.
func adminCartaSede(t *testing.T, sedes ...string) string {
	t.Helper()
	rolConPermisos(t, "T_DISP_ADMIN", "CARTA_ADMINISTRAR")
	user := "disp.adm." + strings.ToLower(sufijo())
	asegurarUsuario(t, user, "DispAdmin2026Clave", "T_DISP_ADMIN", sedes...)
	return token(t, user, "DispAdmin2026Clave")
}

// usuarioRolSede crea un usuario propio con un rol de la migración (COCINA, MESERO...) en las sedes dadas.
func usuarioRolSede(t *testing.T, rol string, sedes ...string) string {
	t.Helper()
	user := "disp." + strings.ToLower(rol) + "." + strings.ToLower(sufijo())
	asegurarUsuario(t, user, "DispRol2026Clave", rol, sedes...)
	return token(t, user, "DispRol2026Clave")
}

func putDisp(t *testing.T, tk, producto, cuerpo string) resp {
	t.Helper()
	return callH(t, http.MethodPut, "/api/v1/productos/"+producto+"/disponibilidad", cuerpo, tk, nil)
}

func cuerpoDisp(sede string, disponible bool, extras ...string) string {
	b := fmt.Sprintf(`{"sucursal_id":"%s","disponible":%t`, sede, disponible)
	for _, e := range extras {
		b += "," + e
	}
	return b + "}"
}

func fijarDisp(t *testing.T, tk, producto, cuerpo string) dispResp {
	t.Helper()
	r := putDisp(t, tk, producto, cuerpo)
	esperar(t, r, http.StatusOK, "")
	var d dispResp
	r.json(t, &d)
	return d
}

// productoNuevo crea un producto propio (categoría propia) y devuelve el producto.
func productoNuevo(t *testing.T) productoResp {
	t.Helper()
	gestor := gestorCarta(t)
	return crearProducto(t, gestor, cuerpoProd(categoriaPropia(t, gestor), codigoProd()))
}

// enVista cuenta las filas de v_carta_disponible del producto en la sede.
func enVista(t *testing.T, sede, producto string) int {
	t.Helper()
	return scalar(t, `SELECT count(*) FROM sch_restaurante.v_carta_disponible WHERE sucursal_id=`+sede+` AND producto_id=`+producto)
}

func precioVista(t *testing.T, sede, producto string) string {
	t.Helper()
	return scalarStr(t, `SELECT precio_venta::text FROM sch_restaurante.v_carta_disponible WHERE sucursal_id=`+sede+` AND producto_id=`+producto)
}

func cuerpoProducto(t *testing.T, tk, id, query string) string {
	t.Helper()
	r := call(t, "GET", "/api/v1/productos/"+id+query, "", tk)
	esperar(t, r, http.StatusOK, "")
	return r.Body.String()
}

func TestDisponibilidadHabilitarPausarYVista(t *testing.T) {
	sede, sedeID := sedeEstaciones(t)
	tk := adminCartaSede(t, sede)
	p := productoNuevo(t)
	antes := cuerpoProducto(t, tk, p.ID, "")

	if enVista(t, sedeID, p.ID) != 0 {
		t.Fatal("sin fila en productos_sucursal el producto no se ofrece")
	}

	// Habilitar crea la fila; sin precio ni estación usa el precio general.
	d := fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, true))
	if d.ID == "" || d.ProductoID != p.ID || d.SucursalID != sedeID || !d.Disponible || !d.EsActivo || d.EstacionID != nil ||
		d.PrecioVenta != nil || d.CreadoPor == "" || d.ActualizadoPor != d.CreadoPor || d.CreadoEn == "" || d.ActualizadoEn == "" {
		t.Fatalf("fila creada: %+v", d)
	}
	if enVista(t, sedeID, p.ID) != 1 || precioVista(t, sedeID, p.ID) != "12.50" {
		t.Fatalf("la vista debe ofrecer el producto al precio general: %d %s", enVista(t, sedeID, p.ID), precioVista(t, sedeID, p.ID))
	}

	// Pausar actualiza la misma fila y lo retira de la vista; rehabilitar lo devuelve (idempotente).
	pausada := fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, false))
	if pausada.ID != d.ID || pausada.Disponible || !pausada.EsActivo || pausada.CreadoEn != d.CreadoEn {
		t.Fatalf("pausa: %+v (creada: %+v)", pausada, d)
	}
	if enVista(t, sedeID, p.ID) != 0 {
		t.Fatal("pausado: no debe estar en v_carta_disponible")
	}
	for i := 0; i < 2; i++ {
		r := fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, true))
		if r.ID != d.ID || !r.Disponible {
			t.Fatalf("rehabilitar #%d: %+v", i, r)
		}
	}
	if enVista(t, sedeID, p.ID) != 1 {
		t.Fatal("rehabilitado: debe volver a la vista")
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.productos_sucursal WHERE producto_id=`+p.ID+` AND sucursal_id=`+sedeID); n != 1 {
		t.Fatalf("el upsert no debe duplicar filas: %d", n)
	}

	// Pausar no toca el maestro: el producto es idéntico byte a byte.
	if despues := cuerpoProducto(t, tk, p.ID, ""); despues != antes {
		t.Fatalf("el maestro cambió:\nantes:   %s\ndespués: %s", antes, despues)
	}

	// Una fila inactiva (caso borde, sin API que la desactive) se reactiva con el upsert.
	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.productos_sucursal SET es_activo=false WHERE id=%s`, d.ID))
	if enVista(t, sedeID, p.ID) != 0 {
		t.Fatal("fila inactiva: no debe estar en la vista")
	}
	r := fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, true))
	if r.ID != d.ID || !r.EsActivo || enVista(t, sedeID, p.ID) != 1 {
		t.Fatalf("reactivación: %+v", r)
	}
}

func TestDisponibilidadPrecioPorSede(t *testing.T) {
	sede, sedeID := sedeEstaciones(t)
	tk := adminCartaSede(t, sede)
	p := productoNuevo(t)

	d := fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, true, `"precio_venta":"18"`))
	if d.PrecioVenta == nil || *d.PrecioVenta != "18.00" || precioVista(t, sedeID, p.ID) != "18.00" {
		t.Fatalf("precio de la sede: %+v vista=%s", d, precioVista(t, sedeID, p.ID))
	}
	// Ausente = sin cambio: pausar y rehabilitar conserva el precio.
	d = fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, false))
	if d.PrecioVenta == nil || *d.PrecioVenta != "18.00" {
		t.Fatalf("ausente debe conservar el precio: %+v", d)
	}
	// Precio 0 es válido (cortesía de la sede).
	d = fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, true, `"precio_venta":"0"`))
	if d.PrecioVenta == nil || *d.PrecioVenta != "0.00" {
		t.Fatalf("precio cero: %+v", d)
	}
	// null = volver al precio general.
	d = fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, true, `"precio_venta":null`))
	if d.PrecioVenta != nil || precioVista(t, sedeID, p.ID) != "12.50" {
		t.Fatalf("null debe volver al precio general: %+v vista=%s", d, precioVista(t, sedeID, p.ID))
	}

	// Validación: sin signo, hasta 2 decimales, máximo 99999999999.99, sin float.
	for _, malo := range []string{`"-1"`, `"1.234"`, `"abc"`, `""`, `"1e3"`, `"100000000000"`, `"99999999999.999"`} {
		esperar(t, putDisp(t, tk, p.ID, cuerpoDisp(sedeID, true, `"precio_venta":`+malo)), http.StatusUnprocessableEntity, "precio_invalido")
	}
	esperar(t, putDisp(t, tk, p.ID, cuerpoDisp(sedeID, true, `"precio_venta":12.5`)), http.StatusBadRequest, "")
	d = fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, true, `"precio_venta":"99999999999.99"`))
	if *d.PrecioVenta != "99999999999.99" {
		t.Fatalf("máximo: %+v", d)
	}
}

func TestDisponibilidadEstacion(t *testing.T) {
	sede, sedeID := sedeEstaciones(t)
	otra, otraID := sedeEstaciones(t)
	tk := adminCartaSede(t, sede)
	gestor := gestorEstaciones(t, sede, otra)
	propia := crearEstacion(t, gestor, sedeID, "COC", "Cocina caliente")
	ajena := crearEstacion(t, gestor, otraID, "COC", "Cocina de otra sede")
	caida := crearEstacion(t, gestor, sedeID, "BAR", "Barra")
	esperar(t, post(t, "/api/v1/estaciones/"+caida.ID+"/baja", ``, gestor), http.StatusOK, "")
	p := productoNuevo(t)

	d := fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, true, `"estacion_id":"`+propia.ID+`"`))
	if d.EstacionID == nil || *d.EstacionID != propia.ID {
		t.Fatalf("estación de la misma sede: %+v", d)
	}
	if got := scalarStr(t, `SELECT estacion_id FROM sch_restaurante.v_carta_disponible WHERE sucursal_id=`+sedeID+` AND producto_id=`+p.ID); got != propia.ID {
		t.Fatalf("la vista debe traer la estación: %s", got)
	}
	// Ausente conserva la estación; null la quita.
	if d = fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, false)); d.EstacionID == nil || *d.EstacionID != propia.ID {
		t.Fatalf("ausente debe conservar la estación: %+v", d)
	}
	if d = fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, true, `"estacion_id":null`)); d.EstacionID != nil {
		t.Fatalf("null debe quitar la estación: %+v", d)
	}

	// De otra sede: 422 referencia_invalida (igual que inexistente, para no revelar otras sedes), también al crear la fila (producto sin fila).
	esperar(t, putDisp(t, tk, p.ID, cuerpoDisp(sedeID, true, `"estacion_id":"`+ajena.ID+`"`)), http.StatusUnprocessableEntity, "referencia_invalida")
	p2 := productoNuevo(t)
	esperar(t, putDisp(t, tk, p2.ID, cuerpoDisp(sedeID, true, `"estacion_id":"`+ajena.ID+`"`)), http.StatusUnprocessableEntity, "referencia_invalida")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.productos_sucursal WHERE producto_id=`+p2.ID) != 0 {
		t.Fatal("un rechazo no debe dejar fila")
	}
	// Inactiva o inexistente: 422 referencia_invalida.
	esperar(t, putDisp(t, tk, p.ID, cuerpoDisp(sedeID, true, `"estacion_id":"`+caida.ID+`"`)), http.StatusUnprocessableEntity, "referencia_invalida")
	esperar(t, putDisp(t, tk, p.ID, cuerpoDisp(sedeID, true, `"estacion_id":"999999999"`)), http.StatusUnprocessableEntity, "referencia_invalida")
	// Mal formada: 400.
	esperar(t, putDisp(t, tk, p.ID, cuerpoDisp(sedeID, true, `"estacion_id":"abc"`)), http.StatusBadRequest, "estacion_id_invalido")

	// Con la estación asignada, la estación no se puede dar de baja (estacion_en_uso).
	fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, true, `"estacion_id":"`+propia.ID+`"`))
	esperar(t, post(t, "/api/v1/estaciones/"+propia.ID+"/baja", ``, gestor), http.StatusConflict, "estacion_en_uso")
}

func TestDisponibilidadProductoYCuerpo(t *testing.T) {
	sede, sedeID := sedeEstaciones(t)
	tk := adminCartaSede(t, sede)
	gestor := gestorCarta(t)
	p := productoNuevo(t)

	// Producto inexistente: 404; dado de baja: 409 producto_inactivo y sin fila nueva.
	esperar(t, putDisp(t, tk, "999999999", cuerpoDisp(sedeID, true)), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/productos/"+p.ID+"/baja", `{"motivo":"Fuera de carta"}`, gestor), http.StatusOK, "")
	esperar(t, putDisp(t, tk, p.ID, cuerpoDisp(sedeID, true)), http.StatusConflict, "producto_inactivo")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.productos_sucursal WHERE producto_id=`+p.ID) != 0 {
		t.Fatal("un producto dado de baja no se habilita")
	}
	// Restaurado vuelve a poder habilitarse.
	esperar(t, post(t, "/api/v1/productos/"+p.ID+"/restauracion", ``, gestor), http.StatusOK, "")
	fijarDisp(t, tk, p.ID, cuerpoDisp(sedeID, true))

	// Dar de baja un producto habilitado lo saca de la vista (el maestro manda).
	esperar(t, post(t, "/api/v1/productos/"+p.ID+"/baja", `{"motivo":"Fuera de carta"}`, gestor), http.StatusOK, "")
	if enVista(t, sedeID, p.ID) != 0 {
		t.Fatal("producto dado de baja: no debe estar en la vista")
	}
	esperar(t, post(t, "/api/v1/productos/"+p.ID+"/restauracion", ``, gestor), http.StatusOK, "")

	// Cuerpo.
	esperar(t, putDisp(t, tk, p.ID, `{"disponible":true}`), http.StatusUnprocessableEntity, "sucursal_requerida")
	esperar(t, putDisp(t, tk, p.ID, `{"sucursal_id":"`+sedeID+`"}`), http.StatusUnprocessableEntity, "disponible_requerido")
	esperar(t, putDisp(t, tk, p.ID, `{"sucursal_id":"abc","disponible":true}`), http.StatusBadRequest, "sucursal_id_invalido")
	esperar(t, putDisp(t, tk, p.ID, `{"sucursal_id":"`+sedeID+`","disponible":"si"}`), http.StatusBadRequest, "")
	esperar(t, putDisp(t, tk, p.ID, `{"sucursal_id":"`+sedeID+`","disponible":true,"otro":1}`), http.StatusBadRequest, "")
	esperar(t, putDisp(t, tk, "abc", cuerpoDisp(sedeID, true)), http.StatusBadRequest, "id_invalido")
	esperar(t, callH(t, http.MethodPut, "/api/v1/productos/"+p.ID+"/disponibilidad", cuerpoDisp(sedeID, true), "", nil), http.StatusUnauthorized, "no_autenticado")
}

func TestDisponibilidadPermisosPorSede(t *testing.T) {
	sede, sedeID := sedeEstaciones(t)
	ajena, ajenaID := sedeEstaciones(t)
	p := productoNuevo(t)
	admin := adminCartaSede(t, sede)
	gestor := gestorEstaciones(t, sede)
	est := crearEstacion(t, gestor, sedeID, "PER", "Permisos")

	// COCINA (CARTA_DISPONIBILIDAD en su sede): pausa y habilita; precio o estación son 403.
	cocina := usuarioRolSede(t, "COCINA", sede)
	if d := fijarDisp(t, cocina, p.ID, cuerpoDisp(sedeID, true)); !d.Disponible {
		t.Fatalf("cocina habilita: %+v", d)
	}
	if d := fijarDisp(t, cocina, p.ID, cuerpoDisp(sedeID, false)); d.Disponible {
		t.Fatalf("cocina pausa: %+v", d)
	}
	esperar(t, putDisp(t, cocina, p.ID, cuerpoDisp(sedeID, true, `"precio_venta":"9.00"`)), http.StatusForbidden, "sede_no_autorizada")
	esperar(t, putDisp(t, cocina, p.ID, cuerpoDisp(sedeID, true, `"precio_venta":null`)), http.StatusForbidden, "sede_no_autorizada")
	esperar(t, putDisp(t, cocina, p.ID, cuerpoDisp(sedeID, true, `"estacion_id":"`+est.ID+`"`)), http.StatusForbidden, "sede_no_autorizada")
	esperar(t, putDisp(t, cocina, p.ID, cuerpoDisp(sedeID, true, `"estacion_id":null`)), http.StatusForbidden, "sede_no_autorizada")
	if d := fijarDisp(t, admin, p.ID, cuerpoDisp(sedeID, true)); d.PrecioVenta != nil || d.EstacionID != nil {
		t.Fatalf("un 403 no debe haber escrito precio ni estación: %+v", d)
	}

	// MESERO solo tiene CARTA_VER: no cambia la disponibilidad.
	mesero := usuarioRolSede(t, "MESERO", sede)
	esperar(t, putDisp(t, mesero, p.ID, cuerpoDisp(sedeID, false)), http.StatusForbidden, "sin_permiso")

	// Sede ajena: el actor no la ve → 404 de sucursal, sin revelar nada, aunque administre otra.
	esperar(t, putDisp(t, admin, p.ID, cuerpoDisp(ajenaID, true)), http.StatusNotFound, "no_encontrado")
	esperar(t, putDisp(t, cocina, p.ID, cuerpoDisp(ajenaID, true)), http.StatusNotFound, "no_encontrado")
	esperar(t, putDisp(t, admin, p.ID, cuerpoDisp("999999999", true)), http.StatusNotFound, "no_encontrado")
	if enVista(t, ajenaID, p.ID) != 0 || scalar(t, `SELECT count(*) FROM sch_restaurante.productos_sucursal WHERE sucursal_id=`+ajenaID) != 0 {
		t.Fatal("una sede ajena no debe recibir filas")
	}

	// Quien administra en varias sedes escribe en todas.
	admin2 := adminCartaSede(t, sede, ajena)
	fijarDisp(t, admin2, p.ID, cuerpoDisp(ajenaID, true, `"precio_venta":"7.5"`))

	// Una sede dada de baja deja de figurar en los permisos del actor: 404.
	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.sucursales SET es_activo=false, motivo_eliminacion='Cierre de prueba' WHERE id=%s`, ajenaID))
	esperar(t, putDisp(t, admin2, p.ID, cuerpoDisp(ajenaID, true)), http.StatusNotFound, "no_encontrado")
	esperar(t, putDisp(t, admin2, p.ID, cuerpoDisp(ajenaID, true)), http.StatusNotFound, "no_encontrado")
}

func TestProductosListadoYDetalleConSucursal(t *testing.T) {
	sede, sedeID := sedeEstaciones(t)
	ajena, ajenaID := sedeEstaciones(t)
	admin := adminCartaSede(t, sede)
	gestor := gestorCarta(t)
	cat := categoriaPropia(t, gestor)
	nuevo := func() productoResp { return crearProducto(t, gestor, cuerpoProd(cat, codigoProd())) }
	habilitado, pausado, conPrecio, sinFila := nuevo(), nuevo(), nuevo(), nuevo()

	gestorEst := gestorEstaciones(t, sede)
	est := crearEstacion(t, gestorEst, sedeID, "LST", "Listado")
	fijarDisp(t, admin, habilitado.ID, cuerpoDisp(sedeID, true, `"estacion_id":"`+est.ID+`"`))
	fijarDisp(t, admin, pausado.ID, cuerpoDisp(sedeID, false))
	fijarDisp(t, admin, conPrecio.ID, cuerpoDisp(sedeID, true, `"precio_venta":"20"`))

	lista := func(query string, tk string) productosSedePagina {
		t.Helper()
		r := call(t, "GET", "/api/v1/productos"+conLimite("?categoria_id="+cat+query), "", tk)
		esperar(t, r, http.StatusOK, "")
		var pg productosSedePagina
		r.json(t, &pg)
		return pg
	}
	porID := func(pg productosSedePagina, id string) productoSedeResp {
		t.Helper()
		for _, x := range pg.Items {
			if x.ID == id {
				return x
			}
		}
		t.Fatalf("producto %s ausente: %v", id, pg.ids())
		return productoSedeResp{}
	}

	// Sin sucursal_id el campo se omite del JSON.
	r := call(t, "GET", "/api/v1/productos"+conLimite("?categoria_id="+cat), "", admin)
	esperar(t, r, http.StatusOK, "")
	if strings.Contains(r.Body.String(), "en_sucursal") {
		t.Fatalf("sin sucursal_id no debe haber en_sucursal: %s", r.Body.String())
	}

	// Con sucursal_id: cada ítem trae su estado o null.
	pg := lista("&sucursal_id="+sedeID, admin)
	if got := pg.ids(); !slices.Equal(got, []string{habilitado.ID, pausado.ID, conPrecio.ID, sinFila.ID}) {
		t.Fatalf("ítems: %v", got)
	}
	h := porID(pg, habilitado.ID).EnSucursal
	if h == nil || !h.Disponible || h.PrecioSucursal != nil || h.PrecioEfectivo != "12.50" || h.EstacionID == nil || *h.EstacionID != est.ID {
		t.Fatalf("habilitado: %+v", h)
	}
	if x := porID(pg, pausado.ID).EnSucursal; x == nil || x.Disponible || x.PrecioEfectivo != "12.50" {
		t.Fatalf("pausado: %+v", x)
	}
	if x := porID(pg, conPrecio.ID).EnSucursal; x == nil || !x.Disponible || x.PrecioSucursal == nil || *x.PrecioSucursal != "20.00" || x.PrecioEfectivo != "20.00" {
		t.Fatalf("con precio de sede: %+v", x)
	}
	if x := porID(pg, sinFila.ID).EnSucursal; x != nil {
		t.Fatalf("sin fila debe ser null: %+v", x)
	}
	if !strings.Contains(call(t, "GET", "/api/v1/productos"+conLimite("?categoria_id="+cat+"&sucursal_id="+sedeID), "", admin).Body.String(), `"en_sucursal":null`) {
		t.Fatal("sin fila el JSON debe traer en_sucursal:null")
	}

	// Filtro disponible: true = habilitados, false = pausados (no incluye los que no tienen fila).
	if got := lista("&sucursal_id="+sedeID+"&disponible=true", admin).ids(); !slices.Equal(got, []string{habilitado.ID, conPrecio.ID}) {
		t.Fatalf("disponible=true: %v", got)
	}
	if got := lista("&sucursal_id="+sedeID+"&disponible=false", admin).ids(); !slices.Equal(got, []string{pausado.ID}) {
		t.Fatalf("disponible=false: %v", got)
	}
	// Paginación con sucursal_id.
	p1 := lista("&sucursal_id="+sedeID+"&limit=3", admin)
	if len(p1.Items) != 3 || p1.NextCursor == nil {
		t.Fatalf("página 1: %v %v", p1.ids(), p1.NextCursor)
	}
	if p2 := lista("&sucursal_id="+sedeID+"&limit=3&cursor="+*p1.NextCursor, admin); !slices.Equal(p2.ids(), []string{sinFila.ID}) {
		t.Fatalf("página 2: %v", p2.ids())
	}

	// En otra sede (visible para quien la ve) el mismo producto no tiene fila.
	admin2 := adminCartaSede(t, sede, ajena)
	if x := porID(lista("&sucursal_id="+ajenaID, admin2), habilitado.ID).EnSucursal; x != nil {
		t.Fatalf("otra sede: %+v", x)
	}

	// Errores: disponible sin sucursal_id, valores mal formados, sede no visible.
	esperar(t, call(t, "GET", "/api/v1/productos?disponible=true", "", admin), http.StatusBadRequest, "disponible_requiere_sucursal")
	esperar(t, call(t, "GET", "/api/v1/productos?sucursal_id="+sedeID+"&disponible=quizas", "", admin), http.StatusBadRequest, "disponible_invalido")
	esperar(t, call(t, "GET", "/api/v1/productos?sucursal_id=abc", "", admin), http.StatusBadRequest, "sucursal_id_invalido")
	esperar(t, call(t, "GET", "/api/v1/productos?sucursal_id="+ajenaID, "", admin), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/productos?sucursal_id=999999999", "", admin), http.StatusNotFound, "no_encontrado")

	// Detalle con sucursal_id: lo mismo; sin él, sin el campo.
	var d productoSedeResp
	r = call(t, "GET", "/api/v1/productos/"+conPrecio.ID+"?sucursal_id="+sedeID, "", admin)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &d)
	if d.EnSucursal == nil || d.EnSucursal.PrecioEfectivo != "20.00" || d.PrecioVenta != "12.50" {
		t.Fatalf("detalle: %+v / %+v", d.EnSucursal, d.productoResp)
	}
	if strings.Contains(cuerpoProducto(t, admin, conPrecio.ID, ""), "en_sucursal") {
		t.Fatal("detalle sin sucursal_id no debe traer en_sucursal")
	}
	if !strings.Contains(cuerpoProducto(t, admin, sinFila.ID, "?sucursal_id="+sedeID), `"en_sucursal":null`) {
		t.Fatal("detalle sin fila: en_sucursal:null")
	}
	esperar(t, call(t, "GET", "/api/v1/productos/"+conPrecio.ID+"?sucursal_id="+ajenaID, "", admin), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/productos/999999999?sucursal_id="+sedeID, "", admin), http.StatusNotFound, "no_encontrado")

	// COCINA y MESERO (CARTA_VER) ven el estado de su sede; no el de una sede ajena.
	cocina := usuarioRolSede(t, "COCINA", sede)
	if x := porID(lista("&sucursal_id="+sedeID+"&disponible=false", cocina), pausado.ID).EnSucursal; x == nil || x.Disponible {
		t.Fatalf("cocina ve lo pausado: %+v", x)
	}
	mesero := usuarioRolSede(t, "MESERO", sede)
	esperar(t, call(t, "GET", "/api/v1/productos"+conLimite("?sucursal_id="+sedeID), "", mesero), http.StatusOK, "")
	esperar(t, call(t, "GET", "/api/v1/productos?sucursal_id="+ajenaID, "", mesero), http.StatusNotFound, "no_encontrado")
}
