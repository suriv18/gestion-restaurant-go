package integration

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

type categoriaResp struct {
	ID             string  `json:"id"`
	Nombre         string  `json:"nombre"`
	Descripcion    *string `json:"descripcion"`
	Orden          int     `json:"orden_visualizacion"`
	EsActivo       bool    `json:"es_activo"`
	CreadoEn       string  `json:"creado_en"`
	CreadoPor      string  `json:"creado_por"`
	ActualizadoEn  string  `json:"actualizado_en"`
	ActualizadoPor string  `json:"actualizado_por"`
}

type categoriasPagina struct {
	Items      []categoriaResp `json:"items"`
	NextCursor *string         `json:"nextCursor"`
}

func (p categoriasPagina) ids() []string {
	out := make([]string, 0, len(p.Items))
	for _, c := range p.Items {
		out = append(out, c.ID)
	}
	return out
}

// gestorCarta devuelve el token de un usuario con CARTA_ADMINISTRAR.
func gestorCarta(t *testing.T) string {
	t.Helper()
	rolConPermisos(t, "T_CARTA_GESTOR", "CARTA_ADMINISTRAR")
	asegurarUsuario(t, "carta.gestor", "CartaGestor2026Clave", "T_CARTA_GESTOR", "PRINCIPAL")
	return token(t, "carta.gestor", "CartaGestor2026Clave")
}

func crearCategoria(t *testing.T, tk, cuerpo string) categoriaResp {
	t.Helper()
	r := post(t, "/api/v1/categorias", cuerpo, tk)
	esperar(t, r, http.StatusCreated, "")
	var c categoriaResp
	r.json(t, &c)
	return c
}

func TestCategoriasPermisos(t *testing.T) {
	// Sin sesión.
	esperar(t, call(t, "GET", "/api/v1/categorias", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, call(t, "GET", "/api/v1/categorias/1", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, post(t, "/api/v1/categorias", `{"nombre":"X"}`, ""), http.StatusUnauthorized, "no_autenticado")

	// Un rol sin permisos de carta no ve ni escribe.
	rolConPermisos(t, "T_CARTA_NADA", "CLIENTE_VER")
	asegurarUsuario(t, "carta.nada", "CartaNada2026Clave", "T_CARTA_NADA", "PRINCIPAL")
	nada := token(t, "carta.nada", "CartaNada2026Clave")
	esperar(t, call(t, "GET", "/api/v1/categorias", "", nada), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/categorias/1", "", nada), http.StatusForbidden, "sin_permiso")
	esperar(t, post(t, "/api/v1/categorias", `{"nombre":"No debe crearse"}`, nada), http.StatusForbidden, "sin_permiso")

	// MESERO (CARTA_VER por la migración 00010) solo lee.
	gestor := gestorCarta(t)
	c := crearCategoria(t, gestor, fmt.Sprintf(`{"nombre":"Perm %s"}`, sufijo()))
	asegurarUsuario(t, "carta.mesero", "CartaMesero2026Clave", "MESERO", "PRINCIPAL")
	mesero := token(t, "carta.mesero", "CartaMesero2026Clave")
	esperar(t, call(t, "GET", "/api/v1/categorias"+conLimite(""), "", mesero), http.StatusOK, "")
	esperar(t, call(t, "GET", "/api/v1/categorias/"+c.ID, "", mesero), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/categorias", `{"nombre":"Mesero no crea"}`, mesero), http.StatusForbidden, "sin_permiso")
	esperar(t, patch(t, "/api/v1/categorias/"+c.ID, `{"nombre":"Mesero no edita"}`, mesero), http.StatusForbidden, "sin_permiso")
	esperar(t, post(t, "/api/v1/categorias/"+c.ID+"/baja", ``, mesero), http.StatusForbidden, "sin_permiso")
	esperar(t, post(t, "/api/v1/categorias/"+c.ID+"/restauracion", ``, mesero), http.StatusForbidden, "sin_permiso")
}

func TestCategoriaAltaIdempotenciaYValidaciones(t *testing.T) {
	tk := gestorCarta(t)
	nombre := "Entradas " + sufijo()
	cuerpo := fmt.Sprintf(`{"nombre":"  %s  ","descripcion":"Línea uno\nlínea dos","orden_visualizacion":7}`, nombre)

	clave := map[string]string{"Idempotency-Key": nuevaClave()}
	r := callH(t, "POST", "/api/v1/categorias", cuerpo, tk, clave)
	esperar(t, r, http.StatusCreated, "")
	var c categoriaResp
	r.json(t, &c)
	if c.Nombre != nombre || c.Descripcion == nil || *c.Descripcion != "Línea uno\nlínea dos" || c.Orden != 7 || !c.EsActivo ||
		c.CreadoPor != idDe(t, "usuarios", "username", "carta.gestor") || c.ActualizadoPor != c.CreadoPor || c.CreadoEn == "" {
		t.Fatalf("categoría creada: %+v", c)
	}
	// Misma clave y mismo cuerpo: misma respuesta, sin duplicar.
	rep := callH(t, "POST", "/api/v1/categorias", cuerpo, tk, clave)
	esperar(t, rep, http.StatusCreated, "")
	var c2 categoriaResp
	rep.json(t, &c2)
	if c2.ID != c.ID || scalar(t, `SELECT count(*) FROM sch_restaurante.categorias WHERE nombre=$1`, nombre) != 1 {
		t.Fatalf("la repetición creó otra categoría: %s vs %s", c2.ID, c.ID)
	}
	// Misma clave con otro cuerpo: conflicto de idempotencia.
	esperar(t, callH(t, "POST", "/api/v1/categorias", `{"nombre":"Otra `+sufijo()+`"}`, tk, clave), http.StatusConflict, "idempotency_key_reutilizada")
	esperar(t, call(t, "POST", "/api/v1/categorias", `{"nombre":"Sin clave `+sufijo()+`"}`, tk), http.StatusBadRequest, "idempotency_key_requerida")

	// Valores por defecto: orden 0 y descripción nula.
	def := crearCategoria(t, tk, fmt.Sprintf(`{"nombre":"Defecto %s"}`, sufijo()))
	if def.Orden != 0 || def.Descripcion != nil {
		t.Fatalf("valores por defecto: %+v", def)
	}
	// Una descripción en blanco se guarda como nula.
	blanco := crearCategoria(t, tk, fmt.Sprintf(`{"nombre":"Blanco %s","descripcion":"   "}`, sufijo()))
	if blanco.Descripcion != nil {
		t.Fatalf("descripción en blanco: %+v", blanco)
	}

	// Validaciones.
	esperar(t, post(t, "/api/v1/categorias", `{}`, tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, post(t, "/api/v1/categorias", `{"nombre":"   "}`, tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, post(t, "/api/v1/categorias", `{"nombre":"dos\nlíneas"}`, tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, post(t, "/api/v1/categorias", `{"nombre":"`+strings.Repeat("a", 101)+`"}`, tk), http.StatusUnprocessableEntity, "nombre_invalido")
	// Exactamente 100 caracteres (no bytes) es válido.
	s100 := sufijo()
	esperar(t, post(t, "/api/v1/categorias", `{"nombre":"`+strings.Repeat("ñ", 100-len(s100))+s100+`"}`, tk), http.StatusCreated, "")
	esperar(t, post(t, "/api/v1/categorias", `{"nombre":"D1 `+sufijo()+`","descripcion":"`+strings.Repeat("d", 1001)+`"}`, tk), http.StatusUnprocessableEntity, "descripcion_invalido")
	esperar(t, post(t, "/api/v1/categorias", `{"nombre":"O1 `+sufijo()+`","orden_visualizacion":-1}`, tk), http.StatusUnprocessableEntity, "orden_invalido")
	esperar(t, post(t, "/api/v1/categorias", `{"nombre":"O2 `+sufijo()+`","orden_visualizacion":10000}`, tk), http.StatusUnprocessableEntity, "orden_invalido")
	esperar(t, post(t, "/api/v1/categorias", `{"nombre":"O3 `+sufijo()+`","orden_visualizacion":9999}`, tk), http.StatusCreated, "")
	esperar(t, post(t, "/api/v1/categorias", `{"nombre":"X","campo_raro":1}`, tk), http.StatusBadRequest, "")
	esperar(t, post(t, "/api/v1/categorias", `{"nombre":"X","orden_visualizacion":"uno"}`, tk), http.StatusBadRequest, "")

	// El nombre es único, también frente a una categoría dada de baja.
	esperar(t, post(t, "/api/v1/categorias", fmt.Sprintf(`{"nombre":"%s"}`, nombre), tk), http.StatusConflict, "categoria_duplicada")
	esperar(t, post(t, "/api/v1/categorias/"+c.ID+"/baja", ``, tk), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/categorias", fmt.Sprintf(`{"nombre":"%s"}`, nombre), tk), http.StatusConflict, "categoria_duplicada")
}

func TestCategoriaLecturaYEdicion(t *testing.T) {
	tk := gestorCarta(t)
	nombre := "Edit " + sufijo()
	c := crearCategoria(t, tk, fmt.Sprintf(`{"nombre":"%s","descripcion":"Original","orden_visualizacion":3}`, nombre))
	otra := crearCategoria(t, tk, fmt.Sprintf(`{"nombre":"Otra edit %s"}`, sufijo()))

	var leida categoriaResp
	g := call(t, "GET", "/api/v1/categorias/"+c.ID, "", tk)
	esperar(t, g, http.StatusOK, "")
	g.json(t, &leida)
	if leida.ID != c.ID || leida.Nombre != nombre || leida.Orden != 3 {
		t.Fatalf("lectura: %+v", leida)
	}
	esperar(t, call(t, "GET", "/api/v1/categorias/999999999", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/categorias/abc", "", tk), http.StatusBadRequest, "id_invalido")

	// PATCH parcial: solo el orden; el resto queda igual.
	r := patch(t, "/api/v1/categorias/"+c.ID, `{"orden_visualizacion":12}`, tk)
	esperar(t, r, http.StatusOK, "")
	var ed categoriaResp
	r.json(t, &ed)
	if ed.Nombre != nombre || ed.Orden != 12 || ed.Descripcion == nil || *ed.Descripcion != "Original" {
		t.Fatalf("PATCH parcial: %+v", ed)
	}
	// Nombre nuevo y descripción vaciada con null.
	nuevo := nombre + " editado"
	r = patch(t, "/api/v1/categorias/"+c.ID, fmt.Sprintf(`{"nombre":" %s ","descripcion":null}`, nuevo), tk)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &ed)
	if ed.Nombre != nuevo || ed.Descripcion != nil || ed.Orden != 12 {
		t.Fatalf("PATCH con null: %+v", ed)
	}
	// Descripción nueva; el orden puede volver a 0.
	r = patch(t, "/api/v1/categorias/"+c.ID, `{"descripcion":"Con\nsaltos","orden_visualizacion":0}`, tk)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &ed)
	if ed.Descripcion == nil || *ed.Descripcion != "Con\nsaltos" || ed.Orden != 0 {
		t.Fatalf("PATCH descripción/orden: %+v", ed)
	}

	esperar(t, patch(t, "/api/v1/categorias/"+c.ID, `{}`, tk), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, patch(t, "/api/v1/categorias/"+c.ID, `{"nombre":"  "}`, tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, patch(t, "/api/v1/categorias/"+c.ID, `{"orden_visualizacion":10000}`, tk), http.StatusUnprocessableEntity, "orden_invalido")
	esperar(t, patch(t, "/api/v1/categorias/"+c.ID, `{"descripcion":"`+strings.Repeat("d", 1001)+`"}`, tk), http.StatusUnprocessableEntity, "descripcion_invalido")
	esperar(t, patch(t, "/api/v1/categorias/"+c.ID, `{"campo_raro":1}`, tk), http.StatusBadRequest, "")
	esperar(t, patch(t, "/api/v1/categorias/"+c.ID, fmt.Sprintf(`{"nombre":"%s"}`, otra.Nombre), tk), http.StatusConflict, "categoria_duplicada")
	esperar(t, patch(t, "/api/v1/categorias/999999999", `{"nombre":"Z"}`, tk), http.StatusNotFound, "no_encontrado")

	// Una categoría dada de baja no se edita.
	esperar(t, post(t, "/api/v1/categorias/"+c.ID+"/baja", ``, tk), http.StatusOK, "")
	esperar(t, patch(t, "/api/v1/categorias/"+c.ID, `{"nombre":"No"}`, tk), http.StatusConflict, "categoria_inactiva")
}

func TestCategoriasListadoFiltrosYPaginacion(t *testing.T) {
	tk := gestorCarta(t)
	marca := "Lst" + sufijo()
	var ids []string
	for i := 1; i <= 3; i++ {
		c := crearCategoria(t, tk, fmt.Sprintf(`{"nombre":"%s %d"}`, marca, i))
		ids = append(ids, c.ID)
	}
	esperar(t, post(t, "/api/v1/categorias/"+ids[2]+"/baja", ``, tk), http.StatusOK, "")

	lista := func(consulta string) categoriasPagina {
		var p categoriasPagina
		r := call(t, "GET", "/api/v1/categorias"+conLimite(consulta), "", tk)
		esperar(t, r, http.StatusOK, "")
		r.json(t, &p)
		return p
	}
	if got := lista("?q=" + marca).ids(); len(got) != 3 || !slices.Contains(got, ids[0]) {
		t.Fatalf("q=%s: %v", marca, got)
	}
	if got := lista("?q=" + strings.ToLower(marca)).ids(); len(got) != 3 {
		t.Fatalf("q en minúsculas: %v", got)
	}
	// Un comodín literal no coincide con todo.
	if got := lista("?q=%25").ids(); slices.Contains(got, ids[0]) {
		t.Fatalf("q=%%25 trató el comodín como patrón: %v", got)
	}
	if got := lista("?q=" + marca + "&es_activo=true").ids(); len(got) != 2 || slices.Contains(got, ids[2]) {
		t.Fatalf("es_activo=true: %v", got)
	}
	if got := lista("?q=" + marca + "&es_activo=false").ids(); len(got) != 1 || got[0] != ids[2] {
		t.Fatalf("es_activo=false: %v", got)
	}
	// Orden por id ascendente y paginación por cursor sin repetir ni saltar.
	if todos := lista("?q=" + marca).ids(); !slices.Equal(todos, ids) {
		t.Fatalf("orden por id: %v, quería %v", todos, ids)
	}
	p1 := lista("?q=" + marca + "&limit=2")
	if len(p1.Items) != 2 || p1.NextCursor == nil {
		t.Fatalf("página 1: %+v", p1)
	}
	p2 := lista("?q=" + marca + "&limit=2&cursor=" + *p1.NextCursor)
	if len(p2.Items) != 1 || p2.NextCursor != nil || slices.Contains(p1.ids(), p2.Items[0].ID) {
		t.Fatalf("página 2: %+v", p2)
	}
	esperar(t, call(t, "GET", "/api/v1/categorias?es_activo=quizas", "", tk), http.StatusBadRequest, "es_activo_invalido")
	esperar(t, call(t, "GET", "/api/v1/categorias?limit=0", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/categorias?limit=101", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/categorias?cursor=!!!", "", tk), http.StatusBadRequest, "cursor_invalido")
}

func TestCategoriaBajaYRestauracion(t *testing.T) {
	tk := gestorCarta(t)
	c := crearCategoria(t, tk, fmt.Sprintf(`{"nombre":"Baja %s","descripcion":"d"}`, sufijo()))
	baja := "/api/v1/categorias/" + c.ID + "/baja"
	rest := "/api/v1/categorias/" + c.ID + "/restauracion"

	// Una restauración de algo activo es un conflicto.
	esperar(t, post(t, rest, ``, tk), http.StatusConflict, "ya_activa")

	// Baja idempotente: la misma clave devuelve la misma respuesta.
	clave := map[string]string{"Idempotency-Key": nuevaClave()}
	r := callH(t, "POST", baja, "", tk, clave)
	esperar(t, r, http.StatusOK, "")
	var b categoriaResp
	r.json(t, &b)
	if b.ID != c.ID || b.EsActivo {
		t.Fatalf("baja: %+v", b)
	}
	esperar(t, callH(t, "POST", baja, "", tk, clave), http.StatusOK, "")
	esperar(t, call(t, "POST", baja, "", tk), http.StatusBadRequest, "idempotency_key_requerida")
	esperar(t, post(t, baja, ``, tk), http.StatusConflict, "ya_dada_de_baja")
	esperar(t, post(t, baja, `{"motivo":"x"}`, tk), http.StatusBadRequest, "")
	esperar(t, post(t, "/api/v1/categorias/999999999/baja", ``, tk), http.StatusNotFound, "no_encontrado")

	// Restauración: misma fila, mismo id; admite cuerpo vacío o {}.
	r = post(t, rest, `{}`, tk)
	esperar(t, r, http.StatusOK, "")
	var rs categoriaResp
	r.json(t, &rs)
	if rs.ID != c.ID || !rs.EsActivo || rs.Nombre != c.Nombre {
		t.Fatalf("restauración: %+v", rs)
	}
	esperar(t, post(t, rest, ``, tk), http.StatusConflict, "ya_activa")
	esperar(t, post(t, "/api/v1/categorias/999999999/restauracion", ``, tk), http.StatusNotFound, "no_encontrado")

	// Se puede dar de baja de nuevo tras restaurar.
	esperar(t, post(t, baja, ``, tk), http.StatusOK, "")
}

func TestCategoriaBajaBloqueadaPorProductosActivos(t *testing.T) {
	tk := gestorCarta(t)
	c := crearCategoria(t, tk, fmt.Sprintf(`{"nombre":"ConProd %s"}`, sufijo()))
	codigo := "CP" + numeroUnico(12)
	asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.productos(categoria_id, codigo, nombre, tipo, precio_venta, afectacion_igv)
		VALUES (%s, '%s', 'Plato de prueba', 'PREPARADO', 10, '20')`, c.ID, codigo))

	baja := "/api/v1/categorias/" + c.ID + "/baja"
	esperar(t, post(t, baja, ``, tk), http.StatusConflict, "categoria_con_productos_activos")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.categorias WHERE id=`+c.ID+` AND es_activo`) != 1 {
		t.Fatal("la categoría debe seguir activa")
	}

	// Con el producto dado de baja, la categoría ya puede darse de baja.
	asSystem(t, `UPDATE sch_restaurante.productos SET es_activo=false, motivo_eliminacion='Retirado' WHERE codigo='`+codigo+`'`)
	esperar(t, post(t, baja, ``, tk), http.StatusOK, "")
}
