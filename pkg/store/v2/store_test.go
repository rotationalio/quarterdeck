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
	tsuite "go.rtnl.ai/tidal/suite"
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

// TestReadOnlyStore verifies writes fail after opening with ReadOnly config.
func (s *openSuite) TestReadOnlyStore() {
	uri := s.DSN()
	url := uri.String()

	// Setup: open writable store and seed a user.
	rw, err := db.Open(config.DatabaseConfig{URL: url})
	s.Require().NoError(err)

	_, err = rw.CreateUser(context.Background(), &models.User{Email: "a@b.com", Password: "x"})
	s.Require().NoError(err, "should be able to create a user in writeable mode")
	s.Require().NoError(rw.Close())

	// Action: reopen read-only and attempt another write.
	st, err := db.Open(config.DatabaseConfig{URL: url, ReadOnly: true})
	s.Require().NoError(err)

	_, err = st.CreateUser(context.Background(), &models.User{Email: "a@b.com", Password: "x"})
	// Assert: write is rejected.
	s.Require().ErrorIs(err, errors.ErrReadOnly, "should not be able to create a user in read-only mode")
	s.Require().NoError(st.Close())
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
