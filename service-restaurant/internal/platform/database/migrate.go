package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

// NewMigrator crea un proveedor Goose sobre las migraciones embebidas.
func NewMigrator(db *sql.DB, migrations fs.FS) (*goose.Provider, error) {
	// Las migraciones fijan search_path = sch_restaurante, pg_catalog; la tabla de
	// versiones se califica para que Goose la encuentre dentro de esa transaccion.
	p, err := goose.NewProvider(goose.DialectPostgres, db, migrations,
		goose.WithTableName("public.goose_db_version"))
	if err != nil {
		return nil, fmt.Errorf("database: migrator: %w", err)
	}
	return p, nil
}

// MigrateUp aplica todas las migraciones pendientes.
func MigrateUp(ctx context.Context, db *sql.DB, migrations fs.FS) error {
	p, err := NewMigrator(db, migrations)
	if err != nil {
		return err
	}
	if _, err := p.Up(ctx); err != nil {
		return fmt.Errorf("database: migrate up: %w", err)
	}
	return nil
}
