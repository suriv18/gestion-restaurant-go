package controller

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"service-restaurant/internal/modules/atencion/web/dto/request"
	"service-restaurant/internal/shared/kernel/apperror"
)

func TestIdCadenaSoloAceptaDecimalCanonico(t *testing.T) {
	validos := map[string]int64{"1": 1, "5": 5, "42": 42, "9223372036854775807": 9223372036854775807}
	for entrada, quiere := range validos {
		if got, err := idCadena(entrada, "zona_id"); err != nil || got != quiere {
			t.Errorf("%q: obtuvo %d, %v; quería %d", entrada, got, err, quiere)
		}
	}
	for _, entrada := range []string{"", "0", "-1", "+5", "007", "05", "1.0", "1e3", " 5", "5 ", "abc", "9223372036854775808"} {
		_, err := idCadena(entrada, "zona_id")
		ae, ok := err.(*apperror.AppError)
		if !ok || ae.Kind != apperror.KindBadRequest || ae.Code != "zona_id_invalido" {
			t.Errorf("%q debía rechazarse con zona_id_invalido: %v", entrada, err)
		}
	}
}

// cuerpoFragmentado arma una petición cuyo tamaño se desconoce de antemano (Transfer-Encoding:
// chunked), como la que llega de un cliente que no envía Content-Length.
func cuerpoFragmentado(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/zonas/1/baja", io.NopCloser(strings.NewReader(cuerpo)))
	r.ContentLength = -1
	r.TransferEncoding = []string{"chunked"}
	if cuerpo != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func TestLeerCuerpoOpcionalVacio(t *testing.T) {
	casos := []struct {
		nombre string
		req    func() *http.Request
		code   string // vacío: sin error
	}{
		{"sin cuerpo", func() *http.Request { return httptest.NewRequest(http.MethodPost, "/x", nil) }, ""},
		{"longitud cero", func() *http.Request { return httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("")) }, ""},
		{"fragmentado vacío", func() *http.Request { return cuerpoFragmentado("") }, ""},
		{"objeto vacío", func() *http.Request { return httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("{}")) }, ""},
		{"fragmentado con objeto vacío", func() *http.Request { return cuerpoFragmentado("{}") }, ""},
		{"fragmentado con campo desconocido", func() *http.Request { return cuerpoFragmentado(`{"x":1}`) }, "json_invalido"},
		{"fragmentado con JSON roto", func() *http.Request { return cuerpoFragmentado(`{`) }, "json_invalido"},
		{"con longitud y campo desconocido", func() *http.Request {
			r := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"x":1}`))
			r.Header.Set("Content-Type", "application/json")
			return r
		}, "json_invalido"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			var dst request.Confirmacion
			err := leerCuerpoOpcionalVacio(c.req(), &dst)
			if c.code == "" {
				if err != nil {
					t.Fatalf("no debía fallar: %v", err)
				}
				return
			}
			ae, ok := err.(*apperror.AppError)
			if !ok || ae.Code != c.code {
				t.Fatalf("error %v, quería %s", err, c.code)
			}
		})
	}
}

func TestEnumDe(t *testing.T) {
	q := url.Values{"estado": {"REGISTRADO"}, "tipo": {"MOSTRADOR"}}
	if v, err := enumDe(q, "estado", "REGISTRADO", "LISTO"); err != nil || v != "REGISTRADO" {
		t.Fatalf("valor válido: %q %v", v, err)
	}
	if v, err := enumDe(url.Values{}, "estado", "REGISTRADO"); err != nil || v != "" {
		t.Fatalf("ausente no filtra: %q %v", v, err)
	}
	if _, err := enumDe(q, "tipo", "SALON", "LLEVAR"); err == nil {
		t.Fatal("un valor fuera del conjunto debe rechazarse")
	}
}

func TestPathIDDe(t *testing.T) {
	r := httptest.NewRequest("GET", "/x", nil)
	r.SetPathValue("detalle_id", "12")
	if id, err := pathIDDe(r, "detalle_id"); err != nil || id != 12 {
		t.Fatalf("pathIDDe = %d, %v", id, err)
	}
	r.SetPathValue("detalle_id", "0")
	if _, err := pathIDDe(r, "detalle_id"); err == nil {
		t.Fatal("el id debe ser positivo")
	}
}
