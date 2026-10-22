package integration

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"service-restaurant/internal/modules/atencion/infrastructure/persistence/write/adapter"
)

// sqlstateLockNoDisponible es lock_not_available: lo devuelve PostgreSQL al vencer lock_timeout.
const sqlstateLockNoDisponible = "55P03"

// iniciarTx abre una transacción con un lock_timeout corto: si otra transacción retiene el
// bloqueo que se pide, la espera termina en 55P03 en lugar de colgar la prueba.
func iniciarTx(t *testing.T) *sql.Tx {
	t.Helper()
	tx, err := testDB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	if _, err := tx.ExecContext(context.Background(), `SET LOCAL lock_timeout = '300ms'`); err != nil {
		t.Fatal(err)
	}
	return tx
}

func esLockNoDisponible(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == sqlstateLockNoDisponible
}

// La baja de una zona toma la fila FOR UPDATE; el alta, edición o restauración de mesas la toma
// FOR SHARE. Con dos transacciones abiertas a la vez el orden es determinista: la que llega
// segunda no puede avanzar mientras la primera no termine, así que no queda una mesa activa en
// una zona dada de baja.
func TestZonaBajaSeSerializaConElAltaDeMesas(t *testing.T) {
	ctx := context.Background()
	zona := crearZonaSQL(t, "PRINCIPAL", "Zona serial "+sufijo())
	zonaID, err := strconv.ParseInt(zona, 10, 64)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("la baja de la zona espera a una mesa en alta", func(t *testing.T) {
		alta := iniciarTx(t)
		if _, found, err := adapter.NewMesaWriteAdapter(alta).ZonaDeMesa(ctx, zonaID); err != nil || !found {
			t.Fatalf("el alta debía bloquear la zona: found=%v err=%v", found, err)
		}
		baja := iniciarTx(t)
		_, _, err := adapter.NewZonaWriteAdapter(baja).Obtener(ctx, zonaID)
		if !esLockNoDisponible(err) {
			t.Fatalf("la baja debía esperar al alta en curso (55P03), obtuvo: %v", err)
		}
		// Terminada el alta, el bloqueo se libera y la baja puede leer la zona.
		if err := alta.Commit(); err != nil {
			t.Fatal(err)
		}
		libre := iniciarTx(t)
		if _, found, err := adapter.NewZonaWriteAdapter(libre).Obtener(ctx, zonaID); err != nil || !found {
			t.Fatalf("liberada la zona, la baja debía poder leerla: found=%v err=%v", found, err)
		}
	})

	t.Run("el alta de mesas espera a la baja de la zona", func(t *testing.T) {
		baja := iniciarTx(t)
		if _, found, err := adapter.NewZonaWriteAdapter(baja).Obtener(ctx, zonaID); err != nil || !found {
			t.Fatalf("la baja debía bloquear la zona: found=%v err=%v", found, err)
		}
		alta := iniciarTx(t)
		_, _, err := adapter.NewMesaWriteAdapter(alta).ZonaDeMesa(ctx, zonaID)
		if !esLockNoDisponible(err) {
			t.Fatalf("el alta debía esperar a la baja en curso (55P03), obtuvo: %v", err)
		}
		if err := baja.Rollback(); err != nil {
			t.Fatal(err)
		}
		libre := iniciarTx(t)
		if _, found, err := adapter.NewMesaWriteAdapter(libre).ZonaDeMesa(ctx, zonaID); err != nil || !found {
			t.Fatalf("liberada la zona, el alta debía poder leerla: found=%v err=%v", found, err)
		}
	})
}
