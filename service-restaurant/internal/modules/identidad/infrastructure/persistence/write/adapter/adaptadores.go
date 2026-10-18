// Package adapter implementa los puertos de salida de escritura sobre los repositorios SQL.
package adapter

import (
	"context"
	"database/sql"
	"time"

	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/model"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/write/mapper"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// UsuarioWriteAdapter implementa outbound.UsuarioStore y outbound.ActorContext.
type UsuarioWriteAdapter struct {
	db   database.DBTX
	repo *repository.UsuarioSQLWriter
}

func NewUsuarioWriteAdapter(db database.DBTX) *UsuarioWriteAdapter {
	return &UsuarioWriteAdapter{db: db, repo: repository.NewUsuarioSQLWriter(db)}
}

var (
	_ outbound.UsuarioStore = (*UsuarioWriteAdapter)(nil)
	_ outbound.ActorContext = (*UsuarioWriteAdapter)(nil)
)

func (a *UsuarioWriteAdapter) BuscarPorUsernameParaActualizar(ctx context.Context, key string) (model.Usuario, bool, error) {
	row, found, err := a.repo.FindByKeyForUpdate(ctx, key)
	if err != nil || !found {
		return model.Usuario{}, false, err
	}
	return mapper.UsuarioAModelo(row), true, nil
}

func (a *UsuarioWriteAdapter) GuardarIntentos(ctx context.Context, u model.Usuario) error {
	var hasta sql.NullTime
	if u.BloqueadoHasta != nil {
		hasta = sql.NullTime{Time: *u.BloqueadoHasta, Valid: true}
	}
	return a.repo.UpdateIntentos(ctx, u.ID, u.IntentosFallidos, hasta)
}

func (a *UsuarioWriteAdapter) RegistrarAcceso(ctx context.Context, u model.Usuario, ahora time.Time) error {
	return a.repo.UpdateAcceso(ctx, u.ID, ahora)
}

// Fijar establece app.usuario_id en la transacción en curso (a.db debe ser el *sql.Tx).
func (a *UsuarioWriteAdapter) Fijar(ctx context.Context, usuarioID int64) error {
	return database.SetActor(ctx, a.db, usuarioID)
}

func (a *UsuarioWriteAdapter) UsuarioSistemaID(ctx context.Context) (int64, error) {
	return a.repo.SistemaID(ctx)
}

// SesionWriteAdapter implementa outbound.SesionWriter.
type SesionWriteAdapter struct{ repo *repository.SesionSQLWriter }

func NewSesionWriteAdapter(db database.DBTX) *SesionWriteAdapter {
	return &SesionWriteAdapter{repo: repository.NewSesionSQLWriter(db)}
}

var _ outbound.SesionWriter = (*SesionWriteAdapter)(nil)

func (a *SesionWriteAdapter) Crear(ctx context.Context, s model.Sesion) (model.Sesion, error) {
	id, err := a.repo.Insert(ctx, mapper.SesionARow(s))
	if err != nil {
		return model.Sesion{}, err
	}
	s.ID = id
	return s, nil
}

func (a *SesionWriteAdapter) Revocar(ctx context.Context, sesionID int64, ahora time.Time) error {
	return a.repo.Revoke(ctx, sesionID, ahora)
}

func (a *SesionWriteAdapter) RegistrarUso(ctx context.Context, s model.Sesion) error {
	return a.repo.Touch(ctx, s.ID, s.UltimoUsoEn, s.ExpiraInactividadEn)
}

// InstalacionWriteAdapter implementa outbound.InstalacionStore.
type InstalacionWriteAdapter struct {
	repo *repository.InstalacionSQLWriter
}

func NewInstalacionWriteAdapter(db database.DBTX) *InstalacionWriteAdapter {
	return &InstalacionWriteAdapter{repo: repository.NewInstalacionSQLWriter(db)}
}

var _ outbound.InstalacionStore = (*InstalacionWriteAdapter)(nil)

func (a *InstalacionWriteAdapter) ExisteUsuarioHumano(ctx context.Context) (bool, error) {
	return a.repo.ExisteUsuarioHumano(ctx)
}

func (a *InstalacionWriteAdapter) AsegurarSucursal(ctx context.Context, d outbound.DatosInstalacion) (int64, error) {
	negocioID, found, err := a.repo.NegocioID(ctx)
	if err != nil {
		return 0, err
	}
	if !found {
		if negocioID, err = a.repo.InsertNegocio(ctx, d.RUC, d.RazonSocial, d.NombreComercial, d.DireccionFiscal); err != nil {
			return 0, err
		}
	}
	if id, found, err := a.repo.SucursalPorCodigo(ctx, d.SucursalCodigo); err != nil || found {
		return id, err
	}
	return a.repo.InsertSucursal(ctx, negocioID, d.SucursalCodigo, d.SucursalNombre, d.SucursalDireccion)
}

func (a *InstalacionWriteAdapter) CrearAdministrador(ctx context.Context, username, passwordHash string, sucursalID int64) (int64, error) {
	return a.repo.InsertAdministrador(ctx, username, passwordHash, sucursalID)
}
