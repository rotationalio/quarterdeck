package models_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/ulid"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/mock"
	. "go.rtnl.ai/quarterdeck/pkg/store/models"
)

var (
	modelID  = ulid.MustParse("01JYMS2J4X5XKFWCGKSX5G1JMK")
	created  = time.Date(2025, 4, 7, 12, 21, 33, 00, time.UTC)
	modified = time.Date(2025, 5, 8, 24, 42, 55, 00, time.UTC)
)

func TestUserParams(t *testing.T) {
	// This test ensures that user params are correctly returned in the correct orde
	// to prevent developer typos that may lead to hard to find bugs. It's annoying
	// because anytime you add a new field, you have to update this test, but it
	// will prevent headaches for you later on, I promise.
	user := &User{
		Model: Model{
			ID:       modelID,
			Created:  created,
			Modified: modified,
		},
		Name:      sql.NullString{Valid: true, String: "Carol King"},
		Email:     "cking@example.com",
		Password:  "$argon2id$v=19$m=65536,t=1,p=2$GCSPNYPRVwBT9E559vqOnQ==$QMiOdjzXvvyNiQid3G7WY6E2zprY00UI4xJDCbd1HkM=",
		LastLogin: sql.NullTime{Valid: false},
	}

	CheckParams(t, user.Params(),
		[]string{
			"id", "name", "email", "password", "lastLogin", "created", "modified",
		},
		[]any{
			user.ID, user.Name, user.Email, user.Password, user.LastLogin, user.Created, user.Modified,
		},
	)
}

func TestUserScan(t *testing.T) {
	t.Run("NotNull", func(t *testing.T) {
		data := []any{
			ulid.MakeSecure().String(), // ID
			"Greg Davies",              // Name
			"gdavies@example.com",      // Email
			"$argon2id$v=19$m=65536,t=1,p=2$GCSPNYPRVwBT9E559vqOnQ==$QMiOdjzXvvyNiQid3G7WY6E2zprY00UI4xJDCbd1HkM=", // Password
			time.Now().Add(-1 * time.Hour),  // LastLogin
			time.Now().Add(-14 * time.Hour), // Created
			time.Now().Add(-1 * time.Hour),  // Modified
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		model := &User{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.Name.String, "expected field Name to match data[1]")
		require.Equal(t, data[2], model.Email, "expected field Email to match data[2]")
		require.Equal(t, data[3], model.Password, "expected field Password to match data[3]")
		require.Equal(t, data[4], model.LastLogin.Time, "expected field LastLogin to match data[5]")
		require.Equal(t, data[5], model.Created, "expected field Created to match data[6]")
		require.Equal(t, data[6], model.Modified, "expected field Modified to match data[7]")
	})

	t.Run("Nulls", func(t *testing.T) {
		data := []any{
			ulid.MakeSecure().String(), // ID
			nil,                        // Name (testing null string)
			"email@example.com",        // Email
			"Password",                 // Password
			nil,                        // LastLogin (testing null time)
			time.Now(),                 // Created
			time.Time{},                // Modified (testing zero time)
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		//test
		model := &User{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		require.False(t, model.Name.Valid, "expected field Name to be invalid (null)")
		require.False(t, model.LastLogin.Valid, "expected field LastLogin to be invalid (null)")
		require.True(t, model.Modified.IsZero(), "expected field Modified to be zero time")
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		model := &User{}
		err := model.Scan(mockScanner)
		require.ErrorIs(t, err, ErrModelScan, "expected error when scanning with mock scanner")
	})
}

func TestUserScanSummary(t *testing.T) {
	t.Run("NotNull", func(t *testing.T) {
		data := []any{
			ulid.MakeSecure().String(),    // ID
			"First Last",                  // Name
			"email@example.com",           // Email
			time.Now(),                    // LastLogin
			time.Now(),                    // Created
			time.Now().Add(1 * time.Hour), // Modified
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		model := &User{}
		err := model.ScanSummary(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.Name.String, "expected field Name to match data[1]")
		require.Equal(t, data[2], model.Email, "expected field Email to match data[2]")
		require.Zero(t, model.Password, "important! password should be empty in summary scan")
		require.Equal(t, data[3], model.LastLogin.Time, "expected field LastLogin to match data[4]")
		require.Equal(t, data[4], model.Created, "expected field Created to match data[5]")
		require.Equal(t, data[5], model.Modified, "expected field Modified to match data[6]")
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		model := &User{}
		err := model.ScanSummary(mockScanner)
		require.ErrorIs(t, err, ErrModelScan, "expected error when scanning with mock scanner")
	})
}

func TestUserRoles(t *testing.T) {
	user := &User{
		Model: Model{
			ID:       modelID,
			Created:  created,
			Modified: modified,
		},
		Name:  sql.NullString{Valid: true, String: "Carol King"},
		Email: "cking@example.com",
	}

	_, err := user.Roles()
	require.ErrorIs(t, err, errors.ErrMissingAssociation, "expected error when accessing role before setting it")

	user.SetRoles([]*Role{{
		ID:          int64(410),
		Title:       "Observer",
		Description: "observer role to view the system",
		IsDefault:   true,
		Created:     created,
		Modified:    modified,
	}, {
		ID:          int64(411),
		Title:       "Editor",
		Description: "Editor role with limited permissions",
		IsDefault:   false,
		Created:     created,
		Modified:    modified,
	}})

	roles, err := user.Roles()
	require.NoError(t, err, "expected no error when accessing role after setting it")
	require.Len(t, roles, 2, "expected two roles to be set for the user")
}

func TestUserPermissions(t *testing.T) {
	user := &User{
		Model: Model{
			ID:       modelID,
			Created:  created,
			Modified: modified,
		},
		Name:  sql.NullString{Valid: true, String: "Carol King"},
		Email: "cking@example.com",
	}

	require.Empty(t, user.Permissions(), "expected no permissions before setting them")
	permissions := []string{"read", "write", "delete"}
	user.SetPermissions(permissions)
	require.Equal(t, permissions, user.Permissions())
}
