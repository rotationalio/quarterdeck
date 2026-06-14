package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	db "go.rtnl.ai/quarterdeck/pkg/store/v2"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/suitetest"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/txn"
	tsuite "go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/dsn"
)

// openSuite exercises store.Open against a migrated test database.
type openSuite struct {
	suitetest.BaseSuite
}

//=============================================================================
// Migration Tests
//=============================================================================

// TestMigrationsSQLite verifies SQLite migrations load with expected IDs, names, and SQL.
func TestMigrationsSQLite(t *testing.T) {
	expectedMigrations := map[int]string{
		1: "Primary Schema",
	}
	testMigrations(t, dsn.SQLite3, expectedMigrations)
}

// TestMigrationsPostgres verifies Postgres migrations load with expected IDs, names, and SQL.
func TestMigrationsPostgres(t *testing.T) {
	expectedMigrations := map[int]string{
		1: "Primary Schema",
	}
	testMigrations(t, dsn.Postgres, expectedMigrations)
}

//=============================================================================
// Open Suite Entry Points
//=============================================================================

// TestOpenSQLite runs open/close and read-only checks against SQLite.
func TestOpenSQLite(t *testing.T) {
	runOpenSuite(t, dsn.SQLite3, func(t *testing.T, s *openSuite, m tsuite.Migrations) {
		suitetest.ConfigureSQLite(t, &s.DatabaseSuite, m)
	})
}

// TestOpenPostgres runs open/close and read-only checks against Postgres.
func TestOpenPostgres(t *testing.T) {
	runOpenSuite(t, dsn.Postgres, func(t *testing.T, s *openSuite, m tsuite.Migrations) {
		suitetest.ConfigurePostgres(t, &s.DatabaseSuite, m)
	})
}

//=============================================================================
// Open Suite Tests
//=============================================================================

// TestOpenClose verifies Open returns a store with stats and Close succeeds.
func (s *openSuite) TestOpenClose() {
	uri := s.DSN()
	st, err := db.Open(config.DatabaseConfig{URL: uri.String()})
	s.Require().NoError(err)
	s.Require().NotNil(st.Stats())
	s.Require().NoError(st.Close())
}

// TestReadOnlyStore verifies read-only mode is enforced on every transaction entry
// point and on writes inside read-only transactions.
func (s *openSuite) TestReadOnlyStore() {
	// --- SETUP ---

	// Get a database URI for the test suite.
	ctx := context.Background()
	uri := s.DSN()
	url := uri.String()

	// Open a regular (read-write) connection and create a user to ensure the
	// DB isn't empty, then close it.
	rw, err := db.Open(config.DatabaseConfig{URL: url})
	s.Require().NoError(err)
	created, err := rw.CreateUser(ctx, &models.User{Email: "a@b.com", Password: "x"})
	s.Require().NoError(err)
	s.Require().NoError(rw.Close())

	// Open a new store in read-only mode for subsequent read-only enforcement
	// checks.
	ro, err := db.Open(config.DatabaseConfig{URL: url, ReadOnly: true})
	s.Require().NoError(err)
	defer ro.Close()

	// --- TESTS ---

	// Store methods that write delegate to WithTx; they must fail before hitting SQL.
	s.Run("StoreWrite", func() {
		_, err := ro.CreateUser(ctx, &models.User{Email: "b@b.com", Password: "x"})
		s.Require().ErrorIs(err, errors.ErrReadOnly)
	})

	// RW transactions must be rejected at open time, not only when a write runs.
	s.Run("BeginTx", func() {
		_, err := ro.BeginTx(ctx, nil)
		s.Require().ErrorIs(err, errors.ErrReadOnly)
	})

	// WithTx without ReadOnly opts should fail the same way as BeginTx.
	s.Run("WithTx", func() {
		err := ro.WithTx(ctx, nil, func(tx txn.Tx) error {
			return tx.UpdateUser(&models.User{Email: "a@b.com"})
		})
		s.Require().ErrorIs(err, errors.ErrReadOnly)
	})

	// Read-only txs are allowed, but writes inside them still hit requireWrite.
	s.Run("BeginReadTx", func() {
		tx, err := ro.BeginReadTx(ctx)
		s.Require().NoError(err)
		defer tx.Rollback()

		user, err := tx.RetrieveUser(created.ID)
		s.Require().NoError(err)
		s.Require().Equal(created.Email, user.Email)

		err = tx.UpdateUser(user)
		s.Require().ErrorIs(err, errors.ErrReadOnly)
	})

	// WithReadTx must propagate ErrReadOnly when the callback attempts a write.
	s.Run("WithReadTx", func() {
		err := ro.WithReadTx(ctx, func(tx txn.Tx) error {
			user, err := tx.RetrieveUserByEmail(created.Email)
			if err != nil {
				return err
			}
			return tx.VerifyEmail(user.ID)
		})
		s.Require().ErrorIs(err, errors.ErrReadOnly)
	})

	// A read-only callback that only reads should complete without error.
	s.Run("WithReadTxRead", func() {
		err := ro.WithReadTx(ctx, func(tx txn.Tx) error {
			_, err := tx.RetrieveUser(created.ID)
			return err
		})
		s.Require().NoError(err)
	})

	// List operations use read-only txs internally and should still work.
	s.Run("ListUsers", func() {
		cursor, err := ro.ListUsers(ctx, nil)
		s.Require().NoError(err)
		defer cursor.Close()

		var count int
		for cursor.Next() {
			count++
		}
		s.Require().NoError(cursor.Err())
		s.Require().Positive(count)
	})
}

// TestWritableTransactions verifies read-write transaction entry points commit work.
func (s *openSuite) TestWritableTransactions() {
	ctx := context.Background()
	uri := s.DSN()
	st, err := db.Open(config.DatabaseConfig{URL: uri.String()})
	s.Require().NoError(err)
	defer st.Close()

	var userID ulid.ULID
	err = st.WithTx(ctx, nil, func(tx txn.Tx) error {
		user, err := tx.CreateUser(&models.User{Email: "tx@b.com", Password: "x"})
		if err != nil {
			return err
		}
		userID = user.ID
		return nil
	})
	s.Require().NoError(err)

	tx, err := st.BeginTx(ctx, nil)
	s.Require().NoError(err)
	s.Require().NoError(tx.VerifyEmail(userID))
	s.Require().NoError(tx.Commit())

	user, err := st.RetrieveUser(ctx, userID)
	s.Require().NoError(err)
	s.Require().True(user.EmailVerified)
}

//=============================================================================
// Helpers
//=============================================================================

// testMigrations asserts each migration has a non-empty SQL body and expected metadata.
func testMigrations(t *testing.T, provider string, expectedMigrations map[int]string) {
	m, err := db.LoadMigrations(provider)
	require.NoError(t, err)

	require.Len(t, m, len(expectedMigrations), "should have loaded the correct number of migrations")
	for _, mig := range m {
		expectedName, ok := expectedMigrations[mig.ID]
		require.Truef(t, ok, "unexpected migration ID: %d", mig.ID)
		require.Equalf(t, expectedName, mig.Name, "migration name for ID %d should match", mig.ID)

		sql, err := mig.SQL()
		require.NoErrorf(t, err, "should have been able to load the migration SQL for migration %d", mig.ID)
		require.NotEmptyf(t, sql, "migration SQL for migration %d should not be empty", mig.ID)
	}
}

// runOpenSuite loads migrations, configures the provider, and runs open-suite tests.
func runOpenSuite(t *testing.T, provider string, configure func(*testing.T, *openSuite, tsuite.Migrations)) {
	migrations, err := db.LoadMigrations(provider)
	require.NoError(t, err)

	s := &openSuite{}
	configure(t, s, migrations)
	tsuite.Run(t, s)
}
