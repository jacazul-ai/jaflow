package sqlite

import "embed"

// migrationFS contains the native Jaflow schema migrations.
//
//go:embed migrations/*.sql
var embeddedMigrations embed.FS
