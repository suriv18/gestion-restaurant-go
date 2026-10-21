// Package config compone el módulo atencion: conecta los puertos de aplicación con sus
// adaptadores concretos y decora los casos de uso con transacciones.
package config

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/platform/database"
	"service-restaurant/internal/platform/idempotency"
	"service-restaurant/internal/platform/transaction"
	appresult "service-restaurant/internal/shared/kernel/result"
)

var errInvalidResult = errors.New("el caso de uso devolvió un Result inválido")

// txQuerier es lo que los repositorios necesitan del *sql.Tx; la aplicación nunca lo ve.
type txQuerier = database.DBTX

// ejecutor es el caso de uso que se decora.
type ejecutor[C, T any] interface {
	Execute(context.Context, C) (appresult.Result[T], error)
}

// escritura decora un caso de uso de escritura: lo ejecuta en una transacción con el actor
// autenticado fijado (app.usuario_id) y, si la solicitud trae clave de idempotencia, registra
// la clave en esa misma transacción. Solo un Result exitoso se confirma; un rechazo o un error
// revierte todo y no deja rastro, de modo que el reintento se evalúa de nuevo.
type escritura[C, T any] struct {
	db     *sql.DB
	logger *slog.Logger
	status int // estado HTTP que se recuerda para repeticiones idempotentes
	build  func(txQuerier) ejecutor[C, T]
	meta   func(C) (actorID int64, intencion cmd.Intencion)
}

func (e escritura[C, T]) Execute(ctx context.Context, c C) (appresult.Result[T], error) {
	actorID, in := e.meta(c)
	trabajo := func(ctx context.Context, tx *sql.Tx) (appresult.Result[T], error) {
		return e.build(tx).Execute(ctx, c)
	}

	if in.Clave == "" {
		res, err := transaction.Run(ctx, e.db, nil, e.logger, trabajo, appresult.CanCommit[T], transaction.WithActor(actorID))
		return validar(res, err)
	}
	out, err := idempotency.Run(ctx, e.db, e.logger, idempotency.Request{
		Key: in.Clave, UsuarioID: actorID, Operacion: in.Operacion, Huella: in.Huella,
		Status: e.status, Now: time.Now().UTC(),
	}, trabajo, transaction.WithActor(actorID))
	if err != nil {
		return appresult.Result[T]{}, err
	}
	return validar(out.Result, nil)
}

func validar[T any](res appresult.Result[T], err error) (appresult.Result[T], error) {
	if err != nil {
		return appresult.Result[T]{}, err
	}
	if !res.IsValid() {
		return appresult.Result[T]{}, errInvalidResult
	}
	return res, nil
}
