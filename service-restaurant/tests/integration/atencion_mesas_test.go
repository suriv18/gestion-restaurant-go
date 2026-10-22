package integration

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestMesasLecturaRespetaLasSedesDelActor(t *testing.T) {
	crearSedeSQL(t, "MESA-AJENA")
	crearZonaSQL(t, "PRINCIPAL", "Zona mesas L")
	crearZonaSQL(t, "MESA-AJENA", "Zona mesas ajena")
	crearMesaSQL(t, "PRINCIPAL", "Zona mesas L", "ML-1", 4)
	crearMesaSQL(t, "MESA-AJENA", "Zona mesas ajena", "MAJ-1", 2)
	tk := lectorMesas(t)

	numeros := listarMesas(t, "", tk).numeros()
	if !slices.Contains(numeros, "ML-1") || slices.Contains(numeros, "MAJ-1") {
		t.Fatalf("mesas visibles = %v: debe incluir ML-1 y no MAJ-1", numeros)
	}
	esperar(t, call(t, "GET", "/api/v1/mesas?sucursal_id="+idDe(t, "sucursales", "codigo", "MESA-AJENA"), "", tk),
		http.StatusNotFound, "no_encontrado")

	id := idDe(t, "mesas", "numero_mesa", "ML-1")
	r := call(t, "GET", "/api/v1/mesas/"+id, "", tk)
	esperar(t, r, http.StatusOK, "")
	var m mesaResp
	r.json(t, &m)
	if m.NumeroMesa != "ML-1" || m.Capacidad != 4 || !m.Habilitada || !m.EsActivo || m.PosicionX != nil || m.Motivo != nil ||
		m.SucursalID != idDe(t, "sucursales", "codigo", "PRINCIPAL") || m.ZonaID != idDe(t, "zonas", "nombre", "Zona mesas L") {
		t.Fatalf("mesa: %+v", m)
	}
	esperar(t, call(t, "GET", "/api/v1/mesas/"+idDe(t, "mesas", "numero_mesa", "MAJ-1"), "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/mesas/999999", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/mesas/abc", "", tk), http.StatusBadRequest, "id_invalido")
}

func TestMesasExigeAlgunPermisoDeMesa(t *testing.T) {
	rolConPermisos(t, "T_SIN_MESA", "NEGOCIO_VER")
	asegurarUsuario(t, "mesa.sin", "MesaSin2026Clave", "T_SIN_MESA", "PRINCIPAL")
	tk := token(t, "mesa.sin", "MesaSin2026Clave")
	esperar(t, call(t, "GET", "/api/v1/mesas", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/mesas/1", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/mesas", "", ""), http.StatusUnauthorized, "no_autenticado")
}

func TestMesasFiltroYPaginacion(t *testing.T) {
	crearZonaSQL(t, "PRINCIPAL", "Zona pag mesas")
	crearMesaSQL(t, "PRINCIPAL", "Zona pag mesas", "PM-1", 2)
	crearMesaSQL(t, "PRINCIPAL", "Zona pag mesas", "PM-2", 2)
	crearMesaSQL(t, "PRINCIPAL", "Zona pag mesas", "PM-3", 2)
	asSystem(t, `UPDATE sch_restaurante.mesas SET es_activo=false, motivo_eliminacion='Retirada' WHERE numero_mesa='PM-3'`)
	tk := lectorMesas(t)

	if got := listarMesas(t, "?es_activo=true", tk).numeros(); slices.Contains(got, "PM-3") || !slices.Contains(got, "PM-1") {
		t.Fatalf("es_activo=true: %v", got)
	}
	inactivas := listarMesas(t, "?es_activo=false", tk)
	if got := inactivas.numeros(); !slices.Contains(got, "PM-3") || slices.Contains(got, "PM-1") {
		t.Fatalf("es_activo=false: %v", got)
	}
	for _, m := range inactivas.Items {
		if m.NumeroMesa == "PM-3" && (m.Motivo == nil || *m.Motivo != "Retirada" || m.EliminadoPor == nil) {
			t.Fatalf("una mesa inactiva debe exponer motivo y actor: %+v", m)
		}
	}
	esperar(t, call(t, "GET", "/api/v1/mesas?es_activo=quizas", "", tk), http.StatusBadRequest, "es_activo_invalido")
	esperar(t, call(t, "GET", "/api/v1/mesas?limit=101", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/mesas?cursor=!!!", "", tk), http.StatusBadRequest, "cursor_invalido")

	p1 := listarMesas(t, "?limit=1", tk)
	if len(p1.Items) != 1 || p1.NextCursor == nil {
		t.Fatalf("primera página: %+v", p1)
	}
	p2 := listarMesas(t, "?limit=1&cursor="+*p1.NextCursor, tk)
	if len(p2.Items) != 1 {
		t.Fatalf("la segunda página debe traer 1 mesa: %+v", p2)
	}
	id1, err1 := strconv.Atoi(p1.Items[0].ID)
	id2, err2 := strconv.Atoi(p2.Items[0].ID)
	if err1 != nil || err2 != nil {
		t.Fatalf("ids no numéricos: %q (%v) / %q (%v)", p1.Items[0].ID, err1, p2.Items[0].ID, err2)
	}
	if id1 >= id2 {
		t.Fatalf("la segunda página debe continuar en orden ascendente: %+v / %+v", p1, p2)
	}
}

func altaMesa(sucursalID, zonaID, numero string, capacidad int) string {
	return fmt.Sprintf(`{"sucursal_id":"%s","zona_id":"%s","numero_mesa":"%s","capacidad":%d,"posicion_x":"12.50"}`,
		sucursalID, zonaID, numero, capacidad)
}

func TestMesaCrearConIdempotenciaYValidaciones(t *testing.T) {
	s := sufijo()
	crearSedeSQL(t, "MESA-AJENA")
	zona := crearZonaSQL(t, "PRINCIPAL", "Zona crear mesa "+s)
	zonaAjena := crearZonaSQL(t, "MESA-AJENA", "Zona crear ajena "+s)
	zonaInactiva := crearZonaSQL(t, "PRINCIPAL", "Zona crear inactiva "+s)
	asSystem(t, `UPDATE sch_restaurante.zonas SET es_activo=false WHERE id=`+zonaInactiva)
	tk := gestorMesas(t)
	principal := idDe(t, "sucursales", "codigo", "PRINCIPAL")
	h := map[string]string{"Idempotency-Key": nuevaClave()}
	num := func(base string) string { return base + "-" + s }

	r1 := callH(t, "POST", "/api/v1/mesas", altaMesa(principal, zona, num("NM-1"), 4), tk, h)
	esperar(t, r1, http.StatusCreated, "")
	var m mesaResp
	r1.json(t, &m)
	if m.NumeroMesa != num("NM-1") || m.Capacidad != 4 || !m.Habilitada || !m.EsActivo || m.ZonaID != zona || m.SucursalID != principal ||
		m.PosicionX == nil || *m.PosicionX != "12.50" || m.PosicionY != nil || m.CreadoPor != idDe(t, "usuarios", "username", "mesa.gestor") {
		t.Fatalf("mesa creada: %+v", m)
	}
	r2 := callH(t, "POST", "/api/v1/mesas", altaMesa(principal, zona, num("NM-1"), 4), tk, h)
	esperar(t, r2, http.StatusCreated, "")
	var rep mesaResp
	r2.json(t, &rep)
	if rep.ID != m.ID || scalar(t, `SELECT count(*) FROM sch_restaurante.mesas WHERE zona_id=$1`, zona) != 1 {
		t.Fatalf("la repetición debe devolver la misma mesa: %+v vs %+v", rep, m)
	}
	esperar(t, callH(t, "POST", "/api/v1/mesas", altaMesa(principal, zona, num("NM-9"), 2), tk, h), http.StatusConflict, "idempotency_key_reutilizada")

	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, zona, num("NM-1"), 2), tk), http.StatusConflict, "numero_mesa_duplicado")
	esperar(t, call(t, "POST", "/api/v1/mesas", altaMesa(principal, zona, num("NM-2"), 2), tk), http.StatusBadRequest, "idempotency_key_requerida")
	esperar(t, post(t, "/api/v1/mesas", `{"sucursal_id":"`+principal+`","zona_id":"`+zona+`","numero_mesa":"`+num("NM-3")+`","capacidad":2,"creado_por":"1"}`, tk),
		http.StatusBadRequest, "json_invalido")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, "abc", num("NM-4"), 2), tk), http.StatusBadRequest, "zona_id_invalido")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, zona, " ", 2), tk), http.StatusUnprocessableEntity, "numero_mesa_invalido")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, zona, num("NM-5"), 0), tk), http.StatusUnprocessableEntity, "capacidad_invalida")
	esperar(t, post(t, "/api/v1/mesas", `{"sucursal_id":"`+principal+`","zona_id":"`+zona+`","numero_mesa":"`+num("NM-6")+`","capacidad":2,"posicion_x":"1.234"}`, tk),
		http.StatusUnprocessableEntity, "posicion_x_invalida")
	// La zona debe existir, estar activa y ser de la misma sede.
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, zonaAjena, num("NM-7"), 2), tk), http.StatusUnprocessableEntity, "zona_invalida")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, zonaInactiva, num("NM-8"), 2), tk), http.StatusUnprocessableEntity, "zona_invalida")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, "999999", num("NM-10"), 2), tk), http.StatusUnprocessableEntity, "zona_invalida")

	ajena := idDe(t, "sucursales", "codigo", "MESA-AJENA")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(ajena, zonaAjena, num("NM-11"), 2), tk), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, zona, num("NM-12"), 2), lectorMesas(t)), http.StatusForbidden, "sin_permiso")
	// Ningún rechazo dejó mesas: solo existe la creada con éxito (cada zona de la prueba es nueva).
	if scalar(t, `SELECT count(*) FROM sch_restaurante.mesas WHERE zona_id IN ($1,$2,$3)`, zona, zonaAjena, zonaInactiva) != 1 {
		t.Fatal("los rechazos no deben dejar mesas")
	}

	// La posición viaja como cadena y vuelve igual al leer la mesa creada.
	r := call(t, "GET", "/api/v1/mesas/"+m.ID, "", tk)
	esperar(t, r, http.StatusOK, "")
	var leida mesaResp
	r.json(t, &leida)
	if leida.PosicionX == nil || *leida.PosicionX != "12.50" || leida.PosicionY != nil {
		t.Fatalf("la posición debe volver como \"12.50\": %+v", leida)
	}
}

func TestMesaLimitesDeValidacion(t *testing.T) {
	s := sufijo()
	zona := crearZonaSQL(t, "PRINCIPAL", "Zona limites "+s)
	tk := gestorMesas(t)
	principal := idDe(t, "sucursales", "codigo", "PRINCIPAL")
	cuerpo := func(numero string, capacidad int, extra string) string {
		return fmt.Sprintf(`{"sucursal_id":"%s","zona_id":"%s","numero_mesa":"%s","capacidad":%d%s}`, principal, zona, numero+"-"+s, capacidad, extra)
	}
	mesasEnZona := func() int { return scalar(t, `SELECT count(*) FROM sch_restaurante.mesas WHERE zona_id=$1`, zona) }

	// Capacidad: 1..500.
	esperar(t, post(t, "/api/v1/mesas", cuerpo("L-501", 501, ""), tk), http.StatusUnprocessableEntity, "capacidad_invalida")
	esperar(t, post(t, "/api/v1/mesas", cuerpo("L-NEG", -1, ""), tk), http.StatusUnprocessableEntity, "capacidad_invalida")
	if mesasEnZona() != 0 {
		t.Fatal("capacidades fuera de rango no deben crear mesas")
	}
	r := post(t, "/api/v1/mesas", cuerpo("L-500", 500, ""), tk)
	esperar(t, r, http.StatusCreated, "")
	var tope mesaResp
	r.json(t, &tope)
	if tope.Capacidad != 500 {
		t.Fatalf("500 es válida: %+v", tope)
	}

	// Posición numeric(10,2): hasta 8 enteros y 2 decimales, con signo opcional.
	invalidas := []string{`"123456789"`, `"-123456789"`, `"123456789.5"`, `"1.234"`, `"-"`, `"--1"`, `"+5"`, `"-.5"`, `"1."`, `"1,5"`, `"abc"`, `"1e3"`}
	for _, v := range invalidas {
		esperar(t, post(t, "/api/v1/mesas", cuerpo("L-PX", 2, `,"posicion_x":`+v), tk), http.StatusUnprocessableEntity, "posicion_x_invalida")
		esperar(t, post(t, "/api/v1/mesas", cuerpo("L-PY", 2, `,"posicion_y":`+v), tk), http.StatusUnprocessableEntity, "posicion_y_invalida")
	}
	if mesasEnZona() != 1 {
		t.Fatal("las posiciones inválidas no deben crear mesas")
	}
	r = post(t, "/api/v1/mesas", cuerpo("L-POS", 2, `,"posicion_x":"-99999999.99","posicion_y":"99999999.99"`), tk)
	esperar(t, r, http.StatusCreated, "")
	var pos mesaResp
	r.json(t, &pos)
	if pos.PosicionX == nil || *pos.PosicionX != "-99999999.99" || pos.PosicionY == nil || *pos.PosicionY != "99999999.99" {
		t.Fatalf("los extremos del rango deben conservarse: %+v", pos)
	}
	r = post(t, "/api/v1/mesas", cuerpo("L-NEG05", 2, `,"posicion_x":"-0.5","posicion_y":"7"`), tk)
	esperar(t, r, http.StatusCreated, "")
	var neg mesaResp
	r.json(t, &neg)
	if neg.PosicionX == nil || *neg.PosicionX != "-0.50" || neg.PosicionY == nil || *neg.PosicionY != "7.00" {
		t.Fatalf("negativo y entero deben normalizarse a 2 decimales: %+v", neg)
	}

	// Motivo de baja: máximo 2000 caracteres.
	id := neg.ID
	esperar(t, post(t, "/api/v1/mesas/"+id+"/baja", `{"motivo":"`+strings.Repeat("a", 2001)+`"}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.mesas WHERE id=$1 AND es_activo`, id) != 1 {
		t.Fatal("un motivo excedido no debe dar de baja la mesa")
	}
	r = post(t, "/api/v1/mesas/"+id+"/baja", `{"motivo":"`+strings.Repeat("a", 2000)+`"}`, tk)
	esperar(t, r, http.StatusOK, "")
	var baja mesaResp
	r.json(t, &baja)
	if baja.EsActivo || baja.Motivo == nil || len([]rune(*baja.Motivo)) != 2000 {
		t.Fatalf("2000 caracteres son válidos: %+v", baja)
	}
}

func TestMesaEditarYHabilitar(t *testing.T) {
	s := sufijo()
	crearSedeSQL(t, "MESA-AJENA")
	crearZonaSQL(t, "PRINCIPAL", "Zona editar A "+s)
	zonaB := crearZonaSQL(t, "PRINCIPAL", "Zona editar B "+s)
	zonaAjena := crearZonaSQL(t, "MESA-AJENA", "Zona editar ajena "+s)
	id := crearMesaSQL(t, "PRINCIPAL", "Zona editar A "+s, "ED-1-"+s, 4)
	crearMesaSQL(t, "PRINCIPAL", "Zona editar A "+s, "ED-2-"+s, 2)
	ajenaID := crearMesaSQL(t, "MESA-AJENA", "Zona editar ajena "+s, "ED-AJ-"+s, 2)
	tk := gestorMesas(t)

	// Habilitar o deshabilitar no pide motivo.
	r := callH(t, "PATCH", "/api/v1/mesas/"+id, `{"habilitada":false}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	var m mesaResp
	r.json(t, &m)
	if m.Habilitada || m.NumeroMesa != "ED-1-"+s || m.Capacidad != 4 {
		t.Fatalf("deshabilitar: %+v", m)
	}
	r = callH(t, "PATCH", "/api/v1/mesas/"+id,
		fmt.Sprintf(`{"habilitada":true,"capacidad":6,"zona_id":"%s","posicion_x":"1.5","posicion_y":"-2"}`, zonaB), tk, nil)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &m)
	if !m.Habilitada || m.Capacidad != 6 || m.ZonaID != zonaB || m.PosicionX == nil || *m.PosicionX != "1.50" || m.PosicionY == nil || *m.PosicionY != "-2.00" {
		t.Fatalf("edición: %+v", m)
	}
	r = callH(t, "PATCH", "/api/v1/mesas/"+id, `{"posicion_x":null}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &m)
	if m.PosicionX != nil || m.PosicionY == nil || m.Capacidad != 6 {
		t.Fatalf("null debe vaciar solo la posición X: %+v", m)
	}

	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{}`, tk, nil), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{"sucursal_id":"1"}`, tk, nil), http.StatusBadRequest, "json_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{"capacidad":0}`, tk, nil), http.StatusUnprocessableEntity, "capacidad_invalida")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{"capacidad":501}`, tk, nil), http.StatusUnprocessableEntity, "capacidad_invalida")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{"numero_mesa":" "}`, tk, nil), http.StatusUnprocessableEntity, "numero_mesa_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{"posicion_y":"123456789"}`, tk, nil), http.StatusUnprocessableEntity, "posicion_y_invalida")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{"numero_mesa":"ED-2-`+s+`"}`, tk, nil), http.StatusConflict, "numero_mesa_duplicado")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, fmt.Sprintf(`{"zona_id":"%s"}`, zonaAjena), tk, nil),
		http.StatusUnprocessableEntity, "zona_invalida")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/999999", `{"capacidad":2}`, tk, nil), http.StatusNotFound, "no_encontrado")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+ajenaID, `{"capacidad":2}`, tk, nil), http.StatusNotFound, "no_encontrado")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{"capacidad":2}`, lectorMesas(t), nil), http.StatusForbidden, "sin_permiso")
}

func TestMesaBajaConMotivoYRestauracion(t *testing.T) {
	s := sufijo()
	crearZonaSQL(t, "PRINCIPAL", "Zona ciclo mesa "+s)
	id := crearMesaSQL(t, "PRINCIPAL", "Zona ciclo mesa "+s, "CM-1-"+s, 4)
	tk := gestorMesas(t)
	gestor := idDe(t, "usuarios", "username", "mesa.gestor")

	esperar(t, post(t, "/api/v1/mesas/"+id+"/baja", `{}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	esperar(t, post(t, "/api/v1/mesas/"+id+"/baja", `{"motivo":"   "}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	esperar(t, post(t, "/api/v1/mesas/"+id+"/baja", "", tk), http.StatusBadRequest, "cuerpo_requerido")

	h := map[string]string{"Idempotency-Key": nuevaClave()}
	r := callH(t, "POST", "/api/v1/mesas/"+id+"/baja", `{"motivo":"Mesa dañada"}`, tk, h)
	esperar(t, r, http.StatusOK, "")
	var m mesaResp
	r.json(t, &m)
	if m.EsActivo || m.Motivo == nil || *m.Motivo != "Mesa dañada" || m.EliminadoPor == nil || *m.EliminadoPor != gestor {
		t.Fatalf("baja: %+v", m)
	}
	if scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='mesas' AND accion='UPDATE'
		AND registro_id=`+id+` AND creado_por=`+gestor) == 0 {
		t.Fatal("la baja debe quedar auditada con el actor")
	}
	antes := scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='mesas' AND registro_id=`+id)
	esperar(t, callH(t, "POST", "/api/v1/mesas/"+id+"/baja", `{"motivo":"Mesa dañada"}`, tk, h), http.StatusOK, "")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='mesas' AND registro_id=`+id) != antes {
		t.Fatal("repetir la misma clave no debe volver a escribir")
	}
	esperar(t, post(t, "/api/v1/mesas/"+id+"/baja", `{"motivo":"Otra vez"}`, tk), http.StatusConflict, "ya_dada_de_baja")
	// Una mesa dada de baja no se edita ni se habilita hasta restaurarla.
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{"habilitada":true}`, tk, nil), http.StatusConflict, "mesa_inactiva")

	r = post(t, "/api/v1/mesas/"+id+"/restauracion", `{}`, tk)
	esperar(t, r, http.StatusOK, "")
	var rest mesaResp
	r.json(t, &rest)
	if rest.ID != m.ID || !rest.EsActivo || rest.Motivo != nil || rest.EliminadoPor != nil {
		t.Fatalf("restauración debe conservar el ID y limpiar los marcadores: %+v", rest)
	}
	esperar(t, post(t, "/api/v1/mesas/"+id+"/restauracion", "", tk), http.StatusConflict, "ya_activa")
	esperar(t, post(t, "/api/v1/mesas/999999/baja", `{"motivo":"x"}`, tk), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/mesas/"+id+"/baja", `{"motivo":"x"}`, lectorMesas(t)), http.StatusForbidden, "sin_permiso")
}

// La base impide restaurar una mesa cuya zona está dada de baja: llega como regla_de_datos.
func TestMesaNoSeRestauraConLaZonaDadaDeBaja(t *testing.T) {
	s := sufijo()
	zona := crearZonaSQL(t, "PRINCIPAL", "Zona restaurar mesa "+s)
	mesa := crearMesaSQL(t, "PRINCIPAL", "Zona restaurar mesa "+s, "RM-1-"+s, 2)
	tk := gestorMesas(t)

	esperar(t, post(t, "/api/v1/mesas/"+mesa+"/baja", `{"motivo":"Cierre de zona"}`, tk), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/zonas/"+zona+"/baja", "", tk), http.StatusOK, "")

	r := post(t, "/api/v1/mesas/"+mesa+"/restauracion", "", tk)
	esperar(t, r, http.StatusUnprocessableEntity, "regla_de_datos")
	var p problema
	r.json(t, &p)
	if p.Detail != "no se puede restaurar la mesa porque su zona está dada de baja" {
		t.Fatalf("el mensaje debe ser el curado, sin nombres internos: %q", p.Detail)
	}
	if scalar(t, `SELECT count(*) FROM sch_restaurante.mesas WHERE id=$1 AND NOT es_activo`, mesa) != 1 {
		t.Fatal("la mesa debe seguir dada de baja")
	}

	// Restaurada la zona, la mesa ya puede restaurarse.
	esperar(t, post(t, "/api/v1/zonas/"+zona+"/restauracion", "", tk), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/mesas/"+mesa+"/restauracion", "", tk), http.StatusOK, "")
}
