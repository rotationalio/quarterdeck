package backend

import (
	"context"
	"database/sql"

	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/txn"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/x/dsn"
)

//===========================================================================
// Store Implementations & Types
//===========================================================================

// Store embeds a [tidal.DB] and provides store operations with transaction support.
type Store struct {
	*tidal.DB
}

var _ txn.StoreTx = (*Store)(nil)

//===========================================================================
// Store Methods
//===========================================================================

func New(conn *tidal.DB) *Store {
	return &Store{DB: conn}
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) Stats() sql.DBStats {
	return s.DB.Stats()
}

//===========================================================================
// Transaction Methods
//===========================================================================

// When the database is opened read-only, opts must set ReadOnly or ErrReadOnly is returned.
func (s *Store) BeginTx(ctx context.Context, opts *sql.TxOptions) (txn.Tx, error) {
	tidalTx, err := s.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, tidalErr(err)
	}
	return &tx{ctx: ctx, store: s, tx: tidalTx, readOnly: opts != nil && opts.ReadOnly}, nil
}

func (s *Store) BeginReadTx(ctx context.Context) (txn.Tx, error) {
	tidalTx, err := s.DB.BeginReadTx(ctx)
	if err != nil {
		return nil, tidalErr(err)
	}
	return &tx{ctx: ctx, store: s, tx: tidalTx, readOnly: true}, nil
}

// When the database is opened read-only, opts must set ReadOnly or ErrReadOnly is returned.
func (s *Store) WithTx(ctx context.Context, opts *sql.TxOptions, fn func(txn.Tx) error) error {
	return s.DB.WithTx(ctx, opts, func(tidalTx tidal.Tx) error {
		return fn(&tx{ctx: ctx, store: s, tx: tidalTx, readOnly: opts != nil && opts.ReadOnly})
	})
}

func (s *Store) WithReadTx(ctx context.Context, fn func(txn.Tx) error) error {
	return s.DB.WithReadTx(ctx, func(tidalTx tidal.Tx) error {
		return fn(&tx{ctx: ctx, store: s, tx: tidalTx, readOnly: true})
	})
}

//===========================================================================
// Internal Transaction Implementation
//===========================================================================

type tx struct {
	ctx      context.Context
	store    *Store
	tx       tidal.Tx
	readOnly bool
}

var _ txn.Tx = (*tx)(nil)

func (t *tx) Context() context.Context {
	return t.ctx
}

func (t *tx) Commit() error {
	return t.tx.Commit()
}

func (t *tx) Rollback() error {
	return t.tx.Rollback()
}

// Checks that the transaction and database are not read-only.
func (t *tx) requireWrite() error {
	if t.readOnly || t.store.DSN().Options.ReadOnly() {
		return qerrors.ErrReadOnly
	}
	return nil
}

//===========================================================================
// Helpers
//===========================================================================

// list returns a cursor over models in a new read transaction. The cursor owns
// the transaction and must be closed by the caller.
func list[M tidal.Model](s *Store, ctx context.Context, crud *tidal.CRUD[M], filter tidal.ListFilter) (tidal.Cursor[M], error) {
	t, err := s.BeginReadTx(ctx)
	if err != nil {
		return nil, err
	}

	dt, ok := t.(*tx)
	if !ok {
		_ = t.Rollback()
		return nil, qerrors.ErrInternal
	}

	cursor, err := crud.List(dt.tx, filter)
	if err != nil {
		_ = t.Rollback()
		return nil, tidalErr(err)
	}

	return cursor, nil
}

// listInTx returns a cursor over models using the transaction provided.
func listInTx[M tidal.Model](t *tx, crud *tidal.CRUD[M], filter tidal.ListFilter) (tidal.Cursor[M], error) {
	return crud.List(t.tx, filter)
}

// retrieveBy retrieves a model by a single column name and value.
func retrieveBy[M tidal.Model](t *tx, crud *tidal.CRUD[M], column string, value any) (m M, err error) {
	m = tidal.Make[M]()
	query := crud.Queries.Retrieve + column + " = :" + column
	if err = m.Scan(tidal.Retrieve, t.tx.QueryRow(query, sql.Named(column, value))); err != nil {
		return m, tidalErr(err)
	}
	return m, nil
}

// captureInsertID sets the last inserted integer ID on a model; providers differ
// in how that value is obtained.
func captureInsertID(t *tx, result sql.Result, setID func(int64)) error {
	var id int64
	switch t.store.DSN().Provider {
	case dsn.Postgres:
		if err := t.tx.QueryRow("SELECT lastval()").Scan(&id); err != nil {
			return tidalErr(err)
		}
	default:
		var err error
		id, err = result.LastInsertId()
		if err != nil {
			return tidalErr(err)
		}
	}
	setID(id)
	return nil
}
