package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"service-restaurant/internal/platform/web/response"
)

// Recover captura panics de la goroutine de la solicitud y responde 500.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					rid := RequestIDFrom(r.Context())
					logger.Error("panic en solicitud", "panic", rec, "request_id", rid, "stack", string(debug.Stack()))
					response.WriteProblem(w, response.Problem{
						Status: http.StatusInternalServerError, Detail: "error interno", RequestID: rid,
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
