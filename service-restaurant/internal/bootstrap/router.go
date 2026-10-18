package bootstrap

import (
	"net/http"

	"service-restaurant/internal/platform/web/handler"
	"service-restaurant/internal/platform/web/middleware"
	"service-restaurant/internal/platform/web/response"
)

func (a *App) router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("GET /readyz", handler.Adapt(a.logger, func(w http.ResponseWriter, r *http.Request) error {
		if err := a.db.PingContext(r.Context()); err != nil {
			return err
		}
		response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
		return nil
	}))
	// Los módulos registran sus rutas aquí al incorporarse (infrastructure/config/module.go).

	var h http.Handler = mux
	h = http.MaxBytesHandler(h, a.cfg.MaxBodyBytes)
	h = middleware.Recover(a.logger)(h)
	h = middleware.RequestID(h)
	return h
}
