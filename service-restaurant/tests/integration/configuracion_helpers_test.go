package integration

import (
	"context"

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
