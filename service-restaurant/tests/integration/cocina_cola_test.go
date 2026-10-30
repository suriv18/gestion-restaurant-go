package integration

import (
	"net/http"
	"net/url"
	"testing"
)

func colaDe(t *testing.T, tk string, q url.Values) comandasPagina {
	t.Helper()
	r := call(t, "GET", "/api/v1/comandas?"+q.Encode(), "", tk)
	esperar(t, r, http.StatusOK, "")
	var p comandasPagina
	r.json(t, &p)
	return p
}

func (p comandasPagina) ids() []string {
	out := make([]string, len(p.Items))
	for i, c := range p.Items {
		out[i] = c.ID
	}
	return out
}

func TestColaMuestraLoPendienteDeEntregarPorEstacion(t *testing.T) {
	e := nuevoEscenario(t, "4")
	c1 := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	c2 := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "3"})
	porEstacion := url.Values{"estacion_id": {e.estacion}}

	p := colaDe(t, e.cocinero, porEstacion)
	if got := p.ids(); len(got) != 2 || got[0] != c1.ID || got[1] != c2.ID {
		t.Fatalf("cola inicial: %v", got)
	}
	if p.Items[0].Lineas[0].Estado != "PENDIENTE" || p.Items[0].Lineas[0].Cantidad != "1.000" || p.Items[0].MesaID != nil {
		t.Fatalf("contenido de la primera comanda: %+v", p.Items[0])
	}

	// Entregar toda la primera comanda la saca de la cola por defecto, pero sigue visible con estado=ENTREGADO.
	lc := c1.Lineas[0].ID
	transicionarOK(t, e.cocinero, lc, "PREPARACION")
	transicionarOK(t, e.cocinero, lc, "LISTO")
	if got := colaDe(t, e.cocinero, porEstacion).ids(); len(got) != 2 {
		t.Fatalf("una línea LISTO aún está pendiente de entregar: %v", got)
	}
	transicionarOK(t, e.cocinero, lc, "ENTREGADO")
	if got := colaDe(t, e.cocinero, porEstacion).ids(); len(got) != 1 || got[0] != c2.ID {
		t.Fatalf("la cola no debe incluir lo ya entregado: %v", got)
	}
	porEstado := url.Values{"estacion_id": {e.estacion}, "estado": {"ENTREGADO"}}
	if got := colaDe(t, e.cocinero, porEstado).ids(); len(got) != 1 || got[0] != c1.ID {
		t.Fatalf("estado=ENTREGADO: %v", got)
	}
	porListo := url.Values{"estacion_id": {e.estacion}, "estado": {"LISTO"}}
	if got := colaDe(t, e.cocinero, porListo).ids(); len(got) != 0 {
		t.Fatalf("ninguna comanda tiene líneas LISTO: %v", got)
	}
}

func TestColaPaginaPorCursor(t *testing.T) {
	e := nuevoEscenario(t, "3")
	c1 := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	c2 := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	c3 := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})

	q := url.Values{"estacion_id": {e.estacion}, "limit": {"2"}}
	p1 := colaDe(t, e.cocinero, q)
	if got := p1.ids(); len(got) != 2 || got[0] != c1.ID || got[1] != c2.ID || p1.NextCursor == nil {
		t.Fatalf("primera página: %v %v", got, p1.NextCursor)
	}
	q.Set("cursor", *p1.NextCursor)
	p2 := colaDe(t, e.cocinero, q)
	if got := p2.ids(); len(got) != 1 || got[0] != c3.ID || p2.NextCursor != nil {
		t.Fatalf("segunda página: %v %v", got, p2.NextCursor)
	}
}

func TestColaValidaPermisosFiltrosYSedes(t *testing.T) {
	e := nuevoEscenario(t, "1")
	enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})

	esperar(t, call(t, "GET", "/api/v1/comandas", "", e.mesero), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/comandas", "", ""), http.StatusUnauthorized, "")
	esperar(t, call(t, "GET", "/api/v1/comandas?estado=COCINANDO", "", e.cocinero), http.StatusBadRequest, "estado_invalido")
	esperar(t, call(t, "GET", "/api/v1/comandas?limit=0", "", e.cocinero), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/comandas?estacion_id=abc", "", e.cocinero), http.StatusBadRequest, "estacion_id_invalido")
	esperar(t, call(t, "GET", "/api/v1/comandas?cursor=%23%23%23", "", e.cocinero), http.StatusBadRequest, "cursor_invalido")

	crearSedeSQL(t, "COC5")
	esperar(t, call(t, "GET", "/api/v1/comandas?sucursal_id="+sedeID(t, "COC5"), "", e.cocinero),
		http.StatusNotFound, "no_encontrado")
	if got := colaDe(t, e.cocinero, url.Values{"sucursal_id": {sedeID(t, "PRINCIPAL")}, "estacion_id": {e.estacion}}).ids(); len(got) != 1 {
		t.Fatalf("la sede propia debe listar su comanda: %v", got)
	}
}
