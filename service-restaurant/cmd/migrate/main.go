// Comando migrate: paso controlado para aplicar migraciones (no corre dentro de cada replica).
// Uso: migrate [up|down|status]   (DATABASE_URL obligatoria)
package main

import (
	"context"
	"fmt"
	"os"

	"service-restaurant/internal/platform/database"
	"service-restaurant/migrations"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL es obligatoria")
	}
	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	ctx := context.Background()
	db, err := database.Open(ctx, url)
	if err != nil {
		return err
	}
	defer db.Close()
	p, err := database.NewMigrator(db, migrations.FS)
	if err != nil {
		return err
	}
	switch cmd {
	case "up":
		res, err := p.Up(ctx)
		for _, r := range res {
			fmt.Println("aplicada:", r.Source.Path)
		}
		return err
	case "down":
		r, err := p.Down(ctx)
		if r != nil {
			fmt.Println("revertida:", r.Source.Path)
		}
		return err
	case "status":
		st, err := p.Status(ctx)
		for _, s := range st {
			fmt.Printf("%-8s %s\n", s.State, s.Source.Path)
		}
		return err
	}
	return fmt.Errorf("comando desconocido %q (up|down|status)", cmd)
}
