// Package migrations embebe la secuencia unica de migraciones Goose del despliegue.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
