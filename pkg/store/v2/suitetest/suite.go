package suitetest

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal"
	tsuite "go.rtnl.ai/tidal/suite"
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

//============================================================================
// Configuration
//============================================================================

// ConfigureSQLite prepares a DatabaseSuite for SQLite-backed tests.
func ConfigureSQLite(t *testing.T, s *tsuite.DatabaseSuite, migrations tsuite.Migrations) {
	t.Helper()
	s.Provider = &tsuite.SQLiteProvider{}
	s.Migrations = migrations
}

// ConfigurePostgres prepares a DatabaseSuite for Postgres-backed tests.
// Skips the test when Postgres is not configured.
func ConfigurePostgres(t *testing.T, s *tsuite.DatabaseSuite, migrations tsuite.Migrations) {
	t.Helper()
	s.Provider = &tsuite.PostgresProvider{}
	s.Migrations = migrations

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
// microsecond precision, matching Postgres timestamp resolution.
func EqualTime(tb testing.TB, expected, actual time.Time) {
	tb.Helper()
	areEqual := expected.Truncate(time.Microsecond).Equal(actual.Truncate(time.Microsecond))
	require.True(tb, areEqual, "times must be within microsecond precision")
}

//============================================================================
// Test Lifecycle
//============================================================================

// FinishTest truncates table data and cancels the per-test context without the
// expensive DropTables+Migrate cycle in tidal's DatabaseSuite.TearDownTest.
func FinishTest(t testing.TB, s *tsuite.DatabaseSuite) {
	t.Helper()

	TruncateAndPrepare(t, s)
	s.TearDownSubTest()
}

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
//
// Fixtures run on a raw [sql.DB] so literal timestamps (e.g. T11:21:42) are not
// rewritten as tidal named bind parameters.
func LoadFixtures(t testing.TB, db *sql.DB, provider string) {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "resolve suitetest package path")
	dir := filepath.Join(filepath.Dir(filename), "testdata", strings.ToLower(provider))
	paths, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, paths, "no fixture SQL files in %s", dir)
	sort.Strings(paths)

	for _, path := range paths {
		stmt, err := os.ReadFile(path)
		require.NoError(t, err)
		_, err = db.Exec(string(stmt))
		require.NoError(t, err, "fixture %s", path)
	}
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
