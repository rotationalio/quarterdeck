package models_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	store "go.rtnl.ai/quarterdeck/pkg/store/v2"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/suitetest"
	tsuite "go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/dsn"
)

// modelSuite runs model conformance tests against a migrated database.
type modelSuite struct {
	suitetest.BaseSuite
}

//=============================================================================
// Fixtures / Constants
//=============================================================================

// the admin user from suitetest/testdata (created_by FK parent for all fixtures).
var fixtureAdminUserID = ulid.MustParse("01JN2YQ2VE9GMBRVACD15J1TFX")

// fixture constants for model unit tests.
var (
	modelID  = ulid.MustParse("01JYMS2J4X5XKFWCGKSX5G1JMK")
	created  = time.Date(2025, 4, 7, 12, 21, 33, 0, time.UTC)
	modified = time.Date(2025, 5, 8, 24, 42, 55, 0, time.UTC)
)

// fixture error for model unit tests.
var ErrModelScan = errors.New("test scan error")

//=============================================================================
// Suite Entry Points
//=============================================================================

// TestModelsSQLite runs junction CRUD conformance tests against SQLite.
func TestModelsSQLite(t *testing.T) {
	runModelSuite(t, dsn.SQLite3, func(t *testing.T, s *modelSuite, m tsuite.Migrations) {
		suitetest.ConfigureSQLite(t, &s.DatabaseSuite, m)
	})
}

// TestModelsPostgres runs junction CRUD conformance tests against Postgres.
func TestModelsPostgres(t *testing.T) {
	runModelSuite(t, dsn.Postgres, func(t *testing.T, s *modelSuite, m tsuite.Migrations) {
		suitetest.ConfigurePostgres(t, &s.DatabaseSuite, m)
	})
}

//=============================================================================
// Suite Lifecycle
//=============================================================================

func (s *modelSuite) SetupTest() {
	s.DatabaseSuite.SetupTest()
	suitetest.LoadFixtures(s.T(), s.DB, s.DSN().Provider)
}

//=============================================================================
// Helpers
//=============================================================================

// runModelSuite loads migrations, configures the provider, and runs model suite tests.
func runModelSuite(t *testing.T, provider string, configure func(*testing.T, *modelSuite, tsuite.Migrations)) {
	migrations, err := store.LoadMigrations(provider)
	require.NoError(t, err)

	s := &modelSuite{}
	configure(t, s, migrations)
	tsuite.Run(t, s)
}
