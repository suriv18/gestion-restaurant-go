package config

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/usecase/command"
	"service-restaurant/internal/platform/database"
	"service-restaurant/internal/platform/transaction"
	appresult "service-restaurant/internal/shared/kernel/result"
)

var errInvalidResult = errors.New("el caso de uso devolvió un Result inválido")

// txQuerier es lo que los repositorios necesitan del *sql.Tx; la aplicación nunca lo ve.
type txQuerier = database.DBTX

// transactionalIniciarSesion ejecuta el login dentro de una transacción.
//
// Regla clave: un rechazo por contraseña incorrecta SÍ se confirma. Ese rechazo ya
// incrementó intentos_fallidos (y quizá fijó bloqueado_hasta); si se revirtiera, el
// bloqueo nunca se acumularía. Cualquier otro rechazo o error técnico hace rollback.
type transactionalIniciarSesion struct {
	db     *sql.DB
	logger *slog.Logger
	build  func(txQuerier) *command.IniciarSesion
}

func (t *transactionalIniciarSesion) Execute(ctx context.Context, c cmd.IniciarSesion) (appresult.Result[dtoresult.SesionIniciada], error) {
	res, err := transaction.Run(ctx, t.db, nil, t.logger,
		func(ctx context.Context, tx *sql.Tx) (appresult.Result[dtoresult.SesionIniciada], error) {
			return t.build(tx).Execute(ctx, c)
		},
		confirmaLogin,
	)
	if err != nil {
		return appresult.Result[dtoresult.SesionIniciada]{}, err
	}
	if !res.IsValid() {
		return appresult.Result[dtoresult.SesionIniciada]{}, errInvalidResult
	}
	return res, nil
}

func confirmaLogin(r appresult.Result[dtoresult.SesionIniciada]) bool {
	if r.IsOk() {
		return true
	}
	return r.IsValid() && r.Err() != nil && r.Err().Code == mapper.CodigoCredencialesInvalidas
}
