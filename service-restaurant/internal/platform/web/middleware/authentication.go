package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/security"
	"service-restaurant/internal/platform/web/response"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type actorKey struct{}

// ActorFrom devuelve el actor autenticado de la solicitud.
func ActorFrom(ctx context.Context) (appctx.Actor, bool) {
	a, ok := ctx.Value(actorKey{}).(appctx.Actor)
	return a, ok && !a.IsZero()
}

// Authenticate exige un actor verificado; si falla responde 401 uniforme.
// Un error que no es de autenticación se registra y responde 500 opaco.
func Authenticate(a security.Authenticator, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, err := a.Authenticate(r)
			if err != nil {
				rid := RequestIDFrom(r.Context())
				var ae *apperror.AppError
				if errors.As(err, &ae) && ae.Kind == apperror.KindUnauthorized {
					w.Header().Set("WWW-Authenticate", "Bearer")
					response.WriteProblem(w, response.Problem{
						Status: http.StatusUnauthorized, Detail: ae.Message, Code: ae.Code, RequestID: rid,
					})
					return
				}
				logger.Error("autenticación falló por error técnico", "error", err, "request_id", rid)
				response.WriteProblem(w, response.Problem{
					Status: http.StatusInternalServerError, Detail: "error interno", RequestID: rid,
				})
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, actor)))
		})
	}
}
