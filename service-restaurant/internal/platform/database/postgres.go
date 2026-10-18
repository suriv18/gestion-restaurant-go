// Package database abre el pool PostgreSQL y define la interfaz técnica DBTX.
package database

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DBTX la implementan *sql.DB y *sql.Tx; vive solo en infraestructura/plataforma.
type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Open abre y verifica la conexión.
func Open(ctx context.Context, url string) (*sql.DB, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
