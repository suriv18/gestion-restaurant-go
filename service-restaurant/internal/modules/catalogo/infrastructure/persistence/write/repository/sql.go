// Package repository contiene el SQL parametrizado de escritura del directorio de clientes y
// proveedores. Recibe un database.DBTX: en operaciones de escritura es el *sql.Tx de la unidad
// de trabajo.
package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"service-restaurant/internal/modules/catalogo/application/port/outbound"
)

const codigoUniqueViolation = "23505"

// Restricciones únicas que tienen un rechazo propio: el documento de cada maestro.
const (
	restriccionDocumentoCliente   = "clientes_tipo_documento_numero_documento_key"
	restriccionDocumentoProveedor = "proveedores_tipo_documento_numero_documento_key"
	restriccionNombreCategoria    = "categorias_nombre_key"
	restriccionCodigoEstacion     = "estaciones_preparacion_sucursal_id_codigo_key"
	restriccionCodigoProducto     = "productos_codigo_key"
	// FK compuesta (estacion_id, sucursal_id): la estación debe ser de la misma sede.
	restriccionEstacionSede = "productos_sucursal_estacion_id_sucursal_id_fkey"
)

// Translate convierte los errores de PostgreSQL conocidos en los errores de puerto que los
// casos de uso entienden como rechazos esperados. Cualquier otro error pasa sin cambios.
func Translate(err error) error {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	switch pg.Code {
	case codigoUniqueViolation:
		if pg.ConstraintName == restriccionDocumentoCliente || pg.ConstraintName == restriccionDocumentoProveedor {
			return outbound.ErrDocumentoDuplicado
		}
		if pg.ConstraintName == restriccionCodigoEstacion {
			return outbound.ErrEstacionDuplicada
		}
		if pg.ConstraintName == restriccionCodigoProducto {
			return outbound.ErrProductoDuplicado
		}
		if pg.ConstraintName == restriccionNombreCategoria {
			return outbound.ErrCategoriaDuplicada
		}
		return fmt.Errorf("%w (%s)", outbound.ErrDuplicado, pg.ConstraintName)
	case "23P01": // exclusion_violation: vigencias de una tasa solapadas
		return outbound.ErrVigenciaSolapada
	case "23503": // foreign_key_violation
		if pg.ConstraintName == restriccionEstacionSede {
			return outbound.ErrEstacionOtraSede
		}
		return fmt.Errorf("%w (%s)", outbound.ErrReferenciaInvalida, pg.ConstraintName)
	case "23514": // check_violation
		return &outbound.ErrReglaDatos{Mensaje: "el valor no cumple una restricción de datos"}
	case "P0001": // RAISE EXCEPTION de las reglas de la V4: mensajes curados del esquema
		return &outbound.ErrReglaDatos{Mensaje: pg.Message}
	}
	return err
}

func notFound(err error) (bool, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	return false, err
}
