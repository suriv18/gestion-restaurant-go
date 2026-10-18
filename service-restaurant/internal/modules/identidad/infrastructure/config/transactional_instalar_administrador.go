package config

import (
	"context"
	"database/sql"
	"log/slog"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/usecase/command"
	"service-restaurant/internal/platform/transaction"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// AdminParams son los datos del alta inicial (los entrega bootstrap desde cmd/admin).
type AdminParams struct {
	Username, Password                                 string
	RUC, RazonSocial, NombreComercial, DireccionFiscal string
	SucursalCodigo, SucursalNombre, SucursalDireccion  string
}

// AdminInstalado es el resultado de la instalación inicial.
type AdminInstalado struct{ UsuarioID, SucursalID int64 }

type transactionalInstalarAdministrador struct {
	db     *sql.DB
	logger *slog.Logger
	build  func(txQuerier) *command.InstalarAdministrador
}

func (t *transactionalInstalarAdministrador) Execute(ctx context.Context, c cmd.InstalarAdministrador) (appresult.Result[dtoresult.AdministradorInstalado], error) {
	res, err := transaction.Run(ctx, t.db, nil, t.logger,
		func(ctx context.Context, tx *sql.Tx) (appresult.Result[dtoresult.AdministradorInstalado], error) {
			return t.build(tx).Execute(ctx, c)
		},
		appresult.CanCommit[dtoresult.AdministradorInstalado],
	)
	if err != nil {
		return appresult.Result[dtoresult.AdministradorInstalado]{}, err
	}
	if !res.IsValid() {
		return appresult.Result[dtoresult.AdministradorInstalado]{}, errInvalidResult
	}
	return res, nil
}

// InstalarAdministrador crea el primer administrador. Devuelve un *apperror.AppError si la
// instalación se rechaza (datos inválidos o ya instalado).
func (m *Module) InstalarAdministrador(ctx context.Context, p AdminParams) (AdminInstalado, error) {
	res, err := m.instalar.Execute(ctx, cmd.InstalarAdministrador{
		Username: p.Username, Password: p.Password,
		RUC: p.RUC, RazonSocial: p.RazonSocial, NombreComercial: p.NombreComercial, DireccionFiscal: p.DireccionFiscal,
		SucursalCodigo: p.SucursalCodigo, SucursalNombre: p.SucursalNombre, SucursalDireccion: p.SucursalDireccion,
	})
	if err != nil {
		return AdminInstalado{}, err
	}
	if ae := res.Err(); ae != nil {
		return AdminInstalado{}, ae
	}
	return AdminInstalado{UsuarioID: res.Value().UsuarioID, SucursalID: res.Value().SucursalID}, nil
}
