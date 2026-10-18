// Package observability construye el logger estructurado.
package observability

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger devuelve un slog JSON con el nivel indicado.
func NewLogger(level string) *slog.Logger {
	var l slog.Level
	if err := l.UnmarshalText([]byte(strings.ToUpper(level))); err != nil {
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}
