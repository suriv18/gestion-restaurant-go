// Package transaction ofrece la única API de transacciones, sin conocer módulos.
package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"service-restaurant/internal/platform/database"
)

// Option ajusta la transacción recién abierta.
type Option func(ctx context.Context, tx *sql.Tx) error

// WithActor fija app.usuario_id local a la transacción (decisión 12 de la V4). El id debe
// provenir de la sesión validada, nunca del cliente. Sin actor, la base rechaza escribir.
func WithActor(usuarioID int64) Option {
	return func(ctx context.Context, tx *sql.Tx) error { return database.SetActor(ctx, tx, usuarioID) }
}

// Run ejecuta work dentro de una transacción. Hace commit solo si work no
// devuelve error y canCommit(resultado) es true; en otro caso rollback.
// Un rechazo esperado (canCommit=false) devuelve el resultado con error nil.
func Run[T any](
	ctx context.Context,
	db *sql.DB,
	opts *sql.TxOptions,
	logger *slog.Logger,
	work func(ctx context.Context, tx *sql.Tx) (T, error),
	canCommit func(T) bool,
	options ...Option,
) (T, error) {
	var zero T
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return zero, fmt.Errorf("transaction: begin: %w", err)
	}
	done := false
	defer func() {
		if !done {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				logger.Error("transaction: rollback falló", "error", rbErr)
			}
		}
	}()

	for _, o := range options {
		if err := o(ctx, tx); err != nil {
			return zero, fmt.Errorf("transaction: preparar: %w", err)
		}
	}
	res, err := work(ctx, tx)
	if err != nil {
		return zero, err
	}
	done = true
	if !canCommit(res) {
		if rbErr := tx.Rollback(); rbErr != nil {
			logger.Error("transaction: rollback falló", "error", rbErr)
		}
		return res, nil
	}
	if err := tx.Commit(); err != nil {
		return zero, fmt.Errorf("transaction: commit: %w", err)
	}
	return res, nil
}
