// Package config compone el módulo identidad: conecta los puertos de aplicación con sus
// adaptadores concretos y decora los casos de uso con transacciones.
package config

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"

	"service-restaurant/internal/modules/identidad/application/dto/query"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/inbound"
	"service-restaurant/internal/modules/identidad/application/usecase/command"
	uquery "service-restaurant/internal/modules/identidad/application/usecase/query"
	"service-restaurant/internal/modules/identidad/domain/policy"
	readadapter "service-restaurant/internal/modules/identidad/infrastructure/persistence/read/adapter"
	writeadapter "service-restaurant/internal/modules/identidad/infrastructure/persistence/write/adapter"
	"service-restaurant/internal/modules/identidad/infrastructure/security"
	"service-restaurant/internal/modules/identidad/web/controller"
	platsec "service-restaurant/internal/platform/security"
	appctx "service-restaurant/internal/shared/application/context"
)

// Argon2Cost elige el costo de argon2id. Argon2Light existe solo para pruebas.
type Argon2Cost int

const (
	Argon2Default Argon2Cost = iota
	Argon2Light
)

// Deps son las dependencias técnicas del módulo.
type Deps struct {
	DB          *sql.DB
	Logger      *slog.Logger
	JWTSecret   []byte
	JWTIssuer   string
	JWTAudience string
	Argon2      Argon2Cost
	Bloqueo     policy.BloqueoPorIntentos
	Sesion      policy.PoliticaSesion
}

// withDefaults completa las políticas con sus valores por defecto si vienen vacías.
func (d Deps) withDefaults() Deps {
	if d.Bloqueo.MaxIntentos == 0 {
		d.Bloqueo = policy.BloqueoPorDefecto
	}
	if d.Sesion.Inactividad == 0 {
		d.Sesion = policy.SesionPorDefecto
	}
	return d
}

// Module es el módulo identidad ya ensamblado.
type Module struct {
	deps   Deps
	hasher *security.Argon2Hasher
	admin  *administracion

	iniciar    inbound.IniciarSesion
	cerrar     inbound.CerrarSesion
	sesion     inbound.ObtenerSesion
	autenticar inbound.AutenticarToken
	instalar   inbound.InstalarAdministrador
}

// New ensambla el módulo. Falla si la clave JWT es débil o los parámetros son inválidos.
func New(d Deps) (*Module, error) {
	d = d.withDefaults()
	params := security.Argon2ParamsDefault
	if d.Argon2 == Argon2Light {
		params = security.Argon2ParamsTest
	}
	hasher, err := security.NewArgon2Hasher(params)
	if err != nil {
		return nil, err
	}
	tokens, err := security.NewJWTIssuer(security.JWTConfig{Secret: d.JWTSecret, Issuer: d.JWTIssuer, Audience: d.JWTAudience})
	if err != nil {
		return nil, err
	}
	clock := security.SystemClock{}

	m := &Module{deps: d, hasher: hasher}

	// Lecturas y operaciones de una sola sentencia: sobre el pool, sin transacción explícita.
	reader := readadapter.NewSesionReadAdapter(d.DB)
	sesiones := writeadapter.NewSesionWriteAdapter(d.DB)

	m.autenticar = uquery.NewAutenticarToken(reader, sesiones, tokens, clock, d.Sesion)
	m.sesion = uquery.NewObtenerSesion()
	m.cerrar = command.NewCerrarSesion(sesiones, clock)

	// Escrituras con varias sentencias: cada ejecución construye sus adaptadores sobre el
	// mismo *sql.Tx.
	m.iniciar = &transactionalIniciarSesion{db: d.DB, logger: d.Logger, build: func(q txQuerier) *command.IniciarSesion {
		u := writeadapter.NewUsuarioWriteAdapter(q)
		return command.NewIniciarSesion(u, u, writeadapter.NewSesionWriteAdapter(q), hasher, tokens, clock, d.Bloqueo, d.Sesion)
	}}
	m.instalar = &transactionalInstalarAdministrador{db: d.DB, logger: d.Logger, build: func(q txQuerier) *command.InstalarAdministrador {
		return command.NewInstalarAdministrador(writeadapter.NewInstalacionWriteAdapter(q), writeadapter.NewUsuarioWriteAdapter(q), hasher)
	}}
	m.admin = armarAdministracion(d.DB, d.Logger, hasher, clock)
	return m, nil
}

// Authenticator expone la autenticación para el middleware de plataforma.
func (m *Module) Authenticator() platsec.Authenticator { return bearerAuthenticator{uc: m.autenticar} }

// RegisterRoutes monta las rutas del módulo sobre mux (relativas a /api/v1).
func (m *Module) RegisterRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	controller.RegisterAuthRoutes(mux, controller.NewAuthHandler(m.iniciar, m.cerrar, m.sesion), protect, m.deps.Logger)
	controller.RegisterAdminRoutes(mux, m.admin.roles, m.admin.permisos, m.admin.usuarios, protect, m.deps.Logger)
}

// bearerAuthenticator adapta el caso de uso AutenticarToken a platform/security.Authenticator.
type bearerAuthenticator struct{ uc inbound.AutenticarToken }

func (a bearerAuthenticator) Authenticate(r *http.Request) (appctx.Actor, error) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	token = strings.TrimSpace(token)
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return appctx.Actor{}, mapper.NoAutenticado()
	}
	res, err := a.uc.Execute(r.Context(), query.AutenticarToken{Token: token})
	if err != nil {
		return appctx.Actor{}, err
	}
	if !res.IsValid() {
		return appctx.Actor{}, errInvalidResult
	}
	if res.Err() != nil {
		return appctx.Actor{}, res.Err()
	}
	return res.Value().Actor, nil
}
