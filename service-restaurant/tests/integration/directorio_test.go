package integration

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

type clienteResp struct {
	ID              string  `json:"id"`
	TipoDocumento   *string `json:"tipo_documento"`
	NumeroDocumento *string `json:"numero_documento"`
	Nombre          string  `json:"nombre_razon_social"`
	Telefono        *string `json:"telefono"`
	Email           *string `json:"email"`
	Direccion       *string `json:"direccion"`
	EsActivo        bool    `json:"es_activo"`
	Motivo          *string `json:"motivo_eliminacion"`
	EliminadoPor    *string `json:"eliminado_por"`
	CreadoPor       string  `json:"creado_por"`
}

type clientesPagina struct {
	Items      []clienteResp `json:"items"`
	NextCursor *string       `json:"nextCursor"`
}

func (p clientesPagina) ids() []string {
	out := make([]string, 0, len(p.Items))
	for _, c := range p.Items {
		out = append(out, c.ID)
	}
	return out
}

type proveedorResp struct {
	ID              string  `json:"id"`
	TipoDocumento   string  `json:"tipo_documento"`
	NumeroDocumento string  `json:"numero_documento"`
	RazonSocial     string  `json:"razon_social"`
	Contacto        *string `json:"contacto_nombre"`
	Telefono        *string `json:"telefono"`
	Email           *string `json:"email"`
	EsActivo        bool    `json:"es_activo"`
	Motivo          *string `json:"motivo_eliminacion"`
	EliminadoPor    *string `json:"eliminado_por"`
}

type proveedoresPagina struct {
	Items      []proveedorResp `json:"items"`
	NextCursor *string         `json:"nextCursor"`
}

// numeroUnico devuelve n dígitos que no se repiten entre ejecuciones sobre la misma base.
func numeroUnico(n int) string {
	v := time.Now().UnixNano()/1000 + sufijoSeq.Add(1)
	s := fmt.Sprintf("%020d", v)
	return s[len(s)-n:]
}

// gestorDirectorio devuelve el token de un usuario con ver y administrar clientes y proveedores.
func gestorDirectorio(t *testing.T) string {
	t.Helper()
	rolConPermisos(t, "T_DIR_GESTOR", "CLIENTE_VER", "CLIENTE_ADMINISTRAR", "PROVEEDOR_VER", "PROVEEDOR_ADMINISTRAR")
	asegurarUsuario(t, "dir.gestor", "DirGestor2026Clave", "T_DIR_GESTOR", "PRINCIPAL")
	return token(t, "dir.gestor", "DirGestor2026Clave")
}

func patch(t *testing.T, path, body, tk string) resp {
	t.Helper()
	return callH(t, http.MethodPatch, path, body, tk, nil)
}

func TestDirectorioPermisosSembradosEnLaMigracion(t *testing.T) {
	// CAJERO recibe CLIENTE_VER con la migración 00009; no administra ni ve proveedores.
	asegurarUsuario(t, "dir.cajero", "DirCajero2026Clave", "CAJERO", "PRINCIPAL")
	cajero := token(t, "dir.cajero", "DirCajero2026Clave")
	esperar(t, call(t, "GET", "/api/v1/clientes", "", cajero), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/clientes", `{"nombre_razon_social":"No debe crearse"}`, cajero), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/proveedores", "", cajero), http.StatusForbidden, "sin_permiso")

	// ALMACEN administra proveedores y no ve clientes.
	asegurarUsuario(t, "dir.almacen", "DirAlmacen2026Clave", "ALMACEN", "PRINCIPAL")
	almacen := token(t, "dir.almacen", "DirAlmacen2026Clave")
	esperar(t, call(t, "GET", "/api/v1/proveedores", "", almacen), http.StatusOK, "")
	esperar(t, call(t, "GET", "/api/v1/clientes", "", almacen), http.StatusForbidden, "sin_permiso")
	alta := fmt.Sprintf(`{"tipo_documento":"RUC","numero_documento":"20%s","razon_social":"Almacén SAC %s"}`, numeroUnico(9), sufijo())
	esperar(t, post(t, "/api/v1/proveedores", alta, almacen), http.StatusCreated, "")

	esperar(t, call(t, "GET", "/api/v1/clientes", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, call(t, "GET", "/api/v1/proveedores/1", "", ""), http.StatusUnauthorized, "no_autenticado")
}

func TestClienteCicloDeVida(t *testing.T) {
	tk := gestorDirectorio(t)
	dni := numeroUnico(8)
	nombre := "Cliente " + sufijo()
	cuerpo := fmt.Sprintf(`{"tipo_documento":"dni","numero_documento":" %s ","nombre_razon_social":"  %s  ","telefono":"+51 999 888 777","email":"c@ejemplo.pe","direccion":"Av. Lima 123"}`, dni, nombre)

	// Alta: normaliza el documento, recorta el nombre y registra al autor; la repetición idempotente no duplica.
	clave := map[string]string{"Idempotency-Key": nuevaClave()}
	r := callH(t, "POST", "/api/v1/clientes", cuerpo, tk, clave)
	esperar(t, r, http.StatusCreated, "")
	var c clienteResp
	r.json(t, &c)
	if c.Nombre != nombre || c.TipoDocumento == nil || *c.TipoDocumento != "DNI" || c.NumeroDocumento == nil || *c.NumeroDocumento != dni ||
		!c.EsActivo || c.CreadoPor != idDe(t, "usuarios", "username", "dir.gestor") {
		t.Fatalf("cliente creado: %+v", c)
	}
	rep := callH(t, "POST", "/api/v1/clientes", cuerpo, tk, clave)
	esperar(t, rep, http.StatusCreated, "")
	var c2 clienteResp
	rep.json(t, &c2)
	if c2.ID != c.ID {
		t.Fatalf("la repetición creó otro cliente: %s vs %s", c2.ID, c.ID)
	}

	// Lectura.
	var leido clienteResp
	g := call(t, "GET", "/api/v1/clientes/"+c.ID, "", tk)
	esperar(t, g, http.StatusOK, "")
	g.json(t, &leido)
	if leido.ID != c.ID || leido.Nombre != nombre {
		t.Fatalf("lectura: %+v", leido)
	}
	esperar(t, call(t, "GET", "/api/v1/clientes/999999999", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/clientes/abc", "", tk), http.StatusBadRequest, "id_invalido")

	// Validaciones del alta.
	esperar(t, post(t, "/api/v1/clientes", `{"nombre_razon_social":"   "}`, tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, post(t, "/api/v1/clientes", `{"tipo_documento":"DNI","nombre_razon_social":"X"}`, tk), http.StatusUnprocessableEntity, "documento_incompleto")
	esperar(t, post(t, "/api/v1/clientes", `{"tipo_documento":"DNI","numero_documento":"1234567","nombre_razon_social":"X"}`, tk), http.StatusUnprocessableEntity, "documento_invalido")
	esperar(t, post(t, "/api/v1/clientes", `{"nombre_razon_social":"X","email":"no-es-correo"}`, tk), http.StatusUnprocessableEntity, "email_invalido")
	esperar(t, post(t, "/api/v1/clientes", `{"nombre_razon_social":"X","telefono":"abc"}`, tk), http.StatusUnprocessableEntity, "telefono_invalido")
	esperar(t, post(t, "/api/v1/clientes", `{"nombre_razon_social":"X","campo_raro":1}`, tk), http.StatusBadRequest, "")

	// El documento no se repite, ni con otra grafía.
	dup := fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombre_razon_social":"Otro"}`, dni)
	esperar(t, post(t, "/api/v1/clientes", dup, tk), http.StatusConflict, "documento_duplicado")

	// Edición parcial: cambia el nombre, vacía el teléfono (null) y no toca el resto.
	nuevoNombre := nombre + " editado"
	r = patch(t, "/api/v1/clientes/"+c.ID, fmt.Sprintf(`{"nombre_razon_social":"%s","telefono":null}`, nuevoNombre), tk)
	esperar(t, r, http.StatusOK, "")
	var ed clienteResp
	r.json(t, &ed)
	if ed.Nombre != nuevoNombre || ed.Telefono != nil || ed.Email == nil || *ed.Email != "c@ejemplo.pe" {
		t.Fatalf("edición: %+v", ed)
	}
	esperar(t, patch(t, "/api/v1/clientes/"+c.ID, `{}`, tk), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, patch(t, "/api/v1/clientes/"+c.ID, `{"tipo_documento":"DNI"}`, tk), http.StatusUnprocessableEntity, "documento_incompleto")
	r = patch(t, "/api/v1/clientes/"+c.ID, `{"tipo_documento":null,"numero_documento":null}`, tk)
	esperar(t, r, http.StatusOK, "")
	var sinDoc clienteResp
	r.json(t, &sinDoc)
	if sinDoc.TipoDocumento != nil || sinDoc.NumeroDocumento != nil {
		t.Fatalf("documento no se vació: %+v", sinDoc)
	}
	// Se restituye el documento para comprobar que sigue reservado tras la baja.
	esperar(t, patch(t, "/api/v1/clientes/"+c.ID, fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s"}`, dni), tk), http.StatusOK, "")

	// Baja: exige motivo, registra quién y por qué; no se repite ni se edita.
	baja := "/api/v1/clientes/" + c.ID + "/baja"
	esperar(t, post(t, baja, `{"motivo":"   "}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	r = post(t, baja, `{"motivo":"Pidió que lo borren"}`, tk)
	esperar(t, r, http.StatusOK, "")
	var b clienteResp
	r.json(t, &b)
	if b.EsActivo || b.Motivo == nil || *b.Motivo != "Pidió que lo borren" || b.EliminadoPor == nil {
		t.Fatalf("baja: %+v", b)
	}
	esperar(t, post(t, baja, `{"motivo":"otra vez"}`, tk), http.StatusConflict, "ya_dada_de_baja")
	esperar(t, patch(t, "/api/v1/clientes/"+c.ID, `{"nombre_razon_social":"No"}`, tk), http.StatusConflict, "cliente_inactivo")
	// Un documento dado de baja sigue reservado.
	esperar(t, post(t, "/api/v1/clientes", dup, tk), http.StatusConflict, "documento_duplicado")

	// Restauración: misma fila, mismo id, marcadores limpios.
	rest := "/api/v1/clientes/" + c.ID + "/restauracion"
	r = post(t, rest, ``, tk)
	esperar(t, r, http.StatusOK, "")
	var rs clienteResp
	r.json(t, &rs)
	if rs.ID != c.ID || !rs.EsActivo || rs.Motivo != nil || rs.EliminadoPor != nil {
		t.Fatalf("restauración: %+v", rs)
	}
	esperar(t, post(t, rest, ``, tk), http.StatusConflict, "ya_activa")
}

func TestClienteListadoBusquedaYPaginacion(t *testing.T) {
	tk := gestorDirectorio(t)
	marca := "Busq" + sufijo()
	var ids []string
	for i := 1; i <= 3; i++ {
		var c clienteResp
		r := post(t, "/api/v1/clientes", fmt.Sprintf(`{"nombre_razon_social":"%s %d"}`, marca, i), tk)
		esperar(t, r, http.StatusCreated, "")
		r.json(t, &c)
		ids = append(ids, c.ID)
	}
	esperar(t, post(t, "/api/v1/clientes/"+ids[2]+"/baja", `{"motivo":"prueba"}`, tk), http.StatusOK, "")

	lista := func(consulta string) clientesPagina {
		var p clientesPagina
		r := call(t, "GET", "/api/v1/clientes"+conLimite(consulta), "", tk)
		esperar(t, r, http.StatusOK, "")
		r.json(t, &p)
		return p
	}
	// La búsqueda por texto acota; un comodín literal no coincide con todo.
	if got := lista("?q=" + marca).ids(); len(got) != 3 || !slices.Contains(got, ids[0]) {
		t.Fatalf("q=%s: %v", marca, got)
	}
	if got := lista("?q=%25").ids(); slices.Contains(got, ids[0]) {
		t.Fatalf("q=%%25 trató el comodín como patrón: %v", got)
	}
	// El filtro es_activo separa la baja.
	if got := lista("?q=" + marca + "&es_activo=true").ids(); len(got) != 2 || slices.Contains(got, ids[2]) {
		t.Fatalf("es_activo=true: %v", got)
	}
	if got := lista("?q=" + marca + "&es_activo=false").ids(); len(got) != 1 || got[0] != ids[2] {
		t.Fatalf("es_activo=false: %v", got)
	}
	// Paginación por cursor sin repetir ni saltar.
	p1 := lista("?q=" + marca + "&limit=2")
	if len(p1.Items) != 2 || p1.NextCursor == nil {
		t.Fatalf("página 1: %+v", p1)
	}
	p2 := lista("?q=" + marca + "&limit=2&cursor=" + *p1.NextCursor)
	if len(p2.Items) != 1 || p2.NextCursor != nil || slices.Contains(p1.ids(), p2.Items[0].ID) {
		t.Fatalf("página 2: %+v", p2)
	}
	esperar(t, call(t, "GET", "/api/v1/clientes?es_activo=quizas", "", tk), http.StatusBadRequest, "es_activo_invalido")
	esperar(t, call(t, "GET", "/api/v1/clientes?limit=0", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/clientes?cursor=!!!", "", tk), http.StatusBadRequest, "cursor_invalido")
}

func TestProveedorCicloDeVida(t *testing.T) {
	tk := gestorDirectorio(t)
	ruc := "20" + numeroUnico(9)
	razon := "Proveedor " + sufijo()
	cuerpo := fmt.Sprintf(`{"tipo_documento":"ruc","numero_documento":"%s","razon_social":"%s","contacto_nombre":"Ana Pérez","telefono":"(01) 555-1234","email":"ventas@prov.pe","direccion":"Jr. Comercio 45"}`, ruc, razon)

	r := post(t, "/api/v1/proveedores", cuerpo, tk)
	esperar(t, r, http.StatusCreated, "")
	var p proveedorResp
	r.json(t, &p)
	if p.TipoDocumento != "RUC" || p.NumeroDocumento != ruc || p.RazonSocial != razon || !p.EsActivo || p.Contacto == nil || *p.Contacto != "Ana Pérez" {
		t.Fatalf("proveedor creado: %+v", p)
	}

	// El documento es obligatorio y el RUC exige 11 dígitos.
	esperar(t, post(t, "/api/v1/proveedores", `{"razon_social":"Sin documento"}`, tk), http.StatusUnprocessableEntity, "documento_invalido")
	esperar(t, post(t, "/api/v1/proveedores", `{"tipo_documento":"RUC","numero_documento":"123","razon_social":"Corto"}`, tk), http.StatusUnprocessableEntity, "documento_invalido")
	esperar(t, post(t, "/api/v1/proveedores", fmt.Sprintf(`{"tipo_documento":"RUC","numero_documento":"%s","razon_social":" "}`, "20"+numeroUnico(9)), tk), http.StatusUnprocessableEntity, "razon_social_invalido")
	esperar(t, post(t, "/api/v1/proveedores", fmt.Sprintf(`{"tipo_documento":"RUC","numero_documento":"%s","razon_social":"Duplicado"}`, ruc), tk), http.StatusConflict, "documento_duplicado")

	// Edición: un lado del documento viene solo, el otro se toma del registro.
	otroRuc := "20" + numeroUnico(9)
	r = patch(t, "/api/v1/proveedores/"+p.ID, fmt.Sprintf(`{"numero_documento":"%s","contacto_nombre":null}`, otroRuc), tk)
	esperar(t, r, http.StatusOK, "")
	var ed proveedorResp
	r.json(t, &ed)
	if ed.TipoDocumento != "RUC" || ed.NumeroDocumento != otroRuc || ed.Contacto != nil || ed.Email == nil {
		t.Fatalf("edición: %+v", ed)
	}
	esperar(t, patch(t, "/api/v1/proveedores/"+p.ID, `{"tipo_documento":"RUC","numero_documento":"1"}`, tk), http.StatusUnprocessableEntity, "documento_invalido")
	esperar(t, patch(t, "/api/v1/proveedores/"+p.ID, `{}`, tk), http.StatusUnprocessableEntity, "sin_cambios")

	// Baja con motivo, bloqueo de edición y restauración con el mismo id.
	baja := "/api/v1/proveedores/" + p.ID + "/baja"
	esperar(t, post(t, baja, `{}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	r = post(t, baja, `{"motivo":"Dejó de abastecer"}`, tk)
	esperar(t, r, http.StatusOK, "")
	var b proveedorResp
	r.json(t, &b)
	if b.EsActivo || b.Motivo == nil || b.EliminadoPor == nil {
		t.Fatalf("baja: %+v", b)
	}
	esperar(t, patch(t, "/api/v1/proveedores/"+p.ID, `{"razon_social":"No"}`, tk), http.StatusConflict, "proveedor_inactivo")
	esperar(t, post(t, baja, `{"motivo":"otra"}`, tk), http.StatusConflict, "ya_dada_de_baja")
	r = post(t, "/api/v1/proveedores/"+p.ID+"/restauracion", ``, tk)
	esperar(t, r, http.StatusOK, "")
	var rs proveedorResp
	r.json(t, &rs)
	if rs.ID != p.ID || !rs.EsActivo || rs.Motivo != nil {
		t.Fatalf("restauración: %+v", rs)
	}
	esperar(t, call(t, "GET", "/api/v1/proveedores/999999999", "", tk), http.StatusNotFound, "no_encontrado")

	// Búsqueda por inicio del documento.
	var lista proveedoresPagina
	g := call(t, "GET", "/api/v1/proveedores"+conLimite("?q="+strings.ToLower(otroRuc[:8])), "", tk)
	esperar(t, g, http.StatusOK, "")
	g.json(t, &lista)
	if !slices.ContainsFunc(lista.Items, func(x proveedorResp) bool { return x.ID == p.ID }) {
		t.Fatalf("búsqueda por documento no encontró %s: %+v", p.ID, lista.Items)
	}
}
