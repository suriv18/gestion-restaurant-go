package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// RES-RF-030 y RES-RF-036 (criterios de backend), comprobados de forma uniforme en cada entidad
// de I2 con baja. Las entidades «maestro» exigen motivo y guardan motivo, actor y fecha; las de
// «desactivación simple» no los piden ni los aceptan. Un fallo de un criterio es un hueco que
// se anota en la matriz de auditoría; un fallo al preparar la entidad es un defecto de esta prueba.

type bajaResp struct {
	ID       string `json:"id"`
	EsActivo bool   `json:"es_activo"`
}

type entidadBaja struct {
	nombre  string // nombre de la subprueba
	ruta    string // colección bajo /api/v1
	tabla   string // tabla en sch_restaurante
	maestro bool   // true: baja con motivo; false: desactivación simple
	// preparar crea una instancia nueva y devuelve el token que puede darla de baja y su id.
	preparar func(t *testing.T) (tk, id string)
}

// crearPorAPI hace el alta con su Idempotency-Key y devuelve el id creado.
func crearPorAPI(t *testing.T, tk, ruta, cuerpo string) string {
	t.Helper()
	r := post(t, "/api/v1/"+ruta, cuerpo, tk)
	esperar(t, r, http.StatusCreated, "")
	var b bajaResp
	r.json(t, &b)
	if b.ID == "" {
		t.Fatalf("el alta de %s no devolvió id: %s", ruta, r.Body.String())
	}
	return b.ID
}

func entidadesBaja() []entidadBaja {
	return []entidadBaja{
		{"sucursales", "sucursales", "sucursales", true, func(t *testing.T) (string, string) {
			tk := gestorToken(t)
			codigo := "A" + strings.ToUpper(sufijo())
			id := crearPorAPI(t, tk, "sucursales", altaSede(codigo))
			// Quien crea la sede no queda asignado a ella: se le asigna para poder darla de baja.
			asegurarUsuario(t, gestorUser, gestorPass, "T_SUC_GESTOR", codigo)
			return token(t, gestorUser, gestorPass), id
		}},
		{"usuarios", "usuarios", "usuarios", true, func(t *testing.T) (string, string) {
			tk := adminToken(t)
			cuerpo := fmt.Sprintf(`{"username":"aud.%s","password":"Valida2026Clave"}`, strings.ToLower(sufijo()))
			return tk, crearPorAPI(t, tk, "usuarios", cuerpo)
		}},
		{"empleados", "empleados", "empleados", true, func(t *testing.T) (string, string) {
			tk := adminToken(t)
			cuerpo := fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombres":"Aud","apellidos":"Prueba %s"}`,
				numeroUnico(8), sufijo())
			return tk, crearPorAPI(t, tk, "empleados", cuerpo)
		}},
		{"clientes", "clientes", "clientes", true, func(t *testing.T) (string, string) {
			tk := gestorDirectorio(t)
			cuerpo := fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombre_razon_social":"Aud %s"}`,
				numeroUnico(8), sufijo())
			return tk, crearPorAPI(t, tk, "clientes", cuerpo)
		}},
		{"proveedores", "proveedores", "proveedores", true, func(t *testing.T) (string, string) {
			tk := gestorDirectorio(t)
			cuerpo := fmt.Sprintf(`{"tipo_documento":"RUC","numero_documento":"20%s","razon_social":"Aud %s"}`,
				numeroUnico(9), sufijo())
			return tk, crearPorAPI(t, tk, "proveedores", cuerpo)
		}},
		{"productos", "productos", "productos", true, func(t *testing.T) (string, string) {
			tk := gestorCarta(t)
			cat := crearCategoria(t, tk, fmt.Sprintf(`{"nombre":"Aud cat %s"}`, sufijo()))
			return tk, crearPorAPI(t, tk, "productos", cuerpoProd(cat.ID, "AUD"+strings.ToUpper(sufijo())))
		}},
		{"mesas", "mesas", "mesas", true, func(t *testing.T) (string, string) {
			tk := gestorMesas(t)
			principal := idDe(t, "sucursales", "codigo", "PRINCIPAL")
			zona := crearZonaSQL(t, "PRINCIPAL", "Zona aud "+sufijo())
			return tk, crearPorAPI(t, tk, "mesas", altaMesa(principal, zona, "A"+sufijo(), 2))
		}},
		{"categorias", "categorias", "categorias", false, func(t *testing.T) (string, string) {
			tk := gestorCarta(t)
			return tk, crearPorAPI(t, tk, "categorias", fmt.Sprintf(`{"nombre":"Aud %s"}`, sufijo()))
		}},
		{"estaciones", "estaciones", "estaciones_preparacion", false, func(t *testing.T) (string, string) {
			codigo, sede := sedeEstaciones(t)
			tk := gestorEstaciones(t, codigo)
			cuerpo := cuerpoEstacion(sede, "AUD"+strings.ToUpper(sufijo()), "Aud "+sufijo())
			return tk, crearPorAPI(t, tk, "estaciones", cuerpo)
		}},
		{"zonas", "zonas", "zonas", false, func(t *testing.T) (string, string) {
			tk := gestorMesas(t)
			principal := idDe(t, "sucursales", "codigo", "PRINCIPAL")
			return tk, crearPorAPI(t, tk, "zonas", altaZona(principal, "Aud "+sufijo()))
		}},
		{"roles", "roles", "roles", false, func(t *testing.T) (string, string) {
			tk := adminToken(t)
			cuerpo := fmt.Sprintf(`{"codigo":"AUD_%s","nombre":"Aud %s"}`, strings.ToUpper(sufijo()), sufijo())
			return tk, crearPorAPI(t, tk, "roles", cuerpo)
		}},
	}
}

func TestBajasContratoComun(t *testing.T) {
	const motivo = "Baja de la prueba de contrato"
	for _, e := range entidadesBaja() {
		t.Run(e.nombre, func(t *testing.T) {
			tk, id := e.preparar(t)
			url := "/api/v1/" + e.ruta + "/" + id
			baja, rest := url+"/baja", url+"/restauracion"
			tabla := "sch_restaurante." + e.tabla
			cuerpoBaja := ""
			if e.maestro {
				cuerpoBaja = `{"motivo":"` + motivo + `"}`
			}
			eventos := func() int {
				return scalar(t, fmt.Sprintf(`SELECT count(*) FROM sch_restaurante.auditoria_eventos
					WHERE tabla='%s' AND accion='UPDATE' AND registro_id=%s`, e.tabla, id))
			}
			antes := eventos()

			t.Run("sin_idempotency_key", func(t *testing.T) {
				esperar(t, call(t, "POST", baja, cuerpoBaja, tk), http.StatusBadRequest, "idempotency_key_requerida")
			})

			t.Run("motivo", func(t *testing.T) {
				if e.maestro {
					esperar(t, post(t, baja, `{}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
					esperar(t, post(t, baja, `{"motivo":"   "}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
				} else {
					// Una desactivación simple no admite motivo: el cuerpo desconocido se rechaza.
					esperar(t, post(t, baja, `{"motivo":"x"}`, tk), http.StatusBadRequest, "")
				}
			})

			t.Run("campo_de_auditoria", func(t *testing.T) {
				// El actor sale de la sesión: un campo de auditoría en el cuerpo se rechaza y no da de baja.
				cuerpo := `{"eliminado_por":"1"}`
				if e.maestro {
					cuerpo = `{"motivo":"x","eliminado_por":"1"}`
				}
				esperar(t, post(t, baja, cuerpo, tk), http.StatusBadRequest, "")
				if n := scalar(t, fmt.Sprintf(`SELECT count(*) FROM %s WHERE id=%s AND es_activo`, tabla, id)); n != 1 {
					t.Fatalf("un cuerpo con campo de auditoría no debe dar de baja (activas: %d)", n)
				}
			})

			t.Run("baja", func(t *testing.T) {
				r := post(t, baja, cuerpoBaja, tk)
				esperar(t, r, http.StatusOK, "")
				var b bajaResp
				r.json(t, &b)
				if b.ID != id || b.EsActivo {
					t.Fatalf("respuesta de la baja: %+v (id %s)", b, id)
				}
				if e.maestro {
					n := scalar(t, fmt.Sprintf(`SELECT count(*) FROM %s WHERE id=%s AND NOT es_activo
						AND motivo_eliminacion=$1 AND eliminado_por IS NOT NULL AND eliminado_en IS NOT NULL`, tabla, id), motivo)
					if n != 1 {
						t.Fatal("un maestro dado de baja debe guardar motivo, usuario y fecha")
					}
				} else if n := scalar(t, fmt.Sprintf(`SELECT count(*) FROM %s WHERE id=%s AND NOT es_activo`, tabla, id)); n != 1 {
					t.Fatal("la desactivación simple debe dejar la fila inactiva")
				}
			})

			t.Run("restauracion", func(t *testing.T) {
				r := post(t, rest, `{}`, tk)
				esperar(t, r, http.StatusOK, "")
				var b bajaResp
				r.json(t, &b)
				if b.ID != id || !b.EsActivo {
					t.Fatalf("la restauración debe conservar el id y reactivar: %+v (id %s)", b, id)
				}
				if e.maestro {
					n := scalar(t, fmt.Sprintf(`SELECT count(*) FROM %s WHERE id=%s AND es_activo
						AND motivo_eliminacion IS NULL AND eliminado_por IS NULL AND eliminado_en IS NULL`, tabla, id))
					if n != 1 {
						t.Fatal("restaurar un maestro debe limpiar motivo, usuario y fecha de baja")
					}
				}
			})

			t.Run("auditoria", func(t *testing.T) {
				// La baja y la restauración dejan, cada una, un evento de actualización con actor y cambios.
				if despues := eventos(); despues-antes < 2 {
					t.Fatalf("eventos de auditoría nuevos: %d, quería al menos 2", despues-antes)
				}
			})

			t.Run("delete", func(t *testing.T) {
				r := call(t, "DELETE", url, "", tk)
				if r.Code != http.StatusNotFound && r.Code != http.StatusMethodNotAllowed {
					t.Fatalf("DELETE %s respondió %d, debe ser 404 o 405: %s", url, r.Code, r.Body.String())
				}
				if n := scalar(t, fmt.Sprintf(`SELECT count(*) FROM %s WHERE id=%s`, tabla, id)); n != 1 {
					t.Fatal("DELETE no debe borrar la fila")
				}
			})
		})
	}
}
