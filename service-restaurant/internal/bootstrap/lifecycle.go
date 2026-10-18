package bootstrap

import (
	"context"
	"errors"
	"net/http"
)

// Run sirve hasta que ctx se cancele y apaga ordenadamente con deadline.
func (a *App) Run(ctx context.Context) error {
	errc := make(chan error, 1)
	go func() {
		a.logger.Info("servidor escuchando", "addr", a.cfg.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
		close(errc)
	}()

	select {
	case err := <-errc:
		_ = a.db.Close()
		return err
	case <-ctx.Done():
	}

	a.logger.Info("apagando")
	sctx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
	defer cancel()
	err := a.server.Shutdown(sctx)
	if cerr := a.db.Close(); err == nil {
		err = cerr
	}
	return err
}

// Handler devuelve el handler HTTP completo (router y middleware), útil para pruebas.
func (a *App) Handler() http.Handler { return a.server.Handler }
