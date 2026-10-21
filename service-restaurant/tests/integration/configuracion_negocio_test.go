package integration

import (
	"net/http"
	"testing"
)

type negocioResp struct {
	ID              string  `json:"id"`
	Singleton       bool    `json:"singleton"`
	RUC             string  `json:"ruc"`
	RazonSocial     string  `json:"razon_social"`
	NombreComercial string  `json:"nombre_comercial"`
	DireccionFiscal string  `json:"direccion_fiscal"`
	Ubigeo          *string `json:"ubigeo"`
	Telefono        *string `json:"telefono"`
	Email           *string `json:"email"`
	Moneda          string  `json:"moneda"`
	ZonaHoraria     string  `json:"zona_horaria"`
	ActualizadoPor  string  `json:"actualizado_por"`
}

func TestNegocioLecturaYPermisos(t *testing.T) {
	rolConPermisos(t, "T_NEGOCIO_VER", "NEGOCIO_VER")
	asegurarUsuario(t, "neg.ver", "NegVer2026Clave", "T_NEGOCIO_VER", "PRINCIPAL")
	asegurarUsuario(t, "neg.sin", "NegSin2026Clave", "CAJERO", "PRINCIPAL")

	esperar(t, call(t, "GET", "/api/v1/negocio", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, call(t, "GET", "/api/v1/negocio", "", token(t, "neg.sin", "NegSin2026Clave")), http.StatusForbidden, "sin_permiso")

	for _, tk := range []string{adminToken(t), token(t, "neg.ver", "NegVer2026Clave")} {
		r := call(t, "GET", "/api/v1/negocio", "", tk)
		esperar(t, r, http.StatusOK, "")
		var n negocioResp
		r.json(t, &n)
		if n.RUC != "20123456789" || !n.Singleton || n.Moneda != "PEN" || n.ZonaHoraria != "America/Lima" || n.ID == "" {
			t.Fatalf("negocio: %+v", n)
		}
	}
	// Solo lectura: editar exige NEGOCIO_ADMINISTRAR.
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"nombre_comercial":"X"}`, token(t, "neg.ver", "NegVer2026Clave"), nil),
		http.StatusForbidden, "sin_permiso")
}

func TestNegocioEdicionParcialYValidaciones(t *testing.T) {
	tk := adminToken(t)
	t.Cleanup(func() {
		callH(t, "PATCH", "/api/v1/negocio", `{"nombre_comercial":"Prueba","telefono":null,"email":null,"ubigeo":null}`, tk, nil)
	})

	r := callH(t, "PATCH", "/api/v1/negocio",
		`{"nombre_comercial":"La Casona","telefono":"01-555-0100","email":"hola@lacasona.pe","ubigeo":"150101"}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	var n negocioResp
	r.json(t, &n)
	if n.NombreComercial != "La Casona" || n.Telefono == nil || *n.Telefono != "01-555-0100" ||
		n.Email == nil || *n.Email != "hola@lacasona.pe" || n.Ubigeo == nil || *n.Ubigeo != "150101" ||
		n.RUC != "20123456789" || n.RazonSocial != "Restaurante de Prueba SAC" {
		t.Fatalf("edición parcial: %+v", n)
	}
	if n.ActualizadoPor != idDe(t, "usuarios", "username", "admin") {
		t.Errorf("actualizado_por = %s, debe ser el administrador de la sesión", n.ActualizadoPor)
	}

	// null vacía solo el campo indicado.
	r = callH(t, "PATCH", "/api/v1/negocio", `{"telefono":null}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &n)
	if n.Telefono != nil || n.NombreComercial != "La Casona" || n.Email == nil {
		t.Fatalf("null debe vaciar solo el teléfono: %+v", n)
	}

	// Normaliza la moneda.
	r = callH(t, "PATCH", "/api/v1/negocio", `{"moneda":" pen "}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &n)
	if n.Moneda != "PEN" {
		t.Fatalf("moneda = %q", n.Moneda)
	}

	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{}`, tk, nil), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"ruc":"20999999999"}`, tk, nil), http.StatusBadRequest, "json_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"singleton":false}`, tk, nil), http.StatusBadRequest, "json_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"moneda":"pe"}`, tk, nil), http.StatusUnprocessableEntity, "moneda_invalida")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"zona_horaria":"Marte/Olimpo"}`, tk, nil), http.StatusUnprocessableEntity, "zona_horaria_invalida")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"ubigeo":"12"}`, tk, nil), http.StatusUnprocessableEntity, "ubigeo_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"email":"no-es-correo"}`, tk, nil), http.StatusUnprocessableEntity, "email_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"razon_social":"   "}`, tk, nil), http.StatusUnprocessableEntity, "razon_social_invalida")

	// Los rechazos no modifican nada.
	if scalar(t, `SELECT count(*) FROM sch_restaurante.negocio WHERE ruc='20123456789' AND moneda='PEN' AND razon_social='Restaurante de Prueba SAC'`) != 1 {
		t.Fatal("un rechazo no debe alterar el negocio")
	}
}

func TestNegocioEdicionQuedaAuditadaConElActor(t *testing.T) {
	tk := adminToken(t)
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"nombre_comercial":"Prueba"}`, tk, nil), http.StatusOK, "")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos
		WHERE tabla='negocio' AND accion='UPDATE' AND creado_por=`+idDe(t, "usuarios", "username", "admin")); n == 0 {
		t.Fatal("la edición del negocio debe quedar en auditoria_eventos con el actor de la sesión")
	}
}
