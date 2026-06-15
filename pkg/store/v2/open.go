package store

import (
	"context"

	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/backend"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/mock"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/x/dsn"
)

// Open connects to the storage backend described by conf, applies migrations for
// SQL backends, and returns a [Store].
func Open(conf config.DatabaseConfig) (Store, error) {
	uri, err := dsn.Parse(conf.URL)
	if err != nil {
		return nil, errors.Join(errors.ErrDSNParse, err)
	}

	if conf.ReadOnly {
		if uri.Options == nil {
			uri.Options = dsn.Options{}
		}
		uri.Options[dsn.ReadOnly] = "true"
	}

	switch uri.Provider {
	case dsn.Mock:
		return mock.Open(uri)
	case dsn.SQLite3, dsn.Postgres:
		return openTidal(context.Background(), uri)
	default:
		return nil, errors.UnhandledProvider(uri.Provider)
	}
}

// openTidal opens a tidal connection, applies provider migrations, and enables
// SQLite foreign keys when required.
func openTidal(ctx context.Context, uri *dsn.DSN) (Store, error) {
	// Postgres does not support read-only connections, so we need to remove the
	// read-only option from the connection URI and then wrap the connection
	// with the original URI.
	connectURI := uri
	if uri.Provider == dsn.Postgres && uri.Options.ReadOnly() {
		connectURI = uri.Clone()
		delete(connectURI.Options, dsn.ReadOnly)
	}

	conn, err := tidal.Open(ctx, connectURI)
	if err != nil {
		return nil, err
	}

	// If the connection URI was modified, wrap the connection with the original
	// URI so that readonly mode is preserved (see backend/backend.go for details on how
	// database read-only mode is protected).
	if connectURI != uri {
		conn = tidal.Wrap(conn.DB, uri)
	}

	migrations, err := LoadMigrations(uri.Provider)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err = migrations.Apply(ctx, conn, pkg.Version(true)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	// SQLite-specific configurations.
	if uri.Provider == dsn.SQLite3 {
		// Require a path in order to open the database connection (no in-memory
		// databases).
		if uri.Path == "" {
			_ = conn.Close()
			return nil, errors.ErrPathRequired
		}

		// Enable SQLite foreign keys.
		if _, err = conn.ExecContext(ctx, "PRAGMA foreign_keys = on;"); err != nil {
			_ = conn.Close()
			return nil, errors.Join(errors.ErrSQLiteForeignKeys, err)
		}

		// Set the database to readonly mode; must happen after migrations are
		// applied.
		if uri.Options.ReadOnly() {
			if _, err = conn.ExecContext(ctx, "PRAGMA query_only = on;"); err != nil {
				_ = conn.Close()
				return nil, errors.Join(errors.ErrSQLiteQueryOnly, err)
			}
		}
	}

	return backend.New(conn), nil
}
