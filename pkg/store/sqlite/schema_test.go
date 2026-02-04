package sqlite_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/store/sqlite"
)

func TestMigrations(t *testing.T) {
	// Add migrations here as they are created.
	expected := []*sqlite.Migration{
		{
			ID:   0,
			Name: "Migrations",
			Path: "0000_migrations.sql",
		},
		{
			ID:   1,
			Name: "Initial Schema",
			Path: "0001_initial_schema.sql",
		},
		{
			ID:   2,
			Name: "Oidc Clients",
			Path: "0002_oidc_clients.sql",
		},
	}

	migrations, err := sqlite.Migrations()
	require.NoError(t, err, "should have been able to load migrations")
	require.Equal(t, len(migrations), len(expected), "wrong number of migrations, has a migration been added?")

	for i, migration := range migrations {
		if i > len(expected) {
			break
		}

		require.Equal(t, expected[i].ID, migration.ID)
		require.Equal(t, expected[i].Name, migration.Name)
		require.Equal(t, expected[i].Path, migration.Path)

		query, err := migration.SQL()
		require.NoError(t, err, "could not load SQL from the migration")
		require.NotEmpty(t, query, "no SQL was returned for the migration")
	}
}
