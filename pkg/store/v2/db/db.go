package db

import (
	"context"
	"database/sql"

	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/x/dsn"
)

// Embeds a [tidal.DB] and provides helper methods for local database operations.
type DB struct {
	*tidal.DB
}

func New(db *tidal.DB) *DB {
	return &DB{DB: db}
}

func (d *DB) Close() error {
	return d.DB.Close()
}

func (d *DB) Stats() sql.DBStats {
	return d.DB.Stats()
}

func (d *DB) beginTx(ctx context.Context, opts *sql.TxOptions) (tidal.Tx, error) {
	// Ensure that readonly mode is respected.
	if d.DSN().Options.ReadOnly() && (opts == nil || !opts.ReadOnly) {
		return nil, qerrors.ErrReadOnly
	}

	tx, err := d.BeginTx(ctx, opts)
	if err != nil {
		return nil, tidalErr(err)
	}

	return tx, nil
}

func (d *DB) withTx(ctx context.Context, opts *sql.TxOptions, fn func(tidal.Tx) error) error {
	tx, err := d.beginTx(ctx, opts)
	if err != nil {
		return err
	}

	if err = fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DB) withReadTx(ctx context.Context, fn func(tidal.Tx) error) error {
	return d.withTx(ctx, &sql.TxOptions{ReadOnly: true}, fn)
}

// Helper method to list models from the database. The returned cursor owns the
// transaction and the user must call Close() when done.
func list[M tidal.Model](d *DB, ctx context.Context, crud *tidal.CRUD[M], filter tidal.ListFilter) (tidal.Cursor[M], error) {
	// Cannot use withReadTx: the cursor owns this transaciton and so it must
	// stay open after this function returns; the user must call Close() when
	// done.
	tx, err := d.beginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}

	cursor, err := crud.List(tx, filter)
	if err != nil {
		_ = tx.Rollback()
		return nil, tidalErr(err)
	}

	return cursor, nil
}

// Helper method to capture the last inserted ID from the database and set it on
// the model; different providers use different methods to capture the last
// inserted ID.
func captureInsertID(tx tidal.Tx, db *DB, result sql.Result, setID func(int64)) (err error) {
	var id int64
	switch db.DSN().Provider {
	case dsn.Postgres:
		if err = tx.QueryRow("SELECT lastval()").Scan(&id); err != nil {
			return tidalErr(err)
		}
	default:
		id, err = result.LastInsertId()
		if err != nil {
			return tidalErr(err)
		}
	}
	setID(id)
	return nil
}
