package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"service-restaurant/internal/modules/atencion/application/port/outbound"
)

func unica(constraint string) error {
	return fmt.Errorf("insertar: %w", &pgconn.PgError{Code: "23505", ConstraintName: constraint})
}

func TestTraducirDistingueLaRestriccionUnica(t *testing.T) {
	casos := []struct {
		nombre   string
		traducir func(error) error
		err      error
		quiere   error
	}{
		{"número de mesa", traducirMesa, unica(restriccionNumeroMesa), outbound.ErrNumeroMesaDuplicado},
		{"otra restricción en mesas", traducirMesa, unica("mesas_pkey"), outbound.ErrDuplicado},
		{"nombre de zona", traducirZona, unica(restriccionNombreZona), outbound.ErrNombreZonaDuplicado},
		{"otra restricción en zonas", traducirZona, unica("zonas_pkey"), outbound.ErrDuplicado},
		{"el texto del error no cuenta", traducirMesa, fmt.Errorf("%s: %w", restriccionNumeroMesa, &pgconn.PgError{Code: "23505", ConstraintName: "mesas_pkey"}), outbound.ErrDuplicado},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := c.traducir(c.err)
			if !errors.Is(got, c.quiere) {
				t.Fatalf("obtuvo %v, quería %v", got, c.quiere)
			}
			// Una colisión de otra restricción no debe confundirse con la específica.
			if c.quiere == outbound.ErrDuplicado &&
				(errors.Is(got, outbound.ErrNumeroMesaDuplicado) || errors.Is(got, outbound.ErrNombreZonaDuplicado)) {
				t.Fatalf("confundió la restricción: %v", got)
			}
		})
	}
	if traducirMesa(nil) != nil || traducirZona(nil) != nil {
		t.Fatal("nil debe seguir siendo nil")
	}
	tecnico := errors.New("conexión perdida")
	if got := traducirMesa(tecnico); got != tecnico {
		t.Fatalf("un error técnico pasa sin cambios: %v", got)
	}
}
