package database

import (
	"context"
	"strconv"
)

// SetActor fija app.usuario_id local a la transacción (decisión 12 de la V4).
// Debe ejecutarse sobre el *sql.Tx de la operación: set_config(..., true) solo vale
// dentro de la transacción en curso. El id proviene de la sesión validada, nunca del cliente.
func SetActor(ctx context.Context, q DBTX, usuarioID int64) error {
	_, err := q.ExecContext(ctx, `SELECT set_config('app.usuario_id', $1, true)`, strconv.FormatInt(usuarioID, 10))
	return err
}
