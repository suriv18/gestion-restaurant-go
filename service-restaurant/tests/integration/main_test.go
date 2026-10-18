package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"service-restaurant/internal/platform/database"
	"service-restaurant/migrations"
)

// testDB es una base PostgreSQL 18 limpia con todas las migraciones aplicadas.
var testDB *sql.DB

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()
	ctr, err := tcpostgres.Run(ctx, "postgres:18",
		tcpostgres.WithDatabase("restaurant_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(90*time.Second)),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "no se pudo iniciar PostgreSQL (¿Docker activo?):", err)
		return 1
	}
	defer func() { _ = testcontainers.TerminateContainer(ctr) }()

	url, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	testDB, err = database.Open(ctx, url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer testDB.Close()

	if err := database.MigrateUp(ctx, testDB, migrations.FS); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return m.Run()
}
