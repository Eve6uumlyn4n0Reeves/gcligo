package migrations

import "embed"

//go:embed sql/*.sql
var sqlMigrations embed.FS
