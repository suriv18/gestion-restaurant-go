package integration

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

type productoResp struct {
	ID                string  `json:"id"`
	CategoriaID       string  `json:"categoria_id"`
	Codigo            string  `json:"codigo"`
	Nombre            string  `json:"nombre"`
	Descripcion       *string `json:"descripcion"`
	Tipo              string  `json:"tipo"`
	ArticuloID        *string `json:"articulo_id"`
	FactorConsumo     string  `json:"factor_consumo"`
	PrecioVenta       string  `json:"precio_venta"`
	AfectacionIGV     string  `json:"afectacion_igv"`
	ImpuestoID        *string `json:"impuesto_id"`
	ImagenURL         *string `json:"imagen_url"`
	EsActivo          bool    `json:"es_activo"`
	MotivoEliminacion *string `json:"motivo_eliminacion"`
	EliminadoEn       *string `json:"eliminado_en"`
	EliminadoPor      *string `json:"eliminado_por"`
	CreadoEn          string  `json:"creado_en"`
	CreadoPor         string  `json:"creado_por"`
	ActualizadoEn     string  `json:"actualizado_en"`
	ActualizadoPor    string  `json:"actualizado_por"`
}

type productosPagina struct {
	Items      []productoResp `json:"items"`
	NextCursor *string        `json:"nextCursor"`
}

func (p productosPagina) ids() []string {
	out := make([]string, 0, len(p.Items))
	for _, x := range p.Items {
		out = append(out, x.ID)
	}
	return out
}

// categoriaPropia crea por API una categoría con nombre único y devuelve su id.
func categoriaPropia(t *testing.T, tk string) string {
	t.Helper()
	return crearCategoria(t, tk, fmt.Sprintf(`{"nombre":"Cat prod %s"}`, sufijo())).ID
}

// codigoProd devuelve un código de producto único de la prueba (ya en mayúsculas).
func codigoProd() string { return "P-" + strings.ToUpper(sufijo()) }

// cuerpoProd arma el cuerpo de un alta mínima de tipo PREPARADO, exonerado, con extras opcionales
// (fragmentos JSON ya formados, p. ej. `"descripcion":"x"`).
func cuerpoProd(cat, codigo string, extras ...string) string {
	b := fmt.Sprintf(`{"categoria_id":"%s","codigo":"%s","nombre":"Plato %s","tipo":"PREPARADO","precio_venta":"12.50","afectacion_igv":"20"`, cat, codigo, codigo)
	for _, e := range extras {
		b += "," + e
	}
	return b + "}"
}

func crearProducto(t *testing.T, tk, cuerpo string) productoResp {
	t.Helper()
	r := post(t, "/api/v1/productos", cuerpo, tk)
	esperar(t, r, http.StatusCreated, "")
	var p productoResp
	r.json(t, &p)
	return p
}

func listarProductos(t *testing.T, query, tk string) productosPagina {
	t.Helper()
	if strings.Contains(query, "limit=") && !strings.HasPrefix(query, "?") {
		query = "?" + query
	}
	r := call(t, "GET", "/api/v1/productos"+conLimite(query), "", tk)
	esperar(t, r, http.StatusOK, "")
	var p productosPagina
	r.json(t, &p)
	return p
}

func TestProductosPermisos(t *testing.T) {
	esperar(t, call(t, "GET", "/api/v1/productos", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, call(t, "GET", "/api/v1/productos/1", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, post(t, "/api/v1/productos", `{}`, ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, patch(t, "/api/v1/productos/1", `{"nombre":"X"}`, ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, post(t, "/api/v1/productos/1/baja", `{"motivo":"x"}`, ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, post(t, "/api/v1/productos/1/restauracion", ``, ""), http.StatusUnauthorized, "no_autenticado")

	gestor := gestorCarta(t)
	p := crearProducto(t, gestor, cuerpoProd(categoriaPropia(t, gestor), codigoProd()))

	// Sin permisos de carta: ni lee ni escribe.
	rolConPermisos(t, "T_CARTA_NADA", "CLIENTE_VER")
	asegurarUsuario(t, "carta.nada", "CartaNada2026Clave", "T_CARTA_NADA", "PRINCIPAL")
	nada := token(t, "carta.nada", "CartaNada2026Clave")
	esperar(t, call(t, "GET", "/api/v1/productos", "", nada), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/productos/"+p.ID, "", nada), http.StatusForbidden, "sin_permiso")
	esperar(t, post(t, "/api/v1/productos", cuerpoProd("1", "NO"), nada), http.StatusForbidden, "sin_permiso")

	// MESERO (CARTA_VER) lee y no escribe.
	asegurarUsuario(t, "carta.mesero", "CartaMesero2026Clave", "MESERO", "PRINCIPAL")
	mesero := token(t, "carta.mesero", "CartaMesero2026Clave")
	esperar(t, call(t, "GET", "/api/v1/productos"+conLimite(""), "", mesero), http.StatusOK, "")
	esperar(t, call(t, "GET", "/api/v1/productos/"+p.ID, "", mesero), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/productos", cuerpoProd(p.CategoriaID, "NO"), mesero), http.StatusForbidden, "sin_permiso")
	esperar(t, patch(t, "/api/v1/productos/"+p.ID, `{"nombre":"Mesero no edita"}`, mesero), http.StatusForbidden, "sin_permiso")
	esperar(t, post(t, "/api/v1/productos/"+p.ID+"/baja", `{"motivo":"x"}`, mesero), http.StatusForbidden, "sin_permiso")
	esperar(t, post(t, "/api/v1/productos/"+p.ID+"/restauracion", ``, mesero), http.StatusForbidden, "sin_permiso")
}

func TestProductoAltaTiposIdempotenciaYNormalizacion(t *testing.T) {
	tk := gestorCarta(t)
	cat := categoriaPropia(t, tk)
	imp := impuestoPropio(t, true)

	// Gravado completo: código en minúsculas se guarda en mayúsculas; decimales normalizados.
	cod := codigoProd()
	cuerpo := fmt.Sprintf(`{"categoria_id":"%s","codigo":"  %s  ","nombre":"  Lomo saltado  ","descripcion":"Línea uno\nlínea dos",
		"tipo":"PREPARADO","factor_consumo":"0.5","precio_venta":"25","afectacion_igv":"10","impuesto_id":"%s",
		"imagen_url":"https://cdn.example.com/lomo.png"}`, cat, strings.ToLower(cod), imp)
	clave := map[string]string{"Idempotency-Key": nuevaClave()}
	r := callH(t, "POST", "/api/v1/productos", cuerpo, tk, clave)
	esperar(t, r, http.StatusCreated, "")
	var p productoResp
	r.json(t, &p)
	if p.Codigo != cod || p.Nombre != "Lomo saltado" || p.CategoriaID != cat || p.Tipo != "PREPARADO" || p.ArticuloID != nil ||
		p.FactorConsumo != "0.500000" || p.PrecioVenta != "25.00" || p.AfectacionIGV != "10" || p.ImpuestoID == nil || *p.ImpuestoID != imp ||
		p.Descripcion == nil || *p.Descripcion != "Línea uno\nlínea dos" || p.ImagenURL == nil || *p.ImagenURL != "https://cdn.example.com/lomo.png" ||
		!p.EsActivo || p.MotivoEliminacion != nil || p.EliminadoEn != nil || p.EliminadoPor != nil ||
		p.CreadoPor != idDe(t, "usuarios", "username", "carta.gestor") || p.ActualizadoPor != p.CreadoPor || p.CreadoEn == "" {
		t.Fatalf("producto creado: %+v", p)
	}
	// Misma clave y cuerpo: misma respuesta, sin duplicar; otro cuerpo: conflicto.
	rep := callH(t, "POST", "/api/v1/productos", cuerpo, tk, clave)
	esperar(t, rep, http.StatusCreated, "")
	var p2 productoResp
	rep.json(t, &p2)
	if p2.ID != p.ID || scalar(t, `SELECT count(*) FROM sch_restaurante.productos WHERE codigo=$1`, cod) != 1 {
		t.Fatalf("la repetición duplicó: %s vs %s", p2.ID, p.ID)
	}
	esperar(t, callH(t, "POST", "/api/v1/productos", cuerpoProd(cat, codigoProd()), tk, clave), http.StatusConflict, "idempotency_key_reutilizada")
	esperar(t, call(t, "POST", "/api/v1/productos", cuerpoProd(cat, codigoProd()), tk), http.StatusBadRequest, "idempotency_key_requerida")

	// Valores por defecto: factor "1" -> "1.000000", opcionales nulos; precio 0 es válido.
	def := crearProducto(t, tk, fmt.Sprintf(`{"categoria_id":"%s","codigo":"%s","nombre":"Defecto","tipo":"SERVICIO","precio_venta":"0","afectacion_igv":"30"}`, cat, codigoProd()))
	if def.FactorConsumo != "1.000000" || def.PrecioVenta != "0.00" || def.Descripcion != nil || def.ImagenURL != nil || def.ImpuestoID != nil {
		t.Fatalf("valores por defecto: %+v", def)
	}
	// Texto opcional en blanco se guarda como nulo.
	bl := crearProducto(t, tk, cuerpoProd(cat, codigoProd(), `"descripcion":"   "`, `"imagen_url":"  "`))
	if bl.Descripcion != nil || bl.ImagenURL != nil {
		t.Fatalf("blancos: %+v", bl)
	}

	// Los tipos sin artículo funcionan; el límite superior de precio y de factor, también.
	for _, tipo := range []string{"PREPARADO", "ADICIONAL", "SERVICIO"} {
		x := crearProducto(t, tk, fmt.Sprintf(`{"categoria_id":"%s","codigo":"%s","nombre":"T","tipo":"%s","precio_venta":"99999999999.99",
			"factor_consumo":"999999999999.999999","afectacion_igv":"20"}`, cat, codigoProd(), tipo))
		if x.Tipo != tipo || x.PrecioVenta != "99999999999.99" || x.FactorConsumo != "999999999999.999999" {
			t.Fatalf("tipo %s: %+v", tipo, x)
		}
	}

	// REVENTA: sin artículo, 422; con un artículo inexistente (I5 aún no los crea), referencia_invalida
	// y no queda ninguna fila.
	reventa := func(extra string) string {
		return fmt.Sprintf(`{"categoria_id":"%s","codigo":"%s","nombre":"R","tipo":"REVENTA","precio_venta":"5","afectacion_igv":"20"%s}`, cat, codigoProd(), extra)
	}
	esperar(t, post(t, "/api/v1/productos", reventa(``), tk), http.StatusUnprocessableEntity, "articulo_requerido")
	rr := post(t, "/api/v1/productos", reventa(`,"articulo_id":"999999999"`), tk)
	esperar(t, rr, http.StatusUnprocessableEntity, "referencia_invalida")
	if !strings.Contains(rr.Body.String(), "artículo") || strings.Contains(strings.ToLower(rr.Body.String()), "referencia articulo_id") {
		t.Fatalf("mensaje no curado: %s", rr.Body.String())
	}
}

func TestProductoValidacionesDelAlta(t *testing.T) {
	tk := gestorCarta(t)
	cat := categoriaPropia(t, tk)
	imp := impuestoPropio(t, true)
	// alta arma un cuerpo válido y sustituye un campo (clave -> fragmento JSON del valor).
	alta := func(cambios map[string]string) string {
		campos := map[string]string{
			"categoria_id": `"` + cat + `"`, "codigo": `"` + codigoProd() + `"`, "nombre": `"Valido"`, "tipo": `"PREPARADO"`,
			"precio_venta": `"10.00"`, "afectacion_igv": `"20"`,
		}
		for k, v := range cambios {
			if v == "" {
				delete(campos, k)
			} else {
				campos[k] = v
			}
		}
		partes := make([]string, 0, len(campos))
		for k, v := range campos {
			partes = append(partes, `"`+k+`":`+v)
		}
		return "{" + strings.Join(partes, ",") + "}"
	}
	invalido := func(code string, cambios map[string]string) {
		t.Helper()
		esperar(t, post(t, "/api/v1/productos", alta(cambios), tk), http.StatusUnprocessableEntity, code)
	}
	valido := func(cambios map[string]string) {
		t.Helper()
		esperar(t, post(t, "/api/v1/productos", alta(cambios), tk), http.StatusCreated, "")
	}

	invalido("categoria_requerida", map[string]string{"categoria_id": ""})
	// Formato de id mal formado: 400 (no es un valor de negocio).
	esperar(t, post(t, "/api/v1/productos", alta(map[string]string{"categoria_id": `"abc"`}), tk), http.StatusBadRequest, "categoria_id_invalido")
	esperar(t, post(t, "/api/v1/productos", alta(map[string]string{"categoria_id": `"0"`}), tk), http.StatusBadRequest, "categoria_id_invalido")
	esperar(t, post(t, "/api/v1/productos", alta(map[string]string{"articulo_id": `"x"`}), tk), http.StatusBadRequest, "articulo_id_invalido")

	// Código.
	for _, c := range []string{`""`, `"   "`, `"A B"`, `"A\nB"`, `"Ñ1"`, `"A/B"`, `"` + strings.Repeat("A", 41) + `"`} {
		invalido("codigo_invalido", map[string]string{"codigo": c})
	}
	invalido("codigo_invalido", map[string]string{"codigo": ""})
	s40 := sufijo()
	valido(map[string]string{"codigo": `"` + strings.Repeat("a", 40-len(s40)) + s40 + `"`})
	valido(map[string]string{"codigo": `"a.b_c-` + sufijo() + `"`})

	// Nombre.
	invalido("nombre_invalido", map[string]string{"nombre": ""})
	invalido("nombre_invalido", map[string]string{"nombre": `"   "`})
	invalido("nombre_invalido", map[string]string{"nombre": `"dos\nlíneas"`})
	invalido("nombre_invalido", map[string]string{"nombre": `"` + strings.Repeat("a", 181) + `"`})
	valido(map[string]string{"nombre": `"` + strings.Repeat("ñ", 180) + `"`})

	// Descripción (2000, multilínea).
	invalido("descripcion_invalido", map[string]string{"descripcion": `"` + strings.Repeat("d", 2001) + `"`})
	valido(map[string]string{"descripcion": `"` + strings.Repeat("d", 2000) + `"`})

	// Tipo y afectación.
	for _, v := range []string{"", `""`, `"preparado"`, `"OTRO"`} {
		invalido("tipo_invalido", map[string]string{"tipo": v})
	}
	for _, v := range []string{"", `""`, `"40"`, `"1"`, `"010"`} {
		invalido("afectacion_invalida", map[string]string{"afectacion_igv": v})
	}
	invalido("impuesto_requerido", map[string]string{"afectacion_igv": `"10"`})
	valido(map[string]string{"afectacion_igv": `"10"`, "impuesto_id": `"` + imp + `"`})
	// Un impuesto con afectación exonerada se admite (queda informativo).
	valido(map[string]string{"afectacion_igv": `"30"`, "impuesto_id": `"` + imp + `"`})
	invalido("articulo_no_permitido", map[string]string{"articulo_id": `"5"`})

	// Precio: decimales como cadena, sin float.
	for _, v := range []string{"", `""`, `"abc"`, `"-1"`, `"-0"`, `"+1"`, `" 5"`, `"5 "`, `"1e3"`, `"0.001"`, `"10.555"`, `".5"`, `"5."`,
		`"1,5"`, `"100000000000.00"`, `"100000000000"`} {
		invalido("precio_invalido", map[string]string{"precio_venta": v})
	}
	valido(map[string]string{"precio_venta": `"99999999999.99"`})
	// Un número JSON no es una cadena: 400.
	esperar(t, post(t, "/api/v1/productos", alta(map[string]string{"precio_venta": `12.5`}), tk), http.StatusBadRequest, "")

	// Factor: > 0 con hasta 6 decimales.
	for _, v := range []string{`""`, `"0"`, `"0.000000"`, `"-1"`, `"0.0000001"`, `" 1"`, `"1e3"`, `"1000000000000"`} {
		invalido("factor_consumo_invalido", map[string]string{"factor_consumo": v})
	}
	valido(map[string]string{"factor_consumo": `"0.000001"`})

	// Imagen.
	for _, v := range []string{`"ftp://x.com/a.png"`, `"javascript:alert(1)"`, `"/a.png"`, `"//x.com/a"`, `"https://"`, `"https://x .com/a"`,
		`"` + "https://x.com/" + strings.Repeat("a", 1987) + `"`} {
		invalido("imagen_url_invalido", map[string]string{"imagen_url": v})
	}
	valido(map[string]string{"imagen_url": `"http://x.com/a.png"`})
	valido(map[string]string{"imagen_url": `"` + "https://x.com/" + strings.Repeat("a", 1986) + `"`})

	// Cuerpo.
	esperar(t, post(t, "/api/v1/productos", `{"campo_raro":1}`, tk), http.StatusBadRequest, "")
	esperar(t, post(t, "/api/v1/productos", `{"id":"5"}`, tk), http.StatusBadRequest, "")
}

func TestProductoCodigoDuplicadoYReferenciasInactivas(t *testing.T) {
	tk := gestorCarta(t)
	cat := categoriaPropia(t, tk)
	cod := codigoProd()
	p := crearProducto(t, tk, cuerpoProd(cat, cod))

	// Duplicado: también con otra capitalización y también frente a un producto dado de baja.
	esperar(t, post(t, "/api/v1/productos", cuerpoProd(cat, strings.ToLower(cod)), tk), http.StatusConflict, "codigo_duplicado")
	esperar(t, post(t, "/api/v1/productos/"+p.ID+"/baja", `{"motivo":"Retirado"}`, tk), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/productos", cuerpoProd(cat, cod), tk), http.StatusConflict, "codigo_duplicado")

	// Categoría inexistente, categoría inactiva, impuesto inactivo e inexistente: 422 con mensaje legible.
	r := post(t, "/api/v1/productos", cuerpoProd("999999999", codigoProd()), tk)
	esperar(t, r, http.StatusUnprocessableEntity, "referencia_invalida")
	if !strings.Contains(r.Body.String(), "categoría") || strings.Contains(r.Body.String(), "categoria_id=") {
		t.Fatalf("mensaje no curado: %s", r.Body.String())
	}
	inactiva := categoriaPropia(t, tk)
	esperar(t, post(t, "/api/v1/categorias/"+inactiva+"/baja", ``, tk), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/productos", cuerpoProd(inactiva, codigoProd()), tk), http.StatusUnprocessableEntity, "referencia_invalida")

	impBaja := impuestoPropio(t, false)
	r = post(t, "/api/v1/productos", cuerpoProd(cat, codigoProd(), `"afectacion_igv":"10"`, `"impuesto_id":"`+impBaja+`"`), tk)
	esperar(t, r, http.StatusUnprocessableEntity, "referencia_invalida")
	if !strings.Contains(r.Body.String(), "impuesto") {
		t.Fatalf("mensaje no curado: %s", r.Body.String())
	}
	esperar(t, post(t, "/api/v1/productos", cuerpoProd(cat, codigoProd(), `"afectacion_igv":"10"`, `"impuesto_id":"999999999"`), tk),
		http.StatusUnprocessableEntity, "referencia_invalida")
}

func TestProductoLecturaYEdicion(t *testing.T) {
	tk := gestorCarta(t)
	cat := categoriaPropia(t, tk)
	cat2 := categoriaPropia(t, tk)
	imp := impuestoPropio(t, true)
	cod := codigoProd()
	p := crearProducto(t, tk, cuerpoProd(cat, cod, `"descripcion":"Original"`, `"imagen_url":"https://x.com/a.png"`))
	url := "/api/v1/productos/" + p.ID

	r := call(t, "GET", url, "", tk)
	esperar(t, r, http.StatusOK, "")
	var g productoResp
	r.json(t, &g)
	if g.ID != p.ID || g.Codigo != cod || g.PrecioVenta != "12.50" {
		t.Fatalf("lectura: %+v", g)
	}
	esperar(t, call(t, "GET", "/api/v1/productos/999999999", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/productos/abc", "", tk), http.StatusBadRequest, "id_invalido")

	// PATCH parcial: solo cambia lo enviado.
	r = patch(t, url, `{"nombre":"  Nuevo nombre  ","precio_venta":"13"}`, tk)
	esperar(t, r, http.StatusOK, "")
	var e productoResp
	r.json(t, &e)
	if e.Nombre != "Nuevo nombre" || e.PrecioVenta != "13.00" || e.Codigo != cod || e.CategoriaID != cat || e.FactorConsumo != "1.000000" ||
		e.Descripcion == nil || *e.Descripcion != "Original" || e.ImagenURL == nil {
		t.Fatalf("PATCH parcial: %+v", e)
	}
	// Código (se normaliza), categoría, factor y tipo.
	nuevoCod := codigoProd()
	r = patch(t, url, fmt.Sprintf(`{"codigo":"%s","categoria_id":"%s","factor_consumo":"2.5","tipo":"ADICIONAL"}`, strings.ToLower(nuevoCod), cat2), tk)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &e)
	if e.Codigo != nuevoCod || e.CategoriaID != cat2 || e.FactorConsumo != "2.500000" || e.Tipo != "ADICIONAL" {
		t.Fatalf("PATCH campos: %+v", e)
	}
	// null vacía descripción e imagen; en blanco también.
	r = patch(t, url, `{"descripcion":null,"imagen_url":null}`, tk)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &e)
	if e.Descripcion != nil || e.ImagenURL != nil {
		t.Fatalf("PATCH null: %+v", e)
	}
	esperar(t, patch(t, url, `{"descripcion":"Otra","imagen_url":"http://y.com/b.jpg"}`, tk), http.StatusOK, "")
	r = patch(t, url, `{"descripcion":"  "}`, tk)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &e)
	if e.Descripcion != nil || e.ImagenURL == nil {
		t.Fatalf("PATCH en blanco: %+v", e)
	}

	// Invariantes cruzadas sobre el valor resultante.
	// Afectación gravada sin impuesto (el producto no tiene) -> impuesto_requerido.
	esperar(t, patch(t, url, `{"afectacion_igv":"10"}`, tk), http.StatusUnprocessableEntity, "impuesto_requerido")
	// Con el impuesto en la misma solicitud, sí.
	r = patch(t, url, fmt.Sprintf(`{"afectacion_igv":"10","impuesto_id":"%s"}`, imp), tk)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &e)
	if e.AfectacionIGV != "10" || e.ImpuestoID == nil || *e.ImpuestoID != imp {
		t.Fatalf("PATCH afectación: %+v", e)
	}
	// Vaciar el impuesto de un gravado -> impuesto_requerido; cambiar a exonerado junto con vaciarlo, sí.
	esperar(t, patch(t, url, `{"impuesto_id":null}`, tk), http.StatusUnprocessableEntity, "impuesto_requerido")
	r = patch(t, url, `{"afectacion_igv":"20","impuesto_id":null}`, tk)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &e)
	if e.AfectacionIGV != "20" || e.ImpuestoID != nil {
		t.Fatalf("PATCH vaciar impuesto: %+v", e)
	}
	// Tipo/artículo: REVENTA sin artículo, artículo en un no REVENTA, y artículo inexistente.
	esperar(t, patch(t, url, `{"tipo":"REVENTA"}`, tk), http.StatusUnprocessableEntity, "articulo_requerido")
	esperar(t, patch(t, url, `{"articulo_id":"7"}`, tk), http.StatusUnprocessableEntity, "articulo_no_permitido")
	esperar(t, patch(t, url, `{"tipo":"REVENTA","articulo_id":"999999999"}`, tk), http.StatusUnprocessableEntity, "referencia_invalida")
	esperar(t, patch(t, url, `{"articulo_id":null}`, tk), http.StatusOK, "") // ya era null: sin conflicto

	// Validaciones del PATCH y duplicado.
	esperar(t, patch(t, url, `{}`, tk), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, patch(t, url, `{"nombre":null}`, tk), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, patch(t, url, `{"codigo":"A B"}`, tk), http.StatusUnprocessableEntity, "codigo_invalido")
	esperar(t, patch(t, url, `{"nombre":"  "}`, tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, patch(t, url, `{"precio_venta":"0.001"}`, tk), http.StatusUnprocessableEntity, "precio_invalido")
	esperar(t, patch(t, url, `{"factor_consumo":"0"}`, tk), http.StatusUnprocessableEntity, "factor_consumo_invalido")
	esperar(t, patch(t, url, `{"tipo":"X"}`, tk), http.StatusUnprocessableEntity, "tipo_invalido")
	esperar(t, patch(t, url, `{"afectacion_igv":"99"}`, tk), http.StatusUnprocessableEntity, "afectacion_invalida")
	esperar(t, patch(t, url, `{"imagen_url":"ftp://x"}`, tk), http.StatusUnprocessableEntity, "imagen_url_invalido")
	esperar(t, patch(t, url, `{"descripcion":"`+strings.Repeat("d", 2001)+`"}`, tk), http.StatusUnprocessableEntity, "descripcion_invalido")
	esperar(t, patch(t, url, `{"id":"9"}`, tk), http.StatusBadRequest, "")
	esperar(t, patch(t, url, `{"campo_raro":1}`, tk), http.StatusBadRequest, "")
	otro := crearProducto(t, tk, cuerpoProd(cat, codigoProd()))
	esperar(t, patch(t, "/api/v1/productos/"+otro.ID, fmt.Sprintf(`{"codigo":"%s"}`, nuevoCod), tk), http.StatusConflict, "codigo_duplicado")
	// Categoría inactiva o inexistente en el PATCH.
	inactiva := categoriaPropia(t, tk)
	esperar(t, post(t, "/api/v1/categorias/"+inactiva+"/baja", ``, tk), http.StatusOK, "")
	esperar(t, patch(t, url, fmt.Sprintf(`{"categoria_id":"%s"}`, inactiva), tk), http.StatusUnprocessableEntity, "referencia_invalida")
	esperar(t, patch(t, url, `{"categoria_id":"999999999"}`, tk), http.StatusUnprocessableEntity, "referencia_invalida")
	esperar(t, patch(t, "/api/v1/productos/999999999", `{"nombre":"X"}`, tk), http.StatusNotFound, "no_encontrado")

	// Un producto dado de baja no se edita.
	esperar(t, post(t, url+"/baja", `{"motivo":"Fuera de carta"}`, tk), http.StatusOK, "")
	esperar(t, patch(t, url, `{"nombre":"No"}`, tk), http.StatusConflict, "producto_inactivo")
}

func TestProductoBajaYRestauracion(t *testing.T) {
	tk := gestorCarta(t)
	cat := categoriaPropia(t, tk)
	p := crearProducto(t, tk, cuerpoProd(cat, codigoProd()))
	baja := "/api/v1/productos/" + p.ID + "/baja"
	rest := "/api/v1/productos/" + p.ID + "/restauracion"

	esperar(t, post(t, rest, ``, tk), http.StatusConflict, "ya_activa")
	// El motivo es obligatorio (RF-030).
	esperar(t, post(t, baja, `{}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	esperar(t, post(t, baja, `{"motivo":"   "}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	esperar(t, post(t, baja, `{"motivo":"`+strings.Repeat("m", 2001)+`"}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	esperar(t, post(t, baja, ``, tk), http.StatusBadRequest, "")
	esperar(t, post(t, baja, `{"motivo":"x","otro":1}`, tk), http.StatusBadRequest, "")
	esperar(t, call(t, "POST", baja, `{"motivo":"x"}`, tk), http.StatusBadRequest, "idempotency_key_requerida")
	esperar(t, post(t, "/api/v1/productos/999999999/baja", `{"motivo":"x"}`, tk), http.StatusNotFound, "no_encontrado")

	// Baja idempotente con motivo; registra quién, cuándo y por qué.
	clave := map[string]string{"Idempotency-Key": nuevaClave()}
	r := callH(t, "POST", baja, `{"motivo":"  Sin stock permanente  "}`, tk, clave)
	esperar(t, r, http.StatusOK, "")
	var b productoResp
	r.json(t, &b)
	if b.ID != p.ID || b.EsActivo || b.MotivoEliminacion == nil || *b.MotivoEliminacion != "Sin stock permanente" ||
		b.EliminadoPor == nil || *b.EliminadoPor != b.ActualizadoPor || b.EliminadoEn == nil {
		t.Fatalf("baja: %+v", b)
	}
	esperar(t, callH(t, "POST", baja, `{"motivo":"  Sin stock permanente  "}`, tk, clave), http.StatusOK, "")
	esperar(t, post(t, baja, `{"motivo":"otra vez"}`, tk), http.StatusConflict, "ya_dada_de_baja")
	// Un producto dado de baja sigue visible por id.
	esperar(t, call(t, "GET", "/api/v1/productos/"+p.ID, "", tk), http.StatusOK, "")

	// Restauración: mismo id, marcadores limpios; admite cuerpo vacío o {}.
	r = post(t, rest, `{}`, tk)
	esperar(t, r, http.StatusOK, "")
	var rs productoResp
	r.json(t, &rs)
	if rs.ID != p.ID || !rs.EsActivo || rs.MotivoEliminacion != nil || rs.EliminadoEn != nil || rs.EliminadoPor != nil || rs.Codigo != p.Codigo {
		t.Fatalf("restauración: %+v", rs)
	}
	esperar(t, post(t, rest, ``, tk), http.StatusConflict, "ya_activa")
	esperar(t, post(t, "/api/v1/productos/999999999/restauracion", ``, tk), http.StatusNotFound, "no_encontrado")

	// Restaurar con la categoría dada de baja: 409 referencia_inactiva con mensaje legible; el
	// producto sigue de baja. Al restaurar la categoría, ya se puede.
	esperar(t, post(t, baja, `{"motivo":"Otra baja"}`, tk), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/categorias/"+cat+"/baja", ``, tk), http.StatusOK, "")
	r = post(t, rest, ``, tk)
	esperar(t, r, http.StatusConflict, "referencia_inactiva")
	if !strings.Contains(r.Body.String(), "categoría") || strings.Contains(r.Body.String(), "categoria_id=") {
		t.Fatalf("mensaje no curado: %s", r.Body.String())
	}
	if scalar(t, `SELECT count(*) FROM sch_restaurante.productos WHERE id=`+p.ID+` AND NOT es_activo`) != 1 {
		t.Fatal("el producto debe seguir de baja")
	}
	esperar(t, post(t, "/api/v1/categorias/"+cat+"/restauracion", ``, tk), http.StatusOK, "")
	esperar(t, post(t, rest, ``, tk), http.StatusOK, "")

	// Impuesto dado de baja: igual.
	imp := impuestoPropio(t, true)
	g := crearProducto(t, tk, cuerpoProd(cat, codigoProd(), `"afectacion_igv":"10"`, `"impuesto_id":"`+imp+`"`))
	esperar(t, post(t, "/api/v1/productos/"+g.ID+"/baja", `{"motivo":"Prueba"}`, tk), http.StatusOK, "")
	asSystem(t, `UPDATE sch_restaurante.impuestos SET es_activo=false WHERE id=`+imp)
	r = post(t, "/api/v1/productos/"+g.ID+"/restauracion", ``, tk)
	esperar(t, r, http.StatusConflict, "referencia_inactiva")
	if !strings.Contains(r.Body.String(), "impuesto") {
		t.Fatalf("mensaje no curado: %s", r.Body.String())
	}
}

func TestProductoBajaDeCategoriaConProductoActivoPorAPI(t *testing.T) {
	tk := gestorCarta(t)
	cat := categoriaPropia(t, tk)
	p := crearProducto(t, tk, cuerpoProd(cat, codigoProd()))

	esperar(t, post(t, "/api/v1/categorias/"+cat+"/baja", ``, tk), http.StatusConflict, "categoria_con_productos_activos")
	esperar(t, post(t, "/api/v1/productos/"+p.ID+"/baja", `{"motivo":"Retirado"}`, tk), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/categorias/"+cat+"/baja", ``, tk), http.StatusOK, "")
}

func TestProductosListadoFiltrosYPaginacion(t *testing.T) {
	tk := gestorCarta(t)
	cat := categoriaPropia(t, tk)
	otraCat := categoriaPropia(t, tk)
	marca := "Zq" + strings.ToLower(sufijo())
	var ids []string
	for i := 0; i < 5; i++ {
		c := cat
		if i == 4 {
			c = otraCat
		}
		x := crearProducto(t, tk, fmt.Sprintf(`{"categoria_id":"%s","codigo":"%s-%d","nombre":"%s plato %d","tipo":"PREPARADO","precio_venta":"1","afectacion_igv":"20"}`,
			c, strings.ToUpper(marca), i, marca, i))
		ids = append(ids, x.ID)
	}
	// ids[1] dado de baja.
	esperar(t, post(t, "/api/v1/productos/"+ids[1]+"/baja", `{"motivo":"Prueba"}`, tk), http.StatusOK, "")

	// Por defecto solo activos; es_activo=false las bajas; "todos" (extensión) ambos.
	pag := listarProductos(t, "q="+marca, tk)
	if want := []string{ids[0], ids[2], ids[3], ids[4]}; !slices.Equal(pag.ids(), want) {
		t.Fatalf("activos por defecto: %v, quiero %v", pag.ids(), want)
	}
	if got := listarProductos(t, "q="+marca+"&es_activo=false", tk).ids(); !slices.Equal(got, []string{ids[1]}) {
		t.Fatalf("es_activo=false: %v", got)
	}
	if got := listarProductos(t, "q="+marca+"&es_activo=true", tk).ids(); len(got) != 4 {
		t.Fatalf("es_activo=true: %v", got)
	}
	if got := listarProductos(t, "q="+marca+"&es_activo=todos", tk).ids(); !slices.Equal(got, ids) {
		t.Fatalf("es_activo=todos: %v, quiero %v", got, ids)
	}
	// Categoría.
	if got := listarProductos(t, "q="+marca+"&es_activo=todos&categoria_id="+otraCat, tk).ids(); !slices.Equal(got, []string{ids[4]}) {
		t.Fatalf("categoria_id: %v", got)
	}
	// Búsqueda por nombre (contiene, sin distinguir mayúsculas) y por prefijo de código.
	if got := listarProductos(t, "es_activo=todos&q="+strings.ToUpper(marca)+"+plato+3", tk).ids(); !slices.Equal(got, []string{ids[3]}) {
		t.Fatalf("q por nombre: %v", got)
	}
	if got := listarProductos(t, "es_activo=todos&q="+strings.ToLower(marca)+"-2", tk).ids(); !slices.Equal(got, []string{ids[2]}) {
		t.Fatalf("q por prefijo de código: %v", got)
	}
	// El código solo se busca por prefijo (no por contenido); los comodines se escapan.
	if got := listarProductos(t, "es_activo=todos&q=-2", tk).ids(); slices.Contains(got, ids[2]) {
		t.Fatalf("el código no debe buscarse por contenido: %v", got)
	}
	if got := listarProductos(t, "es_activo=todos&q=%25", tk).ids(); len(got) != 0 {
		t.Fatalf("q=%% debe escaparse: %v", got)
	}
	if got := listarProductos(t, "es_activo=todos&q=_", tk).ids(); len(got) != 0 {
		t.Fatalf("q=_ debe escaparse: %v", got)
	}

	// Paginación por cursor.
	p1 := listarProductos(t, "q="+marca+"&es_activo=todos&limit=2", tk)
	if !slices.Equal(p1.ids(), ids[:2]) || p1.NextCursor == nil {
		t.Fatalf("página 1: %v cursor=%v", p1.ids(), p1.NextCursor)
	}
	p2 := listarProductos(t, "q="+marca+"&es_activo=todos&limit=2&cursor="+*p1.NextCursor, tk)
	if !slices.Equal(p2.ids(), ids[2:4]) || p2.NextCursor == nil {
		t.Fatalf("página 2: %v", p2.ids())
	}
	p3 := listarProductos(t, "q="+marca+"&es_activo=todos&limit=2&cursor="+*p2.NextCursor, tk)
	if !slices.Equal(p3.ids(), ids[4:]) || p3.NextCursor != nil {
		t.Fatalf("página 3: %v cursor=%v", p3.ids(), p3.NextCursor)
	}

	// Parámetros inválidos.
	for _, q := range []string{"limit=0", "limit=101", "limit=x", "es_activo=quizas", "categoria_id=abc", "cursor=!!!"} {
		esperar(t, call(t, "GET", "/api/v1/productos?"+q, "", tk), http.StatusBadRequest, "")
	}
}
