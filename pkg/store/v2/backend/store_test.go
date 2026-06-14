package backend_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	v2store "go.rtnl.ai/quarterdeck/pkg/store/v2"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/backend"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/suitetest"
	tsuite "go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/x/dsn"
)

// storeSuite runs integration tests against a real database with fixture data
// loaded into a v2 store.
type storeSuite struct {
	suitetest.BaseSuite
	store v2store.Store
}

//=============================================================================
// Suite Entry Points
//=============================================================================

// TestStoreSQLite runs the full store integration suite against SQLite.
func TestStoreSQLite(t *testing.T) {
	runStoreSuite(t, dsn.SQLite3, func(t *testing.T, s *storeSuite, m tsuite.Migrations) {
		suitetest.ConfigureSQLite(t, &s.DatabaseSuite, m)
	})
}

// TestStorePostgres runs the full store integration suite against Postgres.
func TestStorePostgres(t *testing.T) {
	runStoreSuite(t, dsn.Postgres, func(t *testing.T, s *storeSuite, m tsuite.Migrations) {
		suitetest.ConfigurePostgres(t, &s.DatabaseSuite, m)
	})
}

//=============================================================================
// Suite Lifecycle
//=============================================================================

func (s *storeSuite) SetupTest() {
	s.DatabaseSuite.SetupTest()
	s.openStore()
}

func (s *storeSuite) TearDownTest() {
	// The suite owns the database connection; do not close it via the store.
	s.store = nil
	suitetest.FinishTest(s.T(), &s.DatabaseSuite)
}

//=============================================================================
// Helpers
//=============================================================================

// runStoreSuite loads migrations, configures the provider, and runs suite tests.
func runStoreSuite(t *testing.T, provider string, configure func(*testing.T, *storeSuite, tsuite.Migrations)) {
	migrations, err := v2store.LoadMigrations(provider)
	require.NoError(t, err)

	s := &storeSuite{}
	configure(t, s, migrations)
	tsuite.Run(t, s)
}

// openStore constructs a store backed by the suite database and loads SQL fixtures.
func (s *storeSuite) openStore() {
	s.store = backend.New(s.DB)
	suitetest.LoadFixtures(s.T(), s.DB.DB, s.DSN().Provider)
}

// resetStore truncates tables, reapplies provider settings, and reloads fixtures.
// Used by tests that mutate junction data and need a clean slate mid-test.
func (s *storeSuite) resetStore() {
	s.store = nil
	suitetest.TruncateAndPrepare(s.T(), &s.DatabaseSuite)
	s.openStore()
}

// count returns the number of rows in a table via a read-only transaction.
func (s *storeSuite) count(table string) int {
	tx := s.BeginTx(&sql.TxOptions{ReadOnly: true})
	defer tx.Rollback()

	var count int
	s.Require().NoError(tx.QueryRow("SELECT count(*) FROM " + table).Scan(&count))
	s.Require().NoError(tx.Commit())
	return count
}
