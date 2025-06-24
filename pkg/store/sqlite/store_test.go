package sqlite_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/dsn"
	db "go.rtnl.ai/quarterdeck/pkg/store/sqlite"
)

// ===========================================================================
// Top Level Tests
// ===========================================================================
func TestConnectClose(t *testing.T) {
	t.Run("ReadWrite", func(t *testing.T) {
		uri, _ := dsn.Parse("sqlite3:///" + filepath.Join(t.TempDir(), "test.db"))

		store, err := db.Open(uri)
		require.NoError(t, err, "could not open connection to temporary sqlite database")

		tx, err := store.BeginTx(context.Background(), nil)
		require.NoError(t, err, "could not create write transaction")
		tx.Rollback()

		tx, err = store.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
		require.NoError(t, err, "could not create readonly transaction")
		tx.Rollback()

		err = store.Close()
		require.NoError(t, err, "should be able to close the db without error when not connected")
	})

	t.Run("ReadOnly", func(t *testing.T) {
		uri, _ := dsn.Parse("sqlite3:///" + filepath.Join(t.TempDir(), "test.db") + "?readonly=true")

		store, err := db.Open(uri)
		require.NoError(t, err, "could not open connection to temporary sqlite database")

		tx, err := store.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: false})
		require.ErrorIs(t, err, errors.ErrReadOnly, "created write transaction in readonly mode")
		require.Nil(t, tx, "expected no transaction to be returned")

		tx, err = store.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
		require.NoError(t, err, "could not create readonly transaction")
		tx.Rollback()

		err = store.Close()
		require.NoError(t, err, "should be able to close the db without error when not connected")
	})

	t.Run("Failures", func(t *testing.T) {
		tests := []struct {
			uri *dsn.DSN
			err error
		}{
			{
				&dsn.DSN{Scheme: "leveldb"},
				errors.ErrUnknownScheme,
			},
			{
				&dsn.DSN{Scheme: "sqlite3"},
				errors.ErrPathRequired,
			},
		}

		for i, tc := range tests {
			_, err := db.Open(tc.uri)
			require.ErrorIs(t, err, tc.err, "test case %d failed", i)
		}
	})
}

//===========================================================================
// Store Test Suite
//===========================================================================

type storeTestSuite struct {
	suite.Suite
	dsn *dsn.DSN
	db  *db.Store
}

func (s *storeTestSuite) SetupSuite() {
	s.CreateDB()
}

func (s *storeTestSuite) AfterTest(suiteName, testName string) {
	s.ResetDB()
}

func (s *storeTestSuite) CreateDB() {
	var err error
	require := s.Require()

	// Only create the database path on the first call to CreateDB. Otherwise the call
	// to TempDir() will be prefixed with the name of the subtest, which will cause an
	// "attempt to write a read-only database" for subsequent tests because the directory
	// will be deleted when the subtest is complete.
	if s.dsn.Path == "" {
		s.dsn.Path = filepath.Join(s.T().TempDir(), "quarterdeck.db")
	}

	s.db, err = db.Open(s.dsn)
	require.NoError(err, "could not open store in temporary location")

	// Execute any SQL files in the testdata directory
	paths, err := filepath.Glob("testdata/*.sql")
	require.NoError(err, "could not list testdata directory")

	tx, err := s.db.BeginTx(context.Background(), nil)
	require.NoError(err, "could not open transaction")
	defer tx.Rollback()

	for _, path := range paths {
		stmt, err := os.ReadFile(path)
		require.NoError(err, "could not read query from file")

		_, err = tx.Exec(string(stmt))
		require.NoError(err, "could not execute sql query from fixture %s", path)
	}

	require.NoError(tx.Commit(), "could not commit transaction")
}

func (s *storeTestSuite) ResetDB() {
	require := s.Require()
	require.NoError(s.db.Close(), "could not close connection to db")
	require.NoError(os.Remove(s.dsn.Path), "could not delete old database")
	s.CreateDB()
}

func (s *storeTestSuite) ReadOnly() bool {
	return s.dsn.ReadOnly
}

func TestStore(t *testing.T) {
	suite.Run(t, &storeTestSuite{dsn: &dsn.DSN{ReadOnly: false, Scheme: "sqlite3"}})
}

func TestReadOnlyStore(t *testing.T) {
	suite.Run(t, &storeTestSuite{dsn: &dsn.DSN{ReadOnly: true, Scheme: "sqlite3"}})
}
