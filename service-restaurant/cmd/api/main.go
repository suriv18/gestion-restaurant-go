package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"service-restaurant/internal/bootstrap"
	"service-restaurant/internal/platform/config"
	"service-restaurant/internal/platform/observability"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := observability.NewLogger(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := bootstrap.New(ctx, cfg, logger)
	if err != nil {
		return err
	}
	return app.Run(ctx)
}
