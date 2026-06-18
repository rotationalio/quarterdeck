package suitetest

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal"
	tsuite "go.rtnl.ai/tidal/suite"
	tfixtures "go.rtnl.ai/tidal/suite/fixtures"
	"go.rtnl.ai/x/dsn"
)

//============================================================================
// Suite
//============================================================================

// BaseSuite wraps tidal's DatabaseSuite and applies provider-specific settings
// after migrations. Embed this for store/v2 test suites instead of tidal's
// DatabaseSuite.
type BaseSuite struct {
	tsuite.DatabaseSuite
}

func (s *BaseSuite) SetupSuite() {
	s.DatabaseSuite.SetupSuite()
	require.NoError(s.T(), prepareDB(context.Background(), s.DB, s.DSN().Provider))
}

func (s *BaseSuite) TearDownTest() {
	s.DatabaseSuite.TearDownTest()

	if !s.ReadOnly() {
		require.NoError(s.T(), prepareDB(context.Background(), s.DB, s.DSN().Provider))
	}
}

//============================================================================
// Configuration
//============================================================================

// ConfigureSQLite prepares a DatabaseSuite for SQLite-backed tests.
func ConfigureSQLite(t *testing.T, s *tsuite.DatabaseSuite, migrations tsuite.Migrations) {
	t.Helper()
	s.Provider = &tsuite.SQLiteProvider{}
	s.Migrations = migrations
	s.Teardown = tsuite.TeardownTruncate
}

// ConfigurePostgres prepares a DatabaseSuite for Postgres-backed tests.
// Skips the test when Postgres is not configured.
func ConfigurePostgres(t *testing.T, s *tsuite.DatabaseSuite, migrations tsuite.Migrations) {
	t.Helper()
	s.Provider = &tsuite.PostgresProvider{}
	s.Migrations = migrations
	s.Teardown = tsuite.TeardownTruncate

	_, err := s.ResolveDSN("")
	if errors.Is(err, tsuite.ErrNoDatabaseURL) {
		t.Fatal("postgres not configured (set POSTGRES_DATABASE_URL, TEST_DATABASE_URL, TIDAL_DATABASE_URL, DATABASE_URL, or PGHOST)")
	}
	require.NoError(t, err)
}

//============================================================================
// Assertions
//============================================================================

// EqualTime compares two times after normalizing to UTC and truncating to
// second precision for DB round-trip checks, because anything larger than a
// second tends to fail in CI testing. This is not ideal, but it's a compromise
// that we can live with unless we want to use a sync-time library.
func EqualTime(tb testing.TB, expected, actual time.Time) {
	tb.Helper()
	areEqual := expected.UTC().Truncate(time.Second).Equal(actual.UTC().Truncate(time.Second))
	require.Truef(tb, areEqual, "times must be within second precision: %s != %s", expected, actual)
}

//============================================================================
// Test Lifecycle
//============================================================================

// TruncateAndPrepare clears all table data and reapplies provider-specific settings.
func TruncateAndPrepare(t testing.TB, s *tsuite.DatabaseSuite) {
	t.Helper()
	require := require.New(t)

	if !s.ReadOnly() {
		s.TruncateTables()
		require.NoError(prepareDB(context.Background(), s.DB, s.DSN().Provider))
	}
}

//============================================================================
// Fixtures
//============================================================================

// LoadFixtures executes shared SQL seed data for store v2 integration and
// conformance tests. All fixture files live under this package's testdata/
// directory (see testdata/README.md for IDs and relationships).
//
// Files are loaded from testdata/<provider>/ in sorted filename order
// (0001_permissions.sql, 0002_users.sql, …). Provider-specific SQL syntax is
// required: postgres/ uses decode(…, 'hex'); sqlite3/ uses x'…' literals.
func LoadFixtures(t testing.TB, db *tidal.DB, provider string) {
	t.Helper()
	require.NotNil(t, db)

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "resolve suitetest package path")
	pattern := filepath.Join(filepath.Dir(filename), "testdata", strings.ToLower(provider), "*.sql")
	fs, err := tfixtures.Glob(pattern)
	require.NoError(t, err)
	require.NotEmpty(t, fs, "no fixture SQL files matching %s", pattern)
	require.NoError(t, fs.Apply(context.Background(), db, "test"))
}

//============================================================================
// Database Preparation
//============================================================================

// prepareDB applies provider-specific connection settings after migrations.
func prepareDB(ctx context.Context, db *tidal.DB, provider string) error {
	switch provider {
	case dsn.SQLite3:
		_, err := db.ExecContext(ctx, "PRAGMA foreign_keys = on;")
		return err
	default:
		return nil
	}
}
