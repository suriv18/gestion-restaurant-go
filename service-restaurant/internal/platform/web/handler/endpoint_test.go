package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"service-restaurant/internal/platform/web/middleware"
	"service-restaurant/internal/shared/kernel/apperror"
)

func serve(t *testing.T, e Endpoint) *httptest.ResponseRecorder {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := middleware.RequestID(middleware.Recover(logger)(Adapt(logger, e)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	return rec
}

func TestAppErrorMapsToProblem(t *testing.T) {
	rec := serve(t, func(http.ResponseWriter, *http.Request) error {
		return apperror.New(apperror.KindConflict, "dup", "ya existe")
	})
	if rec.Code != http.StatusConflict || rec.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("got %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get(middleware.HeaderRequestID) == "" {
		t.Fatal("falta X-Request-ID")
	}
}

func TestUnknownErrorIsOpaque500(t *testing.T) {
	rec := serve(t, func(http.ResponseWriter, *http.Request) error { return errors.New("secreto de bd") })
	if rec.Code != 500 || strings.Contains(rec.Body.String(), "secreto") {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestPanicRecovered(t *testing.T) {
	rec := serve(t, func(http.ResponseWriter, *http.Request) error { panic("boom") })
	if rec.Code != 500 {
		t.Fatalf("got %d", rec.Code)
	}
}
