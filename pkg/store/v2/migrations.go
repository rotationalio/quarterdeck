package store

import (
	"embed"
	"io/fs"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/tidal/migrations"
	"go.rtnl.ai/x/dsn"
)

// SQLiteMigrationFS embeds SQLite schema migrations applied at Open.
//
//go:embed migrations/sqlite/*.sql
var SQLiteMigrationFS embed.FS

// PostgresMigrationFS embeds Postgres schema migrations applied at Open.
//
//go:embed migrations/postgres/*.sql
var PostgresMigrationFS embed.FS

// LoadMigrations returns embedded migrations for provider (dsn.SQLite3 or
// dsn.Postgres).
func LoadMigrations(provider string) (migrations.Migrations, error) {
	var files fs.FS
	switch provider {
	case dsn.SQLite3:
		files = SQLiteMigrationFS
	case dsn.Postgres:
		files = PostgresMigrationFS
	default:
		return nil, errors.UnhandledProvider(provider)
	}
	return migrations.Load(files)
}
