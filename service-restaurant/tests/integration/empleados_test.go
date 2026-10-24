package integration

import (
	"fmt"
	"net/http"
	"testing"
)

type empleadoResp struct {
	ID              string  `json:"id"`
	UsuarioID       *string `json:"usuario_id"`
	Username        *string `json:"username"`
	TipoDocumento   string  `json:"tipo_documento"`
	NumeroDocumento string  `json:"numero_documento"`
	Nombres         string  `json:"nombres"`
	Apellidos       string  `json:"apellidos"`
	Telefono        *string `json:"telefono"`
	Cargo           *string `json:"cargo"`
	FechaIngreso    string  `json:"fecha_ingreso"`
	FechaCese       *string `json:"fecha_cese"`
	EsActivo        bool    `json:"es_activo"`
	Motivo          *string `json:"motivo_eliminacion"`
	EliminadoPor    *string `json:"eliminado_por"`
}

type empleadosPagina struct {
	Items      []empleadoResp `json:"items"`
	NextCursor *string        `json:"nextCursor"`
}

type equipoPagina struct {
	Items []struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Empleado *struct {
			ID        string  `json:"id"`
			Nombres   string  `json:"nombres"`
			Apellidos string  `json:"apellidos"`
			Cargo     *string `json:"cargo"`
		} `json:"empleado"`
	} `json:"items"`
}

// usuarioConEmpleado devuelve el id de un usuario nuevo (sin empleado) para vincular.
func usuarioSinEmpleado(t *testing.T, nombre string) string {
	t.Helper()
	asegurarUsuario(t, nombre, "Vinculo2026Clave", "CAJERO", "PRINCIPAL")
	return idDe(t, "usuarios", "username", nombre)
}

func TestEmpleadosExigeSusPermisos(t *testing.T) {
	// Solo ADMIN recibe EMPLEADO_* con la migración 00009.
	asegurarUsuario(t, "emp.cajero", "EmpCajero2026Clave", "CAJERO", "PRINCIPAL")
	cajero := token(t, "emp.cajero", "EmpCajero2026Clave")
	esperar(t, call(t, "GET", "/api/v1/empleados", "", cajero), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/empleados/1", "", cajero), http.StatusForbidden, "sin_permiso")
	esperar(t, post(t, "/api/v1/empleados", `{}`, cajero), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/empleados", "", ""), http.StatusUnauthorized, "no_autenticado")

	// EMPLEADO_VER solo consulta: no crea.
	rolConPermisos(t, "T_EMP_LECTOR", "EMPLEADO_VER")
	asegurarUsuario(t, "emp.lector", "EmpLector2026Clave", "T_EMP_LECTOR", "PRINCIPAL")
	lector := token(t, "emp.lector", "EmpLector2026Clave")
	esperar(t, call(t, "GET", "/api/v1/empleados", "", lector), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/empleados", `{"tipo_documento":"DNI","numero_documento":"11111111","nombres":"A","apellidos":"B"}`, lector), http.StatusForbidden, "sin_permiso")
}

func TestEmpleadoCicloDeVida(t *testing.T) {
	tk := adminToken(t)
	dni := numeroUnico(8)
	cuerpo := fmt.Sprintf(`{"tipo_documento":"dni","numero_documento":" %s ","nombres":"  María José ","apellidos":"Quispe Huamán","telefono":"999 888 777","cargo":"Mesera","fecha_ingreso":"2026-03-15"}`, dni)

	clave := map[string]string{"Idempotency-Key": nuevaClave()}
	r := callH(t, "POST", "/api/v1/empleados", cuerpo, tk, clave)
	esperar(t, r, http.StatusCreated, "")
	var e empleadoResp
	r.json(t, &e)
	if e.TipoDocumento != "DNI" || e.NumeroDocumento != dni || e.Nombres != "María José" || e.FechaIngreso != "2026-03-15" ||
		e.UsuarioID != nil || e.Username != nil || !e.EsActivo || e.Cargo == nil || *e.Cargo != "Mesera" {
		t.Fatalf("empleado creado: %+v", e)
	}
	rep := callH(t, "POST", "/api/v1/empleados", cuerpo, tk, clave)
	esperar(t, rep, http.StatusCreated, "")
	var e2 empleadoResp
	rep.json(t, &e2)
	if e2.ID != e.ID {
		t.Fatalf("la repetición creó otro empleado: %s vs %s", e2.ID, e.ID)
	}

	// Sin fecha de ingreso se toma la de hoy.
	r = post(t, "/api/v1/empleados", fmt.Sprintf(`{"tipo_documento":"CE","numero_documento":"CE%s","nombres":"Luis","apellidos":"Rojas"}`, numeroUnico(8)), tk)
	esperar(t, r, http.StatusCreated, "")
	var hoy empleadoResp
	r.json(t, &hoy)
	if len(hoy.FechaIngreso) != 10 {
		t.Fatalf("fecha de ingreso por defecto: %+v", hoy)
	}

	// Validaciones del alta y documento repetido.
	esperar(t, post(t, "/api/v1/empleados", `{"tipo_documento":"DNI","numero_documento":"12345678","nombres":" ","apellidos":"X"}`, tk), http.StatusUnprocessableEntity, "nombres_invalido")
	esperar(t, post(t, "/api/v1/empleados", `{"tipo_documento":"DNI","numero_documento":"12345678","nombres":"X","apellidos":""}`, tk), http.StatusUnprocessableEntity, "apellidos_invalido")
	esperar(t, post(t, "/api/v1/empleados", `{"tipo_documento":"DNI","numero_documento":"123","nombres":"X","apellidos":"Y"}`, tk), http.StatusUnprocessableEntity, "documento_invalido")
	esperar(t, post(t, "/api/v1/empleados", fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombres":"X","apellidos":"Y","fecha_ingreso":"15/03/2026"}`, numeroUnico(8)), tk), http.StatusUnprocessableEntity, "fecha_ingreso_invalida")
	esperar(t, post(t, "/api/v1/empleados", fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombres":"X","apellidos":"Y","telefono":"abc"}`, numeroUnico(8)), tk), http.StatusUnprocessableEntity, "telefono_invalido")
	esperar(t, post(t, "/api/v1/empleados", fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombres":"Otro","apellidos":"Y"}`, dni), tk), http.StatusConflict, "documento_duplicado")

	// Lectura.
	var leido empleadoResp
	g := call(t, "GET", "/api/v1/empleados/"+e.ID, "", tk)
	esperar(t, g, http.StatusOK, "")
	g.json(t, &leido)
	if leido.ID != e.ID || leido.Apellidos != "Quispe Huamán" {
		t.Fatalf("lectura: %+v", leido)
	}
	esperar(t, call(t, "GET", "/api/v1/empleados/999999999", "", tk), http.StatusNotFound, "no_encontrado")

	// Edición parcial y fechas coherentes.
	r = patch(t, "/api/v1/empleados/"+e.ID, `{"cargo":"Jefa de sala","telefono":null}`, tk)
	esperar(t, r, http.StatusOK, "")
	var ed empleadoResp
	r.json(t, &ed)
	if ed.Cargo == nil || *ed.Cargo != "Jefa de sala" || ed.Telefono != nil || ed.Nombres != "María José" {
		t.Fatalf("edición: %+v", ed)
	}
	esperar(t, patch(t, "/api/v1/empleados/"+e.ID, `{}`, tk), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, patch(t, "/api/v1/empleados/"+e.ID, `{"fecha_cese":"2026-01-01"}`, tk), http.StatusUnprocessableEntity, "fechas_incoherentes")
	r = patch(t, "/api/v1/empleados/"+e.ID, `{"fecha_cese":"2026-09-30"}`, tk)
	esperar(t, r, http.StatusOK, "")
	var cese empleadoResp
	r.json(t, &cese)
	if cese.FechaCese == nil || *cese.FechaCese != "2026-09-30" {
		t.Fatalf("cese: %+v", cese)
	}
	esperar(t, patch(t, "/api/v1/empleados/"+e.ID, `{"fecha_cese":null}`, tk), http.StatusOK, "")

	// Baja con motivo; no se edita; restauración conserva el id; el documento sigue reservado.
	baja := "/api/v1/empleados/" + e.ID + "/baja"
	esperar(t, post(t, baja, `{"motivo":" "}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	r = post(t, baja, `{"motivo":"Renunció"}`, tk)
	esperar(t, r, http.StatusOK, "")
	var b empleadoResp
	r.json(t, &b)
	if b.EsActivo || b.Motivo == nil || *b.Motivo != "Renunció" || b.EliminadoPor == nil {
		t.Fatalf("baja: %+v", b)
	}
	esperar(t, post(t, baja, `{"motivo":"otra"}`, tk), http.StatusConflict, "ya_dado_de_baja")
	esperar(t, patch(t, "/api/v1/empleados/"+e.ID, `{"cargo":"No"}`, tk), http.StatusConflict, "empleado_inactivo")
	esperar(t, post(t, "/api/v1/empleados", fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombres":"Otro","apellidos":"Y"}`, dni), tk), http.StatusConflict, "documento_duplicado")
	r = post(t, "/api/v1/empleados/"+e.ID+"/restauracion", ``, tk)
	esperar(t, r, http.StatusOK, "")
	var rs empleadoResp
	r.json(t, &rs)
	if rs.ID != e.ID || !rs.EsActivo || rs.Motivo != nil {
		t.Fatalf("restauración: %+v", rs)
	}
	esperar(t, post(t, "/api/v1/empleados/"+e.ID+"/restauracion", ``, tk), http.StatusConflict, "ya_activo")
}

func TestEmpleadoVinculoConUsuario(t *testing.T) {
	tk := adminToken(t)
	s := sufijo()
	usuario := "vinc" + s
	usuarioID := usuarioSinEmpleado(t, usuario)

	r := post(t, "/api/v1/empleados", fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombres":"Rosa","apellidos":"Vega","usuario_id":"%s"}`, numeroUnico(8), usuarioID), tk)
	esperar(t, r, http.StatusCreated, "")
	var e empleadoResp
	r.json(t, &e)
	if e.UsuarioID == nil || *e.UsuarioID != usuarioID || e.Username == nil || *e.Username != usuario {
		t.Fatalf("vínculo: %+v", e)
	}

	// Un usuario solo tiene un empleado; el usuario debe existir y no ser el técnico.
	esperar(t, post(t, "/api/v1/empleados", fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombres":"Otra","apellidos":"Vega","usuario_id":"%s"}`, numeroUnico(8), usuarioID), tk), http.StatusConflict, "usuario_ya_vinculado")
	esperar(t, post(t, "/api/v1/empleados", fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombres":"Otra","apellidos":"Vega","usuario_id":"999999999"}`, numeroUnico(8)), tk), http.StatusUnprocessableEntity, "usuario_invalido")
	sistema := scalarStr(t, `SELECT id FROM sch_restaurante.usuarios WHERE es_sistema`)
	esperar(t, post(t, "/api/v1/empleados", fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombres":"Otra","apellidos":"Vega","usuario_id":"%s"}`, numeroUnico(8), sistema), tk), http.StatusUnprocessableEntity, "usuario_invalido")
	esperar(t, post(t, "/api/v1/empleados", fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombres":"Otra","apellidos":"Vega","usuario_id":"abc"}`, numeroUnico(8)), tk), http.StatusBadRequest, "usuario_id_invalido")

	// Aparece en Equipo y accesos con su nombre real.
	var equipo equipoPagina
	g := call(t, "GET", "/api/v1/usuarios?limit=100", "", tk)
	esperar(t, g, http.StatusOK, "")
	g.json(t, &equipo)
	var visto bool
	for _, u := range equipo.Items {
		if u.ID == usuarioID {
			visto = u.Empleado != nil && u.Empleado.Nombres == "Rosa" && u.Empleado.ID == e.ID
		}
	}
	if !visto {
		t.Fatalf("GET /usuarios no trae al empleado de %s: %+v", usuarioID, equipo)
	}

	// La baja del empleado se rechaza mientras su usuario siga activo.
	baja := "/api/v1/empleados/" + e.ID + "/baja"
	esperar(t, post(t, baja, `{"motivo":"Prueba"}`, tk), http.StatusConflict, "empleado_con_usuario_activo")

	// Desvincular permite la baja; restaurar con otro usuario dado de baja se rechaza.
	esperar(t, patch(t, "/api/v1/empleados/"+e.ID, `{"usuario_id":null}`, tk), http.StatusOK, "")
	esperar(t, post(t, baja, `{"motivo":"Prueba"}`, tk), http.StatusOK, "")
	esperar(t, patch(t, "/api/v1/empleados/"+e.ID, `{"cargo":"No"}`, tk), http.StatusConflict, "empleado_inactivo")
	esperar(t, post(t, "/api/v1/empleados/"+e.ID+"/restauracion", ``, tk), http.StatusOK, "")

	// Se vuelve a vincular y, tras dar de baja al usuario, el empleado ya se puede dar de baja.
	esperar(t, patch(t, "/api/v1/empleados/"+e.ID, fmt.Sprintf(`{"usuario_id":"%s"}`, usuarioID), tk), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/usuarios/"+usuarioID+"/baja", `{"motivo":"Se fue"}`, tk), http.StatusOK, "")
	esperar(t, post(t, baja, `{"motivo":"Se fue"}`, tk), http.StatusOK, "")
	// Con su usuario dado de baja, el empleado no se puede restaurar hasta restaurar al usuario.
	esperar(t, post(t, "/api/v1/empleados/"+e.ID+"/restauracion", ``, tk), http.StatusUnprocessableEntity, "usuario_invalido")
	esperar(t, post(t, "/api/v1/usuarios/"+usuarioID+"/restauracion", ``, tk), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/empleados/"+e.ID+"/restauracion", ``, tk), http.StatusOK, "")
}

func TestAltaConjuntaDeUsuarioYEmpleado(t *testing.T) {
	tk := adminToken(t)
	s := sufijo()
	usuario := "conj" + s
	dni := numeroUnico(8)
	cuerpo := func(user, doc string) string {
		return fmt.Sprintf(`{"username":"%s","email":"%s@ejemplo.pe","password":"ConjuntaClave2026","empleado":{"tipo_documento":"DNI","numero_documento":"%s","nombres":"Pedro","apellidos":"Alva Ríos","cargo":"Cajero"}}`, user, user, doc)
	}

	r := post(t, "/api/v1/usuarios", cuerpo(usuario, dni), tk)
	esperar(t, r, http.StatusCreated, "")
	var u struct {
		ID string `json:"id"`
	}
	r.json(t, &u)
	var e empleadoResp
	g := call(t, "GET", "/api/v1/empleados"+conLimite("?q="+dni), "", tk)
	esperar(t, g, http.StatusOK, "")
	var lista empleadosPagina
	g.json(t, &lista)
	if len(lista.Items) != 1 {
		t.Fatalf("empleados con documento %s: %+v", dni, lista.Items)
	}
	e = lista.Items[0]
	if e.UsuarioID == nil || *e.UsuarioID != u.ID || e.Nombres != "Pedro" || e.Cargo == nil || *e.Cargo != "Cajero" {
		t.Fatalf("empleado vinculado: %+v (usuario %s)", e, u.ID)
	}

	// Documento repetido: se rechaza todo y el usuario no queda creado (una sola transacción).
	otro := "conj2" + s
	esperar(t, post(t, "/api/v1/usuarios", cuerpo(otro, dni), tk), http.StatusConflict, "documento_duplicado")
	if n := scalarStr(t, `SELECT count(*) FROM sch_restaurante.usuarios WHERE username=$1`, otro); n != "0" {
		t.Fatalf("el usuario %s quedó creado pese al rechazo (%s filas)", otro, n)
	}

	// Datos del empleado inválidos: se rechaza antes de crear nada.
	malo := fmt.Sprintf(`{"username":"%s","password":"ConjuntaClave2026","empleado":{"tipo_documento":"DNI","numero_documento":"1","nombres":"X","apellidos":"Y"}}`, otro)
	esperar(t, post(t, "/api/v1/usuarios", malo, tk), http.StatusUnprocessableEntity, "documento_invalido")
	if n := scalarStr(t, `SELECT count(*) FROM sch_restaurante.usuarios WHERE username=$1`, otro); n != "0" {
		t.Fatalf("el usuario %s quedó creado pese al rechazo", otro)
	}

	// Sin datos de empleado el alta sigue funcionando como antes.
	solo := "solo" + s
	esperar(t, post(t, "/api/v1/usuarios", fmt.Sprintf(`{"username":"%s","password":"ConjuntaClave2026"}`, solo), tk), http.StatusCreated, "")

	// Quien administra accesos pero no empleados no puede crear al empleado.
	rolConPermisos(t, "T_SOLO_USUARIOS", "USUARIO_ADMINISTRAR")
	asegurarUsuario(t, "emp.solouser", "SoloUser2026Clave", "T_SOLO_USUARIOS", "PRINCIPAL")
	sinEmpleados := token(t, "emp.solouser", "SoloUser2026Clave")
	esperar(t, post(t, "/api/v1/usuarios", cuerpo("conj3"+s, numeroUnico(8)), sinEmpleados), http.StatusForbidden, "sin_permiso")
	esperar(t, post(t, "/api/v1/usuarios", fmt.Sprintf(`{"username":"%s","password":"ConjuntaClave2026"}`, "conj4"+s), sinEmpleados), http.StatusCreated, "")
}

func TestEmpleadosListadoBusquedaYPaginacion(t *testing.T) {
	tk := adminToken(t)
	marca := "Zeta" + sufijo()
	var ids []string
	for i := 1; i <= 3; i++ {
		var e empleadoResp
		r := post(t, "/api/v1/empleados", fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombres":"%s","apellidos":"Prueba %d"}`, numeroUnico(8), marca, i), tk)
		esperar(t, r, http.StatusCreated, "")
		r.json(t, &e)
		ids = append(ids, e.ID)
	}
	esperar(t, post(t, "/api/v1/empleados/"+ids[2]+"/baja", `{"motivo":"prueba"}`, tk), http.StatusOK, "")

	lista := func(consulta string) empleadosPagina {
		var p empleadosPagina
		r := call(t, "GET", "/api/v1/empleados"+conLimite(consulta), "", tk)
		esperar(t, r, http.StatusOK, "")
		r.json(t, &p)
		return p
	}
	if got := lista("?q=" + marca); len(got.Items) != 3 {
		t.Fatalf("q=%s: %+v", marca, got.Items)
	}
	// Se busca también por nombre completo.
	if got := lista("?q=" + marca + "%20Prueba%201"); len(got.Items) != 1 || got.Items[0].ID != ids[0] {
		t.Fatalf("q por nombre completo: %+v", got.Items)
	}
	if got := lista("?q=" + marca + "&es_activo=false"); len(got.Items) != 1 || got.Items[0].ID != ids[2] {
		t.Fatalf("es_activo=false: %+v", got.Items)
	}
	p1 := lista("?q=" + marca + "&limit=2")
	if len(p1.Items) != 2 || p1.NextCursor == nil {
		t.Fatalf("página 1: %+v", p1)
	}
	p2 := lista("?q=" + marca + "&limit=2&cursor=" + *p1.NextCursor)
	if len(p2.Items) != 1 || p2.NextCursor != nil {
		t.Fatalf("página 2: %+v", p2)
	}
	esperar(t, call(t, "GET", "/api/v1/empleados?es_activo=quizas", "", tk), http.StatusBadRequest, "es_activo_invalido")
	esperar(t, call(t, "GET", "/api/v1/empleados?cursor=!!!", "", tk), http.StatusBadRequest, "cursor_invalido")
}
