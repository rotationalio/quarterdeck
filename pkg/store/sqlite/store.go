package sqlite

import (
	"context"
	"database/sql"
	"os"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/dsn"
	"go.rtnl.ai/quarterdeck/pkg/store/txn"

	"github.com/mattn/go-sqlite3"
)

// Store implements the store.Store interface using sqlite3 as the storage backend.
type Store struct {
	readonly bool
	conn     *sql.DB
}

// Tx implements the txn.Txn interface for sqlite3 transactions.
type Tx struct {
	tx   *sql.Tx
	opts *sql.TxOptions
}

func Open(uri *dsn.DSN) (_ *Store, err error) {
	// Ensure that only sqlite3 connections can be opened.
	if uri.Scheme != dsn.SQLite && uri.Scheme != dsn.SQLite3 {
		return nil, errors.ErrUnknownScheme
	}

	// Require a path in order to open the database connection (no in-memory databases).
	if uri.Path == "" {
		return nil, errors.ErrPathRequired
	}

	// Check if the database file exists, if it doesn't exist, it will be created and
	// all migrations will be applied to the database. Otherwise only the migrations
	// that have not been applied will be run.
	empty := false
	if _, err = os.Stat(uri.Path); os.IsNotExist(err) {
		empty = true
	}

	// Connect to the database
	s := &Store{readonly: uri.ReadOnly}
	if s.conn, err = sql.Open("sqlite3", uri.Path); err != nil {
		return nil, err
	}

	// Ping the database to establish the connection
	if err = s.conn.Ping(); err != nil {
		return nil, err
	}

	// Ensure that foreign key support is turned on by executing PRAGMA query.
	if _, err = s.conn.Exec("PRAGMA foreign_keys = on;"); err != nil {
		return nil, errors.Fmt("could not enable foreign key support: %w", err)
	}

	// Ensure the schema is initialized.
	if err = s.InitializeSchema(empty); err != nil {
		return nil, err
	}

	// Set the database to readonly mode after initializing the schema.
	if uri.ReadOnly {
		if _, err = s.conn.Exec("PRAGMA query_only = on;"); err != nil {
			return nil, errors.Fmt("could not set database to readonly mode: %w", err)
		}
	}

	return s, nil
}

//===========================================================================
// Store methods
//===========================================================================

func (s *Store) Close() error {
	return s.conn.Close()
}

func (s *Store) Begin(ctx context.Context, opts *sql.TxOptions) (txn.Txn, error) {
	return s.BeginTx(ctx, opts)
}

func (s *Store) BeginTx(ctx context.Context, opts *sql.TxOptions) (_ *Tx, err error) {
	// Ensure the options respect the readonly mode of the store.
	if opts == nil {
		opts = &sql.TxOptions{ReadOnly: s.readonly}
	} else if s.readonly && !opts.ReadOnly {
		return nil, errors.ErrReadOnly
	}

	var tx *sql.Tx
	if tx, err = s.conn.BeginTx(ctx, opts); err != nil {
		return nil, err
	}

	return &Tx{tx: tx, opts: opts}, nil
}

func (s *Store) Stats() sql.DBStats {
	return s.conn.Stats()
}

//===========================================================================
// Tx methods
//===========================================================================

func (t *Tx) Commit() error {
	return t.tx.Commit()
}

func (t *Tx) Rollback() error {
	return t.tx.Rollback()
}

func (t *Tx) Query(query string, args ...any) (*sql.Rows, error) {
	return t.tx.Query(query, args...)
}

func (t *Tx) QueryRow(query string, args ...any) *sql.Row {
	return t.tx.QueryRow(query, args...)
}

func (t *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return t.tx.Exec(query, args...)
}

// ===========================================================================
// Database Helpers
// ===========================================================================
func dbe(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return errors.ErrNotFound
	}

	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		if errors.Is(sqliteErr.Code, sqlite3.ErrReadonly) {
			return errors.ErrReadOnly
		}

		if errors.Is(sqliteErr.Code, sqlite3.ErrConstraint) && errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
			return errors.ErrAlreadyExists
		}
	}

	return errors.Fmt("sqlite3 error: %w", err)
}
