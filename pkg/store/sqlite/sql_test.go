package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	. "go.rtnl.ai/quarterdeck/pkg/store/sqlite"
	"go.rtnl.ai/quarterdeck/pkg/store/tests"
	"go.rtnl.ai/x/dsn"
)

func TestCRUD(t *testing.T) {
	// Create a blank Sqlite3 database in a temporary directory.
	path := filepath.Join(t.TempDir(), "quarterdeck_crud_test.db")
	uri := &dsn.DSN{Path: path, Provider: dsn.SQLite3}

	store, err := Open(uri)
	require.NoError(t, err, "could not open store")
	defer store.Close()

	t.Run("APIKeys", func(t *testing.T) {
		// Create a write transaction.
		tx, err := store.BeginTx(context.Background(), nil)
		require.NoError(t, err, "could not begin write transaction")
		defer tx.Rollback()

		// Create a user for the association with API keys.
		userFactory := &tests.UserFactory{}
		user := userFactory.Make()
		require.NoError(t, tx.CreateUser(user), "could not create user")

		t.Run("", tests.CRUDTests(tx, APIKeyCRUD, &tests.APIKeyFactory{CreatedBy: user.ID}))
	})
}

func TestFields(t *testing.T) {
	require := require.New(t)

	t.Run("APIKey", func(t *testing.T) {
		key := &models.APIKey{}
		require.Equal(key.Fields(models.List), Fields[*models.APIKey](models.List))
		require.Equal(key.Fields(models.Retrieve), Fields[*models.APIKey](models.Retrieve))
	})

	t.Run("OIDCClient", func(t *testing.T) {
		client := &models.OIDCClient{}
		require.Equal(client.Fields(models.List), Fields[*models.OIDCClient](models.List))
		require.Equal(client.Fields(models.Retrieve), Fields[*models.OIDCClient](models.Retrieve))
	})

	t.Run("Permission", func(t *testing.T) {
		permission := &models.Permission{}
		require.Equal(permission.Fields(models.List), Fields[*models.Permission](models.List))
		require.Equal(permission.Fields(models.Retrieve), Fields[*models.Permission](models.Retrieve))
	})

	t.Run("Role", func(t *testing.T) {
		role := &models.Role{}
		require.Equal(role.Fields(models.List), Fields[*models.Role](models.List))
		require.Equal(role.Fields(models.Retrieve), Fields[*models.Role](models.Retrieve))
	})

	t.Run("User", func(t *testing.T) {
		user := &models.User{}
		require.Equal(user.Fields(models.List), Fields[*models.User](models.List))
		require.Equal(user.Fields(models.Retrieve), Fields[*models.User](models.Retrieve))
	})

	t.Run("VeroToken", func(t *testing.T) {
		token := &models.VeroToken{}
		require.Equal(token.Fields(models.List), Fields[*models.VeroToken](models.List))
		require.Equal(token.Fields(models.Retrieve), Fields[*models.VeroToken](models.Retrieve))
	})
}
