package integration

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

type impuestoResp struct {
	ID             string  `json:"id"`
	Codigo         string  `json:"codigo"`
	Nombre         string  `json:"nombre"`
	Descripcion    *string `json:"descripcion"`
	EsActivo       bool    `json:"es_activo"`
	CreadoPor      string  `json:"creado_por"`
	ActualizadoPor string  `json:"actualizado_por"`
}

type impuestosPagina struct {
	Items      []impuestoResp `json:"items"`
	NextCursor *string        `json:"nextCursor"`
}

type tasaResp struct {
	ID             string  `json:"id"`
	ImpuestoID     string  `json:"impuesto_id"`
	Tasa           string  `json:"tasa"`
	VigenteDesde   string  `json:"vigente_desde"`
	VigenteHasta   *string `json:"vigente_hasta"`
	Fundamento     *string `json:"fundamento"`
	CreadoEn       string  `json:"creado_en"`
	CreadoPor      string  `json:"creado_por"`
	ActualizadoEn  string  `json:"actualizado_en"`
	ActualizadoPor string  `json:"actualizado_por"`
}

type tasasPagina struct {
	Items      []tasaResp `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}

func (p tasasPagina) ids() []string {
	out := make([]string, 0, len(p.Items))
	for _, t := range p.Items {
		out = append(out, t.ID)
	}
	return out
}

// impuestoPropio crea (con el actor sistema) un impuesto con código único y devuelve su id: cada
// prueba de tasas usa el suyo para que las vigencias no choquen entre pruebas.
func impuestoPropio(t *testing.T, activo bool) string {
	t.Helper()
	codigo := "T" + sufijo()
	asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.impuestos(codigo, nombre, descripcion) VALUES ('%s', 'Impuesto %s', 'Prueba')`, codigo, codigo))
	if !activo {
		asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.impuestos SET es_activo = false WHERE codigo = '%s'`, codigo))
	}
	return idDe(t, "impuestos", "codigo", codigo)
}

// adminImpuestos devuelve el token de un usuario con IMPUESTO_ADMINISTRAR.
func adminImpuestos(t *testing.T) string {
	t.Helper()
	rolConPermisos(t, "T_IMP_ADMIN", "IMPUESTO_ADMINISTRAR")
	asegurarUsuario(t, "imp.admin", "ImpAdmin2026Clave", "T_IMP_ADMIN", "PRINCIPAL")
	return token(t, "imp.admin", "ImpAdmin2026Clave")
}

func cuerpoTasa(impuesto, tasa, desde, hasta string) string {
	h := "null"
	if hasta != "" {
		h = `"` + hasta + `"`
	}
	return fmt.Sprintf(`{"impuesto_id":"%s","tasa":"%s","vigente_desde":"%s","vigente_hasta":%s}`, impuesto, tasa, desde, h)
}

func crearTasa(t *testing.T, tk, impuesto, tasa, desde, hasta string) tasaResp {
	t.Helper()
	r := post(t, "/api/v1/tasas-impuesto", cuerpoTasa(impuesto, tasa, desde, hasta), tk)
	esperar(t, r, http.StatusCreated, "")
	var v tasaResp
	r.json(t, &v)
	return v
}

func TestTasasPermisos(t *testing.T) {
	esperar(t, call(t, "GET", "/api/v1/impuestos", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, call(t, "GET", "/api/v1/impuestos/1", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, call(t, "GET", "/api/v1/tasas-impuesto", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, call(t, "GET", "/api/v1/tasas-impuesto/1", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, post(t, "/api/v1/tasas-impuesto", `{}`, ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/1", `{"tasa":"1"}`, ""), http.StatusUnauthorized, "no_autenticado")

	imp := impuestoPropio(t, true)
	admin := adminImpuestos(t)
	tasa := crearTasa(t, admin, imp, "18", "2030-01-01", "")

	// Sin permisos de impuestos ni de carta: ni lee ni escribe.
	rolConPermisos(t, "T_IMP_NADA", "CLIENTE_VER")
	asegurarUsuario(t, "imp.nada", "ImpNada2026Clave", "T_IMP_NADA", "PRINCIPAL")
	nada := token(t, "imp.nada", "ImpNada2026Clave")
	for _, ruta := range []string{"/api/v1/impuestos", "/api/v1/impuestos/" + imp, "/api/v1/tasas-impuesto", "/api/v1/tasas-impuesto/" + tasa.ID} {
		esperar(t, call(t, "GET", ruta, "", nada), http.StatusForbidden, "sin_permiso")
	}
	esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(imp, "5", "2040-01-01", ""), nada), http.StatusForbidden, "sin_permiso")
	// Un impuesto_id mal formado no se evalúa antes del permiso: 403 primero.
	for _, malo := range []string{"abc", "+5", "007"} {
		esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(malo, "5", "2040-01-01", ""), nada), http.StatusForbidden, "sin_permiso")
		esperar(t, call(t, "GET", "/api/v1/tasas-impuesto?impuesto_id="+malo, "", nada), http.StatusForbidden, "sin_permiso")
	}

	// MESERO no tiene permisos de impuestos (solo CARTA_VER): no lee.
	asegurarUsuario(t, "imp.mesero", "ImpMesero2026Clave", "MESERO", "PRINCIPAL")
	mesero := token(t, "imp.mesero", "ImpMesero2026Clave")
	esperar(t, call(t, "GET", "/api/v1/impuestos", "", mesero), http.StatusForbidden, "sin_permiso")

	// IMPUESTO_VER lee pero no escribe.
	rolConPermisos(t, "T_IMP_VER", "IMPUESTO_VER")
	asegurarUsuario(t, "imp.ver", "ImpVer2026Clave", "T_IMP_VER", "PRINCIPAL")
	ver := token(t, "imp.ver", "ImpVer2026Clave")
	esperar(t, call(t, "GET", "/api/v1/impuestos"+conLimite(""), "", ver), http.StatusOK, "")
	esperar(t, call(t, "GET", "/api/v1/impuestos/"+imp, "", ver), http.StatusOK, "")
	esperar(t, call(t, "GET", "/api/v1/tasas-impuesto"+conLimite(""), "", ver), http.StatusOK, "")
	esperar(t, call(t, "GET", "/api/v1/tasas-impuesto/"+tasa.ID, "", ver), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(imp, "5", "2040-01-01", ""), ver), http.StatusForbidden, "sin_permiso")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"tasa":"1"}`, ver), http.StatusForbidden, "sin_permiso")

	// CARTA_ADMINISTRAR lee impuestos y tasas (formulario de productos), pero no escribe tasas.
	carta := gestorCarta(t)
	esperar(t, call(t, "GET", "/api/v1/impuestos"+conLimite(""), "", carta), http.StatusOK, "")
	esperar(t, call(t, "GET", "/api/v1/impuestos/"+imp, "", carta), http.StatusOK, "")
	esperar(t, call(t, "GET", "/api/v1/tasas-impuesto"+conLimite(""), "", carta), http.StatusOK, "")
	esperar(t, call(t, "GET", "/api/v1/tasas-impuesto/"+tasa.ID, "", carta), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(imp, "5", "2040-01-01", ""), carta), http.StatusForbidden, "sin_permiso")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"tasa":"1"}`, carta), http.StatusForbidden, "sin_permiso")

	// IMPUESTO_ADMINISTRAR escribe y también lee.
	esperar(t, call(t, "GET", "/api/v1/impuestos/"+imp, "", admin), http.StatusOK, "")
}

func TestImpuestosLectura(t *testing.T) {
	admin := adminImpuestos(t)
	activo := impuestoPropio(t, true)
	inactivo := impuestoPropio(t, false)

	// IGV_TOTAL viene sembrado por la migración 00002.
	var igv impuestoResp
	r := call(t, "GET", "/api/v1/impuestos/"+idDe(t, "impuestos", "codigo", "IGV_TOTAL"), "", admin)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &igv)
	if igv.Codigo != "IGV_TOTAL" || igv.Nombre != "IGV e IPM" || igv.Descripcion == nil || !igv.EsActivo || igv.CreadoPor == "" {
		t.Fatalf("IGV_TOTAL: %+v", igv)
	}
	esperar(t, call(t, "GET", "/api/v1/impuestos/999999999", "", admin), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/impuestos/abc", "", admin), http.StatusBadRequest, "id_invalido")

	lista := func(consulta string) impuestosPagina {
		var p impuestosPagina
		r := call(t, "GET", "/api/v1/impuestos"+conLimite(consulta), "", admin)
		esperar(t, r, http.StatusOK, "")
		r.json(t, &p)
		return p
	}
	ids := func(p impuestosPagina) []string {
		var out []string
		for _, i := range p.Items {
			out = append(out, i.ID)
		}
		return out
	}
	todos := ids(lista(""))
	if !slices.Contains(todos, activo) || !slices.Contains(todos, inactivo) || !slices.Contains(todos, igv.ID) {
		t.Fatalf("listado: %v", todos)
	}
	if !slices.IsSorted(todos) && len(todos) > 1 {
		// Los ids son cadenas: se compara numéricamente por longitud y luego lexicográficamente.
		ord := slices.Clone(todos)
		slices.SortFunc(ord, func(a, b string) int {
			if len(a) != len(b) {
				return len(a) - len(b)
			}
			return strings.Compare(a, b)
		})
		if !slices.Equal(ord, todos) {
			t.Fatalf("orden por id: %v", todos)
		}
	}
	if got := ids(lista("?es_activo=false")); !slices.Contains(got, inactivo) || slices.Contains(got, activo) {
		t.Fatalf("es_activo=false: %v", got)
	}
	if got := ids(lista("?es_activo=true")); !slices.Contains(got, activo) || slices.Contains(got, inactivo) {
		t.Fatalf("es_activo=true: %v", got)
	}
	// Paginación por cursor sin repetir.
	p1 := lista("?limit=1")
	if len(p1.Items) != 1 || p1.NextCursor == nil {
		t.Fatalf("página 1: %+v", p1)
	}
	p2 := lista("?limit=1&cursor=" + *p1.NextCursor)
	if len(p2.Items) != 1 || p2.Items[0].ID == p1.Items[0].ID {
		t.Fatalf("página 2: %+v", p2)
	}
	esperar(t, call(t, "GET", "/api/v1/impuestos?es_activo=quizas", "", admin), http.StatusBadRequest, "es_activo_invalido")
	esperar(t, call(t, "GET", "/api/v1/impuestos?cursor=!!!", "", admin), http.StatusBadRequest, "cursor_invalido")
}

func TestTasaAltaIdempotenciaYValidaciones(t *testing.T) {
	tk := adminImpuestos(t)
	imp := impuestoPropio(t, true)
	cuerpo := fmt.Sprintf(`{"impuesto_id":"%s","tasa":"18","vigente_desde":"2031-01-01","vigente_hasta":"2032-01-01","fundamento":"Línea uno\nlínea dos"}`, imp)

	clave := map[string]string{"Idempotency-Key": nuevaClave()}
	r := callH(t, "POST", "/api/v1/tasas-impuesto", cuerpo, tk, clave)
	esperar(t, r, http.StatusCreated, "")
	var v tasaResp
	r.json(t, &v)
	// La tasa se entrega siempre con 4 decimales (numeric(7,4)); las fechas, como AAAA-MM-DD.
	if v.ImpuestoID != imp || v.Tasa != "18.0000" || v.VigenteDesde != "2031-01-01" || v.VigenteHasta == nil || *v.VigenteHasta != "2032-01-01" ||
		v.Fundamento == nil || *v.Fundamento != "Línea uno\nlínea dos" || v.CreadoEn == "" ||
		v.CreadoPor != idDe(t, "usuarios", "username", "imp.admin") || v.ActualizadoPor != v.CreadoPor {
		t.Fatalf("tasa creada: %+v", v)
	}
	rep := callH(t, "POST", "/api/v1/tasas-impuesto", cuerpo, tk, clave)
	esperar(t, rep, http.StatusCreated, "")
	var v2 tasaResp
	rep.json(t, &v2)
	if v2.ID != v.ID || scalar(t, `SELECT count(*) FROM sch_restaurante.tasas_impuesto WHERE impuesto_id=`+imp) != 1 {
		t.Fatalf("la repetición creó otra tasa: %s vs %s", v2.ID, v.ID)
	}
	esperar(t, callH(t, "POST", "/api/v1/tasas-impuesto", cuerpoTasa(imp, "5", "2040-01-01", ""), tk, clave), http.StatusConflict, "idempotency_key_reutilizada")
	esperar(t, call(t, "POST", "/api/v1/tasas-impuesto", cuerpoTasa(imp, "5", "2040-01-01", ""), tk), http.StatusBadRequest, "idempotency_key_requerida")

	// Vigencia abierta, sin fundamento, y decimales aceptados (se normalizan a 4 decimales).
	abierta := crearTasa(t, tk, imp, "10.5", "2033-01-01", "")
	if abierta.Tasa != "10.5000" || abierta.VigenteHasta != nil || abierta.Fundamento != nil {
		t.Fatalf("vigencia abierta: %+v", abierta)
	}
	// Cada prueba de rango en su propio impuesto para que no se solapen entre sí.
	validas := map[string]string{"0": "0.0000", "100": "100.0000", "100.0000": "100.0000", "17.9999": "17.9999", "0.0001": "0.0001", "007.5": "7.5000", "18.": ""}
	for entrada, esperado := range validas {
		otro := impuestoPropio(t, true)
		if esperado == "" {
			esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(otro, entrada, "2034-01-01", ""), tk), http.StatusUnprocessableEntity, "tasa_invalida")
			continue
		}
		if got := crearTasa(t, tk, otro, entrada, "2034-01-01", ""); got.Tasa != esperado {
			t.Errorf("tasa %q: quería %q, obtuve %q", entrada, esperado, got.Tasa)
		}
	}
	otro := impuestoPropio(t, true)
	for _, mala := range []string{"100.0001", "101", "1000", "-1", "-0.5", "1.12345", "abc", "", " 18", "18 ", "1,5", ".5", "1e2", "+5", "NaN"} {
		esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(otro, mala, "2035-01-01", ""), tk), http.StatusUnprocessableEntity, "tasa_invalida")
	}
	// La tasa debe ser cadena y no número JSON; un campo desconocido responde 400.
	esperar(t, post(t, "/api/v1/tasas-impuesto", fmt.Sprintf(`{"impuesto_id":"%s","tasa":18,"vigente_desde":"2035-01-01"}`, otro), tk), http.StatusBadRequest, "")
	esperar(t, post(t, "/api/v1/tasas-impuesto", fmt.Sprintf(`{"impuesto_id":"%s","tasa":"18","vigente_desde":"2035-01-01","es_activo":true}`, otro), tk), http.StatusBadRequest, "")

	// Fechas.
	for _, mala := range []string{"", "2035-13-01", "2035-02-30", "01/01/2035", "2035-1-1", "2035-01-01T00:00:00Z", "0000-01-01", "1800-01-01", "3000-01-01", "mañana"} {
		esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(otro, "18", mala, ""), tk), http.StatusUnprocessableEntity, "vigente_desde_invalido")
	}
	esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(otro, "18", "2035-01-01", "2035-13-45"), tk), http.StatusUnprocessableEntity, "vigente_hasta_invalido")
	// hasta debe ser estrictamente posterior a desde.
	esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(otro, "18", "2035-01-01", "2035-01-01"), tk), http.StatusUnprocessableEntity, "vigencia_invalida")
	esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(otro, "18", "2035-01-01", "2034-12-31"), tk), http.StatusUnprocessableEntity, "vigencia_invalida")
	esperar(t, post(t, "/api/v1/tasas-impuesto", fmt.Sprintf(`{"impuesto_id":"%s","tasa":"18"}`, otro), tk), http.StatusUnprocessableEntity, "vigente_desde_invalido")

	// Fundamento: hasta 2000 caracteres; en blanco se guarda nulo.
	esperar(t, post(t, "/api/v1/tasas-impuesto", fmt.Sprintf(`{"impuesto_id":"%s","tasa":"18","vigente_desde":"2036-01-01","fundamento":"%s"}`, otro, strings.Repeat("f", 2001)), tk), http.StatusUnprocessableEntity, "fundamento_invalido")
	blanco := post(t, "/api/v1/tasas-impuesto", fmt.Sprintf(`{"impuesto_id":"%s","tasa":"18","vigente_desde":"2036-01-01","vigente_hasta":"2036-06-01","fundamento":"   "}`, otro), tk)
	esperar(t, blanco, http.StatusCreated, "")
	var vb tasaResp
	blanco.json(t, &vb)
	if vb.Fundamento != nil {
		t.Fatalf("fundamento en blanco: %+v", vb)
	}
	esperar(t, post(t, "/api/v1/tasas-impuesto", fmt.Sprintf(`{"impuesto_id":"%s","tasa":"18","vigente_desde":"2037-01-01","fundamento":"%s"}`, otro, strings.Repeat("f", 2000)), tk), http.StatusCreated, "")

	// Impuesto: obligatorio, existente y activo.
	esperar(t, post(t, "/api/v1/tasas-impuesto", `{"tasa":"18","vigente_desde":"2035-01-01"}`, tk), http.StatusUnprocessableEntity, "impuesto_id_invalido")
	for _, malo := range []string{"abc", "+5", "007", "0", "-1"} {
		esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(malo, "18", "2035-01-01", ""), tk), http.StatusBadRequest, "impuesto_id_invalido")
		esperar(t, call(t, "GET", "/api/v1/tasas-impuesto"+conLimite("?impuesto_id="+malo), "", tk), http.StatusBadRequest, "impuesto_id_invalido")
	}
	esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa("999999999", "18", "2035-01-01", ""), tk), http.StatusUnprocessableEntity, "referencia_invalida")
	inactivo := impuestoPropio(t, false)
	esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(inactivo, "18", "2035-01-01", ""), tk), http.StatusUnprocessableEntity, "referencia_invalida")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.tasas_impuesto WHERE impuesto_id=`+inactivo) != 0 {
		t.Fatal("no debe crearse una tasa para un impuesto inactivo")
	}
}

func TestTasaSolapeYVigenciasContiguas(t *testing.T) {
	tk := adminImpuestos(t)
	imp := impuestoPropio(t, true)

	base := crearTasa(t, tk, imp, "18", "2040-01-01", "2040-07-01") // [01-01, 07-01)
	// Alta que se solapa: parcial, contenida y abierta.
	for _, c := range [][2]string{{"2040-06-30", "2040-09-01"}, {"2040-02-01", "2040-03-01"}, {"2039-12-01", "2040-01-02"}, {"2039-01-01", ""}} {
		esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(imp, "10", c[0], c[1]), tk), http.StatusConflict, "vigencia_solapada")
	}
	// Contiguas [a,b)+[b,c): válidas por ambos lados.
	siguiente := crearTasa(t, tk, imp, "10", "2040-07-01", "2040-12-31")
	anterior := crearTasa(t, tk, imp, "5", "2039-06-01", "2040-01-01")
	if siguiente.VigenteDesde != "2040-07-01" || anterior.VigenteHasta == nil || *anterior.VigenteHasta != "2040-01-01" {
		t.Fatalf("contiguas: %+v %+v", siguiente, anterior)
	}
	// Una vigencia abierta posterior también es válida y bloquea cualquier alta posterior.
	abierta := crearTasa(t, tk, imp, "12", "2041-01-01", "")
	esperar(t, post(t, "/api/v1/tasas-impuesto", cuerpoTasa(imp, "1", "2050-01-01", ""), tk), http.StatusConflict, "vigencia_solapada")

	// Dos impuestos distintos con la misma vigencia: válido.
	otro := impuestoPropio(t, true)
	paralela := crearTasa(t, tk, otro, "18", "2040-01-01", "2040-07-01")
	if paralela.ImpuestoID != otro {
		t.Fatalf("paralela: %+v", paralela)
	}

	// Solape en edición: ampliar la tasa base sobre la siguiente, adelantar el desde de la
	// siguiente sobre la base, o cerrar la abierta... y cerrar la abierta hacia atrás.
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+base.ID, `{"vigente_hasta":"2040-08-01"}`, tk), http.StatusConflict, "vigencia_solapada")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+base.ID, `{"vigente_hasta":null}`, tk), http.StatusConflict, "vigencia_solapada")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+siguiente.ID, `{"vigente_desde":"2040-06-30"}`, tk), http.StatusConflict, "vigencia_solapada")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+abierta.ID, `{"vigente_desde":"2040-12-30"}`, tk), http.StatusConflict, "vigencia_solapada")
	// Editar sin cambiar la vigencia nunca se solapa consigo misma.
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+base.ID, `{"tasa":"19"}`, tk), http.StatusOK, "")
	// Mover una vigencia a un hueco libre es válido (contigua por el otro lado).
	r := patch(t, "/api/v1/tasas-impuesto/"+siguiente.ID, `{"vigente_hasta":"2040-12-15"}`, tk)
	esperar(t, r, http.StatusOK, "")
	var ed tasaResp
	r.json(t, &ed)
	if ed.VigenteHasta == nil || *ed.VigenteHasta != "2040-12-15" || ed.VigenteDesde != "2040-07-01" {
		t.Fatalf("PATCH vigencia: %+v", ed)
	}
	// Un rechazo por solape no deja rastro.
	if got := scalarStr(t, `SELECT vigente_hasta::text FROM sch_restaurante.tasas_impuesto WHERE id=`+base.ID); got != "2040-07-01" {
		t.Fatalf("la base cambió tras el rechazo: %s", got)
	}
}

func TestTasaLecturaYEdicion(t *testing.T) {
	tk := adminImpuestos(t)
	imp := impuestoPropio(t, true)
	tasa := crearTasa(t, tk, imp, "18", "2050-01-01", "2051-01-01")

	var leida tasaResp
	g := call(t, "GET", "/api/v1/tasas-impuesto/"+tasa.ID, "", tk)
	esperar(t, g, http.StatusOK, "")
	g.json(t, &leida)
	if leida.ID != tasa.ID || leida.Tasa != "18.0000" || leida.VigenteDesde != "2050-01-01" {
		t.Fatalf("lectura: %+v", leida)
	}
	esperar(t, call(t, "GET", "/api/v1/tasas-impuesto/999999999", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/tasas-impuesto/abc", "", tk), http.StatusBadRequest, "id_invalido")

	// PATCH parcial: solo la tasa; lo demás queda igual.
	r := patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"tasa":"10.25"}`, tk)
	esperar(t, r, http.StatusOK, "")
	var ed tasaResp
	r.json(t, &ed)
	if ed.Tasa != "10.2500" || ed.VigenteDesde != "2050-01-01" || ed.VigenteHasta == nil || *ed.VigenteHasta != "2051-01-01" || ed.ImpuestoID != imp {
		t.Fatalf("PATCH tasa: %+v", ed)
	}
	// Fundamento nuevo; luego vaciado con null.
	r = patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"fundamento":"Ley\nvigente"}`, tk)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &ed)
	if ed.Fundamento == nil || *ed.Fundamento != "Ley\nvigente" {
		t.Fatalf("PATCH fundamento: %+v", ed)
	}
	r = patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"fundamento":null}`, tk)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &ed)
	if ed.Fundamento != nil {
		t.Fatalf("PATCH fundamento null: %+v", ed)
	}
	// vigente_hasta null deja la vigencia abierta; luego se vuelve a cerrar.
	r = patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"vigente_hasta":null}`, tk)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &ed)
	if ed.VigenteHasta != nil {
		t.Fatalf("PATCH vigente_hasta null: %+v", ed)
	}
	r = patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"vigente_desde":"2050-02-01","vigente_hasta":"2050-03-01"}`, tk)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &ed)
	if ed.VigenteDesde != "2050-02-01" || ed.VigenteHasta == nil || *ed.VigenteHasta != "2050-03-01" || ed.ActualizadoPor == "" {
		t.Fatalf("PATCH fechas: %+v", ed)
	}

	// Validaciones de la edición.
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{}`, tk), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"tasa":"100.5"}`, tk), http.StatusUnprocessableEntity, "tasa_invalida")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"tasa":"-1"}`, tk), http.StatusUnprocessableEntity, "tasa_invalida")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"tasa":"1.00001"}`, tk), http.StatusUnprocessableEntity, "tasa_invalida")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"tasa":null}`, tk), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"vigente_desde":"2050-1-1"}`, tk), http.StatusUnprocessableEntity, "vigente_desde_invalido")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"vigente_hasta":"hoy"}`, tk), http.StatusUnprocessableEntity, "vigente_hasta_invalido")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"fundamento":"`+strings.Repeat("f", 2001)+`"}`, tk), http.StatusUnprocessableEntity, "fundamento_invalido")
	// El invariante hasta > desde se valida con el valor resultante (la fila tiene [02-01, 03-01)).
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"vigente_desde":"2050-03-01"}`, tk), http.StatusUnprocessableEntity, "vigencia_invalida")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"vigente_hasta":"2050-01-15"}`, tk), http.StatusUnprocessableEntity, "vigencia_invalida")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"vigente_desde":"2050-04-01","vigente_hasta":"2050-04-01"}`, tk), http.StatusUnprocessableEntity, "vigencia_invalida")
	// Desde posterior pero hasta abierta es válido.
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"vigente_desde":"2050-04-01","vigente_hasta":null}`, tk), http.StatusOK, "")
	// impuesto_id no se edita; los campos desconocidos responden 400.
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, fmt.Sprintf(`{"impuesto_id":"%s"}`, imp), tk), http.StatusBadRequest, "")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/"+tasa.ID, `{"campo_raro":1}`, tk), http.StatusBadRequest, "")
	esperar(t, patch(t, "/api/v1/tasas-impuesto/999999999", `{"tasa":"1"}`, tk), http.StatusNotFound, "no_encontrado")

	// No hay baja de tasas ni de impuestos.
	esperar(t, post(t, "/api/v1/tasas-impuesto/"+tasa.ID+"/baja", ``, tk), http.StatusNotFound, "")
	esperar(t, post(t, "/api/v1/impuestos", `{"codigo":"X","nombre":"X"}`, tk), http.StatusMethodNotAllowed, "")
}

func TestTasasListadoFiltrosYPaginacion(t *testing.T) {
	tk := adminImpuestos(t)
	a, b := impuestoPropio(t, true), impuestoPropio(t, true)
	a1 := crearTasa(t, tk, a, "18", "2060-01-01", "2061-01-01")
	b1 := crearTasa(t, tk, b, "18", "2060-01-01", "2061-01-01")
	a2 := crearTasa(t, tk, a, "10", "2061-01-01", "2062-01-01")
	a3 := crearTasa(t, tk, a, "5", "2062-01-01", "")

	lista := func(consulta string) tasasPagina {
		var p tasasPagina
		r := call(t, "GET", "/api/v1/tasas-impuesto"+conLimite(consulta), "", tk)
		esperar(t, r, http.StatusOK, "")
		r.json(t, &p)
		return p
	}
	if got := lista("?impuesto_id=" + a).ids(); !slices.Equal(got, []string{a1.ID, a2.ID, a3.ID}) {
		t.Fatalf("impuesto A (orden por id): %v", got)
	}
	if got := lista("?impuesto_id=" + b).ids(); !slices.Equal(got, []string{b1.ID}) {
		t.Fatalf("impuesto B: %v", got)
	}
	if todas := lista("").ids(); !slices.Contains(todas, a1.ID) && !slices.Contains(todas, b1.ID) && len(todas) == 0 {
		t.Fatalf("sin filtro: %v", todas)
	}
	// Paginación por cursor sin repetir ni saltar.
	p1 := lista("?impuesto_id=" + a + "&limit=2")
	if len(p1.Items) != 2 || p1.NextCursor == nil {
		t.Fatalf("página 1: %+v", p1)
	}
	p2 := lista("?impuesto_id=" + a + "&limit=2&cursor=" + *p1.NextCursor)
	if len(p2.Items) != 1 || p2.NextCursor != nil || p2.Items[0].ID != a3.ID {
		t.Fatalf("página 2: %+v", p2)
	}
	if got := lista("?impuesto_id=999999999").ids(); len(got) != 0 {
		t.Fatalf("impuesto inexistente: %v", got)
	}
	esperar(t, call(t, "GET", "/api/v1/tasas-impuesto?impuesto_id=abc", "", tk), http.StatusBadRequest, "impuesto_id_invalido")
	esperar(t, call(t, "GET", "/api/v1/tasas-impuesto?impuesto_id=0", "", tk), http.StatusBadRequest, "impuesto_id_invalido")
	esperar(t, call(t, "GET", "/api/v1/tasas-impuesto?limit=0", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/tasas-impuesto?limit=101", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/tasas-impuesto?cursor=!!!", "", tk), http.StatusBadRequest, "cursor_invalido")
}
