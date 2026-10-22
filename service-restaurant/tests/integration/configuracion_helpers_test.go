package integration

import (
	"context"
	"testing"

	"service-restaurant/internal/modules/identidad/infrastructure/security"
	"service-restaurant/internal/platform/database"
)

// asSystemErr es asSystem pero devuelve el error de la sentencia en lugar de abortar la prueba:
// sirve para comprobar que la base rechaza una escritura.
func asSystemErr(stmts ...string) error {
	ctx := context.Background()
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM sch_restaurante.usuarios WHERE es_sistema`).Scan(&id); err != nil {
		return err
	}
	if err := database.SetActor(ctx, tx, id); err != nil {
		return err
	}
	for _, s := range stmts {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// rolConPermisos garantiza un rol (por código) con los permisos indicados. Es idempotente.
func rolConPermisos(t *testing.T, codigo string, permisos ...string) {
	t.Helper()
	app(t)
	asSystem(t, `INSERT INTO sch_restaurante.roles(codigo,nombre)
		SELECT '`+codigo+`','`+codigo+`' WHERE NOT EXISTS (SELECT 1 FROM sch_restaurante.roles WHERE codigo='`+codigo+`')`)
	for _, p := range permisos {
		asSystem(t, `INSERT INTO sch_restaurante.roles_permisos(rol_id,permiso_id)
			SELECT r.id,p.id FROM sch_restaurante.roles r, sch_restaurante.permisos p
			WHERE r.codigo='`+codigo+`' AND p.codigo='`+p+`'
			AND NOT EXISTS (SELECT 1 FROM sch_restaurante.roles_permisos x WHERE x.rol_id=r.id AND x.permiso_id=p.id)`)
	}
}

// asegurarUsuario crea el usuario humano si no existe (con el rol dado) y le asigna las sedes
// indicadas (por código) que aún no tenga. Es idempotente: varias pruebas comparten la base.
func asegurarUsuario(t *testing.T, username, password, rol string, sedes ...string) {
	t.Helper()
	app(t)
	if scalar(t, `SELECT count(*) FROM sch_restaurante.usuarios WHERE username='`+username+`'`) == 0 {
		h, err := security.NewArgon2Hasher(security.Argon2ParamsTest)
		if err != nil {
			t.Fatal(err)
		}
		hash, _ := h.Hash(password)
		asSystem(t,
			`INSERT INTO sch_restaurante.usuarios(username, password_hash) VALUES ('`+username+`', '`+hash+`')`,
			`INSERT INTO sch_restaurante.usuarios_roles(usuario_id, rol_id)
			 SELECT u.id, r.id FROM sch_restaurante.usuarios u, sch_restaurante.roles r
			 WHERE u.username='`+username+`' AND r.codigo='`+rol+`'`)
	}
	for _, s := range sedes {
		asSystem(t, `INSERT INTO sch_restaurante.usuarios_sucursales(usuario_id, sucursal_id)
			SELECT u.id, s.id FROM sch_restaurante.usuarios u, sch_restaurante.sucursales s
			WHERE u.username='`+username+`' AND s.codigo='`+s+`'
			AND NOT EXISTS (SELECT 1 FROM sch_restaurante.usuarios_sucursales x WHERE x.usuario_id=u.id AND x.sucursal_id=s.id)`)
	}
}

// crearSedeSQL inserta una sede activa si no existe (para preparar escenarios).
func crearSedeSQL(t *testing.T, codigo string) {
	t.Helper()
	app(t)
	asSystem(t, `INSERT INTO sch_restaurante.sucursales(negocio_id, codigo, nombre, direccion)
		SELECT id, '`+codigo+`', 'Sede `+codigo+`', 'Calle de prueba 1' FROM sch_restaurante.negocio
		WHERE NOT EXISTS (SELECT 1 FROM sch_restaurante.sucursales WHERE codigo='`+codigo+`')`)
}

// sedeResp es la sucursal tal como la devuelve la API.
type sedeResp struct {
	ID           string  `json:"id"`
	NegocioID    string  `json:"negocio_id"`
	Codigo       string  `json:"codigo"`
	Nombre       string  `json:"nombre"`
	Direccion    string  `json:"direccion"`
	Ubigeo       *string `json:"ubigeo"`
	CodigoSunat  *string `json:"codigo_establecimiento_sunat"`
	Telefono     *string `json:"telefono"`
	EsActivo     bool    `json:"es_activo"`
	Motivo       *string `json:"motivo_eliminacion"`
	EliminadoPor *string `json:"eliminado_por"`
	CreadoPor    string  `json:"creado_por"`
}
