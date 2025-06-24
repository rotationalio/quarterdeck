package store

import (
	"context"
	"database/sql"
	"io"

	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/dsn"
	"go.rtnl.ai/quarterdeck/pkg/store/mock"
	"go.rtnl.ai/quarterdeck/pkg/store/sqlite"
	"go.rtnl.ai/quarterdeck/pkg/store/txn"
)

// Open a directory storage provider with the specified URI. Database URLs should either
// specify protocol+transport://user:pass@host/dbname?opt1=a&opt2=b for servers or
// protocol:///relative/path/to/file for embedded databases (for absolute paths, specify
// protocol:////absolute/path/to/file).
func Open(conf config.DatabaseConfig) (s Store, err error) {
	var uri *dsn.DSN
	if uri, err = dsn.Parse(conf.URL); err != nil {
		return nil, err
	}

	// The configuration overrides any read-only setting in the DSN.
	uri.ReadOnly = conf.ReadOnly

	switch uri.Scheme {
	case dsn.Mock:
		return mock.Open(uri)
	case dsn.SQLite, dsn.SQLite3:
		return sqlite.Open(uri)
	default:
		return nil, errors.Fmt("unhandled database scheme %q", uri.Scheme)
	}
}

// Store is a generic storage interface allowing multiple storage backends such as
// SQLite or Postgres to be used based on the preference of the user.
// NOTE: to prevent import cycles, the txn.Tx interface is in its own package. If an
// interface is added to the Store interface, it should be added to the txn.Tx interface
// as well (to ensure the Txn has the same methods as the Store).
type Store interface {
	io.Closer

	Begin(context.Context, *sql.TxOptions) (txn.Txn, error)
}

// The Stats interface exposes database statistics if it is available from the backend.
type Stats interface {
	Stats() sql.DBStats
}
