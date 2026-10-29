package integration

import (
	"net/http"
	"testing"
)

func transicionar(t *testing.T, tk, lineaComanda, estado string) resp {
	t.Helper()
	return post(t, "/api/v1/comanda-detalles/"+lineaComanda+"/transiciones", `{"estado":"`+estado+`"}`, tk)
}

func transicionarOK(t *testing.T, tk, lineaComanda, estado string) comandaLineaResp {
	t.Helper()
	r := transicionar(t, tk, lineaComanda, estado)
	esperar(t, r, http.StatusOK, "")
	var l comandaLineaResp
	r.json(t, &l)
	return l
}

// RES-CP-012: la cocina avanza PENDIENTE → PREPARACION → LISTO → ENTREGADO y queda la marca de cada paso.
func TestLineaDeComandaRecorreSusEstados(t *testing.T) {
	e := nuevoEscenario(t, "2")
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	lc := c.Lineas[0].ID

	l := transicionarOK(t, e.cocinero, lc, "PREPARACION")
	if l.Estado != "PREPARACION" || l.IniciadoEn == nil || l.ListoEn != nil || l.EntregadoEn != nil {
		t.Fatalf("PREPARACION: %+v", l)
	}
	l = transicionarOK(t, e.cocinero, lc, "LISTO")
	if l.Estado != "LISTO" || l.IniciadoEn == nil || l.ListoEn == nil || l.EntregadoEn != nil {
		t.Fatalf("LISTO: %+v", l)
	}
	l = transicionarOK(t, e.cocinero, lc, "ENTREGADO")
	if l.Estado != "ENTREGADO" || l.EntregadoEn == nil || l.Cantidad != "2.000" || l.ComandaID != c.ID {
		t.Fatalf("ENTREGADO: %+v", l)
	}
}

func TestLineaDeComandaRechazaTransicionesInvalidas(t *testing.T) {
	e := nuevoEscenario(t, "2")
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	lc := c.Lineas[0].ID

	esperar(t, transicionar(t, e.cocinero, lc, "LISTO"), http.StatusConflict, "transicion_invalida")
	esperar(t, transicionar(t, e.cocinero, lc, "ENTREGADO"), http.StatusConflict, "transicion_invalida")
	esperar(t, transicionar(t, e.cocinero, lc, "CANCELADO"), http.StatusConflict, "transicion_invalida")
	esperar(t, transicionar(t, e.cocinero, lc, "COCINANDO"), http.StatusUnprocessableEntity, "estado_invalido")

	transicionarOK(t, e.cocinero, lc, "PREPARACION")
	esperar(t, transicionar(t, e.cocinero, lc, "PREPARACION"), http.StatusConflict, "transicion_invalida")
	transicionarOK(t, e.cocinero, lc, "LISTO")
	esperar(t, transicionar(t, e.cocinero, lc, "PREPARACION"), http.StatusConflict, "transicion_invalida")

	esperar(t, transicionar(t, e.cocinero, "999999999", "PREPARACION"), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/comanda-detalles/"+lc+"/transiciones", `{"estado":"LISTO","otro":1}`, e.cocinero),
		http.StatusBadRequest, "")
}

func TestLaEntregaLaConfirmaElMeseroPeroNoLaPreparacion(t *testing.T) {
	e := nuevoEscenario(t, "2")
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	lc := c.Lineas[0].ID

	esperar(t, transicionar(t, e.mesero, lc, "PREPARACION"), http.StatusForbidden, "sin_permiso")
	transicionarOK(t, e.cocinero, lc, "PREPARACION")
	transicionarOK(t, e.cocinero, lc, "LISTO")
	esperar(t, transicionar(t, lectorPedidos(t), lc, "ENTREGADO"), http.StatusForbidden, "sin_permiso")
	if l := transicionarOK(t, e.mesero, lc, "ENTREGADO"); l.Estado != "ENTREGADO" {
		t.Fatalf("el mesero debía confirmar la entrega: %+v", l)
	}
}

func TestTransicionNoCruzaSedes(t *testing.T) {
	e := nuevoEscenario(t, "2")
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	// Un cocinero que solo pertenece a COC4: `tokenPedidos` añadiría PRINCIPAL, así que se arma a mano.
	crearSedeSQL(t, "COC4")
	rolConPermisos(t, "T_COC_OTRA", "COCINA_OPERAR", "PEDIDO_VER")
	asegurarUsuario(t, "coc.cocinero.otra", "Pedidos2026Clave", "T_COC_OTRA", "COC4")
	otra := token(t, "coc.cocinero.otra", "Pedidos2026Clave")
	esperar(t, transicionar(t, otra, c.Lineas[0].ID, "PREPARACION"), http.StatusNotFound, "no_encontrado")
}
